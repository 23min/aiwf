package initrepo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/config"
)

func TestHostLifecycle_CanceledSetupWritesNothing(t *testing.T) {
	t.Parallel()
	for _, refresh := range []bool{false, true} {
		t.Run(map[bool]string{false: "init", true: "refresh"}[refresh], func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			var err error
			if refresh {
				_, err = RefreshArtifacts(ctx, root, RefreshOptions{SkipHooks: true})
			} else {
				_, err = Init(ctx, root, Options{ActorOverride: "human/test", SkipHook: true})
			}
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("canceled setup = %v", err)
			}
			entries, readErr := os.ReadDir(root)
			if readErr != nil || len(entries) != 0 {
				t.Fatalf("canceled setup wrote files: %v, %v", entries, readErr)
			}
		})
	}
}

func TestHostLifecycle_CodexCollisionPreservesUserSkill(t *testing.T) {
	t.Parallel()
	for _, refresh := range []bool{false, true} {
		t.Run(map[bool]string{false: "init", true: "refresh"}[refresh], func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if err := config.Write(root, &config.Config{Hosts: &[]string{"codex"}}); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, ".agents", "skills", "aiwf-check", "SKILL.md")
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("personal skill\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			var err error
			if refresh {
				_, err = RefreshArtifacts(context.Background(), root, RefreshOptions{SkipHooks: true})
			} else {
				_, err = Init(context.Background(), root, Options{ActorOverride: "human/test", SkipHook: true})
			}
			if err == nil {
				t.Fatal("unowned Codex skill was overwritten")
			}
			assertAgentsFile(t, path, "personal skill\n", 0o600)
			for _, relative := range []string{".agents/skills/.aiwf-owned", ".agents/aiwf/templates", "AGENTS.md"} {
				if _, statErr := os.Stat(filepath.Join(root, relative)); !os.IsNotExist(statErr) {
					t.Errorf("partial Codex output %s: %v", relative, statErr)
				}
			}
		})
	}
}

func TestHostLifecycle_CodexDryRunReportsSelectionWithoutWrites(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := config.Write(root, &config.Config{Hosts: &[]string{"codex"}}); err != nil {
		t.Fatal(err)
	}
	result, err := RefreshArtifacts(context.Background(), root, RefreshOptions{SkipHooks: true, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	want := config.HostSelection{Hosts: []config.Host{config.HostCodex}, Source: config.HostsConfigured}
	if diff := cmp.Diff(want, result.HostSelection); diff != "" {
		t.Fatalf("selection (-want +got):\n%s", diff)
	}
	if !result.DryRun {
		t.Fatal("dry-run result was not marked")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	if diff := cmp.Diff([]string{config.FileName}, names); diff != "" {
		t.Fatalf("dry-run files (-want +got):\n%s", diff)
	}
}

func TestHostLifecycle_ReportsSelectedHostWriteAndReadFailures(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, host, path string
		unreadable       bool
	}{
		{"claude skill directory", "claude-code", ".claude/skills", false},
		{"claude guidance output", "claude-code", ".claude/aiwf-guidance.md", false},
		{"claude root guidance", "claude-code", "CLAUDE.md", true},
		{"codex root guidance", "codex", "AGENTS.md", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if err := config.Write(root, &config.Config{Hosts: &[]string{tc.host}}); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, tc.path)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			switch {
			case tc.unreadable:
				if err := os.WriteFile(path, []byte("user instructions"), 0); err != nil {
					t.Fatal(err)
				}
				if _, err := os.ReadFile(path); err == nil {
					t.Skip("process can read mode-000 files")
				}
			case tc.path == ".claude/skills":
				if err := os.WriteFile(path, []byte("user file"), 0o600); err != nil {
					t.Fatal(err)
				}
			default:
				if err := os.Mkdir(path, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			if result, err := RefreshArtifacts(context.Background(), root, RefreshOptions{SkipHooks: true, WireClaudeMd: true}); err == nil || result != nil {
				t.Fatalf("host failure hidden: result=%+v err=%v", result, err)
			}
		})
	}
}
