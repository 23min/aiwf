package policies

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
)

// TestPolicy_ThisRepoDriftCheckClean asserts `aiwf check` against this
// repo's own active tree produces zero `entity-id-narrow-width`
// findings.
//
// The claim is stronger than it was under the rule's earlier
// tree-state form, which fired only when narrow and canonical widths
// coexisted: every active id is now required to be canonical outright,
// so this asserts the whole active tree, not the absence of a mix.
//
// A failure means an active entity's filename carries a narrow id.
// Since no verb can produce one, the cause is a hand-edit or a file
// move — the table-driven fixtures in
// internal/check/entity_id_narrow_width_test.go should catch a
// regression in the rule itself first.
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
