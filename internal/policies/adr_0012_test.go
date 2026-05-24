package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// loadADR0012 reads ADR-0012 (Typed Coded error pattern) from disk by
// resolving the id through the loader, per CLAUDE.md *Testing* §"Policy
// tests that read entity files must resolve via the loader" — never a
// hardcoded path, so the test survives rename and archive sweeps.
func loadADR0012(t *testing.T) string {
	t.Helper()
	root, tr := sharedRepoTree(t)
	e := tr.ByID("ADR-0012")
	if e == nil {
		t.Fatal("ADR-0012 not found in tree (active or archive)")
	}
	data, err := os.ReadFile(filepath.Join(root, e.Path))
	if err != nil {
		t.Fatalf("reading ADR-0012 at %s: %v", e.Path, err)
	}
	return string(data)
}

// TestADR0012_AC5_Allocation asserts M-0138/AC-5: the ADR exists at the
// canonical docs/adr/ path with a matching frontmatter id, and the body
// cross-references the epic, this milestone, and the two gaps the
// pattern closes — as bare ids so finder tooling resolves them.
func TestADR0012_AC5_Allocation(t *testing.T) {
	t.Parallel()
	body := loadADR0012(t)

	// Frontmatter id matches the canonical ADR id.
	if !regexp.MustCompile(`(?m)^id:\s*ADR-0012\s*$`).MatchString(body) {
		t.Error("AC-5: ADR-0012 frontmatter must contain `id: ADR-0012`")
	}

	// Bare-id cross-references — the canonical id form so `aiwf history`
	// and finder tools resolve them.
	for _, ref := range []string{"E-0036", "M-0138", "G-0141", "G-0142"} {
		if !strings.Contains(body, ref) {
			t.Errorf("AC-5: ADR-0012 body must cross-reference %q", ref)
		}
	}
}

// TestADR0012_AC5_DecisionSections asserts M-0138/AC-5: the ADR's
// `## Decision` section contains exactly the five named decision
// subsections, each with non-empty prose and the per-section literals
// that pin the realized pattern's contract.
//
// Per CLAUDE.md *Testing* §"Substring assertions are not structural
// assertions", every literal is asserted inside the relevant
// subsection — a literal floating in another section would not satisfy
// the AC. The level-3 count is the drift guard: a sixth subsection
// added silently (or one of the five renamed away) fails here.
func TestADR0012_AC5_DecisionSections(t *testing.T) {
	t.Parallel()
	body := loadADR0012(t)

	decision := extractMarkdownSection(body, 2, "Decision")
	if decision == "" {
		t.Fatal("AC-5: ADR-0012 must have a `## Decision` section")
	}

	// Per-subsection required literals. Identifier literals (Go symbols,
	// ids) are matched case-sensitively; prose words are matched against
	// a lower-cased copy of the section.
	type sectionSpec struct {
		name              string
		identifierLiteral []string // case-sensitive Contains
		proseWords        []string // matched against lower-cased section
	}
	required := []sectionSpec{
		{
			name:              "Behavioral interface",
			identifierLiteral: []string{"Coded", "errors.As", "entity.Code"},
		},
		{
			name:              "Typed errors",
			identifierLiteral: []string{"FSMTransitionError", "AuthorizeKindError"},
			proseWords:        []string{"preserve"},
		},
		{
			name:              "Named code constants",
			identifierLiteral: []string{"CodeFSMTransitionIllegal", "CodeAuthorizeKindNotAllowed", "G-0129"},
		},
		{
			name:       "Scope",
			proseWords: []string{"legality", "malformed", "yagni"},
		},
		{
			name:              "Scanner recognition",
			identifierLiteral: []string{"collectImplFindingCodes", "AC-4", "check.Finding"},
		},
	}

	for _, spec := range required {
		section := extractSubsection(decision, spec.name)
		if section == "" {
			t.Errorf("AC-5: `### %s` subsection missing under `## Decision`", spec.name)
			continue
		}
		if !hasNonEmptyProse(section) {
			t.Errorf("AC-5: `### %s` subsection is empty / placeholder only", spec.name)
			continue
		}
		for _, lit := range spec.identifierLiteral {
			if !strings.Contains(section, lit) {
				t.Errorf("AC-5: `### %s` subsection must contain identifier %q", spec.name, lit)
			}
		}
		lower := strings.ToLower(section)
		for _, word := range spec.proseWords {
			if !strings.Contains(lower, word) {
				t.Errorf("AC-5: `### %s` subsection must convey %q", spec.name, word)
			}
		}
	}

	// Drift guard: exactly the five named subsections under `## Decision`.
	count := countLevel3Headings(decision)
	if count != len(required) {
		t.Errorf("AC-5: expected %d level-3 sub-headings under `## Decision`, found %d — if a decision section was added or removed, update %s",
			len(required), count, t.Name())
	}
}
