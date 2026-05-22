package policies

import (
	"regexp"
	"testing"

	"github.com/23min/aiwf/internal/workflows/spec"
)

// TestM0123_AC3_AntiRulesCount asserts AntiRules() returns exactly 12
// entries (Pass B §10's eleven R-FP-0166..R-FP-0176 anti-rules plus Q10's
// zero-milestone-active addition). The list is closed-set; new anti-rules
// land via spec amendment commits, not silent growth.
func TestM0123_AC3_AntiRulesCount(t *testing.T) {
	t.Parallel()

	got := len(spec.AntiRules())
	if got != 12 {
		t.Fatalf("spec.AntiRules() count: want 12, got %d", got)
	}
}

// TestM0123_AC3_AntiRuleIDShape asserts every entry's ID matches the
// canonical ANTI-NNNN four-digit shape (ADR-0008 width discipline).
func TestM0123_AC3_AntiRuleIDShape(t *testing.T) {
	t.Parallel()

	re := regexp.MustCompile(`^ANTI-\d{4}$`)
	for i, ar := range spec.AntiRules() {
		if !re.MatchString(ar.ID) {
			t.Errorf("AntiRules()[%d]: ID=%q does not match ANTI-NNNN shape", i, ar.ID)
		}
	}
}

// TestM0123_AC3_AntiRuleIDsUnique asserts no two anti-rules share an ID.
// Duplicate IDs would break any cross-reference to a specific anti-rule.
func TestM0123_AC3_AntiRuleIDsUnique(t *testing.T) {
	t.Parallel()

	seen := map[string]int{}
	for i, ar := range spec.AntiRules() {
		if prev, ok := seen[ar.ID]; ok {
			t.Errorf("AntiRules()[%d] and AntiRules()[%d] share ID %q", prev, i, ar.ID)
		}
		seen[ar.ID] = i
	}
}

// TestM0123_AC3_AntiRuleStatementNonEmpty asserts every entry's Statement
// field is non-empty. The Statement is the load-bearing surface — an empty
// one would silently degrade the catalog to an ID list.
func TestM0123_AC3_AntiRuleStatementNonEmpty(t *testing.T) {
	t.Parallel()

	for i, ar := range spec.AntiRules() {
		if ar.Statement == "" {
			t.Errorf("AntiRules()[%d] (ID=%q): Statement is empty", i, ar.ID)
		}
	}
}

// TestM0123_AC3_AntiRuleReasoningNonEmpty asserts every entry's Reasoning
// field is non-empty. Without reasoning, a future contributor can't tell
// whether the anti-rule is load-bearing or vestigial.
func TestM0123_AC3_AntiRuleReasoningNonEmpty(t *testing.T) {
	t.Parallel()

	for i, ar := range spec.AntiRules() {
		if ar.Reasoning == "" {
			t.Errorf("AntiRules()[%d] (ID=%q): Reasoning is empty", i, ar.ID)
		}
	}
}
