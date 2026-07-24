package policies

import (
	"strings"
	"testing"
)

// m0274TddCycleFixturePath is the canonical authoring location of the
// wf-tdd-cycle ritual skill whose RED step M-0274/AC-4 rewrites.
const m0274TddCycleFixturePath = "internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-tdd-cycle/SKILL.md"

// TestM0274_TddCycleRedPromoteIsLiveMandatory pins M-0274/AC-4: the
// wf-tdd-cycle RED step names the "" → red phase promote as a live,
// mandatory step run the moment the failing test is written and shown to
// fail — and no longer tells the operator to skip it because the AC was
// "already seeded at red." Since `aiwf add ac` now seeds the pre-cycle
// empty phase, the promote is always a live transition, never a redundant
// re-run.
//
// Structural per CLAUDE.md *Substring assertions are not structural
// assertions*: scoped to the `### RED — Write` section, not grepped
// file-wide.
func TestM0274_TddCycleRedPromoteIsLiveMandatory(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, m0274TddCycleFixturePath)

	red := sectionUnder(body, "RED — Write")
	if red == "" {
		t.Fatal("wf-tdd-cycle has no 'RED — Write ...' section")
	}
	lower := strings.ToLower(red)

	// The live red promote command must be named in the RED step.
	if !strings.Contains(red, "aiwf promote M-NNN/AC-<N> --phase red") {
		t.Error("AC-4: the RED step must drive the live `aiwf promote M-NNN/AC-<N> --phase red` promote")
	}
	// It must be framed as a live, mandatory step tied to the failing test.
	for _, want := range []string{"live", "mandatory", "the moment the failing test"} {
		if !strings.Contains(lower, want) {
			t.Errorf("AC-4: the RED promote must be framed as a live mandatory step run when the test fails — missing %q", want)
		}
	}
	// The stale born-red skip guidance must be gone: an AC is no longer
	// "already seeded at red," so there is no step to skip, no "redundant"
	// re-run to warn about, and no re-run to mislabel as "idempotent" (the
	// FSM refuses red → red, so a re-run errors — it is never a silent
	// no-op; this preserves the honesty guard the retired G-0297 RED
	// assertion used to hold).
	for _, gone := range []string{"skip this step", "already seeded at", "redundant", "idempotent"} {
		if strings.Contains(lower, gone) {
			t.Errorf("AC-4: stale born-red skip/re-run guidance must be gone from the RED step — found %q", gone)
		}
	}
}
