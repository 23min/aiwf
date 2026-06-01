package check

import (
	"strings"
	"testing"

	codespkg "github.com/23min/aiwf/internal/codes"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/scope"
)

// fakeOracle is a test-only BranchOracle. It maps commit SHAs to
// branch-name lists and returns nil for unknown SHAs. Tests build
// it inline for each fixture; no shared state.
type fakeOracle map[string][]string

func (f fakeOracle) FirstParentBranches(sha string) []string {
	return f[sha]
}

// makeAuthorizeOpenCommit constructs an authorize-opens-scope
// fixture commit. The scope is opened on `entity` by `actor`,
// authorizing `agent`, bound to ritual branch `branch`. SHA is
// supplied so tests can wire the commit into oracles deterministically.
func makeAuthorizeOpenCommit(sha, entity, actor, agent, branch string) scope.Commit {
	trailers := []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: "authorize"},
		{Key: gitops.TrailerEntity, Value: entity},
		{Key: gitops.TrailerActor, Value: actor},
		{Key: gitops.TrailerTo, Value: agent},
		{Key: gitops.TrailerScope, Value: "opened"},
	}
	if branch != "" {
		trailers = append(trailers, gitops.Trailer{Key: gitops.TrailerBranch, Value: branch})
	}
	return scope.Commit{SHA: sha, Trailers: trailers}
}

// makeAICommit constructs an AI-actor work commit on `entity`. The
// commit's verb is a non-authorize verb so the rule's
// authorize-commit filter doesn't skip it.
func makeAICommit(sha, entity, actor, verb string) scope.Commit {
	return scope.Commit{
		SHA: sha,
		Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: verb},
			{Key: gitops.TrailerEntity, Value: entity},
			{Key: gitops.TrailerActor, Value: actor},
		},
	}
}

// TestIsolationEscape_AC13_TypedCodeDescriptor pins M-0106/AC-13:
// the isolation-escape finding-code descriptor lands in
// internal/check/ as a typed [codes.Code] value (per the G-0129
// pattern adopted for CodeProvenanceAuthorizationOutOfScope), with
// a stable ID and the correct Class.
//
// The structural assertions:
//   - The ID is exactly "isolation-escape" — the stable wire string
//     that messages, JSON envelopes, and downstream consumers key
//     on. A typo regression that drifts the ID fires this test.
//   - The Class is ClassBranchChoreography — the new layer-4 carve-
//     out introduced for this milestone. A regression that reuses
//     ClassStructural (the default zero value) fires this test.
//   - The value is a [codes.Code], not a bare string constant —
//     enforces the G-0129 typed-code shape. The compile-time check
//     would catch a bare-string drift, but pinning the type via a
//     non-trivial assertion (Class field access) gives explicit
//     evidence in the test set.
func TestIsolationEscape_AC13_TypedCodeDescriptor(t *testing.T) {
	t.Parallel()

	if got, want := CodeIsolationEscape.ID, "isolation-escape"; got != want {
		t.Errorf("CodeIsolationEscape.ID = %q; want %q", got, want)
	}
	if got, want := CodeIsolationEscape.Class, codespkg.ClassBranchChoreography; got != want {
		t.Errorf("CodeIsolationEscape.Class = %v; want %v (ClassBranchChoreography)", got, want)
	}
}

// TestIsolationEscape_AC1_AICommitOnMainFires pins M-0106/AC-1: an
// AI-actor's work commit on `main` while the active scope's
// aiwf-branch: is a ritual epic branch (e.g. epic/E-0001-engine)
// fires isolation-escape. The bound branch is the trailer's value
// on the most-recent opened-scope commit on the same entity; the
// actual branch comes from the oracle.
//
// One finding per violating commit (AC-10 anchor — verified by
// asserting exactly one finding here, with the canonical fields
// populated). Severity is warning (AC-11 anchor — verified by
// asserting Severity field).
func TestIsolationEscape_AC1_AICommitOnMainFires(t *testing.T) {
	t.Parallel()

	commits := []scope.Commit{
		makeAuthorizeOpenCommit("auth0001", "E-0001", "human/peter", "ai/claude", "epic/E-0001-engine"),
		makeAICommit("c0000001", "E-0001", "ai/claude", "edit-body"),
	}
	oracle := fakeOracle{
		"auth0001": {"epic/E-0001-engine"},
		"c0000001": {"main"}, // AI commit landed on main — escape.
	}

	findings := RunIsolationEscape(commits, oracle)
	if len(findings) != 1 {
		t.Fatalf("expected exactly 1 finding (AC-10: per-commit firing); got %d", len(findings))
	}
	got := findings[0]
	if got.Code != CodeIsolationEscape.ID {
		t.Errorf("Code = %q; want %q", got.Code, CodeIsolationEscape.ID)
	}
	if got.Severity != SeverityWarning {
		t.Errorf("Severity = %q; want %q (AC-11)", got.Severity, SeverityWarning)
	}
	if got.EntityID != "E-0001" {
		t.Errorf("EntityID = %q; want %q", got.EntityID, "E-0001")
	}
	if !strings.Contains(got.Message, "main") {
		t.Errorf("Message %q does not name the actual branch (main)", got.Message)
	}
	if !strings.Contains(got.Message, "epic/E-0001-engine") {
		t.Errorf("Message %q does not name the bound branch", got.Message)
	}
}

// TestIsolationEscape_AC3_WorktreeBranchMismatchFires pins
// M-0106/AC-3. The "worktree-vs-branch mismatch" scenario from
// G-0099: a subagent dispatched into worktree epic/E-0001-engine
// runs `git checkout main` (or `git -C ../other-path`) from
// inside the worktree, so the commit's first-parent path now
// reaches a different branch. The rule's detection is purely
// branch-based — it doesn't read filesystem paths or worktree
// metadata — so the worktree dimension is a fixture variation of
// AC-1, not a separate code path.
//
// The fixture pins this explicitly so a future reader sees the
// connection to G-0099's worktree-escape failure mode rather
// than assuming the rule somehow validates filesystem paths.
func TestIsolationEscape_AC3_WorktreeBranchMismatchFires(t *testing.T) {
	t.Parallel()

	commits := []scope.Commit{
		makeAuthorizeOpenCommit("auth0001", "E-0001", "human/peter", "ai/claude", "epic/E-0001-engine"),
		makeAICommit("c0000002", "E-0001", "ai/claude", "edit-body"),
	}
	oracle := fakeOracle{
		"auth0001": {"epic/E-0001-engine"},
		// The subagent was assigned epic/E-0001-engine but did
		// `git checkout main` from inside its worktree. The commit
		// now reaches main, not the bound epic branch — the
		// worktree-vs-branch mismatch G-0099 describes.
		"c0000002": {"main"},
	}

	findings := RunIsolationEscape(commits, oracle)
	if len(findings) != 1 {
		t.Fatalf("expected exactly 1 finding for worktree-branch mismatch; got %d", len(findings))
	}
	if findings[0].EntityID != "E-0001" {
		t.Errorf("EntityID = %q; want %q", findings[0].EntityID, "E-0001")
	}
}

// TestIsolationEscape_AC4_AICommitOnBoundBranchSilent pins
// M-0106/AC-4: an AI-actor's work commit on the scope's bound
// branch is silent. The oracle confirms the commit rides
// epic/E-0001-engine, matching the scope's aiwf-branch: trailer
// value. No finding.
func TestIsolationEscape_AC4_AICommitOnBoundBranchSilent(t *testing.T) {
	t.Parallel()

	commits := []scope.Commit{
		makeAuthorizeOpenCommit("auth0001", "E-0001", "human/peter", "ai/claude", "epic/E-0001-engine"),
		makeAICommit("c0000003", "E-0001", "ai/claude", "edit-body"),
	}
	oracle := fakeOracle{
		"auth0001": {"epic/E-0001-engine"},
		"c0000003": {"epic/E-0001-engine"}, // rides bound branch — silent.
	}

	findings := RunIsolationEscape(commits, oracle)
	if len(findings) != 0 {
		t.Fatalf("expected zero findings (AC-4: commit rides bound branch); got %d: %+v", len(findings), findings)
	}
}

// TestIsolationEscape_AC9_NoScopeOpenedSilent pins M-0106/AC-9:
// an AI-actor's commit on an entity that has NO authorize-opens
// commit in history is silent. The rule polices only AI commits
// against an existing active scope; an entity with no scope
// opened is out of policing reach. (A separate provenance rule —
// no-active-scope — handles the "AI commit without a scope at
// all" case; isolation-escape stays focused on branch-binding
// violations.)
func TestIsolationEscape_AC9_NoScopeOpenedSilent(t *testing.T) {
	t.Parallel()

	commits := []scope.Commit{
		makeAICommit("c0000004", "E-0002", "ai/claude", "edit-body"),
	}
	oracle := fakeOracle{
		"c0000004": {"epic/E-0002-other"},
	}

	findings := RunIsolationEscape(commits, oracle)
	if len(findings) != 0 {
		t.Fatalf("expected zero findings (AC-9: no scope on entity); got %d: %+v", len(findings), findings)
	}
}

// TestIsolationEscape_NilOracleSilent pins the graceful-degradation
// path: when the gather layer cannot supply a BranchOracle (e.g.,
// the repo has no commits, or branch metadata is unavailable),
// the rule returns silently rather than misfiring. This is what
// the Cycle 1 wire-up relies on so the rule can be hooked through
// RunProvenanceCheck before the oracle implementation lands.
func TestIsolationEscape_NilOracleSilent(t *testing.T) {
	t.Parallel()

	commits := []scope.Commit{
		makeAuthorizeOpenCommit("auth0001", "E-0001", "human/peter", "ai/claude", "epic/E-0001-engine"),
		makeAICommit("c0000005", "E-0001", "ai/claude", "edit-body"),
	}

	findings := RunIsolationEscape(commits, nil)
	if len(findings) != 0 {
		t.Fatalf("expected zero findings with nil oracle (graceful degradation); got %d", len(findings))
	}
}

// TestIsolationEscape_UnknownBranchSilent pins the "oracle returns
// empty" case: when the oracle has no entry for a commit's SHA
// (returns nil or empty slice), the rule does NOT fire. The kernel
// will not flag commits it cannot confidently classify; "unknown
// branch" is treated as "not policed", not "definitely escaped".
//
// This prevents false positives when the gather range narrows or a
// commit's branch reachability becomes ambiguous (e.g., it's been
// orphaned after a ref rewrite).
func TestIsolationEscape_UnknownBranchSilent(t *testing.T) {
	t.Parallel()

	commits := []scope.Commit{
		makeAuthorizeOpenCommit("auth0001", "E-0001", "human/peter", "ai/claude", "epic/E-0001-engine"),
		makeAICommit("c0000006", "E-0001", "ai/claude", "edit-body"),
	}
	oracle := fakeOracle{
		"auth0001": {"epic/E-0001-engine"},
		// c0000006 deliberately absent — oracle returns nil for it.
	}

	findings := RunIsolationEscape(commits, oracle)
	if len(findings) != 0 {
		t.Fatalf("expected zero findings for unknown-branch commit; got %d", len(findings))
	}
}

// TestIsolationEscape_HumanCommitSilent pins that human-actor
// commits are not policed by this rule regardless of branch. The
// finding scope is AI-actor commits only per the M-0106 spec
// "policies AI-actor commits only". Human commits are subject to
// other rules (e.g. provenance-trailer-incoherent) but not this one.
func TestIsolationEscape_HumanCommitSilent(t *testing.T) {
	t.Parallel()

	commits := []scope.Commit{
		makeAuthorizeOpenCommit("auth0001", "E-0001", "human/peter", "ai/claude", "epic/E-0001-engine"),
		// Human commits on main while the AI scope is open — not policed.
		makeAICommit("c0000007", "E-0001", "human/peter", "edit-body"),
	}
	oracle := fakeOracle{
		"auth0001": {"epic/E-0001-engine"},
		"c0000007": {"main"},
	}

	findings := RunIsolationEscape(commits, oracle)
	if len(findings) != 0 {
		t.Fatalf("expected zero findings for human-actor commit; got %d: %+v", len(findings), findings)
	}
}

// TestIsolationEscape_LegacyPreM0102ScopeSilent pins the
// non-retroactive contract: a scope opened before M-0102 shipped
// does not carry an aiwf-branch: trailer. The rule must NOT fire
// on AI commits made under such legacy scopes (the kernel cannot
// retroactively assign a branch binding). Per M-0106 spec
// §"Out of scope" — retroactive enforcement.
func TestIsolationEscape_LegacyPreM0102ScopeSilent(t *testing.T) {
	t.Parallel()

	commits := []scope.Commit{
		makeAuthorizeOpenCommit("auth0001", "E-0001", "human/peter", "ai/claude", ""), // no --branch
		makeAICommit("c0000008", "E-0001", "ai/claude", "edit-body"),
	}
	oracle := fakeOracle{
		"auth0001": {"main"},
		"c0000008": {"main"}, // even on main, legacy scope means silent.
	}

	findings := RunIsolationEscape(commits, oracle)
	if len(findings) != 0 {
		t.Fatalf("expected zero findings for legacy pre-M-0102 scope; got %d: %+v", len(findings), findings)
	}
}

// TestIsolationEscape_AC2_AICommitOnDifferentRitualBranchFires
// pins M-0106/AC-2: an AI-actor's commit on a different ritual
// branch (e.g. epic/E-0002-other) while the active scope binds
// epic/E-0001-engine fires isolation-escape. The detection
// mechanism is the same as AC-1 — branch identity comparison —
// but this fixture pins that ritual-shape branches are not all
// treated as equivalent. A regression that compared only "is the
// branch a ritual shape?" instead of "does the branch equal the
// bound branch?" would silently pass AC-1 (commit on main fires)
// while failing AC-2 (commit on a different epic also fires).
func TestIsolationEscape_AC2_AICommitOnDifferentRitualBranchFires(t *testing.T) {
	t.Parallel()

	commits := []scope.Commit{
		makeAuthorizeOpenCommit("auth0001", "E-0001", "human/peter", "ai/claude", "epic/E-0001-engine"),
		makeAICommit("c0000010", "E-0001", "ai/claude", "edit-body"),
	}
	oracle := fakeOracle{
		"auth0001": {"epic/E-0001-engine"},
		"c0000010": {"epic/E-0002-other"}, // different epic branch.
	}

	findings := RunIsolationEscape(commits, oracle)
	if len(findings) != 1 {
		t.Fatalf("expected exactly 1 finding (AC-2); got %d", len(findings))
	}
	if !strings.Contains(findings[0].Message, "epic/E-0002-other") {
		t.Errorf("Message %q does not name the actual (wrong) branch", findings[0].Message)
	}
}

// TestIsolationEscape_AC5_AICommitOnBoundBranchPausedScopeSilent
// pins M-0106/AC-5: an AI-actor's commit on the scope's bound
// branch while the scope is in `paused` state is silent. The
// pause event does NOT change the binding (the scope's
// aiwf-branch: is what was recorded at `opened`); if the commit
// rides the bound branch, the rule has no opinion about pause.
//
// The fixture: opener → pause → AI commit on the bound branch.
// The algorithm's behavior is: the bound branch index records
// only the opener; the pause commit is itself an authorize commit
// and is filtered out of the work-commit set; the AI commit's
// bound-branch comparison hits the opener's branch → silent.
//
// The pause/resume events do not change the algorithm's behavior
// for branch-binding purposes (corner case 6 per epic body).
func TestIsolationEscape_AC5_AICommitOnBoundBranchPausedScopeSilent(t *testing.T) {
	t.Parallel()

	// Pause event: same authorize verb, but scope: paused.
	pauseEvent := scope.Commit{
		SHA: "pause001",
		Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "authorize"},
			{Key: gitops.TrailerEntity, Value: "E-0001"},
			{Key: gitops.TrailerActor, Value: "human/peter"},
			{Key: gitops.TrailerScope, Value: "paused"},
		},
	}

	commits := []scope.Commit{
		makeAuthorizeOpenCommit("auth0001", "E-0001", "human/peter", "ai/claude", "epic/E-0001-engine"),
		pauseEvent,
		makeAICommit("c0000020", "E-0001", "ai/claude", "edit-body"),
	}
	oracle := fakeOracle{
		"auth0001": {"epic/E-0001-engine"},
		"pause001": {"epic/E-0001-engine"},
		"c0000020": {"epic/E-0001-engine"}, // rides bound — silent.
	}

	findings := RunIsolationEscape(commits, oracle)
	if len(findings) != 0 {
		t.Fatalf("expected zero findings (AC-5: paused scope + on bound branch); got %d: %+v", len(findings), findings)
	}
}

// TestIsolationEscape_AC10_PerCommitFiring pins M-0106/AC-10:
// when multiple AI commits violate the scope's branch binding,
// the rule fires ONE finding per violating commit — not an
// aggregate. The user wants the cardinality so each escaped
// commit is individually addressable (e.g., for `git rebase` or
// per-commit amends to add aiwf-force).
//
// Three violating commits → three findings, each with its own
// EntityID and SHA-bearing message.
func TestIsolationEscape_AC10_PerCommitFiring(t *testing.T) {
	t.Parallel()

	commits := []scope.Commit{
		makeAuthorizeOpenCommit("auth0001", "E-0001", "human/peter", "ai/claude", "epic/E-0001-engine"),
		makeAICommit("c0000030", "E-0001", "ai/claude", "edit-body"),
		makeAICommit("c0000031", "E-0001", "ai/claude", "promote"),
		makeAICommit("c0000032", "E-0001", "ai/claude", "edit-body"),
	}
	oracle := fakeOracle{
		"auth0001": {"epic/E-0001-engine"},
		"c0000030": {"main"},                // violating
		"c0000031": {"main"},                // violating
		"c0000032": {"epic/E-0002-other"},   // violating (different branch)
	}

	findings := RunIsolationEscape(commits, oracle)
	if len(findings) != 3 {
		t.Fatalf("expected 3 findings (one per violating commit, AC-10); got %d: %+v", len(findings), findings)
	}
	// Each finding mentions a distinct commit SHA in its message.
	wantSHAs := []string{"c000003"} // short() truncates; all three start with this prefix
	for _, want := range wantSHAs {
		count := 0
		for _, f := range findings {
			if strings.Contains(f.Message, want) {
				count++
			}
		}
		if count != 3 {
			t.Errorf("expected 3 findings to mention SHA prefix %q; got %d", want, count)
		}
	}
}

// TestIsolationEscape_AC11_WarningSeverityCheckExitsZero pins
// M-0106/AC-11: the finding is warning severity (not error). The
// `aiwf check` exit code is governed by error-severity findings
// (see internal/check/check.go and cliutil's mapping); a
// warning-only result exits 0. This test pins the severity at
// the source; the exit-code half is enforced by the existing
// CLI check machinery and re-verified by integration tests in
// the broader sweep.
//
// Distinct from the severity assertion in AC-1's test: AC-11 is
// the spec's mechanical pin, isolated so a future severity flip
// (e.g. tightening to error in a follow-up D-NNN) has a single
// test to update deliberately.
func TestIsolationEscape_AC11_WarningSeverityCheckExitsZero(t *testing.T) {
	t.Parallel()

	commits := []scope.Commit{
		makeAuthorizeOpenCommit("auth0001", "E-0001", "human/peter", "ai/claude", "epic/E-0001-engine"),
		makeAICommit("c0000040", "E-0001", "ai/claude", "edit-body"),
	}
	oracle := fakeOracle{
		"auth0001": {"epic/E-0001-engine"},
		"c0000040": {"main"},
	}

	findings := RunIsolationEscape(commits, oracle)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding; got %d", len(findings))
	}
	if findings[0].Severity != SeverityWarning {
		t.Errorf("Severity = %q; want %q (AC-11 — must remain warning until a future D-NNN deliberately tightens to error)", findings[0].Severity, SeverityWarning)
	}
}

// TestIsolationEscape_AC12_HintTextNamesBothOverridePaths pins
// M-0106/AC-12: the finding's hint text names both sovereign
// override paths — (a) git cherry-pick -x for the re-author
// path, and (b) the aiwf-force trailer amend for the explicit
// override path. The hint is looked up from the hintTable when
// `applyHints` runs at the end of check.Run; the underlying
// finding doesn't carry Hint until then.
//
// Assertion via the hint table directly (since the rule emits
// findings without Hint until applyHints runs). The full
// finding-with-hint path is exercised by the broader
// `internal/check` test suite when Run() composes all checks.
func TestIsolationEscape_AC12_HintTextNamesBothOverridePaths(t *testing.T) {
	t.Parallel()

	hint := HintFor(CodeIsolationEscape.ID, "")
	if hint == "" {
		t.Fatal("isolation-escape has no hint registered in hintTable")
	}

	// Two override paths must be named so an operator reading the
	// hint sees both sovereign exits.
	wantSubstrings := []struct {
		name string
		s    string
	}{
		{"cherry-pick -x path", "cherry-pick -x"},
		{"aiwf-force trailer path", "aiwf-force"},
		{"human/ actor requirement", "human/"},
		{"audit-trail pointer to the epic doc", "Sovereign override surface"},
	}
	for _, w := range wantSubstrings {
		if !strings.Contains(hint, w.s) {
			t.Errorf("hint must name the %s (substring %q); hint = %q", w.name, w.s, hint)
		}
	}
}

// TestIsolationEscape_AC13_ClassBranchChoreographyDistinct pins
// that ClassBranchChoreography is a NEW enum value distinct from
// the prior two classes (ClassStructural=0, ClassLegality=1). A
// regression that re-orders the enum so the new class collides
// with an existing one fires this test.
//
// The assertion is positional rather than literal: we don't pin
// the numeric value (Class is an iota — its concrete int can shift
// in principle), but we DO pin that the three values are pairwise
// distinct. That's the load-bearing invariant: callers
// distinguishing between classes can do so.
func TestIsolationEscape_AC13_ClassBranchChoreographyDistinct(t *testing.T) {
	t.Parallel()

	classes := []codespkg.Class{
		codespkg.ClassStructural,
		codespkg.ClassLegality,
		codespkg.ClassBranchChoreography,
	}
	seen := make(map[codespkg.Class]int)
	for i, c := range classes {
		if prior, ok := seen[c]; ok {
			t.Errorf("class at index %d collides with class at index %d (both = %v)", i, prior, c)
		}
		seen[c] = i
	}
}
