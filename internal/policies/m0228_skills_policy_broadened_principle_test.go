package policies

import "testing"

// TestPolicy_M0228SkillsPolicyBroadenedPrinciple runs the section-scoped
// assertion against the real CLAUDE.md. Green once the §"Skills policy"
// paragraph names the full shipped-surface list and the content class.
// The synthetic firing fixtures (missing file, missing section, missing
// markers) live in firing_fixtures_multi_site_test.go — they cover the
// policy's dark construction sites and the not-green branches.
func TestPolicy_M0228SkillsPolicyBroadenedPrinciple(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyM0228SkillsPolicyBroadenedPrinciple)
}
