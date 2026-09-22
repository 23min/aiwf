package initrepo

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
// Replaces the prior `freshGitRepo` t.Setenv block — incompatible
// with t.Parallel adoption per M-0091.
//
// Serial: TestInit_InstallsExplicitProjectGuidance,
// TestProjectGuidanceRefresh_ReportsFailuresWithoutClaimingInstallation, and
// TestProjectGuidanceRefresh_OptOutAndDryRun isolate personal-guidance environment
// variables. Other tests own separate repositories.
func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	testsupport.HardenGitTestEnv()
	// Claude setup fixtures require command presence; selection tests isolate PATH.
	os.Exit(testsupport.RunWithClaudeOnPATH(m.Run))
}
