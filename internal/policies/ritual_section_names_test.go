package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// Ritual skills that enumerate an entity kind's body sections for the author
// filling a template in. These are the readers of the shipped prose templates:
// they tell a consumer which sections to write, so a ritual naming a section
// the template no longer has is the epic's own failure mode — one surface
// stating the section set and drifting from the owner — reproduced on the
// surface a consumer actually follows.
const (
	aiwfxPlanEpicSkillPath       = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-plan-epic/SKILL.md"
	aiwfxStartEpicSkillPath      = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-epic/SKILL.md"
	aiwfxPlanMilestonesSkillPath = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-plan-milestones/SKILL.md"
	aiwfxRecordDecisionSkillPath = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-record-decision/SKILL.md"
)

// sectionInstruction is one ritual passage that tells an author which sections
// to write for a kind. anchor names the passage; the region runs from the
// anchor's line through the bullet list beneath it.
type sectionInstruction struct {
	path   string
	anchor string
	kind   entity.Kind
}

var ritualSectionInstructions = []sectionInstruction{
	{aiwfxPlanEpicSkillPath, "at `.claude/templates/epic-spec.md`", entity.KindEpic},
	{aiwfxPlanMilestonesSkillPath, "at `.claude/templates/milestone-spec.md`", entity.KindMilestone},
	{aiwfxRecordDecisionSkillPath, "For an ADR: read `.claude/templates/adr.md`. Fill in:", entity.KindADR},
	{aiwfxRecordDecisionSkillPath, "For a D-NNNN: read `.claude/templates/decision.md`. Fill in:", entity.KindDecision},
	{aiwfxStartEpicSkillPath, "Confirm the", entity.KindEpic},
}

// optionalityMarkedSection matches an optionality marker anywhere in an
// instruction passage. The passage is a short bulleted list of section names, so
// there is nothing else the word can be marking; a form-specific pattern only
// invites a spelling it does not cover, which is what an earlier bolded-or-heading
// alternation did — its heading arm could not match a bulleted `- ## Name
// (optional)` and was dead.
var optionalityMarkedSection = regexp.MustCompile(`(?i)\(optional\)`)

// TestRitualsNameTheOwnedSectionsWithoutMarkers pins that a ritual passage
// instructing an author to fill in a body names every section its kind
// carries, and names none of them by a marker-suffixed form.
//
// Both halves are failures measured on live surfaces. The epic ritual told an
// author to write "Scope — in / out", the substructure the epic template used
// before its out-of-scope heading moved to top level, so a body written by
// following it carried no `out_of_scope` key on any read path. The decision
// ritual named `Validation (optional)` and `Consequences (optional)` after
// those markers left the templates, which would have had aiwf's own ritual
// emitting the `validation_optional` key the retirement exists to foreclose.
//
// The containment check is scoped to the instruction passage, not the file. A
// ritual mentioning a section name anywhere — the `aiwf add` skeleton, a prose
// aside — would otherwise satisfy it while the instruction itself omitted the
// section, which is exactly how the epic ritual read before this milestone.
//
// Expected names come from entity.RequiredSections, so a kind gaining a section
// fails here until the rituals that instruct authors follow.
func TestRitualsNameTheOwnedSectionsWithoutMarkers(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	for _, instr := range ritualSectionInstructions {
		t.Run(filepath.Base(filepath.Dir(instr.path))+"/"+string(instr.kind), func(t *testing.T) {
			t.Parallel()
			raw, err := os.ReadFile(filepath.Join(root, instr.path))
			if err != nil {
				t.Fatalf("reading %s: %v", instr.path, err)
			}
			region, found := instructionRegion(string(raw), instr.anchor)
			if !found {
				t.Fatalf("%s: no passage anchored on %q; the instruction moved or was reworded", instr.path, instr.anchor)
			}
			if optionalityMarkedSection.MatchString(region) {
				t.Errorf("%s: the passage at %q marks a section optional in its name; no template carries such a heading, so an author following this writes a section whose slug folds the marker into the key. State optionality in the description after the section name instead",
					instr.path, instr.anchor)
			}
			for _, section := range entity.RequiredSections(instr.kind) {
				if !strings.Contains(strings.ToLower(region), strings.ToLower(section)) {
					t.Errorf("%s: the passage at %q instructs an author to fill a %s body but never names its %q section; a body written by following it omits that section, which no surface reports (G-0571)",
						instr.path, instr.anchor, instr.kind, section)
				}
			}
		})
	}
}

// instructionRegion returns the passage beginning at the line containing anchor
// and continuing through the bullet list beneath it, stopping at the first line
// that is neither a bullet nor blank.
func instructionRegion(doc, anchor string) (region string, found bool) {
	lines := strings.Split(doc, "\n")
	start := -1
	for i, line := range lines {
		if strings.Contains(line, anchor) {
			start = i
			break
		}
	}
	if start < 0 {
		return "", false
	}
	out := []string{lines[start]}
	for _, line := range lines[start+1:] {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(trimmed, "- ") {
			break
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n"), true
}
