package verb_test

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// field_verbs_same_state_noop_test.go covers M-0281/AC-7: the four
// field-mutation verbs left outside the same-state NoOp convention after
// AC-1..AC-5. Each drives the verb with input that already equals current
// state and asserts convergence to Result.NoOp rather than a refusal.

// TestSetArea_SameState_ReturnsNoOp covers both of SetArea's same-state
// guards: re-tagging with the member already set, and clearing an already
// untagged entity.
func TestSetArea_SameState_ReturnsNoOp(t *testing.T) {
	t.Parallel()
	members := []string{"platform"}

	t.Run("already tagged", func(t *testing.T) {
		t.Parallel()
		r := newRunner(t)
		r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{Area: "platform"}))

		res, err := verb.SetArea(r.ctx, r.tree(), members, "E-0001", "platform", false, testActor)
		if err != nil {
			t.Fatalf("set-area to the current member returned a Go error, want a NoOp: %v", err)
		}
		if !res.NoOp {
			t.Errorf("res.NoOp = false, want true")
		}
		if res.Plan != nil {
			t.Errorf("res.Plan = %+v, want nil", res.Plan)
		}
	})

	t.Run("already untagged, clear", func(t *testing.T) {
		t.Parallel()
		r := newRunner(t)
		r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))

		res, err := verb.SetArea(r.ctx, r.tree(), members, "E-0001", "", true, testActor)
		if err != nil {
			t.Fatalf("clearing an untagged entity returned a Go error, want a NoOp: %v", err)
		}
		if !res.NoOp {
			t.Errorf("res.NoOp = false, want true")
		}
		if res.Plan != nil {
			t.Errorf("res.Plan = %+v, want nil", res.Plan)
		}
	})
}

// TestSetPriority_SameState_ReturnsNoOp covers both of SetPriority's
// same-state guards: re-setting the level already recorded, and clearing an
// already-unset priority.
func TestSetPriority_SameState_ReturnsNoOp(t *testing.T) {
	t.Parallel()

	t.Run("already at level", func(t *testing.T) {
		t.Parallel()
		r := newRunner(t)
		r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Stray gap", testActor,
			verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap), Priority: "high"}))

		res, err := verb.SetPriority(r.ctx, r.tree(), "G-0001", "high", false, testActor)
		if err != nil {
			t.Fatalf("set-priority to the current level returned a Go error, want a NoOp: %v", err)
		}
		if !res.NoOp {
			t.Errorf("res.NoOp = false, want true")
		}
		if res.Plan != nil {
			t.Errorf("res.Plan = %+v, want nil", res.Plan)
		}
	})

	t.Run("already unset, clear", func(t *testing.T) {
		t.Parallel()
		r := newRunner(t)
		r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Stray gap", testActor,
			verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap)}))

		res, err := verb.SetPriority(r.ctx, r.tree(), "G-0001", "", true, testActor)
		if err != nil {
			t.Fatalf("clearing an unset priority returned a Go error, want a NoOp: %v", err)
		}
		if !res.NoOp {
			t.Errorf("res.NoOp = false, want true")
		}
		if res.Plan != nil {
			t.Errorf("res.Plan = %+v, want nil", res.Plan)
		}
	})
}

// RenameArea's same-name NoOp lives in renamearea_test.go (package verb), where
// its areaTree/mustReadAreaDoc fixtures already are.

// TestMilestoneDependsOn_SameList_ReturnsNoOp: re-declaring the depends_on list
// a milestone already carries wrote byte-identical content and landed a commit
// with an empty diff. Apply's zero-Ops guard does not catch it — the plan has
// one write Op, and git commit-tree has no same-tree refusal — so the guard has
// to live in the verb. Order-sensitivity is deliberate: `--on` is
// replace-not-append, so a reordered list is a real (if cosmetic) change and
// must still commit.
func TestMilestoneDependsOn_SameList_ReturnsNoOp(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor,
		verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Second", testActor,
		verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.MilestoneDependsOn(r.ctx, r.tree(), "M-0001", []string{"M-0002"}, false, testActor, ""))
	before := countCommits(t, r.root)

	res, err := verb.MilestoneDependsOn(r.ctx, r.tree(), "M-0001", []string{"M-0002"}, false, testActor, "")
	if err != nil {
		t.Fatalf("re-declaring the same depends_on list returned a Go error, want a NoOp: %v", err)
	}
	if !res.NoOp {
		t.Errorf("res.NoOp = false, want true (the list is already exactly M-0002)")
	}
	if res.Plan != nil {
		t.Fatalf("res.Plan = %+v, want nil — an identical list must not land an empty-diff commit", res.Plan)
	}
	if got := countCommits(t, r.root); got != before {
		t.Errorf("commit count = %s, want %s (the NoOp must append no commit)", got, before)
	}
}

// TestMilestoneDependsOn_ClearAlreadyEmpty_ReturnsNoOp covers the --clear arm:
// clearing a milestone that carries no depends_on list is likewise a no-op.
func TestMilestoneDependsOn_ClearAlreadyEmpty_ReturnsNoOp(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Solo", testActor,
		verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	before := countCommits(t, r.root)

	res, err := verb.MilestoneDependsOn(r.ctx, r.tree(), "M-0001", nil, true, testActor, "")
	if err != nil {
		t.Fatalf("clearing an already-empty depends_on returned a Go error, want a NoOp: %v", err)
	}
	if !res.NoOp {
		t.Errorf("res.NoOp = false, want true")
	}
	if got := countCommits(t, r.root); got != before {
		t.Errorf("commit count = %s, want %s", got, before)
	}
}

// TestMilestoneDependsOn_ReorderedList_StillCommits is the guard against an
// over-broad NoOp: `--on` is replace-not-append, so a list with the same members
// in a different order is a real change and must still produce a plan.
func TestMilestoneDependsOn_ReorderedList_StillCommits(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "First", testActor,
		verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Second", testActor,
		verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Third", testActor,
		verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	r.must(verb.MilestoneDependsOn(r.ctx, r.tree(), "M-0001", []string{"M-0002", "M-0003"}, false, testActor, ""))

	res, err := verb.MilestoneDependsOn(r.ctx, r.tree(), "M-0001", []string{"M-0003", "M-0002"}, false, testActor, "")
	if err != nil {
		t.Fatalf("reordering the depends_on list errored: %v", err)
	}
	if res.NoOp {
		t.Errorf("res.NoOp = true, want false (a reordered replace-not-append list is a real change)")
	}
	if res.Plan == nil {
		t.Error("res.Plan = nil, want a plan")
	}
}

// TestMilestoneTDD_SamePolicy_ReturnsNoOp is the correctness half of AC-7:
// setting a milestone's TDD policy to the value it already carries produced a
// real commit with an empty diff — a no-op commit polluting history on every
// re-run, the same "re-running creates duplicates" shape acknowledge-illegal
// carried. It must converge to a NoOp and write nothing.
func TestMilestoneTDD_SamePolicy_ReturnsNoOp(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Platform", testActor, verb.AddOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Cache", testActor,
		verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	before := countCommits(t, r.root)

	res, err := verb.MilestoneTDD(r.ctx, r.tree(), "M-0001", "none", testActor, "")
	if err != nil {
		t.Fatalf("milestone tdd to the current policy returned a Go error, want a NoOp: %v", err)
	}
	if !res.NoOp {
		t.Errorf("res.NoOp = false, want true (the policy is already none)")
	}
	if res.Plan != nil {
		t.Fatalf("res.Plan = %+v, want nil — a same-policy call must not land an empty-diff commit", res.Plan)
	}
	if got := countCommits(t, r.root); got != before {
		t.Errorf("commit count = %s, want %s (the NoOp must append no commit)", got, before)
	}
}
