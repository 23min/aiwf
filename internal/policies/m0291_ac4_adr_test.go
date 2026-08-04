package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// loadADR0040 reads ADR-0040 by resolving its id through the loader,
// per CLAUDE.md §"Policy tests that read entity files resolve via the
// loader" — a hardcoded path breaks on a retitle or an archive sweep.
func loadADR0040(t *testing.T) string {
	t.Helper()
	root, tr := sharedRepoTree(t)
	e := tr.ByID("ADR-0040")
	if e == nil {
		t.Fatal("ADR-0040 not found in tree (active or archive)")
	}
	data, err := os.ReadFile(filepath.Join(root, e.Path))
	if err != nil {
		t.Fatalf("reading ADR-0040 at %s: %v", e.Path, err)
	}
	return string(data)
}

// TestM0291AC4_ADRRecordsTheTwoRouteStance is M-0291/AC-4.
//
// The claims are asserted inside the named section that must carry
// them, not by a flat grep: per CLAUDE.md §"Substring assertions are
// not structural assertions", these phrases are short enough to appear
// incidentally elsewhere in the document, and a decision stated only in
// the Context section has not been decided.
func TestM0291AC4_ADRRecordsTheTwoRouteStance(t *testing.T) {
	t.Parallel()
	doc := loadADR0040(t)

	if !strings.Contains(doc, "status: accepted") {
		t.Error("ADR-0040 is not at status accepted; AC-4 requires a ratified record, not a proposal")
	}

	decision := extractMarkdownSection(doc, 2, "Decision")
	if decision == "" {
		t.Fatal("ADR-0040 has no ## Decision section")
	}

	// The seam. Naming it is the whole point: the surfaces this epic
	// corrects were wrong precisely because they named a chokepoint that
	// did not exist.
	if !strings.Contains(decision, "verb.Apply") {
		t.Error("## Decision does not name verb.Apply as the seam that refuses")
	}

	// Both halves of the stance. Recording only the prevention half
	// would misstate it as a closed door rather than a gated one.
	for _, claim := range []string{"verb route", "history route"} {
		if !strings.Contains(decision, claim) {
			t.Errorf("## Decision does not name the %q; the stance has two halves and both are load-bearing", claim)
		}
	}

	// Why one seam suffices where the HEAD-divergence ADR needed two.
	// Without this the reader has a rule with no way to tell whether the
	// single seam is a considered choice or an oversight.
	context := extractMarkdownSection(doc, 2, "Context")
	if context == "" {
		t.Fatal("ADR-0040 has no ## Context section")
	}
	if !strings.Contains(context+decision, "ADR-0038") {
		t.Error("neither ## Context nor ## Decision relates the single seam to ADR-0038's two-seam pattern")
	}

	consequences := extractMarkdownSection(doc, 2, "Consequences")
	if consequences == "" {
		t.Fatal("ADR-0040 has no ## Consequences section")
	}
	// The two things a reader meets by surprise otherwise.
	//
	// The seam refuses only the force-predicated rules, which reads as an
	// arbitrary subset until the reason is stated: a verb outside the
	// provenance-decoration layer cannot satisfy the others by any
	// invocation, so enforcing them there is a closed door rather than a
	// rule. And the refusal exits as a legality refusal rather than an
	// internal failure, which is what a consumer routing on the exit code
	// needs to know and what the ADR would otherwise leave them to
	// discover by running it.
	if !strings.Contains(consequences, "force trailer") {
		t.Error("## Consequences does not record that the seam enforces only the force-predicated rules")
	}
	if !strings.Contains(consequences, "legality refusal") {
		t.Error("## Consequences does not record the refusal's exit class")
	}
}
