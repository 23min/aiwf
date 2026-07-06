package worktree_test

import (
	"os"
	"testing"
)

// TestMain seeds GIT identity env vars once for the test binary's
// lifetime. os.Setenv (not t.Setenv) because t.Setenv panics under
// t.Parallel.
//
// Serial tests: TestRun_GitFailureSurfacesDirectly,
// TestRun_BaseRejectedForExistingBranch, TestRun_MissingAiwfYamlInNewWorktree,
// TestRun_HookConflictReturnsExitFindings, TestRun_PrintPath_UnitLevel, and
// TestRun_JSONSuccessEnvelope use testutil.CaptureRun/CaptureStdout, which
// swap process-global os.Stdout/os.Stderr — incompatible with t.Parallel.
func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	os.Exit(m.Run())
}
