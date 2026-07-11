package stresstest

import (
	"os"
	"testing"

	"github.com/23min/aiwf/internal/testsupport"
)

// TestMain seeds the four GIT identity vars once at startup so test
// functions can call t.Parallel safely (os.Setenv not t.Setenv, which
// panics under parallel). See CLAUDE.md §"Test discipline".
//
// Serial tests: TestResolvePrebuiltBinary, TestSharedBinaryHelpers_PrebuiltEnvVar
// — both use t.Setenv, which panics under a parallel test. Every other
// Test* function in this package builds its own binary into its own
// t.TempDir() and shares no state.
func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	testsupport.HardenGitTestEnv()
	os.Exit(m.Run())
}
