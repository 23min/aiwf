package policies

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// TestM0270_AC6_GapTracksUpgradeRollbackAbsence asserts M-0270/AC-6:
// a gap entity records that `aiwf upgrade` has no automated rollback
// (F12) — discovered in this milestone, never silently closed, and
// its title still names the rollback absence so a future retitle
// can't quietly drift the anchor away from what this AC actually
// claims.
//
// "Never silently closed" means the gap is either open or addressed by
// an accepted decision: the decision exit D-0043 names. D-0108 rules out
// its other exit, a shipped rollback. A decision that is still proposed,
// or was later superseded, does not count: the closure would then rest on
// a judgment nobody holds.
//
// The entity is resolved through the loader (never a hardcoded
// work/gaps path) per .guidance/project/go-conventions.md's "Policy tests that read entity
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
	switch e.Status {
	case entity.StatusOpen:
	case entity.StatusAddressed:
		if !addressedByAcceptedDecision(tr, e.AddressedBy) {
			t.Errorf("AC-6: G-0430 is addressed, but addressed_by %v names no accepted decision; closing it needs an accepted decision, D-0043's decision exit", e.AddressedBy)
		}
	default:
		t.Errorf("AC-6: G-0430 status = %q, want open, or addressed by an accepted decision", e.Status)
	}
	lower := strings.ToLower(e.Title)
	if !strings.Contains(lower, "upgrade") || !strings.Contains(lower, "rollback") {
		t.Errorf("AC-6: G-0430 title %q must still name both `upgrade` and `rollback` — a retitle away from this anchor would silently orphan the AC's claim", e.Title)
	}
}

// addressedByAcceptedDecision reports whether any id in addressedBy
// resolves to a decision whose status is accepted.
func addressedByAcceptedDecision(tr *tree.Tree, addressedBy []string) bool {
	for _, id := range addressedBy {
		if d := tr.ByID(id); d != nil && d.Kind == entity.KindDecision && d.Status == entity.StatusAccepted {
			return true
		}
	}
	return false
}
