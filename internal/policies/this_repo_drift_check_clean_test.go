package policies

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
)

// TestPolicy_ThisRepoDriftCheckClean is the M-083 AC-5 chokepoint:
// running `aiwf check` against this repo's active tree (post-M-082
// uniform-canonical) produces zero findings of code
// `entity-id-narrow-width`.
//
// This is the load-bearing assertion for the M-083 milestone outcome:
// the rule fires only on mixed state; M-082's `aiwf rewidth --apply`
// made the active tree uniform-canonical; therefore the rule is
// silent on this repo.
//
// If this assertion fails, it indicates either:
//
//   - The rule's tree-state computation is wrong (regression in
//     M-083/AC-1; the table-driven fixture tests in
//     internal/check/entity_id_narrow_width_test.go should catch it
//     first).
//   - M-082's apply step missed an active-tree file (regression in
//     M-082; the rewidth verb's idempotence test in
//     internal/verb/rewidth_test.go and M-082 AC-5 should catch it
//     first).
//
// Per CLAUDE.md "framework correctness must not depend on the LLM's
// behavior," AC-5's discipline lives in this test, not in reviewer
// recall.
func TestPolicy_ThisRepoDriftCheckClean(t *testing.T) {
	t.Parallel()
	_, tr := sharedRepoTree(t)
	loadErrs := sharedRepoTreeLoadErrs(t)
	findings := check.Run(tr, loadErrs)
	var unwanted []check.Finding
	for _, f := range findings {
		if f.Code == "entity-id-narrow-width" {
			unwanted = append(unwanted, f)
		}
	}
	if len(unwanted) > 0 {
		var lines []string
		for _, f := range unwanted {
			lines = append(lines, "  "+f.Code+": "+f.Message+" ("+f.EntityID+" at "+f.Path+")")
		}
		t.Errorf("AC-5: %d entity-id-narrow-width findings on this repo's tree (regression in M-082 apply step or M-083 rule logic):\n%s",
			len(unwanted), strings.Join(lines, "\n"))
	}
}
