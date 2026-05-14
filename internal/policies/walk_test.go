package policies

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWalkGoFiles_SkipsExcludedDirs pins the directory-name skip
// list in WalkGoFiles. The .claude exclusion is the load-bearing
// new entry per G-0095: Claude Code's worktree directories live at
// `.claude/worktrees/agent-*/` and contain full kernel-source
// clones. Without the skip, every policy that walks Go files
// re-flags the sibling worktree's intentional definitions of
// trailer keys, --force references, etc. as production violations
// of the kernel.
func TestWalkGoFiles_SkipsExcludedDirs(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	// Files that should be picked up.
	want := map[string]bool{
		"main.go":             true,
		"internal/foo.go":     true,
		"cmd/bar/bar.go":      true,
		"cmd/bar/bar_test.go": true,
	}
	// Files that must NOT be picked up — each lives under an
	// excluded directory name.
	excluded := []string{
		"vendor/lib.go",
		"node_modules/pkg.go",
		".git/hook.go",
		".claude/worktrees/agent-x/internal/gitops/trailers.go",
		".claude/skills/aiwf-foo/something.go",
	}

	for path := range want {
		writeFixture(t, root, path)
	}
	for _, path := range excluded {
		writeFixture(t, root, path)
	}

	got, err := WalkGoFiles(root, false)
	if err != nil {
		t.Fatalf("WalkGoFiles: %v", err)
	}

	gotRel := map[string]bool{}
	for _, e := range got {
		gotRel[filepath.ToSlash(e.Path)] = true
	}

	for path := range want {
		if !gotRel[path] {
			t.Errorf("missing expected file %q in WalkGoFiles result", path)
		}
	}
	for _, path := range excluded {
		if gotRel[path] {
			t.Errorf("excluded file %q surfaced from WalkGoFiles — directory skip-list is too narrow", path)
		}
	}
}

func writeFixture(t *testing.T, root, rel string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(abs), err)
	}
	if err := os.WriteFile(abs, []byte("package x\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", abs, err)
	}
}
