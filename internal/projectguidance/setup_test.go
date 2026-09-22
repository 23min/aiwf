package projectguidance

import (
	"os"
	"testing"

	"github.com/23min/aiwf/internal/testsupport"
)

// Serial tests use t.Setenv to configure the Git process boundary:
// TestRetrieve_GitConfigurationWithoutCheckout, TestRetrieve_CleanupFailureIsReported,
// TestRetrieve_CancelDuringClone, TestCheckCompatibility_InspectsInstalledChecker,
// and TestCheckCompatibility_MissingCheckerDetectsLegacyDelivery. Other tests own separate repositories.
func TestMain(m *testing.M) {
	os.Setenv("GIT_AUTHOR_NAME", "aiwf-test")
	os.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	os.Setenv("GIT_COMMITTER_NAME", "aiwf-test")
	os.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
	testsupport.HardenGitTestEnv()
	os.Exit(m.Run())
}
