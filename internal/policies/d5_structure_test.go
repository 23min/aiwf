package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Structural tests for the D5 "findings become checks" force and the surfaces
// that cite it (G-0489).
//
// WHAT EARNS AN ASSERTION HERE, AND WHY THE LINE SITS WHERE IT DOES.
//
// These tests pin document *structure* — a heading exists, a section holds N
// forces, a labelled paragraph opens a line, a cross-file citation resolves.
// They do not check that a paragraph still *means* what it meant.
//
// That line is deliberate and was drawn from measurement. An assertion of the
// form "this phrase appears in this section" pins a reading, not a rule, and a
// reading drifts in more ways than an assertion can enumerate: the phrase can
// pre-exist elsewhere in scope; a later edit can give it a second occurrence so
// even deleting the rule leaves it matching; the negator that makes it binding
// can sit outside the asserted span, so `never X` becomes `or X` with the test
// green; the rule can be widened by appending rather than by inverting. Four
// review rounds over this work produced roughly thirty findings and more than
// half were defects in phrase-level assertions rather than in the prose they
// guarded — the checks generated more work than they caught, while a green
// suite implied an assurance it could not deliver.
//
// So: structure is mechanically checkable and is checked here. Content
// correctness — does this rule still say the right thing, does it contradict
// its neighbour — is held at review, which is the disposition D5 itself
// prescribes for what cannot be pinned. Adding a phrase-content assertion to
// this file re-opens that trade; if you are about to, the bar is that breaking
// it would be a structural break, not a rewording.

// TestWfCodebaseHealth_SectionDCarriesFiveForcesIncludingD5 pins that the
// rubric still carries D5 alongside the original four. Deleting or renumbering
// the force fails here, which is what the surfaces citing it depend on.
func TestWfCodebaseHealth_SectionDCarriesFiveForcesIncludingD5(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfCodebaseHealthFixturePath)

	sec := extractMarkdownSection(body, 2, "D. Tests that pin behavior")
	if sec == "" {
		t.Fatal("wf-codebase-health must have a `## D. Tests that pin behavior…` section")
	}
	if got := countSubHeadings(sec, 3); got != 5 {
		t.Errorf("section D has %d `###` forces; want exactly 5 (D1-D4 plus D5 findings-become-checks)", got)
	}
	if !strings.Contains(sec, "### D5. Findings become checks") {
		t.Error("section D must carry a `### D5. Findings become checks` force")
	}
}

// TestWfCodebaseHealth_D5CarriesLabelledStopRuleParagraph pins the stop rule as
// its own labelled paragraph rather than a clause buried in prose — the shape
// that lets calling surfaces cite the terminator precisely. Checked against raw
// text at line start, since the claim is about paragraph structure.
func TestWfCodebaseHealth_D5CarriesLabelledStopRuleParagraph(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfCodebaseHealthFixturePath)

	d5 := extractMarkdownSection(body, 3, "D5.")
	if d5 == "" {
		t.Fatal("wf-codebase-health must have a `### D5. …` findings-become-checks force")
	}
	if !hasLineStartingWith(d5, "**Stop rule.**") {
		t.Error("D5 must open a paragraph with the `**Stop rule.**` label, so calling surfaces can cite the terminator")
	}
}

// TestWfReviewCode_VerdictCarriesLabelledLoopTerminator pins that the verdict
// step exists and states when the loop ends as its own labelled paragraph. The
// milestone wrap cites this paragraph by name.
func TestWfReviewCode_VerdictCarriesLabelledLoopTerminator(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfReviewCodeFixturePath)

	verdict := extractMarkdownSection(body, 3, "8. Verdict")
	if verdict == "" {
		t.Fatal("wf-review-code must have a `### 8. Verdict` step")
	}
	if !hasLineStartingWith(verdict, "**When the loop ends.**") {
		t.Error("the verdict step must state when the review loop ends as its own labelled paragraph")
	}
}

// TestEmbeddedGuidance_PrimingCarriesD5Bullet pins that the always-on priming
// subset still lists D5 alongside the other primed forces, so the ratchet is
// primed every turn rather than only reachable by opening the full rubric.
func TestEmbeddedGuidance_PrimingCarriesD5Bullet(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, g0343GuidanceFixturePath)

	priming := extractMarkdownSection(body, 2, "Code-health priming")
	if priming == "" {
		t.Fatal("aiwf-guidance.md must have a `## Code-health priming` section")
	}
	if !hasLineStartingWith(priming, "- **D5 —") {
		t.Error("the always-on priming subset must carry a `- **D5 — …**` bullet")
	}
}

// skillSectionCitation matches a cross-skill section reference of the shape
// `skill-name` §"Section Name" — the convention the embedded rituals use to
// point at a section of another skill.
var skillSectionCitation = regexp.MustCompile("`([a-z0-9-]+)` §\"([^\"]+)\"")

// TestEmbeddedRituals_CrossSkillCitationsResolve walks every embedded ritual
// and asserts each cross-skill section citation names a heading that actually
// exists in the cited skill.
//
// This is the check with real reach: a citation rots silently when the target
// is renamed or renumbered, and nothing else in the tree notices. It is also
// why the rituals cite sections by name rather than by number — a numeric
// reference like §8 breaks invisibly when a step is inserted, whereas a named
// one breaks here.
func TestEmbeddedRituals_CrossSkillCitationsResolve(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	ritualsDir := filepath.Join(root, "internal", "skills", "embedded-rituals")

	var checked int
	err := filepath.WalkDir(ritualsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, readErr := os.ReadFile(path) //nolint:gosec // walking a repo-relative tree under test
		if readErr != nil {
			return readErr
		}
		rel, _ := filepath.Rel(root, path)
		for _, m := range skillSectionCitation.FindAllStringSubmatch(string(data), -1) {
			skill, section := m[1], m[2]
			target := findEmbeddedSkill(root, skill)
			if target == "" {
				// Not every backticked token before a § is a skill name; only
				// citations whose target resolves are in scope.
				continue
			}
			checked++
			targetBody, targetErr := os.ReadFile(target) //nolint:gosec // resolved from the same tree
			if targetErr != nil {
				t.Errorf("%s cites `%s` §%q but that skill is unreadable: %v", rel, skill, section, targetErr)
				continue
			}
			if !hasHeadingNamed(string(targetBody), section) {
				t.Errorf("%s cites `%s` §%q, but %s has no heading by that name — the citation is dangling", rel, skill, section, skill)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded rituals: %v", err)
	}
	if checked == 0 {
		t.Error("no cross-skill section citations found; this check has stopped covering anything")
	}
}

// findEmbeddedSkill returns the absolute path of the named ritual skill's
// SKILL.md, or "" when no such skill exists.
func findEmbeddedSkill(root, name string) string {
	matches, err := filepath.Glob(filepath.Join(root, "internal", "skills", "embedded-rituals", "plugins", "*", "skills", name, "SKILL.md"))
	if err != nil || len(matches) == 0 {
		return ""
	}
	return matches[0]
}

// headingStepNumber matches the ordinal a ritual's numbered step heading
// carries — `### 8. Verdict` — so a citation can name the step rather than its
// position. Naming is the point: a step inserted above renumbers the heading,
// and a citation written as §"8. Verdict" would then be silently wrong.
var headingStepNumber = regexp.MustCompile(`^\d+\.\s*`)

// hasHeadingNamed reports whether body carries a markdown heading whose text
// begins with name, at any level, ignoring any leading step number. Prefix
// matching tolerates a heading carrying a trailing qualifier after the cited
// name, e.g. §"Independence" resolving `## Independence — who runs this matters`.
func hasHeadingNamed(body, name string) bool {
	for line := range strings.SplitSeq(body, "\n") {
		trimmed := strings.TrimLeft(line, "#")
		if len(trimmed) == len(line) || !strings.HasPrefix(trimmed, " ") {
			continue
		}
		text := strings.TrimSpace(trimmed)
		if strings.HasPrefix(text, name) || strings.HasPrefix(headingStepNumber.ReplaceAllString(text, ""), name) {
			return true
		}
	}
	return false
}

// hasLineStartingWith reports whether any line of raw markdown begins with
// prefix, ignoring leading indentation. Used for paragraph-label assertions,
// where "the label opens a line" is the property under test.
func hasLineStartingWith(raw, prefix string) bool {
	for line := range strings.SplitSeq(raw, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			return true
		}
	}
	return false
}
