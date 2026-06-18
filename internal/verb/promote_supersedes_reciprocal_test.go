package verb_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/verb"
)

// TestPromote_SupersededByWritesReciprocalSupersedes pins G-0255: a
// `promote <old> superseded --superseded-by <new>` must record the
// link on BOTH ADRs in the verb's single commit — `superseded_by` on
// the superseded ADR and the reciprocal `supersedes` on the
// superseding ADR — so the adr-supersession-mutual rule is satisfied.
// Before the fix the verb wrote only `superseded_by`, the warning
// fired permanently, and no verb could clear it.
func TestPromote_SupersededByWritesReciprocalSupersedes(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindADR, "Old decision", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindADR, "New decision", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "ADR-0001", "accepted", testActor, "", false, verb.PromoteOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "ADR-0002", "accepted", testActor, "", false, verb.PromoteOptions{}))

	r.must(verb.Promote(r.ctx, r.tree(), "ADR-0001", "superseded", testActor, "superseded by the new call", false,
		verb.PromoteOptions{SupersededBy: "ADR-0002"}))

	tr := r.tree()
	old := tr.ByID("ADR-0001")
	if old == nil || old.SupersededBy != "ADR-0002" {
		t.Fatalf("ADR-0001.superseded_by = %+v, want ADR-0002", old)
	}
	neu := tr.ByID("ADR-0002")
	if neu == nil || len(neu.Supersedes) != 1 || neu.Supersedes[0] != "ADR-0001" {
		t.Fatalf("ADR-0002.supersedes = %+v, want [ADR-0001]", neu)
	}

	// The headline closure: adr-supersession-mutual is silent because
	// both sides of the link are now recorded.
	for _, f := range check.Run(tr, nil) {
		if f.Code == check.CodeADRSupersessionMutual {
			t.Errorf("adr-supersession-mutual should be silent after a two-sided supersession; got %+v", f)
		}
	}
}

// TestPromote_SupersededByReciprocalIdempotent exercises the
// already-links-back branch: when the superseding ADR already lists
// the superseded ADR in its supersedes set, the verb writes only the
// superseded ADR (one file op) — no duplicate entry, no redundant
// second write.
func TestPromote_SupersededByReciprocalIdempotent(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindADR, "Old decision", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindADR, "New decision", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "ADR-0001", "accepted", testActor, "", false, verb.PromoteOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "ADR-0002", "accepted", testActor, "", false, verb.PromoteOptions{}))

	// Pre-seed the reciprocal supersedes link on ADR-0002 so the
	// supersession below has nothing to add. Hand-edit + commit keeps
	// the tree clean for the verb's projection.
	adr2 := r.tree().ByID("ADR-0002")
	adr2Path := filepath.Join(r.root, adr2.Path)
	raw, err := os.ReadFile(adr2Path)
	if err != nil {
		t.Fatal(err)
	}
	seeded := strings.Replace(string(raw), "status: accepted\n", "status: accepted\nsupersedes:\n  - ADR-0001\n", 1)
	if err := os.WriteFile(adr2Path, []byte(seeded), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Add(r.ctx, r.root, adr2.Path); err != nil {
		t.Fatal(err)
	}
	if err := gitops.Commit(r.ctx, r.root, "seed supersedes on ADR-0002", "", nil); err != nil {
		t.Fatal(err)
	}

	res := r.must(verb.Promote(r.ctx, r.tree(), "ADR-0001", "superseded", testActor, "", false,
		verb.PromoteOptions{SupersededBy: "ADR-0002"}))

	if len(res.Plan.Ops) != 1 {
		t.Errorf("expected 1 file op (reciprocal already present), got %d: %+v", len(res.Plan.Ops), res.Plan.Ops)
	}
	neu := r.tree().ByID("ADR-0002")
	if len(neu.Supersedes) != 1 || neu.Supersedes[0] != "ADR-0001" {
		t.Errorf("ADR-0002.supersedes = %v, want [ADR-0001] (no duplicate)", neu.Supersedes)
	}
	for _, f := range check.Run(r.tree(), nil) {
		if f.Code == check.CodeADRSupersessionMutual {
			t.Errorf("mutual link already present; warning should be silent; got %+v", f)
		}
	}
}

// TestPromote_SupersededByAbsentTargetNoReciprocal exercises the
// nil-target guard: when --superseded-by names an entity that does not
// exist, the reciprocal helper skips (no nil-deref) and the verb's
// projection blocks on the dangling superseded_by via refs-resolve. No
// plan is produced.
func TestPromote_SupersededByAbsentTargetNoReciprocal(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindADR, "Old decision", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "ADR-0001", "accepted", testActor, "", false, verb.PromoteOptions{}))

	res, err := verb.Promote(r.ctx, r.tree(), "ADR-0001", "superseded", testActor, "", false,
		verb.PromoteOptions{SupersededBy: "ADR-0099"})
	if err != nil {
		t.Fatalf("unexpected Go error (expected findings, not error): %v", err)
	}
	if res == nil || res.Plan != nil {
		t.Fatalf("expected findings with no plan; got %+v", res)
	}
	if !check.HasErrors(res.Findings) {
		t.Errorf("expected a refs-resolve error for the dangling superseded_by; got %+v", res.Findings)
	}
}
