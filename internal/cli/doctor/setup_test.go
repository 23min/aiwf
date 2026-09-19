package doctor_test

import (
	"os"
	"testing"

	"github.com/23min/aiwf/internal/testsupport"
)

// TestMain seeds GIT identity once at startup so tests can run with
// t.Parallel() without t.Setenv panics.
// TestDoctor_ConfiguredHostWithoutExecutableReportsRemediation stays serial:
// it isolates process PATH with t.Setenv.
func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	testsupport.HardenGitTestEnv()
	// Claude setup fixtures require command presence; selection tests isolate PATH.
	os.Exit(testsupport.RunWithClaudeOnPATH(m.Run))
}
