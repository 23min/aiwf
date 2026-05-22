package policies

import (
	"os"
	"testing"
)

// TestMain seeds GIT identity env vars once for the test binary's
// lifetime. os.Setenv (not t.Setenv) because t.Setenv panics under
// t.Parallel; the values are immutable for the lifetime of the
// test binary, so once-setup is correct.
//
// Policies tests run against the live repo tree (memoized via
// sharedRepoTree, see shared_tree_test.go per M-0091/AC-4) and a
// handful of fixtures under testdata/. Some tests shell out to git
// (e.g. log queries for the design-doc anchor and SHA-recording
// audits); a stable identity keeps those reproducible.
//
// Serial tests: none. Every Test* function reads-only against the
// shared *Tree (do not mutate) or uses t.TempDir for fixture work.
func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	// Clear ambient git locator env vars that a parent git hook
	// invocation passes down. Tests under this package may shell out
	// to `git init` in a t.TempDir (e.g. M-0124's per-cell positive
	// driver via cellcoverage); inherited GIT_DIR/GIT_WORK_TREE etc.
	// would steer those into the parent repo's gitdir and produce
	// config-lock contention against the worktree's parent
	// .git/config. Clearing here keeps the test binary insulated
	// from the invoking shell's git context — same hygiene as a
	// fresh subshell.
	os.Unsetenv("GIT_DIR")
	os.Unsetenv("GIT_WORK_TREE")
	os.Unsetenv("GIT_COMMON_DIR")
	os.Unsetenv("GIT_INDEX_FILE")
	os.Unsetenv("GIT_OBJECT_DIRECTORY")
	os.Exit(m.Run())
}
