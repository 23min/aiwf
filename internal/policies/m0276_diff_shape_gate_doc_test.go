package policies

import (
	"strings"
	"testing"
)

// m0276TddCycleFixturePath is the canonical authoring location of the
// wf-tdd-cycle ritual skill whose cycle M-0276/AC-7 documents with the
// red/green diff-shape gate.
const m0276TddCycleFixturePath = "internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-tdd-cycle/SKILL.md"

// TestM0276_TddCycleDocumentsDiffShapeGate pins M-0276/AC-7: the wf-tdd-cycle
// skill documents the red/green diff-shape gate in a named section — that a
// `--phase red` promote wants test-only dirtiness, a `--phase green` promote
// wants implementation dirtiness, and both refusals are `--force`-overridable —
// and the RED and GREEN steps reference it so an operator understands why a
// promote may refuse.
//
// Structural per CLAUDE.md *Substring assertions are not structural
// assertions*: scoped to the named gate section (and the RED / GREEN steps),
// not grepped file-wide.
func TestM0276_TddCycleDocumentsDiffShapeGate(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, m0276TddCycleFixturePath)

	gate := sectionUnder(body, "diff-shape gate")
	if gate == "" {
		t.Fatal("wf-tdd-cycle has no 'diff-shape gate' section documenting the red/green gate")
	}
	lower := strings.ToLower(gate)

	// The gate section names both gated promotes, the test-path config surface,
	// and the --force escape hatch.
	for _, want := range []string{"--phase red", "--phase green", "--force", "test-path"} {
		if !strings.Contains(lower, strings.ToLower(want)) {
			t.Errorf("AC-7: the diff-shape gate section must document %q", want)
		}
	}
	// Directional semantics: red wants the test, green wants the implementation.
	if !strings.Contains(lower, "test") || !strings.Contains(lower, "implementation") {
		t.Error("AC-7: the gate section must state red wants the test and green wants the implementation")
	}

	// The RED and GREEN steps reference the gate so an operator understands why
	// a promote may refuse.
	for _, sec := range []string{"RED — Write", "GREEN — Make"} {
		if !strings.Contains(strings.ToLower(sectionUnder(body, sec)), "diff-shape gate") {
			t.Errorf("AC-7: the %q step must reference the diff-shape gate", sec)
		}
	}
}
