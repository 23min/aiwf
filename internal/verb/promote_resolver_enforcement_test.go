package verb_test

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// TestPromote_GapAddressedRequiresResolver pins G-0096's verb-time
// invariant: `aiwf promote <gap> addressed` is rejected when neither
// `--by` nor `--by-commit` is set. Before this fix, the verb accepted
// the empty case and the gap-resolved-has-resolver warning surfaced
// post-hoc — but warnings don't block the pre-push hook, so the gap
// landed in `addressed` with no resolver and no path back without
// --force (since `addressed` is terminal in the gap FSM). G-0096
// closes that hole at the verb chokepoint.
func TestPromote_GapAddressedRequiresResolver(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Empty resolver", testActor, verb.AddOptions{}))

	_, err := verb.Promote(r.ctx, r.tree(), "G-0001", "addressed", testActor, "", false, verb.PromoteOptions{})
	if err == nil {
		t.Fatal("expected error when promoting gap to addressed without --by/--by-commit; got nil")
	}
	if !strings.Contains(err.Error(), "--by") || !strings.Contains(err.Error(), "--by-commit") {
		t.Errorf("error should mention both --by and --by-commit; got %v", err)
	}
}

// TestPromote_ADRSupersededRequiresResolver pins the parallel rule
// for ADRs: promoting to `superseded` requires --superseded-by. Same
// failure mode as the gap case — adr-supersession-mutual is a
// warning, not blocking, so the verb is the chokepoint.
func TestPromote_ADRSupersededRequiresResolver(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindADR, "Old decision", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "ADR-0001", "accepted", testActor, "", false, verb.PromoteOptions{}))

	_, err := verb.Promote(r.ctx, r.tree(), "ADR-0001", "superseded", testActor, "", false, verb.PromoteOptions{})
	if err == nil {
		t.Fatal("expected error when promoting ADR to superseded without --superseded-by; got nil")
	}
	if !strings.Contains(err.Error(), "--superseded-by") {
		t.Errorf("error should mention --superseded-by; got %v", err)
	}
}

// TestPromote_ResolverRequirementBypassedByForce pins the sovereign-
// override path: even with the verb-time enforcement, --force lets
// the empty-resolver promote land. The user takes the consequence
// (a follow-up gap-resolved-has-resolver warning) but the option is
// preserved for genuine exceptional cases.
func TestPromote_ResolverRequirementBypassedByForce(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Force-resolved", testActor, verb.AddOptions{}))

	r.must(verb.Promote(r.ctx, r.tree(), "G-0001", "addressed", testActor, "manual cleanup", true, verb.PromoteOptions{}))

	g := r.tree().ByID("G-0001")
	if g == nil || g.Status != "addressed" {
		t.Fatalf("force-promote should have landed addressed; got %+v", g)
	}
	if len(g.AddressedBy) != 0 || len(g.AddressedByCommit) != 0 {
		t.Errorf("force-promote should not have set resolver fields; got addressed_by=%v addressed_by_commit=%v",
			g.AddressedBy, g.AddressedByCommit)
	}
}

// TestPromote_BackfillResolverOnAddressedGap pins G-0096's same-
// status carve-out: a gap already in `addressed` with empty resolver
// fields can receive --by via a same-status promote. The status
// doesn't change (it's already `addressed`); only the resolver
// metadata is written. Without this carve-out, legacy gaps that
// pre-dated the verb-time enforcement could only be cleaned via
// --force.
func TestPromote_BackfillResolverOnAddressedGap(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Closer", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Stray", testActor, verb.AddOptions{}))
	// Land the gap in the no-resolver `addressed` state via --force,
	// simulating a pre-G-0096 promote. From here the only path to a
	// resolver is the new same-status back-fill.
	r.must(verb.Promote(r.ctx, r.tree(), "G-0001", "addressed", testActor, "simulate legacy state", true, verb.PromoteOptions{}))

	r.must(verb.Promote(r.ctx, r.tree(), "G-0001", "addressed", testActor, "", false,
		verb.PromoteOptions{AddressedBy: []string{"E-0001"}}))

	g := r.tree().ByID("G-0001")
	if g.Status != "addressed" {
		t.Errorf("status = %q, want addressed (back-fill leaves status alone)", g.Status)
	}
	if len(g.AddressedBy) != 1 || g.AddressedBy[0] != "E-0001" {
		t.Errorf("addressed_by = %v, want [E-0001]", g.AddressedBy)
	}

	// Closure: gap-resolved-has-resolver is silent after back-fill.
	for _, f := range check.Run(r.tree(), nil) {
		if f.Code == "gap-resolved-has-resolver" {
			t.Errorf("back-fill should silence gap-resolved-has-resolver; finding still present: %+v", f)
		}
	}
}

// TestPromote_BackfillRejectedWhenResolverAlreadySet pins the
// carve-out's narrow scope: same-status promote is allowed only when
// the entity's resolver is currently empty. Once a resolver is set,
// further "rewrite the pointer" operations need a deliberate path
// (not yet available) or --force. This keeps the carve-out from
// becoming a generic "edit any frontmatter field" surface.
func TestPromote_BackfillRejectedWhenResolverAlreadySet(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "First", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Second", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Already-resolved", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "G-0001", "addressed", testActor, "", false,
		verb.PromoteOptions{AddressedBy: []string{"E-0001"}}))

	_, err := verb.Promote(r.ctx, r.tree(), "G-0001", "addressed", testActor, "", false,
		verb.PromoteOptions{AddressedBy: []string{"E-0002"}})
	if err == nil {
		t.Fatal("expected FSM error on same-status promote with resolver already set; got nil")
	}
	// The error comes from ValidateTransition (not the back-fill
	// branch), so it speaks of the FSM transition, not the resolver.
	if !strings.Contains(err.Error(), "addressed") {
		t.Errorf("error should reference the addressed status; got %v", err)
	}
}

// TestPromote_ADRSupersededBackfill pins that the same-status carve-
// out applies symmetrically to ADR superseded. A pre-G-0096 ADR in
// `superseded` with empty `superseded_by` can be back-filled via the
// same path.
func TestPromote_ADRSupersededBackfill(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindADR, "Old", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindADR, "New", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "ADR-0001", "accepted", testActor, "", false, verb.PromoteOptions{}))
	// Force the no-superseded-by superseded state (legacy simulation).
	r.must(verb.Promote(r.ctx, r.tree(), "ADR-0001", "superseded", testActor, "simulate legacy", true, verb.PromoteOptions{}))

	r.must(verb.Promote(r.ctx, r.tree(), "ADR-0001", "superseded", testActor, "", false,
		verb.PromoteOptions{SupersededBy: "ADR-0002"}))

	a := r.tree().ByID("ADR-0001")
	if a.Status != "superseded" {
		t.Errorf("status = %q, want superseded", a.Status)
	}
	if a.SupersededBy != "ADR-0002" {
		t.Errorf("superseded_by = %q, want ADR-0002", a.SupersededBy)
	}
}
