package verb_test

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// TestPromote_SameStatus_NoResolverFlag_ReturnsNoOp pins M-0281/AC-1: a
// promote whose target status already equals the entity's current status,
// with no resolver flag, converges to a NoOp Result instead of returning a
// Go error. The canonical operator case is `aiwf promote M-NNNN done` run a
// second time (interactively, or from a forgotten script). The guard is
// kind-agnostic; an ADR is the cleanest fixture — `accepted` carries no
// resolver requirement, needs no ACs, and is not a sovereign act.
func TestPromote_SameStatus_NoResolverFlag_ReturnsNoOp(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindADR, "Render envelope", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindADR)}))
	r.must(verb.Promote(r.ctx, r.tree(), "ADR-0001", "accepted", testActor, "", false, verb.PromoteOptions{}))

	res, err := verb.Promote(r.ctx, r.tree(), "ADR-0001", "accepted", testActor, "", false, verb.PromoteOptions{})
	if err != nil {
		t.Fatalf("same-status promote returned a Go error, want a clean NoOp: %v", err)
	}
	if !res.NoOp {
		t.Errorf("res.NoOp = false, want true (re-promoting to the current status is a no-op)")
	}
	if res.Plan != nil {
		t.Errorf("res.Plan = %+v, want nil (a NoOp produces no commit)", res.Plan)
	}
}

// TestPromote_SameStatus_ResolverBackfill_StillMutates guards the AC-1
// wrinkle: the same-status NoOp must NOT swallow the G-0096 resolver-backfill
// carve-out. A gap forced to `addressed` with an empty resolver is the stray
// state backfill repairs; re-promoting `addressed` with a resolver flag must
// still produce a Plan (write the resolver), never a NoOp.
func TestPromote_SameStatus_ResolverBackfill_StillMutates(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Cache", testActor,
		verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Stray gap", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap)}))
	// Force `addressed` with an empty resolver — the pre-G-0096 stray state.
	r.must(verb.Promote(r.ctx, r.tree(), "G-0001", "addressed", testActor, "stray fixture", true, verb.PromoteOptions{}))

	res, err := verb.Promote(r.ctx, r.tree(), "G-0001", "addressed", testActor, "", false,
		verb.PromoteOptions{AddressedBy: []string{"M-0001"}})
	if err != nil {
		t.Fatalf("resolver backfill returned a Go error: %v", err)
	}
	if res.NoOp {
		t.Errorf("res.NoOp = true, want false (a resolver-backfill same-status promote mutates)")
	}
	if res.Plan == nil {
		t.Fatal("res.Plan = nil, want a plan (backfill writes the resolver)")
	}
}

// TestPromote_SameStatus_Force_StillNoOp pins that --force does not turn a
// no-change same-status promote into a no-diff commit attempt: force relaxes
// the FSM transition rule, but there is still nothing to change, so the guard
// returns a NoOp regardless of force.
func TestPromote_SameStatus_Force_StillNoOp(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindADR, "Render envelope", testActor,
		verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindADR)}))
	r.must(verb.Promote(r.ctx, r.tree(), "ADR-0001", "accepted", testActor, "", false, verb.PromoteOptions{}))

	res, err := verb.Promote(r.ctx, r.tree(), "ADR-0001", "accepted", testActor, "forced rerun", true, verb.PromoteOptions{})
	if err != nil {
		t.Fatalf("forced same-status promote errored: %v", err)
	}
	if !res.NoOp {
		t.Errorf("res.NoOp = false, want true (nothing to change even under --force)")
	}
}
