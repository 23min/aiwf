package initrepo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/config"
)

func TestRetainedHostSteps_ReportsOnlyExistingUnselectedPaths(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		selected []config.Host
		want     []string
	}{
		{"none", nil, []string{".claude", "CLAUDE.md", ".agents", "AGENTS.md"}},
		{"claude", []config.Host{config.HostClaudeCode}, []string{".agents", "AGENTS.md"}},
		{"codex", []config.Host{config.HostCodex}, []string{".claude", "CLAUDE.md"}},
		{"both", []config.Host{config.HostClaudeCode, config.HostCodex}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if got := retainedHostSteps(root, config.HostSelection{Hosts: tc.selected}); len(got) != 0 {
				t.Fatalf("empty repo reported retention: %+v", got)
			}
			for _, path := range []string{".claude", "CLAUDE.md", ".agents", "AGENTS.md"} {
				// Dangling links count as retained paths without reading their targets.
				if err := os.Symlink("missing-target", filepath.Join(root, path)); err != nil {
					t.Fatal(err)
				}
			}
			var got []string
			for _, step := range retainedHostSteps(root, config.HostSelection{Hosts: tc.selected}) {
				if step.Action != ActionPreserved {
					t.Errorf("action = %s", step.Action)
				}
				got = append(got, step.What)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("retained paths (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRetainedHostSteps_UnreadableRootReportsSkippedInspection(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.Chmod(root, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(root, 0o700); err != nil {
			t.Error(err)
		}
	})
	if _, err := os.Lstat(filepath.Join(root, ".claude")); !os.IsPermission(err) {
		t.Skip("process bypasses directory permissions")
	}
	steps := retainedHostSteps(root, config.HostSelection{})
	if len(steps) != 4 {
		t.Fatalf("inspection failures = %+v", steps)
	}
	for _, step := range steps {
		if step.Action != ActionSkipped {
			t.Errorf("inspection failure = %+v", step)
		}
	}
}
