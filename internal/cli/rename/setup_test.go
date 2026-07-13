package rename_test

import (
	"os"
	"testing"

	"github.com/23min/aiwf/internal/testsupport"
)

// Serial tests (do NOT call t.Parallel):
//   - TestRun_ResolveActorFailure (rename_error_paths_test.go) — via
//     testutil.BrokenGitIdentity, uses t.Setenv (HOME, XDG_CONFIG_HOME,
//     GIT_CONFIG_NOSYSTEM), which panics under t.Parallel.

func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	testsupport.HardenGitTestEnv()
	os.Exit(m.Run())
}
