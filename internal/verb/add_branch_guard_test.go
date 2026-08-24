package verb_test

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// add_branch_guard_test.go pins D-0074: `aiwf add epic` refuses off
// trunk. An epic created on a ritual branch cannot be activated — the
// promote guard expects trunk, and moving there puts the entity out of
// view, because the file exists only on the branch that created it
// (G-0616). Refusing at creation costs nothing; refusing at activation
// strands the entity.

// TestAdd_Epic_RefusesOnRitualBranch is the guard itself.
func TestAdd_Epic_RefusesOnRitualBranch(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	gitCheckoutNewBranch(t, r.root, "epic/E-0001-existing")

	_, err := verb.Add(r.ctx, r.tree(), entity.KindEpic, "Front-end auth widgets", testActor, verb.AddOptions{})
	if err == nil {
		t.Fatal("expected refusal for epic creation on a ritual branch")
	}
	if !strings.Contains(err.Error(), "epic/E-0001-existing") {
		t.Errorf("refusal should name the branch it refused on, got: %v", err)
	}
	if got := r.tree().ByID("E-0001"); got != nil {
		t.Errorf("refused add must create nothing; found %+v", got)
	}
}

// TestAdd_Epic_SucceedsOnNonRitualBranch pins the predicate as the
// branch's rung rather than inequality with trunk. A repo whose trunk
// is "master" while the configured trunk name is the default "main"
// has every branch unequal to trunk; a guard written that way refuses
// every epic creation in that repo. Such a branch carries no ritual
// rung, so it does not reach the refusal.
func TestAdd_Epic_SucceedsOnNonRitualBranch(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	gitCheckoutNewBranch(t, r.root, "master")

	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	if got := r.tree().ByID("E-0001"); got == nil {
		t.Fatal("epic creation on a non-ritual, non-trunk branch must succeed")
	}
}

// TestAdd_Epic_SucceedsOnTrunk is the baseline the guard must not
// break: newRunner's repo starts on "main", the unconfigured default
// trunk name.
func TestAdd_Epic_SucceedsOnTrunk(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	if got := r.tree().ByID("E-0001"); got == nil {
		t.Fatal("epic creation on trunk must succeed")
	}
}

// TestAdd_Epic_ForceOverridesOffTrunk pins the bypass D-0074 names:
// the existing sovereign --force --reason on add, not a new flag.
func TestAdd_Epic_ForceOverridesOnRitualBranch(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	gitCheckoutNewBranch(t, r.root, "epic/E-0001-existing")

	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Deliberate off-trunk epic", testActor,
		verb.AddOptions{Force: true, Reason: "drafting from the branch on purpose"}))
	if got := r.tree().ByID("E-0001"); got == nil {
		t.Fatal("--force must let the off-trunk creation through")
	}
}

// TestAdd_NonEpicKinds_UnaffectedOffTrunk pins the scope D-0074 sets.
// ADR-0010 sanctions gaps discovered during ritual work landing on the
// branch, so a blanket guard would break a documented flow.
func TestAdd_NonEpicKinds_UnaffectedOffTrunk(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	gitCheckoutNewBranch(t, r.root, "epic/E-0001-existing")

	r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Defect found mid-work", testActor,
		verb.AddOptions{BodyOverride: []byte("## What's missing\n\nSomething.\n\n## Why it matters\n\nIt breaks.\n")}))
	if got := r.tree().ByID("G-0001"); got == nil {
		t.Fatal("gap creation off trunk must remain legal (ADR-0010 Tier 2)")
	}
}
