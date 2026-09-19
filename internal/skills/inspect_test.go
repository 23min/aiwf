package skills

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestInspectArtifacts_MatchesMaterializationAndAgentTiers(t *testing.T) {
	t.Parallel()
	for _, target := range []Target{ClaudeTarget, CodexTarget()} {
		t.Run(target.Name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			tiers := map[string]AgentTier{"builder": {Model: "sonnet", Effort: "high"}, "unknown": {Model: "opus"}}
			if err := materializeTo(root, target, tiers); err != nil {
				t.Fatal(err)
			}
			statuses, err := InspectArtifacts(context.Background(), root, target, tiers)
			if err != nil {
				t.Fatal(err)
			}
			counts := map[ArtifactFamily]int{}
			for _, status := range statuses {
				counts[status.Family]++
				if status.State != ArtifactCurrent {
					t.Errorf("fresh output: %+v", status)
				}
			}
			for _, family := range []ArtifactFamily{FamilySkills, FamilyRituals, FamilyTemplates} {
				if counts[family] == 0 {
					t.Errorf("missing family %s", family)
				}
			}
			if (counts[FamilyAgents] > 0) != (target.AgentsDir != "") {
				t.Errorf("agent family mismatch: %v", counts)
			}
		})
	}
}

func TestInspectArtifacts_RejectsCanceledAndInvalidRendering(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := InspectArtifacts(ctx, t.TempDir(), ClaudeTarget, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled = %v", err)
	}
	if _, err := InspectArtifacts(context.Background(), t.TempDir(), Target{}, nil); !errors.Is(err, ErrMissingRenderBinding) {
		t.Fatalf("invalid target = %v", err)
	}
}

func TestInspectArtifact_DiskStates(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		want ArtifactState
	}{
		{"missing", ArtifactMissing},
		{"current", ArtifactCurrent},
		{"drifted", ArtifactDrifted},
		{"directory", ArtifactBlocked},
		{"symlink", ArtifactBlocked},
		{"parent link", ArtifactBlocked},
		{"unreadable", ArtifactBlocked},
		{"invalid path", ArtifactBlocked},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			relative := "artifact"
			path := filepath.Join(root, relative)
			switch tc.name {
			case "current", "drifted", "unreadable":
				content := []byte("expected")
				if tc.name == "drifted" {
					content = []byte("different")
				}
				mode := os.FileMode(0o644)
				if tc.name == "unreadable" {
					mode = 0
				}
				if err := os.WriteFile(path, content, mode); err != nil {
					t.Fatal(err)
				}
				if tc.name == "unreadable" {
					if _, err := os.ReadFile(path); err == nil {
						t.Skip("process bypasses permissions")
					}
				}
			case "directory":
				if err := os.Mkdir(path, 0o755); err != nil {
					t.Fatal(err)
				}
			case "symlink", "parent link":
				if err := os.Symlink(t.TempDir(), path); err != nil {
					t.Fatal(err)
				}
				if tc.name == "parent link" {
					relative += "/file"
				}
			case "invalid path":
				relative = "../outside"
			}
			status := InspectArtifact(context.Background(), root, FamilyTemplates, relative, []byte("expected"))
			if status.State != tc.want || status.Path != relative || status.Family != FamilyTemplates {
				t.Fatalf("got %+v, want %s", status, tc.want)
			}
		})
	}
}

func TestInspectArtifacts_RejectsIncompatibleProvenancePaths(t *testing.T) {
	t.Parallel()
	for _, family := range []string{"templates", "agents"} {
		t.Run(family, func(t *testing.T) {
			t.Parallel()
			target := ClaudeTarget
			if family == "templates" {
				target.TemplatesDir = t.TempDir()
			} else {
				target.AgentsDir = t.TempDir()
			}
			root := t.TempDir()
			if statuses, err := InspectArtifacts(context.Background(), root, target, nil); err == nil || statuses != nil {
				t.Fatalf("invalid layout accepted: %+v %v", statuses, err)
			}
			entries, err := os.ReadDir(root)
			if err != nil || len(entries) != 0 {
				t.Fatalf("inspection wrote files: %v %v", entries, err)
			}
		})
	}
}

func TestInspectArtifacts_EmptyAndPartiallyMissingTrees(t *testing.T) {
	t.Parallel()
	for _, target := range []Target{ClaudeTarget, CodexTarget(), {Name: "noagents", SkillsDir: ".x/skills", TemplatesDir: ".x/templates"}} {
		t.Run(target.Name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			statuses, inspectErr := InspectArtifacts(context.Background(), root, target, nil)
			if inspectErr != nil {
				t.Fatal(inspectErr)
			}
			if len(statuses) == 0 {
				t.Fatal("empty tree has no expected artifacts")
			}
			for _, status := range statuses {
				if status.State != ArtifactMissing {
					t.Errorf("empty tree: %+v", status)
				}
			}
			if err := MaterializeTo(root, target); err != nil {
				t.Fatal(err)
			}
			relative := filepath.Join(target.TemplatesDir, "epic-spec.md")
			if target.AgentsDir != "" {
				relative = filepath.Join(target.AgentsDir, "planner.md")
			}
			if err := os.Remove(filepath.Join(root, relative)); err != nil {
				t.Fatal(err)
			}
			statuses, inspectErr = InspectArtifacts(context.Background(), root, target, nil)
			if inspectErr != nil {
				t.Fatal(inspectErr)
			}
			missing := 0
			for _, status := range statuses {
				if status.Path == relative {
					missing++
					if status.State != ArtifactMissing {
						t.Errorf("removed artifact: %+v", status)
					}
				} else if status.State != ArtifactCurrent {
					t.Errorf("unaffected artifact: %+v", status)
				}
				if target.AgentsDir == "" && status.Family == FamilyAgents {
					t.Errorf("unsupported agent family: %+v", status)
				}
			}
			if missing != 1 {
				t.Fatalf("removed artifact reported %d times", missing)
			}
		})
	}
}
