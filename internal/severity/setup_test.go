package severity

import (
	"os"
	"testing"

	"github.com/23min/aiwf/internal/testsupport"
)

// TestMain seeds GIT identity env vars once for the test binary's
// lifetime. os.Setenv (not t.Setenv) because t.Setenv panics under
// t.Parallel.
//
// Serial tests:
//   - TestLoad_EmptyRootDoesNotReadTheProcessWorkingDirectory — t.Chdir,
//     which mutates process-wide state and panics under t.Parallel.
//
// Every other Test* function builds its fixtures in a t.TempDir or in
// memory, so concurrent execution is safe.
func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	testsupport.HardenGitTestEnv()
	os.Exit(m.Run())
}
