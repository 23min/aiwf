package verb_test

import (
	"os"
	"testing"
)

// TestMain seeds GIT identity env vars once for the test binary's
// lifetime. os.Setenv (not t.Setenv) because t.Setenv panics under
// t.Parallel; the values are immutable for the lifetime of the
// test binary, so once-setup is correct.
//
// Replaces the prior `newApplyTestRepo` / `newRunner` / per-test
// t.Setenv blocks — all incompatible with t.Parallel adoption per
// M-0091.
//
// Serial tests:
//   - TestApply_RollsBackOnCommitFailure (apply_test.go) — deliberately
//     clears GIT_{AUTHOR,COMMITTER}_{NAME,EMAIL} and overrides
//     GIT_CONFIG_GLOBAL/SYSTEM to provoke a commit failure; this is
//     fundamentally an env-mutating test and cannot run in parallel.
func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	os.Exit(m.Run())
}
