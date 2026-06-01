package authorize_test

import (
	"os/exec"
	"path/filepath"
	"sort"
	"testing"

	"github.com/23min/aiwf/internal/cli/authorize"
	"github.com/23min/aiwf/internal/cli/cliutil"
)

func TestNewCmd_SmokeShape(t *testing.T) {
	t.Parallel()
	cmd := authorize.NewCmd()
	if cmd == nil {
		t.Fatal("NewCmd returned nil")
	}
	if cmd.Use != "authorize <id>" {
		t.Errorf("Use = %q", cmd.Use)
	}
	for _, flag := range []string{"actor", "root", "to", "pause", "resume", "reason", "force", "branch"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("missing --%s flag", flag)
		}
	}
}

// TestRun_BranchWithPauseRejected (M-0102/AC-1, cli-layer gate): --branch
// is meaningful only on the open path. --branch + --pause is rejected
// upfront so the operator sees the misuse rather than silently dropping
// the flag. Mirrors the existing --reason + --pause guard.
func TestRun_BranchWithPauseRejected(t *testing.T) {
	t.Parallel()
	// pause supplies the reason; --branch must NOT be combined.
	rc := authorize.Run(
		"E-0001",          // id
		"human/test",      // actor
		"",                // root (unused; we fail before tree load)
		"",                // to
		"blocked by E-09", // pause
		"",                // resume
		"",                // reason
		"epic/E-0001-eng", // branch
		false,             // force
		cliutil.OutputFormat{},
	)
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage (%d)", rc, cliutil.ExitUsage)
	}
}

// TestRun_BranchWithResumeRejected: mirror of the pause case for the
// resume mode.
func TestRun_BranchWithResumeRejected(t *testing.T) {
	t.Parallel()
	rc := authorize.Run(
		"E-0001",
		"human/test",
		"",
		"",
		"",                // pause
		"resume work now", // resume
		"",
		"epic/E-0001-eng",
		false,
		cliutil.OutputFormat{},
	)
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage (%d)", rc, cliutil.ExitUsage)
	}
}

// TestRitualLocalBranches_FiltersRitualShape (M-0102/AC-6): the
// --branch completion's underlying enumerator returns only local
// branch names matching the ADR-0010 ritual shape (epic/E-NNNN-...,
// milestone/M-NNNN-..., patch/[Gg]-NNNN-...). Non-ritual branches
// (main, fix/*, chore/*, patch/<topic-without-id>) are filtered out.
func TestRitualLocalBranches_FiltersRitualShape(t *testing.T) {
	t.Parallel()
	root := mustNewGitRepo(t)
	mustGit(t, root, "commit", "--allow-empty", "-m", "init")
	for _, b := range []string{
		"epic/E-0010-cobra",
		"milestone/M-0007-cache",
		"patch/g-0099-iso",
		"patch/G-0050-other",
		"fix/some-bug",
		"chore/dep-bump",
		"patch/refactor-stuff",
		"feature/x",
	} {
		mustGit(t, root, "branch", b)
	}

	got := authorize.RitualLocalBranchesForTest(root)
	sort.Strings(got)
	want := []string{
		"epic/E-0010-cobra",
		"milestone/M-0007-cache",
		"patch/G-0050-other",
		"patch/g-0099-iso",
	}
	if len(got) != len(want) {
		t.Fatalf("len(got)=%d, want=%d\ngot:  %v\nwant: %v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}

// TestRitualLocalBranches_NonGitDirReturnsNil: best-effort behavior —
// a non-git directory yields nil rather than erroring, so the shell
// falls through to its default (no completion suggestions).
func TestRitualLocalBranches_NonGitDirReturnsNil(t *testing.T) {
	t.Parallel()
	got := authorize.RitualLocalBranchesForTest(t.TempDir())
	if got != nil {
		t.Errorf("non-git dir: got %v, want nil", got)
	}
}

// TestRitualLocalBranches_EmptyRepoReturnsNil: a fresh git repo with no
// commits has no branches in refs/heads/ — the helper returns nil
// (no panic, no error surfacing to the shell).
func TestRitualLocalBranches_EmptyRepoReturnsNil(t *testing.T) {
	t.Parallel()
	root := mustNewGitRepo(t)
	got := authorize.RitualLocalBranchesForTest(root)
	if got != nil {
		t.Errorf("empty repo: got %v, want nil", got)
	}
}

// TestCurrentBranch_ReturnsCheckedOutBranch (M-0103/AC-3): the helper
// returns the short name of the current checkout. Used by the verb's
// preflight to detect implicit ritual-branch context.
func TestCurrentBranch_ReturnsCheckedOutBranch(t *testing.T) {
	t.Parallel()
	root := mustNewGitRepo(t)
	mustGit(t, root, "commit", "--allow-empty", "-m", "init")
	mustGit(t, root, "checkout", "-b", "epic/E-0001-engine")
	got := authorize.CurrentBranchForTest(root)
	if got != "epic/E-0001-engine" {
		t.Errorf("CurrentBranchForTest = %q, want %q", got, "epic/E-0001-engine")
	}
}

// TestCurrentBranch_NonGitDirReturnsEmpty: best-effort behavior — a
// non-git directory yields an empty string, so the verb's preflight
// interprets it as "no ritual context detected" and refuses when the
// gate fires. No panic, no error surfacing to the shell.
func TestCurrentBranch_NonGitDirReturnsEmpty(t *testing.T) {
	t.Parallel()
	if got := authorize.CurrentBranchForTest(t.TempDir()); got != "" {
		t.Errorf("CurrentBranchForTest non-git: got %q, want empty", got)
	}
}

// TestCurrentBranch_DetachedHEADReturnsEmpty: when HEAD is detached
// (no symbolic-ref), the helper returns empty. The verb interprets
// this the same way as a non-git directory — no ritual context.
func TestCurrentBranch_DetachedHEADReturnsEmpty(t *testing.T) {
	t.Parallel()
	root := mustNewGitRepo(t)
	mustGit(t, root, "commit", "--allow-empty", "-m", "init")
	mustGit(t, root, "commit", "--allow-empty", "-m", "second")
	// Detach HEAD by checking out the previous commit.
	mustGit(t, root, "checkout", "HEAD^")
	if got := authorize.CurrentBranchForTest(root); got != "" {
		t.Errorf("CurrentBranchForTest detached: got %q, want empty", got)
	}
}

// TestBranchExists_True (M-0103/AC-4): the helper returns true when
// refs/heads/<branch> resolves.
func TestBranchExists_True(t *testing.T) {
	t.Parallel()
	root := mustNewGitRepo(t)
	mustGit(t, root, "commit", "--allow-empty", "-m", "init")
	mustGit(t, root, "branch", "epic/E-0001-engine")
	if !authorize.BranchExistsForTest(root, "epic/E-0001-engine") {
		t.Error("BranchExistsForTest on existing branch returned false")
	}
}

// TestBranchExists_False (M-0103/AC-2): the helper returns false for
// a name that doesn't resolve under refs/heads/ — the verb refuses
// with branch-not-found.
func TestBranchExists_False(t *testing.T) {
	t.Parallel()
	root := mustNewGitRepo(t)
	mustGit(t, root, "commit", "--allow-empty", "-m", "init")
	// No branch by that name was created.
	if authorize.BranchExistsForTest(root, "epic/E-9999-typo") {
		t.Error("BranchExistsForTest on missing branch returned true")
	}
}

// TestBranchExists_EmptyName: when --branch wasn't passed, branch is
// empty; the helper short-circuits to false so the verb's preflight
// takes the implicit-current-branch path instead of branch-not-found.
func TestBranchExists_EmptyName(t *testing.T) {
	t.Parallel()
	root := mustNewGitRepo(t)
	mustGit(t, root, "commit", "--allow-empty", "-m", "init")
	if authorize.BranchExistsForTest(root, "") {
		t.Error("BranchExistsForTest on empty name returned true")
	}
}

// TestBranchExists_NonGitDir: best-effort behavior — a non-git dir
// yields false.
func TestBranchExists_NonGitDir(t *testing.T) {
	t.Parallel()
	if authorize.BranchExistsForTest(t.TempDir(), "main") {
		t.Error("BranchExistsForTest non-git returned true")
	}
}

// mustNewGitRepo initializes a fresh git repo under a fresh TempDir
// and returns its root. Identity is set so commit operations succeed.
func mustNewGitRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mustGit(t, root, "init", "-q")
	mustGit(t, root, "config", "user.email", "test@example.com")
	mustGit(t, root, "config", "user.name", "Tester")
	return root
}

// mustGit runs `git <args...>` in rootDir, failing the test on error.
func mustGit(t *testing.T, rootDir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = rootDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, filepath.Base(rootDir), err, out)
	}
}
