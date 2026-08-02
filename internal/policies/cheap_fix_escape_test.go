package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// cheap_fix_escape_test.go — structural pins for the cheap-fix test and the
// reference-phrasing rule.
//
// Both rules narrow a standing mandate rather than adding one, which is what
// makes them worth pinning: a mandate that loses its escape reverts silently to
// the unconditional form, and nothing else in the tree would notice. Each
// assertion is scoped to the named markdown section that must carry the rule,
// per CLAUDE.md *Testing* §"Substring assertions are not structural
// assertions" — the literals here ("cheap-fix test") are short enough that
// file-wide presence would prove nothing about placement.
//
// The guidance fragment's own copies are additionally held by
// PolicyM0211GuidanceOperatingAnchors' curated anchor set; these tests pin the
// ritual surfaces, which that policy does not read.

// milestoneSpecTemplatePath is the shipped milestone-spec template — the copy a
// builder reads in-line while filling `## Deferrals`, at the moment the decision
// is actually made, rather than the skill body they read beforehand.
const milestoneSpecTemplatePath = "internal/skills/embedded-rituals/plugins/aiwf-extensions/templates/milestone-spec.md"

// findStartMilestoneImplementationSection locates the implementation step by
// heading content rather than by its number, matching the convention its sibling
// finders in this package document: a future reshuffle that renumbers the step
// must not silently stop the structural check from finding anything.
func findStartMilestoneImplementationSection(body string) string {
	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		return ""
	}
	for _, line := range strings.Split(workflow, "\n") {
		if !strings.HasPrefix(line, "### ") {
			continue
		}
		text := strings.TrimPrefix(line, "### ")
		if strings.Contains(strings.ToLower(text), "implementation") {
			return extractMarkdownSection(body, 3, text)
		}
	}
	return ""
}

// TestFindStartMilestoneImplementationSection_BranchCoverage covers the
// defensive return arms the happy-path fixture test does not reach.
func TestFindStartMilestoneImplementationSection_BranchCoverage(t *testing.T) {
	t.Parallel()

	if got := findStartMilestoneImplementationSection("# No workflow heading here\n"); got != "" {
		t.Errorf("absent Workflow section: want empty, got %q", got)
	}
	if got := findStartMilestoneImplementationSection("## Workflow\n\n### 1. Something else\n\nbody\n"); got != "" {
		t.Errorf("Workflow without a matching step: want empty, got %q", got)
	}
}

// findWrapMilestoneSpecSectionsStep returns the wrap-side spec-sections step of
// the wrap-milestone workflow — the step whose `## Deferrals` bullet decides
// whether a punt becomes a gap entity.
func findWrapMilestoneSpecSectionsStep(body string) string {
	workflow := extractMarkdownSection(body, 2, "Workflow")
	if workflow == "" {
		return ""
	}
	for _, line := range strings.Split(workflow, "\n") {
		if !strings.HasPrefix(line, "### ") {
			continue
		}
		text := strings.TrimPrefix(line, "### ")
		if strings.Contains(strings.ToLower(text), "wrap-side sections") {
			return extractMarkdownSection(body, 3, text)
		}
	}
	return ""
}

// TestFindWrapMilestoneSpecSectionsStep_BranchCoverage covers the defensive
// return arms the happy-path fixture test does not reach.
func TestFindWrapMilestoneSpecSectionsStep_BranchCoverage(t *testing.T) {
	t.Parallel()

	if got := findWrapMilestoneSpecSectionsStep("# No workflow heading here\n"); got != "" {
		t.Errorf("absent Workflow section: want empty, got %q", got)
	}
	if got := findWrapMilestoneSpecSectionsStep("## Workflow\n\n### 1. Something else\n\nbody\n"); got != "" {
		t.Errorf("Workflow without a matching step: want empty, got %q", got)
	}
}

// TestAiwfxWrapMilestone_DeferralBulletCarriesCheapFixTest pins the escape on
// the deferral mandate. Without it the ritual reads as "every punt becomes a
// gap entity", which manufactures a tracked entity for work that is cheaper to
// finish than to file.
func TestAiwfxWrapMilestone_DeferralBulletCarriesCheapFixTest(t *testing.T) {
	t.Parallel()

	step := findWrapMilestoneSpecSectionsStep(loadAiwfxWrapMilestoneFixture(t))
	if step == "" {
		t.Fatalf("wrap-side spec-sections step not found in %s", aiwfxWrapMilestoneFixturePath)
	}
	lower := strings.ToLower(step)
	for _, fragment := range []string{
		"cheap-fix test",            // the test is named, not merely implied
		"already touches",           // its first condition: the file is open anyway
		"make it now",               // the instruction, so the escape is actionable
		"corrective commit",         // where the fix lands, so it never dirties the wrap commit
		"its own branch",            // the counter-condition that still earns a gap
		"survives the test",         // the mandate still binds to what the test does not excuse
		"under `## reviewer notes`", // the fix leaves a trace a later reader can find
	} {
		if !strings.Contains(lower, fragment) {
			t.Errorf("the wrap-side spec-sections step has lost %q — the deferral mandate has reverted to its unconditional form, so every punt files a gap entity again", fragment)
		}
	}
}

// TestAiwfxWrapMilestone_AntiPatternNamesLedgerPadding pins the counterpart
// anti-pattern. The escape alone tells you when a gap is optional; this names
// the failure it exists to prevent, which is the form a reader scanning only
// the anti-pattern list will meet.
func TestAiwfxWrapMilestone_AntiPatternNamesLedgerPadding(t *testing.T) {
	t.Parallel()

	section := extractMarkdownSection(loadAiwfxWrapMilestoneFixture(t), 2, "Anti-patterns")
	if section == "" {
		t.Fatalf("Anti-patterns section not found in %s", aiwfxWrapMilestoneFixturePath)
	}
	lower := strings.ToLower(section)
	if !strings.Contains(lower, "ledger padding") {
		t.Error("the Anti-patterns section no longer names ledger padding — a gap opened and closed inside one wrap reads as normal again")
	}
	if !strings.Contains(lower, "survives the cheap-fix test") {
		t.Error("the silent-deferrals anti-pattern no longer defers to the cheap-fix test, so it contradicts the deferral step it sits beside")
	}
}

// TestAiwfxWrapMilestone_ConstraintDefersToCheapFixTest pins the third copy of
// the rule. Constraints is the section a reader consults for what the ritual
// refuses, so an unconditional bullet there overrides the narrowed step however
// carefully the step is worded.
func TestAiwfxWrapMilestone_ConstraintDefersToCheapFixTest(t *testing.T) {
	t.Parallel()

	section := extractMarkdownSection(loadAiwfxWrapMilestoneFixture(t), 2, "Constraints")
	if section == "" {
		t.Fatalf("Constraints section not found in %s", aiwfxWrapMilestoneFixturePath)
	}
	if !strings.Contains(strings.ToLower(section), "survive the cheap-fix test") {
		t.Error("the Constraints section states the deferral mandate unconditionally again, overriding the narrowed step-4 wording")
	}
}

// TestMilestoneSpecTemplate_DeferralsCarryCheapFixTest pins the rule where the
// decision is actually made. The skill bodies are read before the work; this
// template comment sits in the spec being filled, so an unconditional form here
// is the copy that decides behaviour.
func TestMilestoneSpecTemplate_DeferralsCarryCheapFixTest(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join(repoRoot(t), milestoneSpecTemplatePath))
	if err != nil {
		t.Fatalf("loading %s: %v", milestoneSpecTemplatePath, err)
	}
	section := extractMarkdownSection(string(data), 2, "Deferrals")
	if section == "" {
		t.Fatalf("Deferrals section not found in %s", milestoneSpecTemplatePath)
	}
	lower := strings.ToLower(section)
	for _, fragment := range []string{"cheap-fix test", "already touches", "survives the test"} {
		if !strings.Contains(lower, fragment) {
			t.Errorf("the template's Deferrals guidance has lost %q — the surface a builder reads while filling the section still mandates a gap entity for every punt", fragment)
		}
	}
}

// TestAiwfxStartMilestone_DeferralLineCarriesCheapFixTest pins the same escape
// at the point work is deferred rather than at wrap. Pinning only the wrap copy
// would leave mid-implementation deferrals filing unconditionally, which is
// where most of them surface.
func TestAiwfxStartMilestone_DeferralLineCarriesCheapFixTest(t *testing.T) {
	t.Parallel()

	section := findStartMilestoneImplementationSection(loadAiwfxStartMilestoneFixture(t))
	if section == "" {
		t.Fatalf("implementation step not found in %s", aiwfxStartMilestoneFixturePath)
	}
	lower := strings.ToLower(section)
	for _, fragment := range []string{"cheap-fix test", "make it now rather than filing it"} {
		if !strings.Contains(lower, fragment) {
			t.Errorf("the implementation step has lost %q — a mid-implementation deferral files a gap unconditionally again", fragment)
		}
	}
}
