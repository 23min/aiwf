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
// partial-close) unit-level coverage of RunPromoteOnWrongBranch
// per CLAUDE.md §"Test the seam, not just the layer" + the
// AC-5/AC-7 reviewer pattern: the E2E exercises the production
// wire-up; the unit tests below pin the rule's input/output
// contract against in-memory fixtures.
//
// Branch coverage:
//   - epic activating promote on correct trunk → silent
//   - epic activating promote on wrong branch → fires
//   - milestone activating promote on parent epic → silent
//   - milestone activating promote on wrong branch → fires
//   - non-activating promote (active → done) → silent
//   - non-promote verb (edit-body, etc.) → silent
//   - empty expectedBranches map / missing entity → silent
//   - per-commit aiwf-force override → silent
//   - per-SHA ack via ackedSHAs → silent
//   - unknown branch (oracle returns empty for SHA) → silent
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

// TestPromoteOnWrongBranch_AC8_EpicCorrectBranch_Silent pins
// the silent-good path: epic activating promote on the trunk
// matches the expected branch → no finding.
func TestPromoteOnWrongBranch_AC8_EpicCorrectBranch_Silent(t *testing.T) {
	t.Parallel()
	commits := []scope.Commit{
		makePromoteCommit("aaa1", "E-0001", "active", false),
	}
	oracle := fakeOracle{"aaa1": {"main"}}
	got := RunPromoteOnWrongBranch(commits, map[string]string{"E-0001": "main"}, oracle, nil, nil)
	if len(got) != 0 {
		t.Errorf("expected silent on correct branch; got %d findings: %+v", len(got), got)
	}
}

// TestPromoteOnWrongBranch_AC8_EpicWrongBranch_Fires pins the
// load-bearing claim: epic activating promote on a non-trunk
// branch fires the warning with message naming the expected
// and actual branches.
func TestPromoteOnWrongBranch_AC8_EpicWrongBranch_Fires(t *testing.T) {
	t.Parallel()
	commits := []scope.Commit{
		makePromoteCommit("bbb2", "E-0001", "active", false),
	}
	oracle := fakeOracle{"bbb2": {"epic/E-0001-engine"}}
	got := RunPromoteOnWrongBranch(commits, map[string]string{"E-0001": "main"}, oracle, nil, nil)
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
	if !strings.Contains(f.Message, "epic/E-0001-engine") {
		t.Errorf("Message %q does not name actual branch", f.Message)
	}
	if !strings.Contains(f.Hint, "aiwf acknowledge illegal") {
		t.Errorf("Hint %q does not name the acknowledge-illegal recovery path", f.Hint)
	}
	if !strings.Contains(f.Hint, "aiwf-force:") {
		t.Errorf("Hint %q does not name the per-commit aiwf-force override path", f.Hint)
	}
}

// TestPromoteOnWrongBranch_AC8_MilestoneCorrectParentEpic_Silent
// pins the milestone-side silent-good: milestone in_progress
// on its parent epic's ritual branch.
func TestPromoteOnWrongBranch_AC8_MilestoneCorrectParentEpic_Silent(t *testing.T) {
	t.Parallel()
	commits := []scope.Commit{
		makePromoteCommit("ccc3", "M-0010", "in_progress", false),
	}
	oracle := fakeOracle{"ccc3": {"epic/E-0001-engine"}}
	got := RunPromoteOnWrongBranch(commits, map[string]string{"M-0010": "epic/E-0001-engine"}, oracle, nil, nil)
	if len(got) != 0 {
		t.Errorf("expected silent on parent-epic branch; got %d findings", len(got))
	}
}

// TestPromoteOnWrongBranch_AC8_NonActivatingPromote_Silent pins
// the rule's domain narrowness: epic active → done is OUT of
// the rule's domain regardless of branch.
func TestPromoteOnWrongBranch_AC8_NonActivatingPromote_Silent(t *testing.T) {
	t.Parallel()
	commits := []scope.Commit{
		makePromoteCommit("ddd4", "E-0001", "done", false),
	}
	oracle := fakeOracle{"ddd4": {"epic/E-0001-engine"}}
	got := RunPromoteOnWrongBranch(commits, map[string]string{"E-0001": "main"}, oracle, nil, nil)
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
		makePromoteCommit("eee5", "E-0001", "active", true),
	}
	oracle := fakeOracle{"eee5": {"epic/E-0001-engine"}}
	got := RunPromoteOnWrongBranch(commits, map[string]string{"E-0001": "main"}, oracle, nil, nil)
	if len(got) != 0 {
		t.Errorf("expected silent on forced commit; got %d findings", len(got))
	}
}

// TestPromoteOnWrongBranch_AC8_AcknowledgedSHASilences pins
// the post-hoc override via the shared ackedSHAs map.
func TestPromoteOnWrongBranch_AC8_AcknowledgedSHASilences(t *testing.T) {
	t.Parallel()
	commits := []scope.Commit{
		makePromoteCommit("fff6", "E-0001", "active", false),
	}
	oracle := fakeOracle{"fff6": {"epic/E-0001-engine"}}
	acked := map[string]bool{"fff6": true}
	got := RunPromoteOnWrongBranch(commits, map[string]string{"E-0001": "main"}, oracle, acked, nil)
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
		makePromoteCommit("ggg7", "E-0001", "active", false),
	}
	oracle := fakeOracle{"ggg7": {"epic/E-0001-engine"}}
	// Empty map → no expectation → silent.
	got := RunPromoteOnWrongBranch(commits, map[string]string{}, oracle, nil, nil)
	if len(got) != 0 {
		t.Errorf("expected silent on no-expectation; got %d findings", len(got))
	}
}

// TestPromoteOnWrongBranch_AC8_UnknownBranch_Silent pins the
// fail-shut on commits the oracle can't classify (AC-3 D-0019
// composition).
func TestPromoteOnWrongBranch_AC8_UnknownBranch_Silent(t *testing.T) {
	t.Parallel()
	commits := []scope.Commit{
		makePromoteCommit("hhh8", "E-0001", "active", false),
	}
	// Oracle returns empty for the SHA (unknown branch).
	oracle := fakeOracle{}
	got := RunPromoteOnWrongBranch(commits, map[string]string{"E-0001": "main"}, oracle, nil, nil)
	if len(got) != 0 {
		t.Errorf("expected silent on unknown branch; got %d findings", len(got))
	}
}

// TestPromoteOnWrongBranch_AC8_NonPromoteVerb_Silent pins the
// rule's verb filter — edit-body on an epic doesn't fire even
// if it lands on a non-parent branch.
func TestPromoteOnWrongBranch_AC8_NonPromoteVerb_Silent(t *testing.T) {
	t.Parallel()
	commits := []scope.Commit{{
		SHA: "iii9",
		Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "edit-body"},
			{Key: gitops.TrailerEntity, Value: "E-0001"},
			{Key: gitops.TrailerActor, Value: "human/peter"},
		},
	}}
	oracle := fakeOracle{"iii9": {"epic/E-0001-engine"}}
	got := RunPromoteOnWrongBranch(commits, map[string]string{"E-0001": "main"}, oracle, nil, nil)
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
		// M-0099, correctly on its parent epic's branch.
		makePromoteCommit("aaa1", "M-0099", "in_progress", false),
	}
	oracle := fakeOracle{"aaa1": {"epic/E-0001-platform"}}
	expectedBranches := map[string]string{
		"M-0001": "epic/E-0001-platform",  // current M-0001 (renumbered-forward), parent E-0001
		"M-0099": "epic/E-0009-unrelated", // unrelated live entity that reclaimed the freed id
	}
	got := RunPromoteOnWrongBranch(commits, expectedBranches, oracle, nil, tr)
	if len(got) != 0 {
		t.Fatalf("findings = %v; pre-reallocation promote on the renumbered entity's real parent branch must resolve via prior_ids and stay silent", findingCodes(got))
	}
}

// TestPromoteOnWrongBranch_AC8_ReallocatedEntity_GenuineWrongBranch_StillFires
// pins the negative case for the same fixture: the prior_ids
// resolution is forward attribution, not a blanket suppression. A
// pre-reallocation commit that actually landed on the WRONG branch
// (neither M-0001's nor M-0099's expectation) must still fire, now
// correctly attributed to the renumbered-forward entity's
// expectation rather than silenced or misattributed.
func TestPromoteOnWrongBranch_AC8_ReallocatedEntity_GenuineWrongBranch_StillFires(t *testing.T) {
	t.Parallel()
	tr := buildProvenanceTreeWithRenamedAndCollision(t)
	commits := []scope.Commit{
		makePromoteCommit("bbb2", "M-0099", "in_progress", false),
	}
	oracle := fakeOracle{"bbb2": {"main"}}
	expectedBranches := map[string]string{
		"M-0001": "epic/E-0001-platform",
		"M-0099": "epic/E-0009-unrelated",
	}
	got := RunPromoteOnWrongBranch(commits, expectedBranches, oracle, nil, tr)
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
	oracle := fakeOracle{"ccc3": {"epic/E-0001-platform"}}
	expectedBranches := map[string]string{
		"M-0001": "epic/E-0001-platform",  // resolved-forward entity — matches the commit's actual branch
		"M-0099": "epic/E-0009-unrelated", // freed id's naive (wrong) expectation — must NOT be consulted
	}
	got := RunPromoteOnWrongBranch(commits, expectedBranches, oracle, nil, nil)
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
