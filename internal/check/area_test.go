package check

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// TestRun_AreaTaggedEntitiesAreInert pins AC-4 of M-0171: with no areas block
// (Run takes no config — the checker is area-agnostic today), the `area` field
// is inert. The assertion is metamorphic: two trees identical except for area
// values on the root kinds must produce exactly the same findings, and none
// area-related. It passes by construction now (no rule reads area) and diverges
// the instant a future rule does — which guards the next milestone's
// area-unknown finding against firing when no block is declared (present-but-
// no-block is never flagged, per E-0043).
func TestRun_AreaTaggedEntitiesAreInert(t *testing.T) {
	t.Parallel()
	mk := func(area string) []Finding {
		return Run(makeTree(
			&entity.Entity{ID: "E-0001", Kind: entity.KindEpic, Title: "Plat", Status: "active", Area: area},
			&entity.Entity{
				ID: "M-0001", Kind: entity.KindMilestone, Title: "Cache", Status: "draft", Parent: "E-0001",
				ACs: []entity.AcceptanceCriterion{{ID: "AC-1", Title: "x", Status: "open"}},
			},
			&entity.Entity{ID: "G-0001", Kind: entity.KindGap, Title: "Leak", Status: "open", Area: area},
		), nil)
	}
	tagged := mk("some-undeclared-area")
	untagged := mk("")

	// Inert: area tagging neither adds nor removes findings.
	if len(tagged) != len(untagged) {
		t.Fatalf("area tagging changed finding count: tagged=%d untagged=%d\ntagged=%+v", len(tagged), len(untagged), tagged)
	}
	// And no finding is area-related, even with an undeclared value present.
	for _, f := range tagged {
		if strings.Contains(strings.ToLower(f.Code+" "+f.Message), "area") {
			t.Errorf("unexpected area-related finding with no areas block: %+v", f)
		}
	}
}
