package policies

import "testing"

// TestPolicy_M0211GuidanceOperatingAnchors pins M-0211/AC-2 on the live tree:
// every curated consumer-operating anchor is present in the shippable embedded
// guidance source, so no operating rule has drifted out of the shipped fragment
// (G-0313).
//
// The firing (non-vacuity) side — the reason this is not a vacuous chokepoint —
// is exercised by the m0211/* cases in TestFiringFixtures_MultiSite, which feed
// synthetic guidance (an anchor stripped; the file absent) and assert the policy
// returns a violation.
func TestPolicy_M0211GuidanceOperatingAnchors(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyM0211GuidanceOperatingAnchors)
}
