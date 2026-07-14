package render_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/render"
)

// M-0256/AC-1 backfill: RunRoadmap's and RunSite's ResolveRoot guards,
// plus their bare tree.Load guards (both functions always pass a
// fresh context.Background() with no way to inject a canceled one
// through the public API — unlike internal/cli/check's unexported
// runShapeOnly/runFast), are `//coverage:ignore`d in render.go itself.
// check.WalkHeadCommits' `git log HEAD` failure mirrors the same
// unreachable-once-HasCommits-succeeded class internal/cli/check's own
// headErr guard documents. Every other flagged branch below is
// genuinely triggerable.

// TestRunRoadmap_ExistingReadFailure covers RunRoadmap's
// os.ReadFile(dest) guard: a directory sitting at the resolved
// roadmap path fails to read as a file (not os.ErrNotExist).
func TestRunRoadmap_ExistingReadFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "ROADMAP.md"), 0o755); err != nil {
		t.Fatalf("mkdir ROADMAP.md: %v", err)
	}
	rc := render.RunRoadmap(root, false)
	if rc != cliutil.ExitInternal {
		t.Errorf("rc = %d, want ExitInternal", rc)
	}
}

// TestRunRoadmap_WriteFailure covers RunRoadmap's
// pathutil.AtomicWriteFile guard: a read-only root directory makes
// os.CreateTemp (AtomicWriteFile's first step) fail, since the read-
// only permission bits still permit the earlier os.ReadFile lookup
// (which reports a tolerated os.ErrNotExist) but not a new file
// creation.
func TestRunRoadmap_WriteFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.Chmod(root, 0o500); err != nil {
		t.Fatalf("chmod root: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o755) })
	rc := render.RunRoadmap(root, true)
	if rc != cliutil.ExitInternal {
		t.Errorf("rc = %d, want ExitInternal", rc)
	}
}

// TestRunSite_BadFormat covers RunSite's --format validation guard
// (RunSite is also reachable directly, independent of NewCmd's own
// earlier "missing --format" check).
func TestRunSite_BadFormat(t *testing.T) {
	t.Parallel()
	rc := render.RunSite("", "bogus", "", "", false, false)
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRunSite_HTMLRenderFailure covers RunSite's htmlrender.Render
// guard: a read-only root directory makes htmlrender.Render's own
// os.MkdirAll(outDir) fail, since the default --out resolves to a
// subdirectory of root.
func TestRunSite_HTMLRenderFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.Chmod(root, 0o500); err != nil {
		t.Fatalf("chmod root: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o755) })
	rc := render.RunSite(root, "html", "", "", false, false)
	if rc != cliutil.ExitInternal {
		t.Errorf("rc = %d, want ExitInternal", rc)
	}
}
