package policies

import (
	"strings"
	"testing"
)

// TestM0131_AC3_AuditCatalogReflectsImplementation pins M-0131/AC-3:
// once state-aware CancelTarget for Contract lands (M-0131/AC-1 + AC-2),
// the audit catalog's R-RULE-021 row must drop its "code bug" /
// "current code is not state-aware" qualifiers and the stale gap-id
// (G-0129; reallocated to G-0131 mid-flight). The rule's *statement*
// (the (kind, currentStatus) -> terminal table) stays — the code now
// matches the statement.
//
// Structurally scoped per CLAUDE.md §"Substring assertions are not
// structural assertions": negative assertions run against the
// extracted R-RULE-021 row, not the whole catalog. Positive
// assertions name the implementing milestone (M-0131) and the
// closing gap (G-0131) so a future revert that re-adds the "code
// bug" qualifier without also removing the closer reference fails
// on the positive bar.
func TestM0131_AC3_AuditCatalogReflectsImplementation(t *testing.T) {
	t.Parallel()
	body := loadAuditCatalog(t)
	row := extractRuleRow(t, body, "R-RULE-021")

	// Negative bar: stale "code bug" / "not state-aware" / G-0129
	// language is gone. M-0131 makes the code state-aware; the
	// catalog must not still claim the code is buggy.
	stalePhrases := []string{
		"Code bug",
		"code bug",
		"current code is not state-aware",
		"is *not* state-aware",
		"is not state-aware",
		"returns flat `rejected`",
		"returns flat rejected",
		"tracked as G-0129",
		"tracked as **G-0129**",
		"G-0129",
		"M-0127", // the milestone's prior id (reallocated to M-0131)
	}
	for _, phrase := range stalePhrases {
		if containsCaseInsensitive(row, phrase) {
			t.Errorf("R-RULE-021 row still contains stale phrase %q — AC-3 requires the code-bug qualifiers to be removed once M-0131 lands the state-aware CancelTarget", phrase)
		}
	}

	// Positive bar: the row names the implementing milestone and
	// closing gap so a future revert is caught by both ends.
	mustContain := []string{
		"M-0131", // implementing milestone
		"G-0131", // closing gap (post-reallocation)
	}
	for _, phrase := range mustContain {
		if !strings.Contains(row, phrase) {
			t.Errorf("R-RULE-021 row missing required reference %q — AC-3 requires the row to name the implementing milestone and closing gap", phrase)
		}
	}

	// The rule's statement itself must still encode the state-
	// aware truth — that's the load-bearing claim the row asserts.
	mustContainStatement := []string{
		"Contract `deprecated` → `retired`",
	}
	for _, phrase := range mustContainStatement {
		if !strings.Contains(row, phrase) {
			t.Errorf("R-RULE-021 row missing the state-aware statement %q — the deprecated→retired mapping is the rule's substance, not just a Notes-column artifact", phrase)
		}
	}
}
