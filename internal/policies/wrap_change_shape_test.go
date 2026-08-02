package policies

import (
	"strings"
	"testing"
)

// wrap_change_shape_test.go — structural pins for the change-shape block in the
// wrap-milestone review step.
//
// Every other check at wrap is a floor: ACs terminal, check clean, suite green,
// lint clean — each asks whether something is missing. This block is the only
// one that asks whether something is unnecessary, so losing it restores an
// asymmetry nothing else in the ritual would report.
//
// It is pinned inside the review step specifically. The numbers are worth
// nothing coming from the author who produced them; their value is that an
// independent reviewer derives them, so a copy that drifted into any other step
// would satisfy a file-wide grep while losing the property.

// TestAiwfxWrapMilestone_ReviewStepCarriesChangeShape pins the three measures
// the reviewer is briefed to take. Each catches what the others cannot: the
// bucketed diffstat shows where the milestone spent, the rules count is the
// signature of an obligation that recurs, and the cluster grouping catches
// several spellings of one condition tested once per noun.
func TestAiwfxWrapMilestone_ReviewStepCarriesChangeShape(t *testing.T) {
	t.Parallel()

	step := findWorkflowSubsection(loadAiwfxWrapMilestoneFixture(t), "independent")
	if step == "" {
		t.Fatalf("independent-review step not found in %s", aiwfxWrapMilestoneFixturePath)
	}
	lower := strings.ToLower(step)
	for _, fragment := range []string{
		"git diff --numstat", // the bucketed spend
		// The bold-heading forms, not the bare phrases: each phrase also occurs
		// in the block's own prose, so a bare match survives its heading being
		// renamed away and pins nothing.
		"**recurring obligation.**",
		"**deletions.**",
		"**same-outcome clusters.**",
	} {
		if !strings.Contains(lower, fragment) {
			t.Errorf("the review step has lost %q — the reviewer judges a diff with no measure of its size, and nothing at wrap asks whether the milestone spent more than it needed to", fragment)
		}
	}
}

// TestAiwfxWrapMilestone_ChangeShapeAnswersAreNotOptional pins what makes the
// block non-vacuous. An open question the answerer may say "no" to costs one
// token and nothing contradicts it; each of these three either carries a
// trigger that forbids the cheap answer, or refuses silence outright.
func TestAiwfxWrapMilestone_ChangeShapeAnswersAreNotOptional(t *testing.T) {
	t.Parallel()

	step := findWorkflowSubsection(loadAiwfxWrapMilestoneFixture(t), "independent")
	if step == "" {
		t.Fatalf("independent-review step not found in %s", aiwfxWrapMilestoneFixturePath)
	}
	lower := strings.ToLower(step)
	if !strings.Contains(lower, "cannot be \"none\"") {
		t.Error("the recurring-obligation question no longer contradicts a non-zero rules count, so \"none\" is answerable without reading the diff")
	}
	if !strings.Contains(lower, "silence is not") {
		t.Error("the deletions question no longer refuses silence, so a milestone that retired nothing can pass without saying so")
	}
	if !strings.Contains(lower, "not the author") {
		t.Error("the cluster question no longer assigns the call to the reviewer, returning it to the agent that wrote the tests")
	}
}

// TestAiwfxWrapMilestone_ChangeShapeStaysProjectNeutral pins the property that
// keeps the block usable where it ships. Every surface under the embedded
// rituals materializes into consumer repos, most of which are not Go and none
// of which carry this project's own rule-construction identifiers; a measure
// spelled as one language's grep reports zero there and the trigger goes inert
// without ever failing.
func TestAiwfxWrapMilestone_ChangeShapeStaysProjectNeutral(t *testing.T) {
	t.Parallel()

	step := findWorkflowSubsection(loadAiwfxWrapMilestoneFixture(t), "independent")
	if step == "" {
		t.Fatalf("independent-review step not found in %s", aiwfxWrapMilestoneFixturePath)
	}
	// Bare "Policy" rather than "Policy:" — the spelling that produced the
	// overcount this ban exists to prevent was `grep -c '^+.*Policy'`, which
	// a colon-qualified ban lets straight through.
	for _, leaked := range []string{"_test.go", "func Test", "Policy", "policyID", "go test", "golangci", "internal/"} {
		if strings.Contains(step, leaked) {
			t.Errorf("the review step names %q — a language- and project-specific mechanism on a surface that ships into consumer repos, where it silently measures nothing", leaked)
		}
	}
	if !strings.Contains(strings.ToLower(step), "in the project's own terms") {
		t.Error("the review step no longer defers the measurement to the project's own terms, so a consumer has no instruction for how to count anything")
	}
}
