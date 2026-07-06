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
func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	testsupport.HardenGitTestEnv()
	os.Exit(m.Run())
}
