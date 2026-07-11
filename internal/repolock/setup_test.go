package repolock

import (
	"os"
	"testing"

	"github.com/23min/aiwf/internal/testsupport"
)

// TestMain seeds GIT identity env vars once for the test binary's
// lifetime. os.Setenv (not t.Setenv) because t.Setenv panics under
// t.Parallel; the values are immutable for the lifetime of the
// test binary, so once-setup is correct.
//
// worktree_scoping_test.go (M-0241/AC-4) shells out to real git
// subprocesses (`git init`/`worktree add`), so this TestMain hardens
// the git test env per CLAUDE.md's exec-bearing-TestMain convention
// (G-0250/G-0251) — the rest of this package's tests don't invoke
// git themselves, but the hardening call is package-wide, not
// per-test.
//
// Serial tests: none. Every Test* function uses t.TempDir for
// filesystem isolation and only mutates files inside its own
// tempdir, so concurrent execution is safe.
func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	testsupport.HardenGitTestEnv()
	os.Exit(m.Run())
}
