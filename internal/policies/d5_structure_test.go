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

// skillSectionCitation matches a cross-skill section reference of the shape
// `skill-name` §"Section Name" — the convention the embedded rituals use to
// point at a section of another skill.
var skillSectionCitation = regexp.MustCompile("`([a-z0-9-]+)` *§ *\"([^\"]+)\"")

// shippedMarkdownRoot holds every tree that materializes into a consumer's
// `.claude/`. Walking it whole is what makes "every shipped citation is
// checked" hold by construction: a tree added here is read the day it appears,
// with no list to keep in step. The non-markdown trees under it — the hooks and
// the statusline, which ship shell — carry no citation and contribute nothing.
var shippedMarkdownRoot = filepath.Join("internal", "skills")

// TestShippedSkills_CrossSkillCitationsResolve walks every shipped markdown
// file and asserts each cross-skill section citation names a heading that
// actually exists in the cited skill.
//
// This is the check with real reach: a citation rots silently when the target
// is renamed or renumbered, and nothing else in the tree notices. It is also
// why the shipped surfaces cite sections by name rather than by number — a
// numeric reference like §8 breaks invisibly when a step is inserted, whereas a
// named one breaks here.
//
// Two holes worth knowing. A citation whose *skill* name resolves nowhere is
// skipped rather than reported, because not every backticked token before a §
// is a skill name — so renaming a cited skill's directory passes here. And the
// root below is the definition of scope, so narrowing it reports nothing;
// checking it against a second copy of itself would reinstate the hand list
// walking the root whole exists to avoid.
func TestShippedSkills_CrossSkillCitationsResolve(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)

	var checked int
	err := filepath.WalkDir(filepath.Join(root, shippedMarkdownRoot), func(path string, d os.DirEntry, err error) error {
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
		t.Fatalf("walk %s: %v", shippedMarkdownRoot, err)
	}
	if checked == 0 {
		t.Error("no cross-skill section citations found; this check has stopped covering anything")
	}
}

// The subsection that states what an acceptance criterion's title claims and
// what its body holds. Both halves are asserted: a citation retargeted to
// another part of the same skill leaves the rule with no stated home just as a
// deleted one does.
const (
	acBodyRuleOwner        = "aiwf-add"
	acBodyRuleOwnerSection = "What to write per kind"
)

// acBodyRuleCitingSurfaces are the surfaces an author passes through while
// writing or growing a criterion body. Each cites the owner instead of stating
// a rule of its own, so the assignment holds only while the citations do. The
// list is written here because it has to be machine-readable somewhere to be
// checkable at all; what it cannot reach is a *new* surface that starts stating
// a rule, since "states a rule" has no machine shape.
var acBodyRuleCitingSurfaces = []string{
	filepath.Join(sectionRitualsDir, "templates", "milestone-spec.md"),
	filepath.Join(sectionRitualsDir, "skills", "aiwfx-plan-milestones", "SKILL.md"),
	filepath.Join(sectionRitualsDir, "skills", "aiwfx-start-milestone", "SKILL.md"),
	filepath.Join("internal", "skills", "embedded", "aiwf-edit-body", "SKILL.md"),
}

// TestACBodyRule_CitingSurfacesStillCiteTheOwner asserts each deferring surface
// still points at the owning subsection. The walk above catches a citation
// whose target moved; this catches one deleted or retargeted, which is the
// shape a surface takes on just before it states a second rule of its own.
func TestACBodyRule_CitingSurfacesStillCiteTheOwner(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)

	if findEmbeddedSkill(root, acBodyRuleOwner) == "" {
		t.Fatalf("the acceptance-criterion body rule names owner %q, which ships no skill", acBodyRuleOwner)
	}
	if len(acBodyRuleCitingSurfaces) == 0 {
		t.Fatal("no citing surfaces listed; this check has stopped covering anything")
	}

	for _, rel := range acBodyRuleCitingSurfaces {
		t.Run(filepath.Base(filepath.Dir(rel)), func(t *testing.T) {
			t.Parallel()
			data, err := os.ReadFile(filepath.Join(root, rel)) //nolint:gosec // a repo-relative surface under test
			if err != nil {
				t.Fatalf("reading %s: %v", rel, err)
			}
			for _, m := range skillSectionCitation.FindAllStringSubmatch(string(data), -1) {
				if m[1] == acBodyRuleOwner && m[2] == acBodyRuleOwnerSection {
					return
				}
			}
			t.Errorf("%s does not cite `%s` §%q; what an acceptance criterion body holds has one owner, and a surface that stops pointing at it is free to state a second rule", rel, acBodyRuleOwner, acBodyRuleOwnerSection)
		})
	}
}

// findEmbeddedSkill returns the absolute path of the named skill's SKILL.md, or
// "" when no such skill exists. Both shipped skill trees are in the universe —
// the rituals and the per-verb skills — so a ritual citing a verb skill's
// section resolves on the same terms as one citing another ritual's.
func findEmbeddedSkill(root, name string) string {
	matches, err := filepath.Glob(filepath.Join(root, "internal", "skills", "embedded-rituals", "plugins", "*", "skills", name, "SKILL.md"))
	if err == nil && len(matches) > 0 {
		return matches[0]
	}
	verbSkill := filepath.Join(root, "internal", "skills", "embedded", name, "SKILL.md")
	if _, statErr := os.Stat(verbSkill); statErr == nil {
		return verbSkill
	}
	return ""
}

// TestSkillSectionCitation_SpacingDoesNotHideACitation pins the reader rather
// than the house style. A citation typed with a space around the marker is a
// citation; a reader that missed it would leave one unchecked and say nothing,
// which is a worse trade than tolerating the spelling.
func TestSkillSectionCitation_SpacingDoesNotHideACitation(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		line string
		want bool
	}{
		{"canonical", "see `wf-patch` §\"Workflow\" for the rest", true},
		{"spaced around the marker", "see `wf-patch`  §  \"Workflow\" for the rest", true},
		{"no marker", "see `wf-patch` \"Workflow\" for the rest", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := skillSectionCitation.MatchString(tc.line); got != tc.want {
				t.Errorf("MatchString(%q) = %v, want %v", tc.line, got, tc.want)
			}
		})
	}
}

// TestFindEmbeddedSkill_ResolvesBothShippedTrees pins the universe the citation
// walk above resolves against. A name reachable in neither tree returns "", and
// the walk skips it: that is what keeps a backticked word before a § from being
// read as a skill name, and it is also what would silently drop a whole tree.
func TestFindEmbeddedSkill_ResolvesBothShippedTrees(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)

	for _, tc := range []struct {
		name     string
		skill    string
		wantTree string // "" when the name is expected to resolve nowhere
	}{
		{"ritual skill", "wf-review-code", filepath.Join("internal", "skills", "embedded-rituals")},
		{"verb skill", "aiwf-add", filepath.Join("internal", "skills", "embedded")},
		{"neither tree", "not-a-shipped-skill", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := findEmbeddedSkill(root, tc.skill)
			if tc.wantTree == "" {
				if got != "" {
					t.Errorf("findEmbeddedSkill(%q) = %q, want no match", tc.skill, got)
				}
				return
			}
			// Asserted by tree rather than in full: the rituals lookup is a glob
			// precisely so a plugin can move without every caller moving with it.
			if !strings.HasPrefix(got, filepath.Join(root, tc.wantTree)+string(filepath.Separator)) {
				t.Errorf("findEmbeddedSkill(%q) = %q, want a SKILL.md under %s", tc.skill, got, tc.wantTree)
			}
		})
	}
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
