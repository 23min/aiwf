package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// sovereignActsSection is the audit catalogue's own heading for the
// rules covering provenance and sovereign acts.
const sovereignActsSection = "10.5 Provenance + sovereign acts"

// TestM0324_AC5_CatalogueNamesEverySovereignTransition asserts the
// audit catalogue's sovereign-acts section names every transition in
// `entity.SovereignActShapes()`.
//
// The expectation is built from the closed set rather than written
// out, which is what makes this a relationship between two artefacts
// instead of a phrase pin: adding an entry without describing it in
// the catalogue turns this red and names the transition that went
// undocumented, and rewording a row around the same transition keeps
// it green.
//
// What it does NOT assert is that the rows are *correct* about those
// transitions. R-RULE-001's Note has been false since before this
// milestone — it requires `--force --reason` for an edge a human
// reaches with no flag — and no derivable check catches that. Content
// correctness in the catalogue stays held at review, as it is
// elsewhere in this repo.
func TestM0324_AC5_CatalogueNamesEverySovereignTransition(t *testing.T) {
	t.Parallel()

	section := sovereignActsSectionText(t)
	// Backticks are stripped so a row may mark the state names as code
	// or not; the assertion is about which transition is described,
	// not about how it is typeset.
	haystack := strings.ReplaceAll(section, "`", "")

	shapes := entity.SovereignActShapes()
	if len(shapes) == 0 {
		t.Fatal("the sovereign closed set is empty; this test has no subject and is passing vacuously")
	}
	for _, s := range shapes {
		transition := string(s.From) + " → " + string(s.To)
		if !strings.Contains(haystack, transition) {
			t.Errorf("the catalogue's %q section never names the %s transition %q, which is in the "+
				"kernel's sovereign closed set — a reader following the catalogue would not learn "+
				"that a human is required to make it", sovereignActsSection, s.Kind, transition)
		}
	}
}

// sovereignActsSectionText returns the catalogue's sovereign-acts
// section, failing the test when the heading has moved rather than
// silently searching an empty string — a locator that resolves to
// nothing would make every assertion above pass for the wrong reason.
func sovereignActsSectionText(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "docs", "design", "legal-workflows-audit.md"))
	if err != nil {
		t.Fatalf("reading the audit catalogue: %v", err)
	}
	section := extractMarkdownSection(string(data), 3, sovereignActsSection)
	if strings.TrimSpace(section) == "" {
		t.Fatalf("the audit catalogue has no `### %s` section; the locator is stale", sovereignActsSection)
	}
	return section
}
