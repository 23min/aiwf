package initrepo

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/skills"
)

func TestInstructionWriters_PreserveSymlinksAndTargets(t *testing.T) {
	t.Parallel()
	for _, writer := range []string{"scaffold", "claude", "codex"} {
		for _, kind := range []string{"ordinary", "external", "broken", "loop"} {
			for _, dryRun := range []bool{false, true} {
				t.Run(writer+"/"+kind+"/"+map[bool]string{false: "write", true: "dry run"}[dryRun], func(t *testing.T) {
					t.Parallel()
					root := t.TempDir()
					name := "CLAUDE.md"
					if writer == "codex" {
						name = "AGENTS.md"
					}
					path := filepath.Join(root, name)
					target := filepath.Join(root, "shared-instructions")
					if kind == "external" {
						target = filepath.Join(t.TempDir(), "instructions")
					}
					linkText := target
					if kind == "ordinary" || kind == "external" {
						writeAgentsFixture(t, target, "owned by user\n", 0o600)
					}
					if kind == "ordinary" {
						linkText = "shared-instructions"
					}
					if kind == "loop" {
						linkText = name
					}
					if err := os.Symlink(linkText, path); err != nil {
						t.Fatal(err)
					}
					before, err := os.Lstat(path)
					if err != nil {
						t.Fatal(err)
					}
					step, err := runInstructionWriter(context.Background(), root, writer, dryRun)
					if err != nil {
						t.Fatal(err)
					}
					assertIncompleteGuidance(t, step, name)
					if !strings.Contains(step.Detail, "symlink") {
						t.Fatalf("missing condition: %+v", step)
					}
					after, err := os.Lstat(path)
					if err != nil || !os.SameFile(before, after) {
						t.Fatalf("link replaced: %v", err)
					}
					got, err := os.Readlink(path)
					if err != nil || got != linkText {
						t.Fatalf("link = %q, %v", got, err)
					}
					if kind == "ordinary" || kind == "external" {
						assertAgentsFile(t, target, "owned by user\n", 0o600)
					}
					if kind == "broken" {
						if _, err := os.Lstat(target); !errors.Is(err, fs.ErrNotExist) {
							t.Fatalf("created broken link target: %v", err)
						}
					}
				})
			}
		}
	}
}

func TestInstructionWriters_RefuseAliasesBeforeEitherWrite(t *testing.T) {
	t.Parallel()
	for _, shape := range []string{"hard links", "claude points to agents", "agents points to claude", "both point outside", "chained link", "dangling peer"} {
		for _, order := range [][]string{{"scaffold", "claude", "codex"}, {"codex", "claude", "scaffold"}} {
			t.Run(shape+"/"+order[0], func(t *testing.T) {
				t.Parallel()
				root := t.TempDir()
				claude := filepath.Join(root, "CLAUDE.md")
				agents := filepath.Join(root, "AGENTS.md")
				target := agents
				switch shape {
				case "hard links":
					writeAgentsFixture(t, agents, "shared user text", 0o640)
					if err := os.Link(agents, claude); err != nil {
						t.Fatal(err)
					}
				case "claude points to agents":
					writeAgentsFixture(t, agents, "shared user text", 0o640)
					if err := os.Symlink("AGENTS.md", claude); err != nil {
						t.Fatal(err)
					}
				case "agents points to claude":
					target = claude
					writeAgentsFixture(t, claude, "shared user text", 0o640)
					if err := os.Symlink("CLAUDE.md", agents); err != nil {
						t.Fatal(err)
					}
				case "both point outside":
					target = filepath.Join(t.TempDir(), "instructions")
					writeAgentsFixture(t, target, "shared user text", 0o640)
					for _, path := range []string{claude, agents} {
						if err := os.Symlink(target, path); err != nil {
							t.Fatal(err)
						}
					}
				case "chained link":
					writeAgentsFixture(t, agents, "shared user text", 0o640)
					if err := os.Symlink("AGENTS.md", filepath.Join(root, "bridge")); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink("bridge", claude); err != nil {
						t.Fatal(err)
					}
				case "dangling peer":
					if err := os.Symlink("AGENTS.md", claude); err != nil {
						t.Fatal(err)
					}
				}
				beforeClaude, err := os.Lstat(claude)
				if err != nil {
					t.Fatal(err)
				}
				beforeAgents, err := os.Lstat(agents)
				if err != nil && !errors.Is(err, fs.ErrNotExist) {
					t.Fatal(err)
				}
				for _, writer := range order {
					step, err := runInstructionWriter(context.Background(), root, writer, false)
					if err != nil {
						t.Fatal(err)
					}
					assertIncompleteGuidance(t, step, "CLAUDE.md", "AGENTS.md")
					if !strings.Contains(step.Detail, "alias") {
						t.Fatalf("missing alias diagnosis: %+v", step)
					}
					if shape != "dangling peer" {
						assertAgentsFile(t, target, "shared user text", 0o640)
					}
					afterClaude, err := os.Lstat(claude)
					if err != nil || !os.SameFile(beforeClaude, afterClaude) {
						t.Fatalf("CLAUDE path changed: %v", err)
					}
					afterAgents, err := os.Lstat(agents)
					if beforeAgents == nil {
						if !errors.Is(err, fs.ErrNotExist) {
							t.Fatalf("created aliased target: %v", err)
						}
					} else if err != nil || !os.SameFile(beforeAgents, afterAgents) {
						t.Fatalf("AGENTS path changed: %v", err)
					}
				}
			})
		}
	}
}

func TestInstructionWriters_IndependentRegularFilesRemainWritable(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeAgentsFixture(t, filepath.Join(root, "CLAUDE.md"), "claude instructions", 0o644)
	writeAgentsFixture(t, filepath.Join(root, "AGENTS.md"), "codex instructions", 0o640)
	for _, writer := range []string{"scaffold", "claude", "codex"} {
		step, err := runInstructionWriter(context.Background(), root, writer, false)
		if err != nil || step.Action == ActionSkipped {
			t.Fatalf("%s: %+v, %v", writer, step, err)
		}
	}
	for _, name := range []string{"CLAUDE.md", "AGENTS.md"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), guidanceImportStartMarker) {
			t.Fatalf("%s not wired", name)
		}
	}
}

func TestInstructionWriters_UnresolvablePeerPreventsUnsafeWrites(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	root := t.TempDir()
	private := t.TempDir()
	target := filepath.Join(private, "instructions")
	writeAgentsFixture(t, target, "private", 0o600)
	writeAgentsFixture(t, filepath.Join(root, "CLAUDE.md"), "claude", 0o644)
	if err := os.Symlink(target, filepath.Join(root, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(private, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(private, 0o700) })
	for _, writer := range []string{"scaffold", "claude", "codex"} {
		step, err := runInstructionWriter(context.Background(), root, writer, false)
		if err != nil {
			t.Fatal(err)
		}
		assertIncompleteGuidance(t, step, "CLAUDE.md", "AGENTS.md")
	}
	assertAgentsFile(t, filepath.Join(root, "CLAUDE.md"), "claude", 0o644)
	if err := os.Chmod(private, 0o700); err != nil {
		t.Fatal(err)
	}
	assertAgentsFile(t, target, "private", 0o600)
}

func TestInstructionInspection_PropagatesContextAndFilesystemErrors(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := inspectInstructionFiles(ctx, root); !errors.Is(err, context.Canceled) {
		t.Fatalf("context error = %v", err)
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	if err := os.Chmod(root, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o700) })
	if _, err := inspectInstructionFiles(context.Background(), root); !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("inspection error = %v", err)
	}
	for _, writer := range []string{"scaffold", "claude", "codex"} {
		if _, err := runInstructionWriter(context.Background(), root, writer, false); !errors.Is(err, fs.ErrPermission) {
			t.Fatalf("%s error = %v", writer, err)
		}
	}
}

func TestInitAndRefresh_ReportIncompleteGuidanceAndContinueArtifacts(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"symlink", "alias"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			root := freshGitRepo(t)
			path := filepath.Join(root, "CLAUDE.md")
			target := filepath.Join(root, "user-instructions")
			if mode == "alias" {
				target = filepath.Join(root, "AGENTS.md")
			}
			writeAgentsFixture(t, target, "user instructions", 0o640)
			if err := os.Symlink(filepath.Base(target), path); err != nil {
				t.Fatal(err)
			}
			result, err := Init(context.Background(), root, Options{SkipHook: true})
			if err != nil {
				t.Fatal(err)
			}
			assertGuidanceAndArtifactSteps(t, result.Steps)
			assertAgentsFile(t, target, "user instructions", 0o640)
			for _, relative := range []string{skills.GuidanceFile, ".claude/skills/aiwf-check/SKILL.md", "aiwf.example.yaml", ".gitignore"} {
				if _, statErr := os.Stat(filepath.Join(root, relative)); statErr != nil {
					t.Fatalf("unrelated artifact %s: %v", relative, statErr)
				}
			}
			refresh, err := RefreshArtifacts(context.Background(), root, RefreshOptions{WireClaudeMd: true, SkipHooks: true})
			if err != nil {
				t.Fatal(err)
			}
			if refresh.HookConflict {
				t.Fatalf("refresh: %v, hook conflict %v", err, refresh.HookConflict)
			}
			assertGuidanceAndArtifactSteps(t, refresh.Steps)
			assertAgentsFile(t, target, "user instructions", 0o640)
			got, err := os.Readlink(path)
			if err != nil || got != filepath.Base(target) {
				t.Fatalf("symlink = %q, %v", got, err)
			}
		})
	}
}

func runInstructionWriter(ctx context.Context, root, writer string, dryRun bool) (StepResult, error) {
	switch writer {
	case "scaffold":
		return ensureClaudeMd(ctx, root, dryRun)
	case "claude":
		return ensureGuidanceImport(ctx, root, RefreshOptions{WireClaudeMd: true, DryRun: dryRun})
	default:
		return ensureAgentsGuidance(ctx, root, nil, dryRun)
	}
}

func assertIncompleteGuidance(t *testing.T, step StepResult, names ...string) {
	t.Helper()
	if step.Action != ActionSkipped || !strings.Contains(step.Detail, "guidance incomplete") || !strings.Contains(step.Detail, "regular file") {
		t.Fatalf("not actionable incomplete guidance: %+v", step)
	}
	for _, name := range names {
		if !strings.Contains(step.Detail, name) {
			t.Fatalf("missing path %s: %+v", name, step)
		}
	}
}

func assertGuidanceAndArtifactSteps(t *testing.T, steps []StepResult) {
	t.Helper()
	foundGuidance, foundSkills := false, false
	for _, step := range steps {
		if step.What == "CLAUDE.md (aiwf guidance import)" {
			assertIncompleteGuidance(t, step, "CLAUDE.md")
			foundGuidance = true
		}
		if step.What == ".claude/skills/aiwf-*" && step.Action == ActionUpdated {
			foundSkills = true
		}
	}
	if !foundGuidance || !foundSkills {
		t.Fatalf("missing incomplete guidance or successful artifacts: %+v", steps)
	}
}

func TestInstructionWriters_ResolvedLinkAllowsIndependentPeer(t *testing.T) {
	t.Parallel()
	for _, linked := range []string{"CLAUDE.md", "AGENTS.md"} {
		for _, existing := range []bool{false, true} {
			t.Run(linked+"/"+map[bool]string{false: "absent peer", true: "existing peer"}[existing], func(t *testing.T) {
				t.Parallel()
				root := t.TempDir()
				target := filepath.Join(t.TempDir(), "instructions")
				writeAgentsFixture(t, target, "linked instructions", 0o640)
				path := filepath.Join(root, linked)
				if err := os.Symlink(target, path); err != nil {
					t.Fatal(err)
				}
				peer := "AGENTS.md"
				writers := []string{"codex"}
				if linked == "AGENTS.md" {
					peer = "CLAUDE.md"
					writers = []string{"scaffold", "claude"}
				}
				if existing {
					writeAgentsFixture(t, filepath.Join(root, peer), "independent instructions", 0o644)
				}
				for _, writer := range writers {
					step, err := runInstructionWriter(context.Background(), root, writer, false)
					if err != nil || step.Action == ActionSkipped {
						t.Fatalf("independent %s: %+v, %v", peer, step, err)
					}
				}
				data, err := os.ReadFile(filepath.Join(root, peer))
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(data), guidanceImportStartMarker) {
					t.Fatalf("%s guidance missing", peer)
				}
				assertAgentsFile(t, target, "linked instructions", 0o640)
				link, err := os.Readlink(path)
				if err != nil || link != target {
					t.Fatalf("link = %q, %v", link, err)
				}
			})
		}
	}
}

func TestClaudeScaffold_WriteFailureDoesNotCreateInstructionFile(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	root := t.TempDir()
	if err := os.Chmod(root, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o700) })
	if _, err := ensureClaudeMd(context.Background(), root, false); !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("write error = %v", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("write left files: %v, %v", entries, err)
	}
}
