package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// adr0003Path is the canonical relative path to ADR-0003.
const adr0003Path = "docs/adr/ADR-0003-add-finding-f-nnn-as-a-seventh-entity-kind.md"

// loadADR0003 reads ADR-0003 from disk relative to the repo root.
// Per CLAUDE.md "substring assertions are not structural assertions"
// rule, every section-content claim asserted by AC-2 lives under a
// named heading, and the test extracts the section first.
func loadADR0003(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, adr0003Path))
	if err != nil {
		t.Fatalf("loading %s: %v", adr0003Path, err)
	}
	return string(data)
}

// loadCLAUDEMd reads CLAUDE.md from disk relative to the repo root.
// The file is plain markdown (not an aiwf entity), updated via
// standard commit; the structural test below scopes to the
// `## What aiwf commits to` section so the assertion fails on the
// right scope, not on a stray substring elsewhere in the file.
func loadCLAUDEMd(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("loading CLAUDE.md: %v", err)
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

// TestM083_AC2_CLAUDEMdCommitment2 asserts AC-2 (b):
//
// CLAUDE.md "What aiwf commits to" §2 reads as a single uniform rule
// (every kernel id is 4 digits) rather than the previous per-kind
// list of widths. The amended commitment mentions parser tolerance
// for legacy narrow widths and `aiwf rewidth` for migration; the
// per-kind enumeration `E-NN, M-NNN, ADR-NNNN, G-NNN, D-NNN, C-NNN`
// is gone.
//
// Structural: extract the `## What aiwf commits to` section then
// scope further to the second numbered item ("Stable ids ..."). A
// substring grep over the whole file would pass even if the rewrite
// landed in the wrong section.
func TestM083_AC2_CLAUDEMdCommitment2(t *testing.T) {
	t.Parallel()
	body := loadCLAUDEMd(t)
	section := extractMarkdownSection(body, 2, "What aiwf commits to")
	if section == "" {
		t.Fatal("AC-2(b): CLAUDE.md must have a `## What aiwf commits to` section")
	}

	// Locate the numbered item starting with `2. **Stable ids`.
	// The block starts at that line and ends at the next top-level
	// numbered item (`3. ...`) or the next `##` heading or EOF.
	startRE := regexp.MustCompile(`(?m)^2\. \*\*Stable ids[^*]*\*\*`)
	loc := startRE.FindStringIndex(section)
	if loc == nil {
		t.Fatal("AC-2(b): commitment §2 (`2. **Stable ids ...**`) not found in `## What aiwf commits to`")
	}
	rest := section[loc[0]:]
	endRE := regexp.MustCompile(`(?m)^(?:3\. |## )`)
	if end := endRE.FindStringIndex(rest); end != nil {
		rest = rest[:end[0]]
	}
	commitment := rest

	// Single uniform rule: must mention the canonical 4-digit form.
	if !regexp.MustCompile(`(?i)\b4[- ]?digit`).MatchString(commitment) {
		t.Error("AC-2(b): commitment §2 must state the canonical 4-digit width as a single uniform rule")
	}

	// Per-kind enumeration of the old narrow widths must be gone.
	// The previous form listed every kind verbatim:
	//   `E-NN`, `M-NNN`, `ADR-NNNN`, `G-NNN`, `D-NNN`, `C-NNN`.
	// We assert the narrow tokens are absent inside the commitment.
	for _, narrow := range []string{"E-NN", "M-NNN", "G-NNN", "D-NNN", "C-NNN"} {
		// Narrow tokens are matched as exact width — token not followed
		// by another letter / digit (so `E-NNNN` doesn't match `E-NN`).
		// Go's regexp has no lookahead; approximate with non-word char
		// or end-of-line.
		pat := regexp.MustCompile(`\b` + regexp.QuoteMeta(narrow) + `(?:[^A-Za-z0-9_]|$)`)
		if pat.MatchString(commitment) {
			t.Errorf("AC-2(b): commitment §2 must not enumerate the narrow legacy width %q (per-kind list collapsed to single rule)",
				narrow)
		}
	}

	// Parser-tolerance note: the rule must mention that legacy
	// narrow widths still parse on input. Phrasing flexible — match
	// any of the canonical phrasings.
	tolerantRE := regexp.MustCompile(`(?i)(parser|tolerat|legacy|narrow)`)
	if !tolerantRE.MatchString(commitment) {
		t.Error("AC-2(b): commitment §2 must mention that parsers tolerate narrower legacy widths on input")
	}

	// `aiwf rewidth` migration mention: the verb name must appear so
	// readers can find the migration path from the principle alone.
	if !regexp.MustCompile(`\baiwf rewidth\b`).MatchString(commitment) {
		t.Error("AC-2(b): commitment §2 must reference `aiwf rewidth` as the migration verb for legacy trees")
	}
}
