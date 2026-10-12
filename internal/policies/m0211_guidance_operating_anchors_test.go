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

// TestPolicyM0211GuidanceOperatingAnchors_UnreadableSourceStatesTheFix pins
// M-0337 AC-2: an unreadable guidance source is told what to restore.
func TestPolicyM0211GuidanceOperatingAnchors_UnreadableSourceStatesTheFix(t *testing.T) {
	t.Parallel()
	vs, err := PolicyM0211GuidanceOperatingAnchors(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	mustHaveViolation(t, vs, "restore the file, or repoint this policy at the guidance source's new path")
}
