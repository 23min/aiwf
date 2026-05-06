package verb_test

import (
	"context"
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/gitops"
	"github.com/23min/ai-workflow-v2/internal/verb"
)

// TestPromoteAuditOnly_HappyPath: an entity is already at the named
// state (reached via a manual commit). PromoteAuditOnly produces an
// empty-diff commit carrying the standard trailer block plus
// aiwf-audit-only with the reason.
func TestPromoteAuditOnly_HappyPath(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(context.Background(), r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-01", "active", testActor, "begin", false, verb.PromoteOptions{}))
	// E-01 is now `active`. Audit-only against `active` should pass.
	res, err := verb.PromoteAuditOnly(r.ctx, r.tree(), "E-01", "active", testActor, "manual fixup, recovering trail")
	if err != nil {
		t.Fatalf("PromoteAuditOnly: %v", err)
	}
	if !res.Plan.AllowEmpty {
		t.Error("AllowEmpty = false, want true")
	}
	if len(res.Plan.Ops) != 0 {
		t.Errorf("Ops len = %d, want 0", len(res.Plan.Ops))
	}
	if applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("apply: %v", applyErr)
	}
	tr, err := gitops.HeadTrailers(r.ctx, r.root)
	if err != nil {
		t.Fatal(err)
	}
	mustHaveTrailer(t, tr, "aiwf-verb", "promote")
	mustHaveTrailer(t, tr, "aiwf-entity", "E-01")
	mustHaveTrailer(t, tr, "aiwf-actor", testActor)
	mustHaveTrailer(t, tr, "aiwf-to", "active")
	mustHaveTrailer(t, tr, "aiwf-audit-only", "manual fixup, recovering trail")
	for _, x := range tr {
		if x.Key == "aiwf-force" {
			t.Errorf("aiwf-force present on audit-only commit: %q", x.Value)
		}
	}
}

// TestPromoteAuditOnly_RefusesWhenStateMismatch: --audit-only against
// a state the entity hasn't reached yet is refused — audit-only never
// transitions, only documents.
func TestPromoteAuditOnly_RefusesWhenStateMismatch(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	// E-01 is `proposed`. Audit-only to `done` (which the entity has
	// not reached) must refuse.
	_, err := verb.PromoteAuditOnly(r.ctx, r.tree(), "E-01", "done", testActor, "trying to skip ahead")
	if err == nil || !strings.Contains(err.Error(), "audit-only records what's already true") {
		t.Errorf("expected state-mismatch refusal; got %v", err)
	}
}

// TestPromoteAuditOnly_RequiresReason: audit-only with no reason (or
// whitespace-only) is refused.
func TestPromoteAuditOnly_RequiresReason(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	_, err := verb.PromoteAuditOnly(r.ctx, r.tree(), "E-01", "proposed", testActor, "   ")
	if err == nil || !strings.Contains(err.Error(), "non-empty --reason") {
		t.Errorf("expected non-empty-reason refusal; got %v", err)
	}
}

// TestPromoteAuditOnly_RefusesNonHumanActor: the audit-only commit
// goes through CheckTrailerCoherence and is refused. With the cmd-
// supplied trailer set (no aiwf-principal), the principal-missing-
// for-non-human-actor rule fires first; with a principal supplied,
// audit-only-non-human fires. Both are correct refusals for "non-
// human actor cannot wield audit-only" — the specific rule depends
// on what other trailers came along.
func TestPromoteAuditOnly_RefusesNonHumanActor(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	_, err := verb.PromoteAuditOnly(r.ctx, r.tree(), "E-01", "proposed", "ai/claude", "trying to backfill from a bot")
	if err == nil {
		t.Fatal("expected refusal for non-human actor")
	}
	ce, _ := verb.AsCoherenceError(err)
	if ce == nil {
		t.Fatalf("expected coherence error; got %v", err)
	}
	switch ce.Rule {
	case verb.CoherenceRulePrincipalMissingForNonHumanActor,
		verb.CoherenceRuleAuditOnlyNonHuman:
		// Both are valid refusals — see comment above.
	default:
		t.Errorf("expected non-human refusal rule; got %q (msg %q)", ce.Rule, ce.Message)
	}
}

// TestPromoteAuditOnly_RejectsUnknownStatus: an unknown status
// (typo, wrong kind) fails before the state-mismatch check fires.
func TestPromoteAuditOnly_RejectsUnknownStatus(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	_, err := verb.PromoteAuditOnly(r.ctx, r.tree(), "E-01", "Done", testActor, "wrong case")
	if err == nil || !strings.Contains(err.Error(), "not a recognized") {
		t.Errorf("expected unknown-status refusal; got %v", err)
	}
}

// TestCancelAuditOnly_HappyPath: a gap is already at `wontfix` (the
// kind's terminal-cancel target) via a manual commit. CancelAuditOnly
// produces the audit trailer.
func TestCancelAuditOnly_HappyPath(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Validators leak temp files", testActor, verb.AddOptions{
		DiscoveredIn: "",
	}))
	// Move the gap to `wontfix` via the normal path so the test's
	// fixture ends in a state where audit-only is the meaningful op.
	r.must(verb.Cancel(r.ctx, r.tree(), "G-001", testActor, "decided not to fix", false))
	// Now run audit-only on top: the entity is already at wontfix.
	res, err := verb.CancelAuditOnly(r.ctx, r.tree(), "G-001", testActor, "backfilling earlier manual flip")
	if err != nil {
		t.Fatalf("CancelAuditOnly: %v", err)
	}
	if applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("apply: %v", applyErr)
	}
	tr, err := gitops.HeadTrailers(r.ctx, r.root)
	if err != nil {
		t.Fatal(err)
	}
	mustHaveTrailer(t, tr, "aiwf-verb", "cancel")
	mustHaveTrailer(t, tr, "aiwf-entity", "G-001")
	mustHaveTrailer(t, tr, "aiwf-audit-only", "backfilling earlier manual flip")
	// Cancel does not emit aiwf-to: (target is implicit per kind).
	for _, x := range tr {
		if x.Key == "aiwf-to" {
			t.Errorf("aiwf-to present on cancel audit-only commit: %q", x.Value)
		}
	}
}

// TestCancelAuditOnly_RefusesWhenNotAtTerminal: the entity is still
// open (not yet at the kind's terminal-cancel target). Audit-only
// refuses.
func TestCancelAuditOnly_RefusesWhenNotAtTerminal(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Untriaged gap", testActor, verb.AddOptions{}))
	_, err := verb.CancelAuditOnly(r.ctx, r.tree(), "G-001", testActor, "no")
	if err == nil || !strings.Contains(err.Error(), "audit-only records what's already true") {
		t.Errorf("expected refusal; got %v", err)
	}
}

// TestPromoteACAuditOnly_HappyPath: composite-id audit-only on AC
// status. The AC is already at `met`; the verb records the audit.
func TestPromoteACAuditOnly_HappyPath(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Cache warmup", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "First criterion", testActor, nil))
	r.must(verb.Promote(r.ctx, r.tree(), "M-001/AC-1", "met", testActor, "actually done", false, verb.PromoteOptions{}))

	res, err := verb.PromoteAuditOnly(r.ctx, r.tree(), "M-001/AC-1", "met", testActor, "backfill")
	if err != nil {
		t.Fatalf("PromoteAuditOnly composite: %v", err)
	}
	if applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("apply: %v", applyErr)
	}
	tr, err := gitops.HeadTrailers(r.ctx, r.root)
	if err != nil {
		t.Fatal(err)
	}
	mustHaveTrailer(t, tr, "aiwf-entity", "M-001/AC-1")
	mustHaveTrailer(t, tr, "aiwf-audit-only", "backfill")
}

// TestPromoteACPhaseAuditOnly_HappyPath: --phase audit-only on a
// composite id. AC's tdd_phase already matches; verb records the
// audit.
func TestPromoteACPhaseAuditOnly_HappyPath(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Cache warmup", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "First criterion", testActor, nil))
	// AC starts with tdd_phase="" (the milestone is not tdd: required).
	// Phase audit-only against "" isn't allowed (entry state); flip
	// to red via normal promote first, then audit-only against red.
	r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-001/AC-1", "red", testActor, "begin", false, nil))

	res, err := verb.PromoteACPhaseAuditOnly(r.ctx, r.tree(), "M-001/AC-1", "red", testActor, "backfill")
	if err != nil {
		t.Fatalf("PromoteACPhaseAuditOnly: %v", err)
	}
	if applyErr := verb.Apply(r.ctx, r.root, res.Plan); applyErr != nil {
		t.Fatalf("apply: %v", applyErr)
	}
}

// TestPromoteACPhaseAuditOnly_RefusesUnknownPhase: a typo or invalid
// phase value is rejected before the state-mismatch check fires.
func TestPromoteACPhaseAuditOnly_RefusesUnknownPhase(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Engine", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Cache warmup", testActor, verb.AddOptions{EpicID: "E-01"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "First criterion", testActor, nil))

	_, err := verb.PromoteACPhaseAuditOnly(r.ctx, r.tree(), "M-001/AC-1", "Refactoring", testActor, "wrong case")
	if err == nil || !strings.Contains(err.Error(), "not a recognized tdd_phase") {
		t.Errorf("expected unknown-phase refusal; got %v", err)
	}
}
