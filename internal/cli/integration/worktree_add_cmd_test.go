package integration

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// setupInitedRepo inits a fresh repo, runs the real `aiwf init`
// against it, and commits the result so the repo has a base commit
// `aiwf worktree add` can branch a new worktree from. Returns
// (root, bin).
func setupInitedRepo(t *testing.T) (root, bin string) {
	t.Helper()
	bin = testutil.AiwfBinary(t)
	binDir := filepath.Dir(bin)
	root = t.TempDir()
	// -b main pins the default branch name explicitly: the host's
	// init.defaultBranch config is not guaranteed to be "main", and
	// TestWorktreeAdd_PrintPath_EmptyStdoutOnFailure depends on
	// "main" being the branch actually checked out in root.
	if out, err := testutil.RunGit(root, "init", "-q", "-b", "main"); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "peter@example.com"},
		{"config", "user.name", "Peter Test"},
	} {
		if out, err := testutil.RunGit(root, args...); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if out, err := testutil.RunBin(t, root, binDir, nil, "init"); err != nil {
		t.Fatalf("aiwf init: %v\n%s", err, out)
	}
	if out, err := testutil.RunGit(root, "add", "-A"); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	if out, err := testutil.RunGit(root, "commit", "-q", "-m", "seed"); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
	return root, bin
}

// shq single-quotes s for safe embedding in a `sh -c` command string.
func shq(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// TestWorktreeAdd_PrintPath_OnlyPathOnStdout is M-0233/AC-4's success
// half: --print-path writes exactly one line (the absolute path) to
// stdout and nothing to stderr.
func TestWorktreeAdd_PrintPath_OnlyPathOnStdout(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	root, bin := setupInitedRepo(t)

	stdout, stderr, code := runSplit(t, root, bin, "worktree", "add", "feature/print-only", "--print-path")
	if code != 0 {
		t.Fatalf("exit = %d, want 0\nstdout=%q\nstderr=%q", code, stdout, stderr)
	}
	if stderr != "" {
		t.Errorf("stderr should be empty on success; got %q", stderr)
	}
	if strings.Count(stdout, "\n") != 1 {
		t.Errorf("stdout should be exactly one line (path + trailing newline); got %q", stdout)
	}
	trimmed := strings.TrimRight(stdout, "\n")
	if !filepath.IsAbs(trimmed) {
		t.Errorf("stdout should be a bare absolute path; got %q", stdout)
	}
}

// TestWorktreeAdd_PrintPath_EmptyStdoutOnFailure is M-0233/AC-4's
// failure half: on a `git worktree add` refusal, --print-path emits
// nothing to stdout and exits nonzero, so `cd "$(...)"` fails loudly
// rather than landing somewhere wrong.
func TestWorktreeAdd_PrintPath_EmptyStdoutOnFailure(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	root, bin := setupInitedRepo(t)

	// "main" is already checked out in root itself; asking to check
	// it out again into a second worktree is a real git refusal.
	stdout, stderr, code := runSplit(t, root, bin, "worktree", "add", "main", filepath.Join(t.TempDir(), "wt"), "--print-path")
	if code == 0 {
		t.Fatalf("expected nonzero exit; stdout=%q stderr=%q", stdout, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout must be empty on failure under --print-path; got %q", stdout)
	}
	if stderr == "" {
		t.Error("stderr should explain the failure")
	}
}

// TestWorktreeAdd_PrintPath_ShellComposition is the milestone's
// explicit constraint: a Go-level string-return assertion doesn't
// exercise the shell-composition seam --print-path exists for. This
// runs `cd "$(aiwf worktree add ... --print-path)" && pwd` in a real
// subshell and asserts the landed directory.
func TestWorktreeAdd_PrintPath_ShellComposition(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	root, bin := setupInitedRepo(t)

	shCmd := "cd \"$(" + shq(bin) + " worktree add feature/shell-compose --root " + shq(root) + " --print-path)\" && pwd"
	cmd := exec.Command("sh", "-c", shCmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("shell composition failed: %v\n%s", err, out)
	}

	landed := strings.TrimSpace(string(out))
	want := filepath.Join(root, ".claude", "worktrees", "feature", "shell-compose")
	if resolved, evalErr := filepath.EvalSymlinks(want); evalErr == nil {
		want = resolved
	}
	if landed != want {
		t.Errorf("landed dir = %q, want %q", landed, want)
	}
}
