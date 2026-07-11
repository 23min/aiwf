package check

import (
	"strings"
	"testing"

	codespkg "github.com/23min/aiwf/internal/codes"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/scope"
)

// promote_on_wrong_branch_test.go § G-0308: a commit's aiwf-entity:
// trailer must resolve forward through a reallocation before the
// expectedBranches lookup, else a pre-reallocation promote
// mis-attributes to whatever entity currently claims the freed id.
// buildProvenanceTreeWithRenamedAndCollision (provenance_test.go)
// reproduces the exact G-0308 shape: the id M-0099 was reallocated
// to M-0001 (parent E-0001), and a later, unrelated parallel
// allocation reclaimed M-0099 under a different epic, E-0009.

// promote_on_wrong_branch_test.go — M-0161/AC-8 (G-0209
// partial-close) + G-0270 unit-level coverage of
// RunPromoteOnWrongBranch per CLAUDE.md §"Test the seam, not just
// the layer" + the AC-5/AC-7 reviewer pattern: the E2E exercises the
// production wire-up; the unit tests below pin the rule's
// input/output contract against in-memory fixtures.
//
// G-0270 replaced the rule's BranchOracle dependency (which asked
// "what branches is this commit reachable from," a question that
// required enumerating and name-filtering local branches — and
// silently missed an activation commit that landed on an
// arbitrarily-named, non-ritual-shaped branch) with a direct
// ancestor check against the expected branch's resolved tip SHA, via
// the shared in-memory CommitDAG. The fixtures below build a small
// hand-rolled DAG (testDAG) instead of a fakeOracle.
//
// Full ancestry alone can't tell "correctly on the epic branch,
// later merged into trunk and cleaned up" (silent) apart from
// "skipped the epic branch and landed directly on trunk" (fires) —
// both are full ancestors of trunk's CURRENT tip once a merge has
// happened. RunPromoteOnWrongBranch's unified two-condition check
// (the SAME logic for both epics and milestones — see its doc
// comment for the full rationale) additionally consults trunk's own
// first-parent chain to distinguish them.
//
// Branch coverage:
//   - epic activating promote on correct trunk (commit == trunk tip,
//     i.e. on trunk's own first-parent chain) → silent
//   - epic activating promote on wrong branch (not on trunk's own
//     first-parent chain) → fires
//   - milestone activating promote on parent epic (commit == epic
//     tip, epic branch still live) → silent
//   - milestone activating promote on wrong branch (landed directly
//     on trunk itself) → fires
//   - milestone's epic branch was properly used and has since been
//     merged into trunk and deleted (no longer in branchTips) →
//     silent (the core B1/G-0270 regression this file pins)
//   - the same "epic branch gone" shape, but the milestone commit
//     actually landed directly on trunk's own lineage → still fires
//     (proves the previous fix doesn't overcorrect)
//   - non-activating promote (active → done) → silent
//   - non-promote verb (edit-body, etc.) → silent
//   - empty expectedBranches map / missing entity → silent
//   - per-commit aiwf-force override → silent
//   - per-SHA ack via ackedSHAs → silent
//   - trunk itself unresolvable (branchTips has no entry at all) →
//     fires for an epic activation (not fail-shut)
//   - nil dag → silent
//   - commit unreachable from ANY known branch's ancestry (not just
//     unenumerable-by-name) → fires (G-0270's core fix)
//   - pre-reallocation promote resolves via prior_ids → silent (G-0308)
//   - pre-reallocation promote, genuine wrong branch → still fires (G-0308)
//   - in-window aiwf-prior-entity trailer resolves forward → silent (G-0308)

func makePromoteCommit(sha, entityID, targetStatus string, force bool) scope.Commit {
	trailers := []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: "promote"},
		{Key: gitops.TrailerEntity, Value: entityID},
		{Key: gitops.TrailerActor, Value: "human/peter"},
		{Key: gitops.TrailerTo, Value: targetStatus},
	}
	if force {
		trailers = append(trailers, gitops.Trailer{Key: gitops.TrailerForce, Value: "test"})
	}
	return scope.Commit{SHA: sha, Trailers: trailers}
}

// testDAG builds a *CommitDAG directly from a parent map, bypassing
// BuildCommitDAG's `git rev-list` subprocess — this test package
// needs a fixed, hand-authored ancestry graph, not a real repo.
func testDAG(parents map[string][]string) *CommitDAG {
	return &CommitDAG{parents: parents}
}

// promoteOnWrongBranchFixtureDAG is the shared ancestry graph for
// this file's simpler tests: "main-tip" and "epic-tip" are sibling
// branches both rooted at "root" (so neither is an ancestor of the
// other); "stray-tip" is rooted at its own disconnected
// "stray-root", modeling a commit on a branch with no relation to
// any named branch at all — the shape the old BranchOracle could
// never even classify, since it only indexed the first-parent
// chains of branches it could enumerate by name.
func promoteOnWrongBranchFixtureDAG() *CommitDAG {
	return testDAG(map[string][]string{
		"main-tip":   {"root"},
		"epic-tip":   {"root"},
		"root":       nil,
		"stray-tip":  {"stray-root"},
		"stray-root": nil,
	})
}

func promoteOnWrongBranchFixtureTips() map[string]string {
	return map[string]string{
		"main":               "main-tip",
		"epic/E-0001-engine": "epic-tip",
	}
}

// promoteOnWrongBranchMergedEpicFixtureDAG models an epic ritual
// branch that was used correctly and has since been merged into
// trunk via `--no-ff` and deleted — aiwfx-wrap-epic's normal
// end-of-life for a completed epic. "trunk-merged-tip" is trunk's
// tip AFTER the merge, with two parents: "old-trunk-tip" (trunk's
// own prior first-parent lineage) and "epic-tip" (the merged-in
// epic branch, and also the milestone's own correct activation
// commit) — exactly the shape a real `git merge --no-ff` commit
// produces. No branchTips entry for "epic/E-0001-engine" models the
// branch's deletion after merge.
func promoteOnWrongBranchMergedEpicFixtureDAG() *CommitDAG {
	return testDAG(map[string][]string{
		"trunk-merged-tip": {"old-trunk-tip", "epic-tip"},
		"old-trunk-tip":    {"root"},
		"epic-tip":         {"root"},
		"root":             nil,
	})
}

func promoteOnWrongBranchMergedEpicFixtureTips() map[string]string {
	return map[string]string{
		"main": "trunk-merged-tip",
		// Deliberately no "epic/E-0001-engine" entry — the branch is gone.
	}
}

// TestPromoteOnWrongBranch_AC8_EpicCorrectBranch_Silent pins
// the silent-good path: the activation commit IS the trunk
// branch's current tip — on trunk's own first-parent chain — so
// no finding.
func TestPromoteOnWrongBranch_AC8_EpicCorrectBranch_Silent(t *testing.T) {
	t.Parallel()
	commits := []scope.Commit{
		makePromoteCommit("main-tip", "E-0001", "active", false),
	}
	got := RunPromoteOnWrongBranch(commits, map[string]string{"E-0001": "main"}, promoteOnWrongBranchFixtureDAG(), promoteOnWrongBranchFixtureTips(), "main", nil, nil)
	if len(got) != 0 {
		t.Errorf("expected silent on correct branch; got %d findings: %+v", len(got), got)
	}
}

// TestPromoteOnWrongBranch_AC8_EpicWrongBranch_Fires pins the
// load-bearing claim: an epic activating promote commit that is
// NOT on trunk's own first-parent chain fires the warning.
func TestPromoteOnWrongBranch_AC8_EpicWrongBranch_Fires(t *testing.T) {
	t.Parallel()
	commits := []scope.Commit{
		makePromoteCommit("epic-tip", "E-0001", "active", false),
	}
	got := RunPromoteOnWrongBranch(commits, map[string]string{"E-0001": "main"}, promoteOnWrongBranchFixtureDAG(), promoteOnWrongBranchFixtureTips(), "main", nil, nil)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding; got %d: %+v", len(got), got)
	}
	f := got[0]
	if f.Code != CodePromoteOnWrongBranch.ID {
		t.Errorf("Code = %q; want %q", f.Code, CodePromoteOnWrongBranch.ID)
	}
	if f.Severity != SeverityWarning {
		t.Errorf("Severity = %q; want %q (M-0125 ratchet — warning at first land)", f.Severity, SeverityWarning)
	}
	if !strings.Contains(f.Message, "main") {
		t.Errorf("Message %q does not name expected branch (main)", f.Message)
	}
	if !strings.Contains(f.Hint, "git branch --contains") {
		t.Errorf("Hint %q does not point the operator at finding the commit's actual branch", f.Hint)
	}
	if !strings.Contains(f.Hint, "aiwf acknowledge illegal") {
		t.Errorf("Hint %q does not name the acknowledge-illegal recovery path", f.Hint)
	}
	if !strings.Contains(f.Hint, "aiwf-force:") {
		t.Errorf("Hint %q does not name the per-commit aiwf-force override path", f.Hint)
	}
}

// TestPromoteOnWrongBranch_AC8_G0270_CommitUnreachableFromAnyKnownBranch_Fires
// is the core G-0270 regression pin: "stray-tip" sits on a commit
// graph entirely disconnected from any branch this fixture even
// names — the shape a BranchOracle could never classify (its
// FirstParentBranches would return empty, since it only walks
// branches it can enumerate by name, and the old rule treated an
// empty result as "unknown branch, fail-shut silent"). The
// ancestor-check design fires anyway, because it never asks "what
// branch is this on" — only "is it on trunk's own lineage."
func TestPromoteOnWrongBranch_AC8_G0270_CommitUnreachableFromAnyKnownBranch_Fires(t *testing.T) {
	t.Parallel()
	commits := []scope.Commit{
		makePromoteCommit("stray-tip", "E-0001", "active", false),
	}
	got := RunPromoteOnWrongBranch(commits, map[string]string{"E-0001": "main"}, promoteOnWrongBranchFixtureDAG(), promoteOnWrongBranchFixtureTips(), "main", nil, nil)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding for a commit unreachable from the expected branch; got %d: %+v", len(got), got)
	}
}

// TestPromoteOnWrongBranch_AC8_MilestoneCorrectParentEpic_Silent
// pins the milestone-side silent-good: the activation commit IS
// its parent epic's (still-live) ritual branch tip.
func TestPromoteOnWrongBranch_AC8_MilestoneCorrectParentEpic_Silent(t *testing.T) {
	t.Parallel()
	commits := []scope.Commit{
		makePromoteCommit("epic-tip", "M-0010", "in_progress", false),
	}
	got := RunPromoteOnWrongBranch(commits, map[string]string{"M-0010": "epic/E-0001-engine"}, promoteOnWrongBranchFixtureDAG(), promoteOnWrongBranchFixtureTips(), "main", nil, nil)
	if len(got) != 0 {
		t.Errorf("expected silent on parent-epic branch; got %d findings", len(got))
	}
}

// TestPromoteOnWrongBranch_AC8_MilestoneWrongBranch_Fires mirrors
// the epic case for milestones: the commit landed directly on
// trunk itself, skipping the parent epic branch entirely (the
// integration matrix's AC-8 cell 5 shape) — must still fire.
func TestPromoteOnWrongBranch_AC8_MilestoneWrongBranch_Fires(t *testing.T) {
	t.Parallel()
	commits := []scope.Commit{
		makePromoteCommit("main-tip", "M-0010", "in_progress", false),
	}
	got := RunPromoteOnWrongBranch(commits, map[string]string{"M-0010": "epic/E-0001-engine"}, promoteOnWrongBranchFixtureDAG(), promoteOnWrongBranchFixtureTips(), "main", nil, nil)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding; got %d: %+v", len(got), got)
	}
}

// TestPromoteOnWrongBranch_AC8_MilestoneEpicBranchMergedAndDeleted_Silent
// is the B1 regression pin (found by independent review, confirmed
// empirically against this repo's own history: 94 false positives
// on real, correctly-placed, long-since-wrapped milestone
// activations). Once an epic's ritual branch is properly merged
// into trunk via `--no-ff` and deleted — aiwfx-wrap-epic's ordinary
// end-of-life for a completed epic — a milestone activation commit
// that was correctly made on that branch becomes a full ancestor of
// trunk's tip too, arriving via the merge commit's second parent,
// never itself on trunk's own first-parent lineage. Plain full
// ancestry can't tell this apart from a milestone that skipped the
// epic branch and landed directly on trunk (see the sibling FIRES
// test below); the rule must consult trunk's own first-parent chain
// to distinguish them.
func TestPromoteOnWrongBranch_AC8_MilestoneEpicBranchMergedAndDeleted_Silent(t *testing.T) {
	t.Parallel()
	commits := []scope.Commit{
		makePromoteCommit("epic-tip", "M-0010", "in_progress", false),
	}
	got := RunPromoteOnWrongBranch(commits, map[string]string{"M-0010": "epic/E-0001-engine"}, promoteOnWrongBranchMergedEpicFixtureDAG(), promoteOnWrongBranchMergedEpicFixtureTips(), "main", nil, nil)
	if len(got) != 0 {
		t.Fatalf("findings = %v; a milestone activation correctly merged in via its epic branch (since deleted) must stay silent, not fire a false positive", findingCodes(got))
	}
}

// TestPromoteOnWrongBranch_AC8_MilestoneDirectlyOnTrunkDespiteUnrelatedMerge_Fires
// is the distinguishing negative for the B1 fix above: using the
// SAME "epic branch merged and deleted" fixture, a DIFFERENT
// milestone commit that instead landed directly on trunk's own
// first-parent lineage (skipping any epic branch) must still fire —
// proving the merged-branch allowance doesn't overcorrect into
// silencing every milestone activation reachable from trunk.
func TestPromoteOnWrongBranch_AC8_MilestoneDirectlyOnTrunkDespiteUnrelatedMerge_Fires(t *testing.T) {
	t.Parallel()
	commits := []scope.Commit{
		makePromoteCommit("old-trunk-tip", "M-0010", "in_progress", false),
	}
	got := RunPromoteOnWrongBranch(commits, map[string]string{"M-0010": "epic/E-0001-engine"}, promoteOnWrongBranchMergedEpicFixtureDAG(), promoteOnWrongBranchMergedEpicFixtureTips(), "main", nil, nil)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding; a milestone activation directly on trunk's own lineage must still fire even when an unrelated merge exists elsewhere in the DAG; got %d: %+v", len(got), got)
	}
}

// TestPromoteOnWrongBranch_AC8_NonActivatingPromote_Silent pins
// the rule's domain narrowness: epic active → done is OUT of
// the rule's domain regardless of branch.
func TestPromoteOnWrongBranch_AC8_NonActivatingPromote_Silent(t *testing.T) {
	t.Parallel()
	commits := []scope.Commit{
		makePromoteCommit("epic-tip", "E-0001", "done", false),
	}
	got := RunPromoteOnWrongBranch(commits, map[string]string{"E-0001": "main"}, promoteOnWrongBranchFixtureDAG(), promoteOnWrongBranchFixtureTips(), "main", nil, nil)
	if len(got) != 0 {
		t.Errorf("expected silent on non-activating promote (E-0001 → done); got %d findings: %+v", len(got), got)
	}
}

// TestPromoteOnWrongBranch_AC8_ForceTrailerSuppresses pins the
// per-commit override: an aiwf-force trailer on the promote
// commit suppresses the finding.
func TestPromoteOnWrongBranch_AC8_ForceTrailerSuppresses(t *testing.T) {
	t.Parallel()
	commits := []scope.Commit{
		makePromoteCommit("epic-tip", "E-0001", "active", true),
	}
	got := RunPromoteOnWrongBranch(commits, map[string]string{"E-0001": "main"}, promoteOnWrongBranchFixtureDAG(), promoteOnWrongBranchFixtureTips(), "main", nil, nil)
	if len(got) != 0 {
		t.Errorf("expected silent on forced commit; got %d findings", len(got))
	}
}

// TestPromoteOnWrongBranch_AC8_AcknowledgedSHASilences pins
// the post-hoc override via the shared ackedSHAs map, against a
// commit that would otherwise fire (it is on "epic-tip", not the
// expected trunk tip).
func TestPromoteOnWrongBranch_AC8_AcknowledgedSHASilences(t *testing.T) {
	t.Parallel()
	commits := []scope.Commit{
		makePromoteCommit("epic-tip", "E-0001", "active", false),
	}
	acked := map[string]bool{"epic-tip": true}
	got := RunPromoteOnWrongBranch(commits, map[string]string{"E-0001": "main"}, promoteOnWrongBranchFixtureDAG(), promoteOnWrongBranchFixtureTips(), "main", acked, nil)
	if len(got) != 0 {
		t.Errorf("expected silent on acknowledged SHA; got %d findings", len(got))
	}
}

// TestPromoteOnWrongBranch_AC8_NoExpectation_Silent pins fail-
// shut on missing expectation (parent lookup failed, gap kind,
// etc.).
func TestPromoteOnWrongBranch_AC8_NoExpectation_Silent(t *testing.T) {
	t.Parallel()
	commits := []scope.Commit{
		makePromoteCommit("epic-tip", "E-0001", "active", false),
	}
	// Empty map → no expectation → silent.
	got := RunPromoteOnWrongBranch(commits, map[string]string{}, promoteOnWrongBranchFixtureDAG(), promoteOnWrongBranchFixtureTips(), "main", nil, nil)
	if len(got) != 0 {
		t.Errorf("expected silent on no-expectation; got %d findings", len(got))
	}
}

// TestPromoteOnWrongBranch_AC8_TrunkTipUnresolvable_Fires pins the
// epic-side fail-toward-firing posture: when trunk itself has no
// resolvable tip at all (branchTips empty), an epic activation
// can't be confirmed on trunk's lineage, so the rule fires rather
// than fail-shut silent — a commit can't be confirmed correct
// against a trunk we can't even locate.
func TestPromoteOnWrongBranch_AC8_TrunkTipUnresolvable_Fires(t *testing.T) {
	t.Parallel()
	commits := []scope.Commit{
		makePromoteCommit("epic-tip", "E-0001", "active", false),
	}
	got := RunPromoteOnWrongBranch(commits, map[string]string{"E-0001": "main"}, promoteOnWrongBranchFixtureDAG(), map[string]string{}, "main", nil, nil)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding when trunk's own tip is unresolvable; got %d", len(got))
	}
}

// TestPromoteOnWrongBranch_AC8_NilDAG_Silent pins the top-level
// fail-shut guard: no commit DAG means no ancestry data to test
// against, so the rule stays silent rather than risk a false
// positive.
func TestPromoteOnWrongBranch_AC8_NilDAG_Silent(t *testing.T) {
	t.Parallel()
	commits := []scope.Commit{
		makePromoteCommit("epic-tip", "E-0001", "active", false),
	}
	got := RunPromoteOnWrongBranch(commits, map[string]string{"E-0001": "main"}, nil, promoteOnWrongBranchFixtureTips(), "main", nil, nil)
	if len(got) != 0 {
		t.Errorf("expected silent on nil dag; got %d findings", len(got))
	}
}

// TestPromoteOnWrongBranch_AC8_NonPromoteVerb_Silent pins the
// rule's verb filter — edit-body on an epic doesn't fire even
// if it lands on a non-parent branch.
func TestPromoteOnWrongBranch_AC8_NonPromoteVerb_Silent(t *testing.T) {
	t.Parallel()
	commits := []scope.Commit{{
		SHA: "epic-tip",
		Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "edit-body"},
			{Key: gitops.TrailerEntity, Value: "E-0001"},
			{Key: gitops.TrailerActor, Value: "human/peter"},
		},
	}}
	got := RunPromoteOnWrongBranch(commits, map[string]string{"E-0001": "main"}, promoteOnWrongBranchFixtureDAG(), promoteOnWrongBranchFixtureTips(), "main", nil, nil)
	if len(got) != 0 {
		t.Errorf("expected silent on non-promote verb; got %d findings", len(got))
	}
}

// TestPromoteOnWrongBranch_AC8_ReallocatedEntity_PreReallocationPromote_Silent
// pins the G-0308 fix: a pre-reallocation promote commit carries the
// FREED id (M-0099) in its aiwf-entity: trailer. That id has since
// been reclaimed by an unrelated live entity under a different
// parent epic (E-0009). Without resolving the trailer through
// prior_ids to the renumbered-forward entity (M-0001, parent
// E-0001), the naive lookup finds M-0099's *current* claimant's
// expectation (E-0009's branch) and fires a false positive against
// a commit that correctly landed on M-0001's real parent branch.
func TestPromoteOnWrongBranch_AC8_ReallocatedEntity_PreReallocationPromote_Silent(t *testing.T) {
	t.Parallel()
	tr := buildProvenanceTreeWithRenamedAndCollision(t)
	commits := []scope.Commit{
		// Historical commit: landed while this milestone was still
		// M-0099, correctly on its parent epic's branch — modeled as
		// literally being that branch's tip.
		makePromoteCommit("aaa1", "M-0099", "in_progress", false),
	}
	dag := testDAG(map[string][]string{"aaa1": nil, "zzz9": nil})
	branchTips := map[string]string{
		"epic/E-0001-platform":  "aaa1",
		"epic/E-0009-unrelated": "zzz9",
	}
	expectedBranches := map[string]string{
		"M-0001": "epic/E-0001-platform",  // current M-0001 (renumbered-forward), parent E-0001
		"M-0099": "epic/E-0009-unrelated", // unrelated live entity that reclaimed the freed id
	}
	got := RunPromoteOnWrongBranch(commits, expectedBranches, dag, branchTips, "main", nil, tr)
	if len(got) != 0 {
		t.Fatalf("findings = %v; pre-reallocation promote on the renumbered entity's real parent branch must resolve via prior_ids and stay silent", findingCodes(got))
	}
}

// TestPromoteOnWrongBranch_AC8_ReallocatedEntity_GenuineWrongBranch_StillFires
// pins the negative case for the same fixture: the prior_ids
// resolution is forward attribution, not a blanket suppression. A
// pre-reallocation commit that actually landed on the WRONG branch
// (an ancestor of neither M-0001's nor M-0099's expected tip) must
// still fire, now correctly attributed to the renumbered-forward
// entity's expectation rather than silenced or misattributed.
func TestPromoteOnWrongBranch_AC8_ReallocatedEntity_GenuineWrongBranch_StillFires(t *testing.T) {
	t.Parallel()
	tr := buildProvenanceTreeWithRenamedAndCollision(t)
	commits := []scope.Commit{
		makePromoteCommit("bbb2", "M-0099", "in_progress", false),
	}
	dag := testDAG(map[string][]string{
		"epic-plat-tip":  {"root"},
		"root":           nil,
		"bbb2":           {"unrelated-root"},
		"unrelated-root": nil,
	})
	branchTips := map[string]string{
		"epic/E-0001-platform": "epic-plat-tip",
	}
	expectedBranches := map[string]string{
		"M-0001": "epic/E-0001-platform",
		"M-0099": "epic/E-0009-unrelated",
	}
	got := RunPromoteOnWrongBranch(commits, expectedBranches, dag, branchTips, "main", nil, tr)
	if len(got) != 1 {
		t.Fatalf("findings = %v; genuine wrong-branch promote on a reallocated entity must still fire", findingCodes(got))
	}
	if !strings.Contains(got[0].Message, "epic/E-0001-platform") {
		t.Errorf("Message %q does not name the resolved entity's expected branch", got[0].Message)
	}
}

// TestPromoteOnWrongBranch_AC8_InWindowReallocateTrailer_ResolvesForward_Silent
// covers the sibling resolution path: when the reallocate commit
// itself is within the audited commit window, buildRenameChain +
// walkRenameChain resolve the old id forward from the
// aiwf-prior-entity: trailer alone — no tree lookup needed (t is
// nil here to isolate the in-window path from the prior_ids
// fallback exercised by the tests above). expectedBranches carries
// a deliberately WRONG entry under the freed id (M-0099) so that a
// naive unresolved lookup would misfire; only the resolved id
// (M-0001) reaches the correct (matching) expectation.
func TestPromoteOnWrongBranch_AC8_InWindowReallocateTrailer_ResolvesForward_Silent(t *testing.T) {
	t.Parallel()
	commits := []scope.Commit{
		{
			SHA: "realloc1",
			Trailers: []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "reallocate"},
				{Key: gitops.TrailerEntity, Value: "M-0001"},
				{Key: gitops.TrailerPriorEntity, Value: "M-0099"},
				{Key: gitops.TrailerActor, Value: "human/peter"},
			},
		},
		makePromoteCommit("ccc3", "M-0099", "in_progress", false),
	}
	dag := testDAG(map[string][]string{"ccc3": nil})
	branchTips := map[string]string{
		"epic/E-0001-platform": "ccc3",
	}
	expectedBranches := map[string]string{
		"M-0001": "epic/E-0001-platform",  // resolved-forward entity — matches the commit's actual branch
		"M-0099": "epic/E-0009-unrelated", // freed id's naive (wrong) expectation — must NOT be consulted
	}
	got := RunPromoteOnWrongBranch(commits, expectedBranches, dag, branchTips, "main", nil, nil)
	if len(got) != 0 {
		t.Fatalf("findings = %v; in-window aiwf-prior-entity trailer must resolve the old id forward and stay silent", findingCodes(got))
	}
}

// TestPromoteOnWrongBranch_AC8_CodeIsBranchChoreographyClass
// pins the code-class invariant per ADR-0011 + M-0123.
func TestPromoteOnWrongBranch_AC8_CodeIsBranchChoreographyClass(t *testing.T) {
	t.Parallel()
	if CodePromoteOnWrongBranch.Class != codespkg.ClassBranchChoreography {
		t.Errorf("Class = %v; want ClassBranchChoreography", CodePromoteOnWrongBranch.Class)
	}
	if CodePromoteOnWrongBranch.ID != "promote-on-wrong-branch" {
		t.Errorf("ID = %q; want %q", CodePromoteOnWrongBranch.ID, "promote-on-wrong-branch")
	}
}
