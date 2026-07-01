package policies

import "testing"

// TestPolicy_M0210TrailerCommitDrift pins M-0210/AC-1 and AC-2 on the live
// tree: the trailered-commit prescription (a `git commit --trailer` block
// naming all three kernel-required trailer keys, plus the canonical
// variant-casings caveat) is present at both wrap rituals, and the caveat /
// `git config user.email` identity-resolution rule accompanies every
// trailered-commit block in the embedded ritual snapshot.
//
// The firing (non-vacuity) side — the reason this is not a vacuous chokepoint
// — is exercised by the m0210/* cases in TestFiringFixtures_MultiSite, which
// feed synthetic ritual trees with the prescription stripped and assert the
// policy returns a violation.
func TestPolicy_M0210TrailerCommitDrift(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyM0210TrailerCommitDrift)
}
