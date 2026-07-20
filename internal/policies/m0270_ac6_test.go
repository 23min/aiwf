package policies

import (
	"strings"
	"testing"
)

// TestM0270_AC6_GapTracksUpgradeRollbackAbsence asserts M-0270/AC-6:
// a gap entity records that `aiwf upgrade` has no automated rollback
// (F12) — discovered in this milestone, still open (not silently
// closed without a real fix or an explicit decision landing), and
// its title still names the rollback absence so a future retitle
// can't quietly drift the anchor away from what this AC actually
// claims.
//
// The entity is resolved through the loader (never a hardcoded
// work/gaps path) per CLAUDE.md's "Policy tests that read entity
// files resolve via the loader" rule — archive sweeps move files,
// and a hardcoded path would silently stop matching once G-0430
// eventually moves to work/gaps/archive/.
func TestM0270_AC6_GapTracksUpgradeRollbackAbsence(t *testing.T) {
	t.Parallel()
	_, tr := sharedRepoTree(t)
	e := tr.ByID("G-0430")
	if e == nil {
		t.Fatal("AC-6: G-0430 not found in tree — the gap tracking aiwf upgrade's missing rollback must exist")
	}
	if e.DiscoveredIn != "M-0270" {
		t.Errorf("AC-6: G-0430 discovered_in = %q, want %q", e.DiscoveredIn, "M-0270")
	}
	if e.Status != "open" {
		t.Errorf("AC-6: G-0430 status = %q, want %q (a rollback fix or an explicit decision should accompany any other status)", e.Status, "open")
	}
	lower := strings.ToLower(e.Title)
	if !strings.Contains(lower, "upgrade") || !strings.Contains(lower, "rollback") {
		t.Errorf("AC-6: G-0430 title %q must still name both `upgrade` and `rollback` — a retitle away from this anchor would silently orphan the AC's claim", e.Title)
	}
}
