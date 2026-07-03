package policies

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PolicyM0228SkillsPolicyBroadenedPrinciple asserts that CLAUDE.md's
// `### Skills policy` section (under `## Go conventions`) states the
// broadened authoring principle for shipped surfaces: the id chokepoint
// M-0227 extended from `SKILL.md` bodies to the full shipped-surface set,
// plus the content class M-0228 forbids there. The section must name each
// broadened surface and content-class concept so the principle is stated
// in full, not gestured at — a future edit that reintroduces development
// history or rationale into a shipped surface is then held to a written
// rule at review.
//
// The check is section-scoped, not a whole-file grep (CLAUDE.md
// §"Substring assertions are not structural assertions"): it walks the
// heading hierarchy to the (## Go conventions, ### Skills policy) span and
// asserts each marker appears *within that section*. The required markers
// are deliberately ones absent from the pre-broadening section, so the
// assertion is non-vacuous — it goes red if the paragraph is ever narrowed
// back to just `SKILL.md` bodies.
//
// Pins M-0228/AC-1.
func PolicyM0228SkillsPolicyBroadenedPrinciple(root string) ([]Violation, error) {
	const (
		policyID = "m0228-skills-policy-broadened-principle"
		relPath  = "CLAUDE.md"
	)
	raw, err := os.ReadFile(filepath.Join(root, relPath))
	if err != nil {
		return []Violation{{
			Policy: policyID,
			File:   relPath,
			Detail: fmt.Sprintf("missing or unreadable: %v", err),
		}}, nil
	}

	// Collect body lines under the (## Go conventions, ### Skills policy)
	// span. A `# ` re-anchors the hierarchy; a `## ` sets whether we are
	// under Go conventions; a `### ` sets whether we are in Skills policy.
	inGoConventions := false
	inSkillsPolicy := false
	var body []string
	for _, line := range strings.Split(string(raw), "\n") {
		switch {
		case strings.HasPrefix(line, "# "):
			inGoConventions = false
			inSkillsPolicy = false
		case strings.HasPrefix(line, "## "):
			inGoConventions = strings.TrimSpace(strings.TrimPrefix(line, "## ")) == "Go conventions"
			inSkillsPolicy = false
		case strings.HasPrefix(line, "### "):
			inSkillsPolicy = inGoConventions &&
				strings.TrimSpace(strings.TrimPrefix(line, "### ")) == "Skills policy"
		default:
			if inSkillsPolicy {
				body = append(body, line)
			}
		}
	}

	var vs []Violation
	report := func(detail string) {
		vs = append(vs, Violation{Policy: policyID, File: relPath, Detail: detail})
	}

	if len(body) == 0 {
		report("missing `### Skills policy` section under `## Go conventions` (M-0228/AC-1 requires the broadened authoring principle stated there)")
		return vs, nil
	}

	section := strings.ToLower(strings.Join(body, "\n"))
	// The full shipped-surface list M-0227 broadened the id chokepoint to,
	// plus the content class M-0228 forbids. `description:` is omitted as a
	// required marker on purpose: "description" already appears in the
	// section (skill `name:`/`description:` frontmatter), so it cannot
	// distinguish the broadened statement from the pre-broadening one — the
	// four surface markers below are each absent pre-broadening.
	required := []struct{ marker, names string }{
		{"statusline", "the statusline's comments"},
		{"template", "entity templates"},
		{"agent", "role-agent cards"},
		{"guidance", "the guidance fragment"},
		{"history", "the no-development-history content class"},
		{"rationale", "the no-rationale/war-story content class"},
	}
	for _, r := range required {
		if !strings.Contains(section, r.marker) {
			report(fmt.Sprintf("`### Skills policy` section does not name %s (missing marker %q) — the broadened authoring principle must state the full shipped-surface list and the history/provenance/rationale content class (M-0228/AC-1)", r.names, r.marker))
		}
	}
	return vs, nil
}
