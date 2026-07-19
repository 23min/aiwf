package verb_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// promote_branch_guard_test.go pins G-0269's synchronous pre-commit
// branch guard: an epic proposed -> active or milestone -> in_progress
// promote must land on ADR-0010's expected parent branch (trunk for
// an epic, the parent epic's ritual branch for a milestone) — refused
// outright, before any commit, when the current branch doesn't match.

// gitCheckoutNewBranch cuts and switches to a fresh branch off HEAD.
func gitCheckoutNewBranch(t *testing.T, root, branch string) {
	t.Helper()
	cmd := exec.Command("git", "checkout", "-q", "-b", branch)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git checkout -b %s: %v\n%s", branch, err, out)
	}
}

// gitCheckoutDetached detaches HEAD at its current commit.
func gitCheckoutDetached(t *testing.T, root string) {
	t.Helper()
	cmd := exec.Command("git", "checkout", "-q", "--detach", "HEAD")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git checkout --detach: %v\n%s", err, out)
	}
}

// TestPromote_EpicActive_SucceedsOnTrunk is the baseline: newRunner's
// repo starts on "main" (gitops.Init), which is also the unconfigured
// default trunk name (Config.TrunkBranchShortName) — an epic
// proposed -> active promote right there must not be refused.
func TestPromote_EpicActive_SucceedsOnTrunk(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))

	if e := r.tree().ByID("E-0001"); e == nil || e.Status != "active" {
		t.Errorf("E-0001 = %+v, want status active", e)
	}
}

// TestPromote_EpicActive_RefusesOnRitualBranch: cutting the epic's own
// ritual branch before activating it is the wrong order per ADR-0010
// (activate first, cut the branch second) — the guard refuses.
func TestPromote_EpicActive_RefusesOnRitualBranch(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	gitCheckoutNewBranch(t, r.root, "epic/E-0001-foundations")

	_, err := verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{})
	if err == nil {
		t.Fatal("expected refusal for epic activation off trunk")
	}
	if !strings.Contains(err.Error(), "G-0269") {
		t.Errorf("expected the refusal to name G-0269, got: %v", err)
	}
	if !strings.Contains(err.Error(), `expected on "main"`) {
		t.Errorf("expected the refusal to name the expected trunk branch, got: %v", err)
	}
	if e := r.tree().ByID("E-0001"); e == nil || e.Status != "proposed" {
		t.Errorf("refused promote must not mutate status; E-0001 = %+v", e)
	}
}

// TestPromote_EpicActive_RefusesOnDetachedHEAD: the label rendered for
// a detached HEAD reads as an explicit state, not an empty string.
func TestPromote_EpicActive_RefusesOnDetachedHEAD(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	gitCheckoutDetached(t, r.root)

	_, err := verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{})
	if err == nil {
		t.Fatal("expected refusal for epic activation on detached HEAD")
	}
	if !strings.Contains(err.Error(), "(detached HEAD)") {
		t.Errorf("expected the refusal to label detached HEAD explicitly, got: %v", err)
	}
}

// TestPromote_EpicActive_ForceOverridesGuard: --force lets the
// sovereign override land the commit even off trunk, and stamps the
// usual aiwf-force trailer (pinned elsewhere) — this test only
// confirms the guard itself steps aside.
func TestPromote_EpicActive_ForceOverridesGuard(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	gitCheckoutNewBranch(t, r.root, "epic/E-0001-foundations")

	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "sovereign override", true, verb.PromoteOptions{}))
	if e := r.tree().ByID("E-0001"); e == nil || e.Status != "active" {
		t.Errorf("E-0001 = %+v, want status active", e)
	}
}

// TestPromote_MilestoneInProgress_SucceedsOnParentEpicBranch is the
// baseline for the milestone leg: the parent epic's own ritual branch
// is the expected landing spot.
func TestPromote_MilestoneInProgress_SucceedsOnParentEpicBranch(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Bootstrap", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	// M-0268/AC-1+AC-2: draft -> in_progress now refuses a zero-AC
	// milestone, or one with an empty AC body; seed a real one so this
	// test exercises the branch guard, not the AC-completeness guards.
	r.must(verb.AddACBatch(r.ctx, r.tree(), "M-0001", []string{"Boots up"}, [][]byte{[]byte("Real prose.")}, testActor, nil))
	gitCheckoutNewBranch(t, r.root, "epic/E-0001-foundations")

	r.must(verb.Promote(r.ctx, r.tree(), "M-0001", "in_progress", testActor, "", false, verb.PromoteOptions{}))
	if e := r.tree().ByID("M-0001"); e == nil || e.Status != "in_progress" {
		t.Errorf("M-0001 = %+v, want status in_progress", e)
	}
}

// TestPromote_MilestoneInProgress_RefusesOnTrunk: starting a milestone
// while still on trunk, skipping the parent epic's ritual branch
// entirely, is exactly the wrong-order incident G-0270's own AC-8
// cell 5 detects post-hoc — this guard refuses it synchronously.
func TestPromote_MilestoneInProgress_RefusesOnTrunk(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Bootstrap", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))

	_, err := verb.Promote(r.ctx, r.tree(), "M-0001", "in_progress", testActor, "", false, verb.PromoteOptions{})
	if err == nil {
		t.Fatal("expected refusal for milestone activation on trunk")
	}
	if !strings.Contains(err.Error(), `expected on "epic/E-0001-foundations"`) {
		t.Errorf("expected the refusal to name the parent epic's ritual branch, got: %v", err)
	}
	if e := r.tree().ByID("M-0001"); e == nil || e.Status != "draft" {
		t.Errorf("refused promote must not mutate status; M-0001 = %+v", e)
	}
}

// TestPromote_MilestoneInProgress_RefusesOnSiblingBranch: landing on
// some other ritual-shaped branch (not the parent epic's own) is
// still wrong — the guard compares against the specific expected
// branch, not merely "some ritual branch exists".
func TestPromote_MilestoneInProgress_RefusesOnSiblingBranch(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Bootstrap", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))
	gitCheckoutNewBranch(t, r.root, "milestone/M-9999-other")

	_, err := verb.Promote(r.ctx, r.tree(), "M-0001", "in_progress", testActor, "", false, verb.PromoteOptions{})
	if err == nil {
		t.Fatal("expected refusal for milestone activation on an unrelated branch")
	}
	if !strings.Contains(err.Error(), `refusing to land on "milestone/M-9999-other"`) {
		t.Errorf("expected the refusal to name the actual current branch, got: %v", err)
	}
}

// TestPromote_MilestoneInProgress_ForceOverridesGuard mirrors the
// epic-side force test for the milestone leg.
func TestPromote_MilestoneInProgress_ForceOverridesGuard(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))
	r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Bootstrap", testActor, verb.AddOptions{EpicID: "E-0001", TDD: "none"}))

	r.must(verb.Promote(r.ctx, r.tree(), "M-0001", "in_progress", testActor, "sovereign override", true, verb.PromoteOptions{}))
	if e := r.tree().ByID("M-0001"); e == nil || e.Status != "in_progress" {
		t.Errorf("M-0001 = %+v, want status in_progress", e)
	}
}

// TestPromote_NonActivatingTransition_IgnoresBranchGuard: the guard is
// scoped to exactly the two activating transitions — any other
// promote (here, an already-active epic moving to done) must succeed
// regardless of the current branch.
func TestPromote_NonActivatingTransition_IgnoresBranchGuard(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{}))
	gitCheckoutNewBranch(t, r.root, "epic/E-0001-foundations")

	r.must(verb.Promote(r.ctx, r.tree(), "E-0001", "done", testActor, "", false, verb.PromoteOptions{}))
	if e := r.tree().ByID("E-0001"); e == nil || e.Status != "done" {
		t.Errorf("E-0001 = %+v, want status done", e)
	}
}
