package check

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
)

// initGitRepo creates a minimal git repo at dir for the
// RunGitConfigCheck tests. Uses `git init -b main` so the repo has a
// well-defined initial branch. Returns the absolute path that
// filepath.Abs would resolve `dir` to — useful for the comparison
// assertions below where the expected and actual paths must agree
// exactly. Fails the test on any git failure.
func initGitRepo(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "init", "-b", "main", dir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	return abs
}

// setCoreWorktree writes core.worktree=<value> into the local git
// config at dir. Fails the test on git error.
func setCoreWorktree(t *testing.T, dir, value string) {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "config", "--local", "core.worktree", value)
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config --local core.worktree %s: %v", value, err)
	}
}

// TestRunGitConfigCheck_Unset is the AC-1 case: a healthy repo with
// no core.worktree override returns no findings.
func TestRunGitConfigCheck_Unset(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	initGitRepo(t, dir)
	findings := RunGitConfigCheck(context.Background(), dir)
	if len(findings) != 0 {
		t.Errorf("expected no findings with core.worktree unset; got %+v", findings)
	}
}

// TestRunGitConfigCheck_SetToSelf is the AC-2 case: a repo with
// core.worktree explicitly set to its own path (the legitimate
// pattern for linked worktrees) returns no findings.
func TestRunGitConfigCheck_SetToSelf(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	abs := initGitRepo(t, dir)
	setCoreWorktree(t, dir, abs)
	findings := RunGitConfigCheck(context.Background(), dir)
	if len(findings) != 0 {
		t.Errorf("expected no findings with core.worktree=self; got %+v", findings)
	}
}

// TestRunGitConfigCheck_Misset is the AC-3 case: a repo whose
// core.worktree points elsewhere fires a single error finding with
// the configured value and the expected root in its message.
func TestRunGitConfigCheck_Misset(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	abs := initGitRepo(t, dir)
	// Set core.worktree to a path that is NOT the repo root. The
	// concrete path doesn't matter (the check only compares strings);
	// using a sibling tempdir-shaped path keeps the test hermetic.
	misset := filepath.Join(filepath.Dir(abs), "elsewhere")
	setCoreWorktree(t, dir, misset)

	findings := RunGitConfigCheck(context.Background(), dir)
	if len(findings) != 1 {
		t.Fatalf("expected exactly 1 finding; got %d: %+v", len(findings), findings)
	}
	f := findings[0]
	if f.Code != "git-config-core-worktree-misset" {
		t.Errorf("Code = %q; want git-config-core-worktree-misset", f.Code)
	}
	if f.Severity != "error" {
		t.Errorf("Severity = %q; want error", f.Severity)
	}
	// Message must name both the configured (wrong) value and the
	// expected root so the operator can see the discrepancy at a glance.
	if !contains(f.Message, misset) {
		t.Errorf("Message should name the configured value %q; got %q", misset, f.Message)
	}
	if !contains(f.Message, abs) {
		t.Errorf("Message should name the expected root %q; got %q", abs, f.Message)
	}
	if f.Path != ".git/config" {
		t.Errorf("Path = %q; want .git/config", f.Path)
	}
}

// TestRunGitConfigCheck_NonGitDir guards the "not a git repo"
// edge case: an empty tempdir returns no findings (the git config
// invocation fails, and the check treats that as "nothing to flag" —
// other checks are responsible for surfacing the not-a-repo state).
func TestRunGitConfigCheck_NonGitDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	findings := RunGitConfigCheck(context.Background(), dir)
	if len(findings) != 0 {
		t.Errorf("expected no findings on non-git dir; got %+v", findings)
	}
}

// contains is a tiny substring helper to avoid pulling in strings
// just for this test file. Returns true iff substr is somewhere in s.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
