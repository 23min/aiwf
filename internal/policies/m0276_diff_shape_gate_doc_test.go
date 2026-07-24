package policies

import (
	"strings"
	"testing"
)

// m0276TddCycleFixturePath is the canonical authoring location of the
// wf-tdd-cycle ritual skill whose cycle M-0276/AC-7 documents with the
// red/green diff-shape gate.
const m0276TddCycleFixturePath = "internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-tdd-cycle/SKILL.md"

// TestM0276_TddCycleDocumentsDiffShapeGate pins M-0276/AC-7 (as narrowed by
// D-0049): the wf-tdd-cycle skill documents the red-first diff-shape gate in a
// named section — that a `--phase red` promote wants test-only dirtiness, that
// `--phase green` is not gated, and that the refusal is `--force`-overridable —
// and the RED step references it so an operator understands why a promote may
// refuse.
//
// Structural per CLAUDE.md *Substring assertions are not structural
// assertions*: scoped to the named gate section (and the RED step), not grepped
// file-wide.
func TestM0276_TddCycleDocumentsDiffShapeGate(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, m0276TddCycleFixturePath)

	gate := sectionUnder(body, "diff-shape gate")
	if gate == "" {
		t.Fatal("wf-tdd-cycle has no 'diff-shape gate' section documenting the red-first gate")
	}
	lower := strings.ToLower(gate)

	// The gate section names the gated red promote, the test-path config surface,
	// and the --force escape hatch.
	for _, want := range []string{"--phase red", "--force", "test-path"} {
		if !strings.Contains(lower, strings.ToLower(want)) {
			t.Errorf("AC-7: the diff-shape gate section must document %q", want)
		}
	}
	// Red-only (D-0049): --phase green must be documented as not gated.
	if !strings.Contains(lower, "not gated") {
		t.Error("AC-7: the section must document that --phase green is not gated (red-only, D-0049)")
	}
	// Test-first semantics: the red gate is about the test preceding the implementation.
	if !strings.Contains(lower, "test") || !strings.Contains(lower, "implementation") {
		t.Error("AC-7: the gate section must state the red gate is test-first (test before implementation)")
	}

	// The RED step references the gate so an operator understands why a promote
	// may refuse.
	if !strings.Contains(strings.ToLower(sectionUnder(body, "RED — Write")), "diff-shape gate") {
		t.Error("AC-7: the RED step must reference the diff-shape gate")
	}
}
