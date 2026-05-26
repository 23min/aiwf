package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// loadADR0013 resolves ADR-0013 (global-precondition representation;
// out-of-scope legality classification) through the loader — never a
// hardcoded path, per CLAUDE.md *Testing* §"Policy tests that read
// entity files must resolve via the loader" — so the test survives
// rename and archive sweeps. Returns the raw file contents
// (frontmatter + body).
func loadADR0013(t *testing.T) string {
	t.Helper()
	root, tr := sharedRepoTree(t)
	e := tr.ByID("ADR-0013")
	if e == nil {
		t.Fatal("ADR-0013 not found in tree (active or archive)")
	}
	data, err := os.ReadFile(filepath.Join(root, e.Path))
	if err != nil {
		t.Fatalf("reading ADR-0013 at %s: %v", e.Path, err)
	}
	return string(data)
}

// assertDecisionSubsection asserts a `### <name>` subsection under
// `## Decision` exists, carries real prose, and contains each required
// identifier literal and prose word — every assertion scoped to the
// subsection per CLAUDE.md *Testing* §"Substring assertions are not
// structural assertions", so a literal floating in another section
// does not satisfy the AC.
func assertDecisionSubsection(t *testing.T, body, name string, identifiers, proseWords []string) {
	t.Helper()
	decision := extractMarkdownSection(body, 2, "Decision")
	if decision == "" {
		t.Fatal("ADR-0013 must have a `## Decision` section")
	}
	section := extractSubsection(decision, name)
	if section == "" {
		t.Fatalf("`### %s` subsection missing under `## Decision`", name)
	}
	if !hasNonEmptyProse(section) {
		t.Fatalf("`### %s` subsection is empty / placeholder only", name)
	}
	for _, lit := range identifiers {
		if !strings.Contains(section, lit) {
			t.Errorf("`### %s` subsection must contain identifier %q", name, lit)
		}
	}
	lower := strings.ToLower(section)
	for _, w := range proseWords {
		if !strings.Contains(lower, w) {
			t.Errorf("`### %s` subsection must convey %q", name, w)
		}
	}
}

// TestADR0013_M0144_AC1_GlobalRuleRepresentation is the mechanical
// evidence for M-0144/AC-1: ADR-0013 resolves via the loader, is
// ratified (`status: accepted`), and its `## Decision` names the
// global-rule representation mechanism (the `Global` flag) and how it
// composes with the key-uniqueness + coverage meta-tests and the AC-5
// fourth arm.
func TestADR0013_M0144_AC1_GlobalRuleRepresentation(t *testing.T) {
	t.Parallel()
	body := loadADR0013(t)

	if !regexp.MustCompile(`(?m)^id:\s*ADR-0013\s*$`).MatchString(body) {
		t.Error("AC-1: ADR-0013 frontmatter must contain `id: ADR-0013`")
	}
	if !regexp.MustCompile(`(?m)^status:\s*accepted\s*$`).MatchString(body) {
		t.Error("AC-1: ADR-0013 must be ratified (`status: accepted`)")
	}

	assertDecisionSubsection(t, body, "Global-rule representation",
		[]string{
			"Global", "Rule", "scope-reach", "OutcomeIllegal",
			"RejectionLayerVerbTime", "BlockingStrict",
			"ExpectedErrorCode", "provenance-authorization-out-of-scope",
			"globalRules()", "(Kind, FromState, Verb, Outcome)", "LookupRules",
			"m0123_ac2", "m0123_ac4", "m0124", "m0125", "m0123_ac5",
		},
		[]string{"single source of truth", "uniqueness key", "fourth arm"},
	)
}

// TestADR0013_M0144_AC2_OutOfScopeLegality is the mechanical evidence
// for M-0144/AC-2: the ADR records out-of-scope as `codes.ClassLegality`
// with the dual-emission rationale (verb-time refusal + check-time audit
// are one violation at two surfaces) and the `codes.go` carve-out note.
func TestADR0013_M0144_AC2_OutOfScopeLegality(t *testing.T) {
	t.Parallel()
	body := loadADR0013(t)

	assertDecisionSubsection(t, body, "Out-of-scope classification as legality",
		[]string{
			"codes.ClassLegality", "codes.ClassStructural",
			"provenance-authorization-out-of-scope",
			"verb/allow.go", "check/provenance.go",
			"Code{Class", "ADR-0012", "D-0011", "M-0147",
		},
		[]string{"dual-emit", "two surfaces", "carve-out"},
	)
}

// TestADR0013_M0144_AC3_CellcoverageSizing is the mechanical evidence
// for M-0144/AC-3: the ADR sizes the cellcoverage extension and states
// the explicit fallback condition (dedicated test + recorded exemption
// only if the extension proves its own epic).
func TestADR0013_M0144_AC3_CellcoverageSizing(t *testing.T) {
	t.Parallel()
	body := loadADR0013(t)

	assertDecisionSubsection(t, body, "cellcoverage extension sizing",
		[]string{"cellcoverage", "CellFixture", "authorize", "EvalContext", "M-0146", "ai/"},
		[]string{"tractable", "full integration", "fallback", "dedicated", "exemption"},
	)
}

// TestADR0013_M0144_AllocationAndDriftGuard asserts the ADR
// cross-references the epic, milestone, closed gap, and upstream
// decisions as bare ids (so finder tooling resolves them), and that
// `## Decision` carries exactly the three subsections that map 1:1 to
// M-0144's AC-1/AC-2/AC-3 — the drift guard fails if a subsection is
// added or renamed away.
func TestADR0013_M0144_AllocationAndDriftGuard(t *testing.T) {
	t.Parallel()
	body := loadADR0013(t)

	for _, ref := range []string{
		"E-0037", "M-0144", "G-0171",
		"D-0006", "D-0014", "D-0011",
		"ADR-0011", "ADR-0012", "M-0141",
	} {
		if !strings.Contains(body, ref) {
			t.Errorf("ADR-0013 body must cross-reference %q", ref)
		}
	}

	decision := extractMarkdownSection(body, 2, "Decision")
	if decision == "" {
		t.Fatal("ADR-0013 must have a `## Decision` section")
	}
	if got, want := countLevel3Headings(decision), 3; got != want {
		t.Errorf("expected %d `### ` subsections under `## Decision` (one per M-0144 AC), found %d — update this guard if the AC set changed", want, got)
	}
}
