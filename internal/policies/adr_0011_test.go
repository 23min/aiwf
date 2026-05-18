package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// loadADR0011 reads ADR-0011 (Legal-workflow spec methodology) from
// disk by resolving the id through the loader, per CLAUDE.md
// *Testing* §"Policy tests that read entity files must resolve via
// the loader". Returns both the raw body and a sentinel for the
// error case so individual tests can fail with a clear message.
func loadADR0011(t *testing.T) string {
	t.Helper()
	root, tr := sharedRepoTree(t)
	e := tr.ByID("ADR-0011")
	if e == nil {
		t.Fatal("ADR-0011 not found in tree (active or archive)")
	}
	data, err := os.ReadFile(filepath.Join(root, e.Path))
	if err != nil {
		t.Fatalf("reading ADR-0011 at %s: %v", e.Path, err)
	}
	return string(data)
}

// TestADR0011_AC1_AllocationAndCrossReference asserts M-0120/AC-1:
// the ADR exists at the canonical docs/adr/ path, has a matching
// frontmatter id, is loaded by aiwf as a kind=adr entity, and the
// body cross-references the motivating epic E-0033.
func TestADR0011_AC1_AllocationAndCrossReference(t *testing.T) {
	t.Parallel()
	body := loadADR0011(t)

	// Frontmatter id matches the canonical ADR id.
	if !regexp.MustCompile(`(?m)^id:\s*ADR-0011\s*$`).MatchString(body) {
		t.Error("AC-1: ADR-0011 frontmatter must contain `id: ADR-0011`")
	}

	// Cross-reference to E-0033 must appear at least once in the body.
	// Bare-id reference (not just a slug or filename — the canonical
	// id form so `aiwf history` and finder tools resolve it).
	if !strings.Contains(body, "E-0033") {
		t.Error("AC-1: ADR-0011 body must cross-reference epic `E-0033`")
	}
}

// TestADR0011_AC2_SevenDecisionSections asserts M-0120/AC-2: the ADR
// body contains seven distinct decision-point subsections under
// `## Decision`, each with non-empty content. The seven are the
// load-bearing commitments named in M-0120's spec. A missing or
// emptied section here means the methodology has silently lost a
// commitment.
//
// Per CLAUDE.md *Testing* §"Substring assertions are not structural
// assertions", this walks the markdown section hierarchy rather than
// flat-grepping for headings — a `### Independence` line floating
// outside `## Decision` would not satisfy the AC.
func TestADR0011_AC2_SevenDecisionSections(t *testing.T) {
	t.Parallel()
	body := loadADR0011(t)

	decision := extractMarkdownSection(body, 2, "Decision")
	if decision == "" {
		t.Fatal("AC-2: ADR-0011 must have a `## Decision` section")
	}

	// The seven decision-point sub-headings, in the order they appear
	// in the spec body. Order is not asserted by this test (would be
	// brittle), but each name is unique and required.
	required := []string{
		"Independence",
		"Three-pass methodology",
		"Canonical form",
		"Cell-coverage commitment",
		"Drift policy",
		"Scope",
		"Future-change handling",
	}

	for _, name := range required {
		// Each name must be a level-3 heading directly inside the
		// `## Decision` body. extractMarkdownSection on the body
		// alone would also match a `### Scope` floating in some
		// other top-level section, so we extract the sub-section
		// from the decision section specifically.
		section := extractSubsection(decision, name)
		if section == "" {
			t.Errorf("AC-2: `### %s` subsection missing under `## Decision`", name)
			continue
		}
		// Non-empty content: at least one non-blank, non-heading line
		// of body prose. Empty placeholders are not allowed.
		if !hasNonEmptyProse(section) {
			t.Errorf("AC-2: `### %s` subsection is empty / placeholder only", name)
		}
	}

	// Drift guard: count level-3 sub-headings inside `## Decision`
	// and assert it equals 7. A future PR that adds an 8th decision
	// silently — without updating this test — should fail here so
	// the reviewer notices the new commitment.
	count := countLevel3Headings(decision)
	if count != len(required) {
		t.Errorf("AC-2: expected %d level-3 sub-headings under `## Decision`, found %d — if a new commitment was added, update %s",
			len(required), count, t.Name())
	}
}

// TestADR0011_AC3_StatusAccepted asserts M-0120/AC-3: the ADR's
// frontmatter status is `accepted`. Operationally this fires after
// `aiwf promote ADR-0011 accepted` lands. Before promotion the test
// fails (red); after it passes (green). The FSM enforces the
// transition itself — this test pins the *outcome*.
func TestADR0011_AC3_StatusAccepted(t *testing.T) {
	t.Parallel()
	body := loadADR0011(t)

	if !regexp.MustCompile(`(?m)^status:\s*accepted\s*$`).MatchString(body) {
		t.Error("AC-3: ADR-0011 frontmatter status must be `accepted` (run `aiwf promote ADR-0011 accepted`)")
	}
}

// --- helpers --------------------------------------------------------

// extractSubsection returns the body of a `### <name>` heading inside
// a parent section's body. Differs from extractMarkdownSection in that
// (a) it always works at level 3, (b) it does an exact-name match
// rather than prefix match, and (c) the input is already a section
// body, not a whole document.
func extractSubsection(parentBody, name string) string {
	prefix := "### " + name
	lines := strings.Split(parentBody, "\n")
	start := -1
	inFence := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		// Exact: heading line == prefix, or heading line == prefix + " — ..." trailer.
		if line == prefix || strings.HasPrefix(line, prefix+" ") || strings.HasPrefix(line, prefix+"\t") {
			start = i + 1
			break
		}
	}
	if start == -1 {
		return ""
	}
	end := len(lines)
	inFence = false
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		// Stop at any heading of level ≤ 3 (i.e., `# `, `## `, `### `).
		hashes := 0
		for _, r := range lines[i] {
			if r == '#' {
				hashes++
			} else {
				break
			}
		}
		if hashes >= 1 && hashes <= 3 && hashes < len(lines[i]) && lines[i][hashes] == ' ' {
			end = i
			break
		}
	}
	return strings.Join(lines[start:end], "\n")
}

// hasNonEmptyProse reports whether the section contains at least one
// non-blank, non-heading line. Headings and blank lines alone do not
// qualify — the AC requires real content under each commitment.
func hasNonEmptyProse(section string) bool {
	inFence := false
	for _, line := range strings.Split(section, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			// Fenced content counts as prose for our purposes.
			if trimmed != "" {
				return true
			}
			continue
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		return true
	}
	return false
}

// countLevel3Headings counts `### `-prefixed lines outside fenced
// code blocks within the given section body.
func countLevel3Headings(body string) int {
	inFence := false
	count := 0
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if strings.HasPrefix(line, "### ") {
			count++
		}
	}
	return count
}
