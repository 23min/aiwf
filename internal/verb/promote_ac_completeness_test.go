package verb_test

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// promote_ac_completeness_test.go pins M-0268/AC-1: a milestone with
// an empty acs[] cannot start (draft -> in_progress) without --force.
// Real work needs a contract before it starts; a milestone nobody has
// written any AC for is the clearest case of "no contract yet"
// (D-0039 point 1).
//
// Every scenario here first activates the parent epic and checks out
// its ritual branch, matching G-0269's own branch-guard fixture
// pattern (promote_branch_guard_test.go) — the milestone in_progress
// transition is itself a sovereign activating act guarded by
// requireExpectedBranchForActivatingTransition, and without landing
// on the expected branch that guard's own refusal fires first,
// masking the zero-AC guard under test.

// setupACLessMilestoneOnEpicBranch activates a fresh epic, adds a
// zero-AC milestone under it, and checks out the epic's own ritual
// branch — the shared preamble every test below needs before it can
// exercise the zero-AC guard in isolation from G-0269's branch guard.
func setupACLessMilestoneOnEpicBranch(t *testing.T) *runner {
	t.Helper()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "No ACs yet", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	gitCheckoutNewBranch(t, r.root, "epic/E-0001-platform")
	return r
}

// TestPromote_ZeroACMilestoneRefusedAtDraftToInProgress is the
// headline case.
func TestPromote_ZeroACMilestoneRefusedAtDraftToInProgress(t *testing.T) {
	t.Parallel()
	r := setupACLessMilestoneOnEpicBranch(t)

	_, err := verb.Promote(r.ctx, r.tree(), "M-0001", "in_progress", testActor, "", false, verb.PromoteOptions{})
	if err == nil {
		t.Fatal("expected error promoting a zero-AC milestone to in_progress; got nil")
	}
	if !strings.Contains(err.Error(), "M-0001") {
		t.Errorf("error should name the milestone; got %v", err)
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("error should point at --force as the override; got %v", err)
	}
	if m := r.tree().ByID("M-0001"); m == nil || m.Status != entity.StatusDraft {
		t.Errorf("refused promote must not mutate status; M-0001 = %+v", m)
	}
}

// TestPromote_ZeroACMilestoneForceOverridesRefusal pins the Design-
// notes divergence from the adjacent unconditional structural guards
// (MilestonePromoteNonTerminalACsError / EpicPromoteNonTerminalChildren-
// Error): AC-1's refusal is a soft precondition, not a structural
// invariant, so --force lets it through — mirroring the resolver-
// requirement checks' own --force behavior.
func TestPromote_ZeroACMilestoneForceOverridesRefusal(t *testing.T) {
	t.Parallel()
	r := setupACLessMilestoneOnEpicBranch(t)

	r.must(verb.Promote(r.ctx, r.tree(), "M-0001", "in_progress", testActor, "starting anyway", true, verb.PromoteOptions{}))

	m := r.tree().ByID("M-0001")
	if m == nil || m.Status != entity.StatusInProgress {
		t.Fatalf("force-promote should have landed in_progress; got %+v", m)
	}
}

// TestPromote_ZeroACMilestone_OtherTransitionsUnaffected pins the
// Constraints scope: AC-1's refusal applies only to draft ->
// in_progress. A zero-AC milestone can still legally reach cancelled
// (the FSM already permits draft -> cancelled without any AC
// precondition at that transition, and cancelled is not an
// activating transition so the branch guard does not apply either).
func TestPromote_ZeroACMilestone_OtherTransitionsUnaffected(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "No ACs yet", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))

	r.must(verb.Promote(r.ctx, r.tree(), "M-0001", "cancelled", testActor, "", false, verb.PromoteOptions{}))

	m := r.tree().ByID("M-0001")
	if m == nil || m.Status != entity.StatusCancelled {
		t.Fatalf("draft -> cancelled should be unaffected by the zero-AC guard; got %+v", m)
	}
}

// TestPromote_NonZeroACMilestoneUnaffectedByZeroACGuard is the
// regression companion: a milestone with at least one AC promotes
// draft -> in_progress exactly as before, with no --force needed.
func TestPromote_NonZeroACMilestoneUnaffectedByZeroACGuard(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Has one AC", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	// M-0268/AC-2: draft -> in_progress now also refuses an empty AC
	// body; give this AC real prose so it exercises AC-1's own
	// regression, not AC-2's guard.
	r.must(verb.AddACBatch(r.ctx, r.tree(), "M-0001", []string{"Does the thing"}, [][]byte{[]byte("Real prose.")}, testActor))
	gitCheckoutNewBranch(t, r.root, "epic/E-0001-platform")

	r.must(verb.Promote(r.ctx, r.tree(), "M-0001", "in_progress", testActor, "", false, verb.PromoteOptions{}))

	m := r.tree().ByID("M-0001")
	if m == nil || m.Status != entity.StatusInProgress {
		t.Fatalf("non-zero-AC milestone should promote cleanly; got %+v", m)
	}
}
