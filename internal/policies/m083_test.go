package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// loadADR0003 reads ADR-0003 through the loader, which resolves ids
// across the active tree and archive/ alike. Per .guidance/project/testing.md "substring
// assertions are not structural assertions" rule, every section-content
// claim asserted by AC-2 lives under a named heading, and the test
// extracts the section first.
func loadADR0003(t *testing.T) string {
	t.Helper()
	root, tr := sharedRepoTree(t)
	e := tr.ByID("ADR-0003")
	if e == nil {
		t.Fatal("ADR-0003 not found in tree (active or archive)")
	}
	data, err := os.ReadFile(filepath.Join(root, e.Path))
	if err != nil {
		t.Fatalf("reading ADR-0003 at %s: %v", e.Path, err)
	}
	return string(data)
}

// TestM083_AC2_ADR0003IDStorageSection asserts AC-2 (a):
//
// The ADR-0003 §"Id and storage" sub-section reads `F-NNNN` (was
// `F-NNN`), references the canonical-width family `G-NNNN` and
// `D-NNNN` rather than the narrow legacy forms, and cross-references
// ADR-0008 — the policy precedent — explaining the canonical-pad
// policy.
//
// Structural: the assertion targets the `### Id and storage`
// sub-section under `## Decision`, navigated via the markdown
// heading hierarchy. A flat substring grep would pass even if
// `F-NNNN` floated in an unrelated section; a section-scoped
// assertion fires only when the content lands in the right place.
func TestM083_AC2_ADR0003IDStorageSection(t *testing.T) {
	t.Parallel()
	body := loadADR0003(t)
	section := extractMarkdownSection(body, 3, "Id and storage")
	if section == "" {
		t.Fatal("AC-2(a): ADR-0003 must have a `### Id and storage` sub-section")
	}

	// Canonical width: F-NNNN must appear inside the section and
	// the narrow F-NNN form must not (the doc gets canonicalized
	// to F-NNNN per ADR-0008).
	if !regexp.MustCompile(`\bF-NNNN\b`).MatchString(section) {
		t.Error("AC-2(a): §Id and storage must declare the canonical 4-digit form `F-NNNN`")
	}
	// The narrow `F-NNN` form (i.e. F-NNN not followed by another
	// letter or digit) should be absent inside this section. Go's
	// regexp package has no lookahead; we approximate by matching
	// F-NNN followed by either a non-word non-N char or end-of-line.
	narrowFRE := regexp.MustCompile(`\bF-NNN(?:[^A-Za-z0-9_]|$)`)
	if narrowFRE.MatchString(section) {
		t.Error("AC-2(a): §Id and storage must not declare the narrow `F-NNN` form (sweep to F-NNNN per ADR-0008)")
	}

	// Composite-family reference must reflect the unified
	// canonical width: G-NNNN / D-NNNN, not the narrow G-NNN /
	// D-NNN. (The whole point of ADR-0008 is that every kind
	// canonicalizes to 4 digits; a doc that still says "same family
	// as G-NNN, D-NNN" contradicts the policy.)
	if !regexp.MustCompile(`\bG-NNNN\b`).MatchString(section) {
		t.Error("AC-2(a): §Id and storage must reference G-NNNN (the canonical family) when describing the family shape")
	}
	if !regexp.MustCompile(`\bD-NNNN\b`).MatchString(section) {
		t.Error("AC-2(a): §Id and storage must reference D-NNNN (the canonical family) when describing the family shape")
	}
	narrowGDRE := regexp.MustCompile(`\bG-NNN(?:[^A-Za-z0-9_]|$)|\bD-NNN(?:[^A-Za-z0-9_]|$)`)
	if narrowGDRE.MatchString(section) {
		t.Error("AC-2(a): §Id and storage must not declare the narrow G-NNN / D-NNN family shape (sweep to canonical per ADR-0008)")
	}

	// Cross-reference to ADR-0008 inside the section. The reference
	// can be a markdown link or plain text; both shapes count.
	if !regexp.MustCompile(`\bADR-0008\b`).MatchString(section) {
		t.Error("AC-2(a): §Id and storage must cross-reference ADR-0008 (the canonical-width policy precedent)")
	}
}
