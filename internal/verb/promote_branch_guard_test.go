package verb_test

import (
	"os/exec"
	"path/filepath"
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
	r.must(verb.AddACBatch(r.ctx, r.tree(), "M-0001", []string{"Boots up"}, [][]byte{[]byte("Real prose.")}, testActor))
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

// gitWorktreeAddExisting checks an existing branch out into a linked
// worktree at path, so the branch is held somewhere other than root.
func gitWorktreeAddExisting(t *testing.T, root, path, branch string) {
	t.Helper()
	cmd := exec.Command("git", "worktree", "add", "-q", path, branch)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add %s %s: %v\n%s", path, branch, err, out)
	}
}

// TestPromote_RefusalNamesTheHoldingWorktree pins G-0621: when the
// expected branch is held by another worktree, the refusal must point
// at that worktree's path rather than telling the operator to check the
// branch out. A branch is checked out in one worktree at a time, so
// `git checkout <expected>` fails with "already used by worktree at
// ..." in exactly the situation that produces this refusal — leaving
// the operator a remedy that refuses too.
//
// The expected path is derived by asking git where the branch lives,
// not transcribed, so the assertion tracks the fixture rather than a
// copy of it.
func TestPromote_RefusalNamesTheHoldingWorktree(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Foundations", testActor, verb.AddOptions{}))
	// Park trunk in its own worktree, then leave the main checkout on a
	// different branch so the guard fires with "main" held elsewhere.
	held := filepath.Join(t.TempDir(), "trunk-wt")
	gitCheckoutNewBranch(t, r.root, "epic/E-0001-foundations")
	gitWorktreeAddExisting(t, r.root, held, "main")

	_, err := verb.Promote(r.ctx, r.tree(), "E-0001", "active", testActor, "", false, verb.PromoteOptions{})
	if err == nil {
		t.Fatal("expected refusal for epic activation off trunk")
	}
	if !strings.Contains(err.Error(), held) {
		t.Errorf("refusal must name the worktree holding the expected branch (%s), got: %v", held, err)
	}
	if strings.Contains(err.Error(), "git checkout main") {
		t.Errorf("refusal must not suggest checking out a branch another worktree holds, got: %v", err)
	}
}

// TestPromote_RefusalReportsUnreachableEntity pins the second half of
// D-0074: when the entity is not present on the expected branch, the
// refusal says so. Reaching that branch does not help — the entity goes
// out of view there and the next message names a different problem
// ("entity not found"), which is how G-0616 was measured.
//
// The guard states the fact and prescribes no remedy: which recovery is
// correct is unsettled, and recommending one would assert an answer
// D-0074 does not have.
func TestPromote_RefusalReportsUnreachableEntity(t *testing.T) {
	t.Parallel()
	r := newRunner(t)
	// Give trunk history first: without a commit on it, refs/heads/main
	// does not exist and absence cannot be distinguished from an
	// unreadable ref.
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Auth rewrite", testActor, verb.AddOptions{}))
	// Then create the second epic on a ritual branch, bypassing the
	// creation guard the way an operator with --force would, so it
	// exists only there — the state G-0616 measured.
	gitCheckoutNewBranch(t, r.root, "epic/E-0001-auth-rewrite")
	r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Front-end auth widgets", testActor,
		verb.AddOptions{Force: true, Reason: "reproducing the stranded state"}))

	_, err := verb.Promote(r.ctx, r.tree(), "E-0002", "active", testActor, "", false, verb.PromoteOptions{})
	if err == nil {
		t.Fatal("expected refusal for epic activation off trunk")
	}
	if !strings.Contains(err.Error(), "not present on") {
		t.Errorf("refusal must report that the entity is absent from the expected branch, got: %v", err)
	}
}
