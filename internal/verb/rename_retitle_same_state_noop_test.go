package verb_test

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// TestRename_SameSlug_ReturnsNoOp pins the rename half of M-0281/AC-5: renaming
// an entity to the slug it already carries converges to a NoOp instead of the
// "matches the current slug" error.
func TestRename_SameSlug_ReturnsNoOp(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))

	res, err := verb.Rename(r.ctx, r.tree(), "E-0001", "foundations", testActor, 0)
	if err != nil {
		t.Fatalf("rename to the current slug returned a Go error, want a NoOp: %v", err)
	}
	if !res.NoOp {
		t.Errorf("res.NoOp = false, want true (renaming to the current slug is a no-op)")
	}
	if res.Plan != nil {
		t.Errorf("res.Plan = %+v, want nil (a NoOp produces no commit)", res.Plan)
	}
}

// TestRetitle_SameTitle_ReturnsNoOp pins the retitle half of M-0281/AC-5:
// retitling an entity to the title it already carries converges to a NoOp
// instead of the "title already" error.
func TestRetitle_SameTitle_ReturnsNoOp(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))

	res, err := verb.Retitle(r.ctx, r.tree(), "E-0001", "Foundations", testActor, "", 0)
	if err != nil {
		t.Fatalf("retitle to the current title returned a Go error, want a NoOp: %v", err)
	}
	if !res.NoOp {
		t.Errorf("res.NoOp = false, want true (retitling to the current title is a no-op)")
	}
	if res.Plan != nil {
		t.Errorf("res.Plan = %+v, want nil (a NoOp produces no commit)", res.Plan)
	}
}

// TestRenameRetitleAC_SameTitle_ReturnsNoOp extends AC-5's convergence to the
// composite-id (acceptance-criterion) variants of both verbs: an AC carries a
// title but no slug, so `rename M-NNNN/AC-N` and `retitle M-NNNN/AC-N` both
// operate on that title, and both must converge on a same-title input rather
// than keeping the "title already" error the entity-level paths just shed.
func TestRenameRetitleAC_SameTitle_ReturnsNoOp(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Cache", testActor,
		verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	const acTitle = "cache warms on boot"
	r.must(verb.AddAC(r.ctx, r.tree(), "M-0001", acTitle, testActor))

	t.Run("rename", func(t *testing.T) {
		t.Parallel()
		res, err := verb.Rename(r.ctx, r.tree(), "M-0001/AC-1", acTitle, testActor, 0)
		if err != nil {
			t.Fatalf("rename AC to its current title returned a Go error, want a NoOp: %v", err)
		}
		if !res.NoOp {
			t.Errorf("res.NoOp = false, want true")
		}
		if res.Plan != nil {
			t.Errorf("res.Plan = %+v, want nil", res.Plan)
		}
	})

	t.Run("retitle", func(t *testing.T) {
		t.Parallel()
		res, err := verb.Retitle(r.ctx, r.tree(), "M-0001/AC-1", acTitle, testActor, "", 0)
		if err != nil {
			t.Fatalf("retitle AC to its current title returned a Go error, want a NoOp: %v", err)
		}
		if !res.NoOp {
			t.Errorf("res.NoOp = false, want true")
		}
		if res.Plan != nil {
			t.Errorf("res.Plan = %+v, want nil", res.Plan)
		}
	})
}
