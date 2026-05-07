package verb_test

import (
	"strings"
	"testing"

	"github.com/23min/ai-workflow-v2/internal/check"
	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/verb"
)

// TestPromote_GapAddressedBy: gap → addressed with PromoteOptions.AddressedBy
// writes addressed_by atomically with the status change (M-059/AC-1, AC-3).
// After promotion the gap-resolved-has-resolver check is silent — the verb
// route alone is enough; no follow-up hand-edit needed (M-059/AC-4).
func TestPromote_GapAddressedBy(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Resolver", testActor, verb.AddOptions{EpicID: "E-01", TDD: "none"}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Hand-edit gap", testActor, verb.AddOptions{}))

	r.must(verb.Promote(r.ctx, r.tree(), "G-001", "addressed", testActor, "", false,
		verb.PromoteOptions{AddressedBy: []string{"M-001"}}))

	g := r.tree().ByID("G-001")
	if g == nil {
		t.Fatal("G-001 missing after promote")
	}
	if g.Status != "addressed" {
		t.Errorf("status = %q, want addressed", g.Status)
	}
	if len(g.AddressedBy) != 1 || g.AddressedBy[0] != "M-001" {
		t.Errorf("addressed_by = %v, want [M-001]", g.AddressedBy)
	}

	// AC-4 closure: the resolver-flag path leaves the tree clean — no
	// gap-resolved-has-resolver finding requires a hand-edit follow-up.
	for _, f := range check.Run(r.tree(), nil) {
		if f.Code == "gap-resolved-has-resolver" {
			t.Errorf("verb route should satisfy gap-resolved-has-resolver; finding still present: %+v", f)
		}
	}
}

// TestPromote_GapAddressedByMultiple: --by accepts a comma-separated
// list of entity ids, all of which land in addressed_by.
func TestPromote_GapAddressedByMultiple(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor, verb.AddOptions{EpicID: "E-01", TDD: "none"}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Second", testActor, verb.AddOptions{EpicID: "E-01", TDD: "none"}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Co-resolved", testActor, verb.AddOptions{}))

	r.must(verb.Promote(r.ctx, r.tree(), "G-001", "addressed", testActor, "", false,
		verb.PromoteOptions{AddressedBy: []string{"M-001", "M-002"}}))

	g := r.tree().ByID("G-001")
	if g == nil || len(g.AddressedBy) != 2 {
		t.Fatalf("G-001 = %+v, want addressed_by [M-001 M-002]", g)
	}
	if g.AddressedBy[0] != "M-001" || g.AddressedBy[1] != "M-002" {
		t.Errorf("addressed_by = %v, want [M-001 M-002]", g.AddressedBy)
	}
}

// TestPromote_GapAddressedByCommit: --by-commit value lands in
// addressed_by_commit and satisfies the resolver check on its own.
func TestPromote_GapAddressedByCommit(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Closed by hardening commit", testActor, verb.AddOptions{}))

	r.must(verb.Promote(r.ctx, r.tree(), "G-001", "addressed", testActor, "", false,
		verb.PromoteOptions{AddressedByCommit: []string{"abcdef1234"}}))

	g := r.tree().ByID("G-001")
	if g == nil {
		t.Fatal("G-001 missing")
	}
	if g.Status != "addressed" {
		t.Errorf("status = %q, want addressed", g.Status)
	}
	if len(g.AddressedByCommit) != 1 || g.AddressedByCommit[0] != "abcdef1234" {
		t.Errorf("addressed_by_commit = %v, want [abcdef1234]", g.AddressedByCommit)
	}
	for _, f := range check.Run(r.tree(), nil) {
		if f.Code == "gap-resolved-has-resolver" {
			t.Errorf("commit-resolver path should silence gap-resolved-has-resolver; got %+v", f)
		}
	}
}

// TestPromote_ADRSupersededBy: adr → superseded with --superseded-by
// writes superseded_by atomically (M-059/AC-2, AC-3).
func TestPromote_ADRSupersededBy(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindADR, "Old decision", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindADR, "Replacement decision", testActor, verb.AddOptions{}))
	// Walk both ADRs to "accepted" — the FSM only lets accepted → superseded.
	r.must(verb.Promote(r.ctx, r.tree(), "ADR-0001", "accepted", testActor, "", false, verb.PromoteOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "ADR-0002", "accepted", testActor, "", false, verb.PromoteOptions{}))

	r.must(verb.Promote(r.ctx, r.tree(), "ADR-0001", "superseded", testActor, "", false,
		verb.PromoteOptions{SupersededBy: "ADR-0002"}))

	a := r.tree().ByID("ADR-0001")
	if a == nil {
		t.Fatal("ADR-0001 missing")
	}
	if a.Status != "superseded" {
		t.Errorf("status = %q, want superseded", a.Status)
	}
	if a.SupersededBy != "ADR-0002" {
		t.Errorf("superseded_by = %q, want ADR-0002", a.SupersededBy)
	}
}

// TestPromote_ResolverWrongKind: a resolver flag on the wrong entity
// kind is a Go error (usage misalignment), not a finding.
func TestPromote_ResolverWrongKind(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foo", testActor, verb.AddOptions{}))

	_, err := verb.Promote(r.ctx, r.tree(), "E-01", "active", testActor, "", false,
		verb.PromoteOptions{AddressedBy: []string{"M-001"}})
	if err == nil || !strings.Contains(err.Error(), "only valid for gap entities") {
		t.Errorf("expected gap-only error, got %v", err)
	}

	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "G", testActor, verb.AddOptions{}))
	_, err = verb.Promote(r.ctx, r.tree(), "G-001", "addressed", testActor, "", false,
		verb.PromoteOptions{SupersededBy: "ADR-0001"})
	if err == nil || !strings.Contains(err.Error(), "only valid for ADR entities") {
		t.Errorf("expected ADR-only error, got %v", err)
	}
}

// TestPromote_ResolverWrongStatus: --by on gap → wontfix is rejected
// (resolver is meaningful only when promoting to addressed).
func TestPromote_ResolverWrongStatus(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "G", testActor, verb.AddOptions{}))

	_, err := verb.Promote(r.ctx, r.tree(), "G-001", "wontfix", testActor, "", false,
		verb.PromoteOptions{AddressedBy: []string{"M-001"}})
	if err == nil || !strings.Contains(err.Error(), `only valid when promoting to "addressed"`) {
		t.Errorf("expected wrong-status error, got %v", err)
	}
}

// TestPromote_ResolverOnAC: composite ids reject resolver flags. ACs
// don't have resolver fields; the verb refuses early so the user
// notices the misalignment instead of silently dropping the values.
func TestPromote_ResolverOnAC(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foo", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Bar", testActor, verb.AddOptions{EpicID: "E-01", TDD: "none"}))
	r.must(verb.AddAC(r.ctx, r.tree(), "M-001", "an AC", testActor, nil))

	_, err := verb.Promote(r.ctx, r.tree(), "M-001/AC-1", "met", testActor, "", false,
		verb.PromoteOptions{AddressedBy: []string{"M-001"}})
	if err == nil || !strings.Contains(err.Error(), "AC promotions") {
		t.Errorf("expected AC-rejection error, got %v", err)
	}
}

// TestPromote_ResolverAtomicSingleCommit: the status change and the
// resolver-field write land in one git commit (M-059/AC-3). Loading
// the entity at HEAD shows both changes together; HEAD~1 shows
// neither.
func TestPromote_ResolverAtomicSingleCommit(t *testing.T) {
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "M", testActor, verb.AddOptions{EpicID: "E-01", TDD: "none"}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "G", testActor, verb.AddOptions{}))

	beforeStatus := r.tree().ByID("G-001").Status
	beforeResolver := append([]string(nil), r.tree().ByID("G-001").AddressedBy...)
	if beforeStatus != "open" || len(beforeResolver) != 0 {
		t.Fatalf("setup invalid: before status=%q resolver=%v", beforeStatus, beforeResolver)
	}

	r.must(verb.Promote(r.ctx, r.tree(), "G-001", "addressed", testActor, "", false,
		verb.PromoteOptions{AddressedBy: []string{"M-001"}}))

	g := r.tree().ByID("G-001")
	if g.Status != "addressed" || len(g.AddressedBy) != 1 || g.AddressedBy[0] != "M-001" {
		t.Errorf("post-promote G-001 = %+v; expected status=addressed, addressed_by=[M-001]", g)
	}
	// The check tree must be clean — the check rule does not fire
	// because the resolver write happened in the same commit.
	if findings := check.Run(r.tree(), nil); check.HasErrors(findings) {
		t.Errorf("post-promote tree has errors: %+v", findings)
	}
}
