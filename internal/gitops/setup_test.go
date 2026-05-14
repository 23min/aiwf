package gitops

import (
	"os"
	"testing"
)

// TestMain seeds GIT identity env vars once for the test binary's
// lifetime. os.Setenv (not t.Setenv) because t.Setenv panics under
// t.Parallel; the values are immutable for the lifetime of the
// test binary, so once-setup is correct.
//
// Replaces the prior `gitTestEnv(t)` and `initTestRepo` t.Setenv
// blocks — both incompatible with t.Parallel adoption per M-0091.
//
// Serial tests: none. Every Test* function uses t.TempDir for
// filesystem isolation; git invocations are separate processes
// with their own cwd, so concurrent execution is safe.
func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	os.Exit(m.Run())
}
