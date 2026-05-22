package policies

import (
	"regexp"
	"strings"
	"testing"
)

// TestM0130_AC6_AuditCatalogReflectsImplementation pins M-0130/AC-6:
// once `fsm-history-consistent` lands, the audit catalog's R-RULE-149
// row and §10.1 enforcement-status legend must drop their
// "currently unimplemented" / "pending G-013N" qualifiers and reflect
// the per-subcode severity that actually ships (illegal-transition +
// forced-untrailered = error; manual-edit = warning).
//
// Structurally scoped per CLAUDE.md §"Substring assertions are not
// structural assertions": the negative assertions run against the
// extracted R-RULE-149 row and §10.1 legend prose, not the whole
// catalog. The positive assertions name the implementing milestone
// (M-0130) and closing gap (G-0132) so any future revert that simply
// re-adds the "unimplemented" qualifier without also removing the
// implementation reference fails on the positive bar.
func TestM0130_AC6_AuditCatalogReflectsImplementation(t *testing.T) {
	t.Parallel()
	body := loadAuditCatalog(t)

	t.Run("R-RULE-149 row", func(t *testing.T) {
		t.Parallel()
		row := extractRuleRow(t, body, "R-RULE-149")

		// Negative bar: stale "unimplemented" / "pending" language is gone.
		stalePhrases := []string{
			"currently unimplemented",
			"currently *unimplemented*",
			"pending G-0130",
			"pending G-0132",
			"Until G-0130",
			"Until G-0132",
			"tracked as **G-0130**",
			"tracked as G-0130",
			"target severity once implemented",
			"none yet",
		}
		for _, phrase := range stalePhrases {
			if containsCaseInsensitive(row, phrase) {
				t.Errorf("R-RULE-149 row still contains stale phrase %q — AC-6 requires the implementation qualifiers to be removed", phrase)
			}
		}

		// Positive bar: row names the implementing milestone, closing gap,
		// and the disjoint per-subcode severity reality.
		mustContain := []string{
			"M-0130",                   // implementing milestone
			"G-0132",                   // closing gap
			"`illegal-transition`",     // subcode 1
			"`forced-untrailered`",     // subcode 2
			"`manual-edit`",            // subcode 3
			"`fsm-history-consistent`", // check rule name
			"internal/check/fsm_history_consistent.go", // chokepoint path
		}
		for _, phrase := range mustContain {
			if !strings.Contains(row, phrase) {
				t.Errorf("R-RULE-149 row missing required reference %q — AC-6 requires the row to reflect the shipped reality", phrase)
			}
		}

		// Severity reality: row must encode per-subcode severity, not a
		// single monolithic "hard-reject" claim that would mis-state
		// manual-edit's warning level.
		if !regexp.MustCompile(`(?is)illegal-transition.*?error`).MatchString(row) {
			t.Errorf("R-RULE-149 row does not state illegal-transition = error")
		}
		if !regexp.MustCompile(`(?is)forced-untrailered.*?error`).MatchString(row) {
			t.Errorf("R-RULE-149 row does not state forced-untrailered = error")
		}
		if !regexp.MustCompile(`(?is)manual-edit.*?warning`).MatchString(row) {
			t.Errorf("R-RULE-149 row does not state manual-edit = warning — severity reconciliation is a load-bearing part of AC-6")
		}
	})

	t.Run("§10.1 enforcement-status legend", func(t *testing.T) {
		t.Parallel()
		section := sectionBody(body, "### 10.1 Entity FSM transitions")
		if section == "" {
			t.Fatal("§10.1 section not found in audit catalog")
		}
		// Scope to the prose before the rule table header (`| Rule id |`)
		// so the structural assertion doesn't drift into row content.
		legend := section
		if idx := strings.Index(section, "| Rule id |"); idx != -1 {
			legend = section[:idx]
		}

		stalePhrases := []string{
			"pending G-0130",
			"pending G-0132",
			"Until G-0130",
			"Until G-0132",
			"target, pending",
			"only one of which is active today",
			"target severity",
		}
		for _, phrase := range stalePhrases {
			if containsCaseInsensitive(legend, phrase) {
				t.Errorf("§10.1 legend still contains stale phrase %q — AC-6 requires the legend to reflect the active history-walk chokepoint", phrase)
			}
		}

		// Positive bar: legend names the implementation and all three
		// subcodes with their actual severities.
		mustContain := []string{
			"M-0130",
			"G-0132",
			"`fsm-history-consistent`",
			"`illegal-transition`",
			"`forced-untrailered`",
			"`manual-edit`",
			"both now active", // legend's new framing
		}
		for _, phrase := range mustContain {
			if !strings.Contains(legend, phrase) {
				t.Errorf("§10.1 legend missing required reference %q — AC-6 requires the legend to reflect the shipped reality", phrase)
			}
		}
	})
}

// extractRuleRow returns the single markdown table row whose first cell
// holds the given rule id (e.g. "R-RULE-149"). Scopes substring
// assertions to a single row so generic phrases don't leak across the
// catalog.
func extractRuleRow(t *testing.T, body, ruleID string) string {
	t.Helper()
	// Match the row prefix exactly: leading pipe, optional whitespace,
	// then the rule id followed by ` |` to avoid prefix collisions
	// (e.g., R-RULE-149 vs R-RULE-1490 — hypothetical, but cheap).
	prefix := "| " + ruleID + " |"
	idx := strings.Index(body, prefix)
	if idx == -1 {
		t.Fatalf("rule row %s not found in audit catalog", ruleID)
	}
	rest := body[idx:]
	end := strings.IndexByte(rest, '\n')
	if end == -1 {
		return rest
	}
	return rest[:end]
}
