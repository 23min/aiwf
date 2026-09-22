package update_test

import (
	"os"
	"testing"

	"github.com/23min/aiwf/internal/testsupport"
)

// Serial tests in this package (must NOT call t.Parallel):
//   - TestRefreshStatuslineInPlace_PrintsLedgerForUnmarkedCopy
//     (refresh_statusline_test.go): swaps $HOME via t.Setenv and captures
//     os.Stdout — both process-globals.
//   - TestRun_ProjectGuidanceFailuresPreserveInstallationAndContinue,
//     TestRun_ProjectGuidanceDisabledPreservesInstallationWithoutRetrieval, and
//     TestRun_ProjectGuidanceRemovalAndInterruptedRetry capture os.Stdout;
//     these also isolate personal-guidance environment variables;
//     the disabled-maintenance test sets GIT_TRACE.
//   - TestRun_ProjectGuidanceTracksUpstreamAndConverges isolates personal settings.
//   - TestRun_ProjectGuidanceHandover isolates personal settings and captures stdout.
func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	testsupport.HardenGitTestEnv()
	// Claude setup fixtures require command presence; selection tests isolate PATH.
	os.Exit(testsupport.RunWithClaudeOnPATH(m.Run))
}
