package worktree_test

import (
	"os"
	"testing"

	"github.com/23min/aiwf/internal/testsupport"
)

// TestMain seeds GIT identity env vars once for the test binary's
// lifetime. os.Setenv (not t.Setenv) because t.Setenv panics under
// t.Parallel. HardenGitTestEnv scrubs inherited git-locator env vars
// and disables auto-gc — this package's tests spin up real git repos
// under t.TempDir() in parallel (G-0250/G-0251 flake classes).
//
// Serial tests: TestRun_GitFailureSurfacesDirectly,
// TestRun_BaseRejectedForExistingBranch, TestRun_MissingAiwfYamlInNewWorktree,
// TestRun_HookConflictReturnsExitFindings, TestRun_PrintPath_UnitLevel,
// TestRun_JSONSuccessEnvelope, and TestRun_PrintPathAndJSONMutuallyExclusive
// use testutil.CaptureRun/CaptureStdout, which swap process-global
// os.Stdout/os.Stderr — incompatible with t.Parallel.
func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	testsupport.HardenGitTestEnv()
	os.Exit(m.Run())
}
