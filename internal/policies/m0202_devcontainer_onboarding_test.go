package policies

import "testing"

// TestPolicy_M0202DevcontainerOnboarding pins M-0202/AC-1: the live
// .devcontainer/init.sh banner and README.md carry no retired
// plugin-install instruction and the banner points at `aiwf doctor`'s
// `rituals:` line. The firing (non-vacuity) side is exercised by the
// m0202-onboarding/* cases in TestFiringFixtures_MultiSite.
func TestPolicy_M0202DevcontainerOnboarding(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyM0202DevcontainerOnboarding)
}
