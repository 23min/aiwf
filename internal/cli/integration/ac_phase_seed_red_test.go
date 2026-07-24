package integration

import (
	"context"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/tree"
)

// TestPromoteACPhaseRed_LiveEmptyToRedUnderTDDRequired pins M-0274/AC-2 at
// the CLI seam. On a tdd: required milestone the real `aiwf add ac`
// command seeds the AC at the pre-cycle empty phase (M-0274/AC-1), and the
// real `aiwf promote --phase red` command then fires as a live "" → red
// transition: it exits 0 and the AC comes to rest at red. Before this
// milestone the AC was born at red, so that promote could never fire — the
// phase FSM refuses red → red — which is the exact regression this pins.
// The live event this test asserts is the one the M-0276 ordering gate
// attaches to, so the assertion runs through the actual command surface,
// not the verb function underneath it.
func TestPromoteACPhaseRed_LiveEmptyToRedUnderTDDRequired(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--tdd", "required", "--epic", "E-0001", "--title", "Required", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "ac", "--actor", "human/test", "--root", root, "M-0001", "--title", "Engine")

	// Precondition: the seeded state under tdd: required is the pre-cycle
	// empty phase, not red — otherwise the promote below could not be a
	// live "" → red transition.
	if got := acPhase(t, root, "M-0001"); got != "" {
		t.Fatalf("seeded phase = %q, want empty (pre-cycle)", got)
	}

	// The load-bearing AC-2 claim: the real promote command fires the
	// live "" → red transition and exits cleanly.
	if rc := cli.Execute([]string{"promote", "--actor", "human/test", "--root", root, "M-0001/AC-1", "--phase", "red"}); rc != cliutil.ExitOK {
		t.Fatalf(`promote --phase red: rc=%d, want ExitOK — the live "" → red transition must succeed`, rc)
	}

	// ...and the AC comes to rest at red.
	if got := acPhase(t, root, "M-0001"); got != "red" {
		t.Fatalf("after --phase red: phase = %q, want red", got)
	}
}

// acPhase loads root through the entity loader and returns the tdd_phase
// of the milestone's first AC. Reads via tree.Load rather than parsing the
// markdown so the assertion sees exactly what the kernel sees.
func acPhase(t *testing.T, root, milestoneID string) string {
	t.Helper()
	tr, _, err := tree.Load(context.Background(), root)
	if err != nil {
		t.Fatalf("tree.Load: %v", err)
	}
	m := tr.ByID(milestoneID)
	if m == nil {
		t.Fatalf("milestone %q not found in tree", milestoneID)
	}
	if len(m.ACs) == 0 {
		t.Fatalf("milestone %q has no ACs", milestoneID)
	}
	return m.ACs[0].TDDPhase
}
