package render_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/render"
)

// M-0256/AC-1 backfill: RunRoadmap's and RunSite's ResolveRoot guards
// are `//coverage:ignore`d in render.go itself (ResolveRoot only wraps
// filepath.Abs/os.Getwd, neither triggerable). Their bare tree.Load
// guards are NOT ignored — unlike internal/cli/check's unexported
// runShapeOnly/runFast (which need a canceled context, only reachable
// via a direct in-package call), RunRoadmap/RunSite's tree.Load can
// also fail on an unreadable directory (os.Stat returning a
// permission error, not os.ErrNotExist), which IS reachable through
// the public API — see TestRunRoadmap_TreeLoadFailure /
// TestRunSite_TreeLoadFailure below. check.WalkHeadCommits' `git log
// HEAD` failure mirrors the same unreachable-once-HasCommits-succeeded
// class internal/cli/check's own headErr guard documents. Every other
// flagged branch below is genuinely triggerable.

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

// TestRunRoadmap_TreeLoadFailure covers RunRoadmap's bare tree.Load
// guard: an unreadable root directory (no execute/search permission)
// makes tree.Load's own os.Stat(root/work/epics) fail with a
// permission error, not os.ErrNotExist — the fatal, non-loadErrs path.
// 0o000 (not 0o500) is required so the search bit is also gone;
// 0o500 (as TestRunRoadmap_WriteFailure below uses) still lets
// tree.Load succeed reading, only failing later at the write step.
func TestRunRoadmap_TreeLoadFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.Chmod(root, 0o000); err != nil {
		t.Fatalf("chmod root: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o755) })
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

// TestRunSite_TreeLoadFailure covers RunSite's bare tree.Load guard,
// mirroring TestRunRoadmap_TreeLoadFailure's unreadable-root fixture.
func TestRunSite_TreeLoadFailure(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.Chmod(root, 0o000); err != nil {
		t.Fatalf("chmod root: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o755) })
	rc := render.RunSite(root, "html", "", "", false, false)
	if rc != cliutil.ExitInternal {
		t.Errorf("rc = %d, want ExitInternal", rc)
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
