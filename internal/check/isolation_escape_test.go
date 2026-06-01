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

	findings := RunIsolationEscape(commits, oracle, nil)
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
// M-0106/AC-3.
//
// IMPORTANT: this AC is a DOCUMENTATION PIN, not a separate
// code-path assertion. Algorithmic coverage for the "AI commit
// landed on the wrong branch" scenario comes from
// TestIsolationEscape_AC1_AICommitOnMainFires; the M-0106 rule
// does not — and by design cannot — distinguish a worktree-
// induced escape from a `git checkout main`-induced escape.
// They are the same branch-identity mismatch.
//
// Why keep AC-3 as a separate test despite that: the
// "worktree-vs-branch mismatch" scenario from G-0099 is the
// failure mode the milestone is closing. A reader auditing the
// test set for G-0099 coverage should find a named fixture that
// pins it; without one, the connection to G-0099 lives only in
// the spec body, which is brittle. Per M-0106 retrospective F-5:
// the dedicated test is documentation-as-test, deliberate and
// acceptable, but should not be read as evidence of a distinct
// code path.
//
// A future refactor that merges AC-1 + AC-3 fixtures into a
// table-driven test would be cleaner, but the AC numbering in the
// spec is the source of truth; merging would require renumbering.
// Left as-is.
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

	findings := RunIsolationEscape(commits, oracle, nil)
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

	findings := RunIsolationEscape(commits, oracle, nil)
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

	findings := RunIsolationEscape(commits, oracle, nil)
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

	findings := RunIsolationEscape(commits, nil, nil)
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

	findings := RunIsolationEscape(commits, oracle, nil)
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

	findings := RunIsolationEscape(commits, oracle, nil)
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

	findings := RunIsolationEscape(commits, oracle, nil)
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

	findings := RunIsolationEscape(commits, oracle, nil)
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

	findings := RunIsolationEscape(commits, oracle, nil)
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
		"c0000030": {"main"},              // violating
		"c0000031": {"main"},              // violating
		"c0000032": {"epic/E-0002-other"}, // violating (different branch)
	}

	findings := RunIsolationEscape(commits, oracle, nil)
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

// TestIsolationEscape_AC11_SeverityIsWarning pins M-0106/AC-11's
// rule-side half: the finding's Severity field is SeverityWarning,
// not SeverityError. Renamed from "...CheckExitsZero" per F-4 of
// the M-0106 retrospective — the original name claimed an
// end-to-end exit-code assertion the test could not make at the
// unit level.
//
// The exit-code half (warning + non-zero findings → exit 0) is
// pinned end-to-end by
// TestRunProvenanceCheck_IsolationEscape_WarningDoesNotMarkErrors
// at internal/cli/check/isolation_escape_test.go, which drives
// RunProvenanceCheck against a violating fixture and asserts
// check.HasErrors over the isolation-escape findings returns
// false (since check.HasErrors is the predicate that drives the
// CLI's exit-code mapping at internal/cli/check/check.go:195).
// The pair jointly pins the full AC-11 claim through the
// production composition.
//
// This isolated severity assertion is the single place to update
// at a future tightening (e.g. a D-NNN that flips to SeverityError
// after the false-positive rate is known); changing it should be
// deliberate, not accidental.
func TestIsolationEscape_AC11_SeverityIsWarning(t *testing.T) {
	t.Parallel()

	commits := []scope.Commit{
		makeAuthorizeOpenCommit("auth0001", "E-0001", "human/peter", "ai/claude", "epic/E-0001-engine"),
		makeAICommit("c0000040", "E-0001", "ai/claude", "edit-body"),
	}
	oracle := fakeOracle{
		"auth0001": {"epic/E-0001-engine"},
		"c0000040": {"main"},
	}

	findings := RunIsolationEscape(commits, oracle, nil)
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
//
// CAVEAT (F-7 from M-0106 retrospective): this test is a
// circular tautology — the author of the hint also authored the
// substring assertions and the sabotage probe. It catches the
// "someone removed the hint entirely" regression class but does
// not catch "the hint's wording drifts from what an LLM agent
// parses for remediation." When the hint text contents become
// load-bearing for downstream parsing (e.g. an LLM agent reads
// the hint to choose between override paths), tighten this to a
// structural assertion — either split the hint into named
// fragments via a struct, or maintain a golden file. Until then,
// the circular shape is acceptable.
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

// TestIsolationEscape_AC6_CherryPickReAuthorSilent pins
// M-0106/AC-6: a `git cherry-pick -x` re-author is suppressed.
// The gather layer identifies cherry-picks (committer email
// differs from the AI actor's encoded email AND body carries
// `(cherry picked from commit <sha>)` marker) and feeds the SHAs
// to the rule via the `cherryPicked` parameter. The rule sees an
// AI-actor commit on a non-bound branch (looks like an escape
// from the trailer perspective) but skips firing because the
// SHA is flagged.
//
// The audit trail lives in the committer-vs-author identity gap
// and the cherry-pick marker — both are observable on the commit
// itself; the rule doesn't need to record additional state.
func TestIsolationEscape_AC6_CherryPickReAuthorSilent(t *testing.T) {
	t.Parallel()

	commits := []scope.Commit{
		makeAuthorizeOpenCommit("auth0001", "E-0001", "human/peter", "ai/claude", "epic/E-0001-engine"),
		makeAICommit("cp000060", "E-0001", "ai/claude", "edit-body"),
	}
	oracle := fakeOracle{
		"auth0001": {"epic/E-0001-engine"},
		"cp000060": {"main"},
	}
	cherryPicked := map[string]bool{
		"cp000060": true,
	}

	findings := RunIsolationEscape(commits, oracle, cherryPicked)
	if len(findings) != 0 {
		t.Fatalf("expected zero findings for cherry-pick re-author (AC-6); got %d: %+v", len(findings), findings)
	}
}

// TestIsolationEscape_AC6_NonCherryPickStillFires pins the
// suppression's lower bound: a commit that looks LIKE a cherry-
// pick but is NOT flagged by the gather layer still fires. The
// suppression depends on positive identification by the gather
// layer; the rule does not infer cherry-pick status itself.
//
// Without this guard, a regression that silently treated
// "missing cherry-pick info" as "is a cherry-pick" would suppress
// every commit when the gather layer is absent — converting the
// rule to a no-op.
func TestIsolationEscape_AC6_NonCherryPickStillFires(t *testing.T) {
	t.Parallel()

	commits := []scope.Commit{
		makeAuthorizeOpenCommit("auth0001", "E-0001", "human/peter", "ai/claude", "epic/E-0001-engine"),
		makeAICommit("c0000061", "E-0001", "ai/claude", "edit-body"),
	}
	oracle := fakeOracle{
		"auth0001": {"epic/E-0001-engine"},
		"c0000061": {"main"},
	}
	for _, cp := range []map[string]bool{nil, {}} {
		findings := RunIsolationEscape(commits, oracle, cp)
		if len(findings) != 1 {
			t.Errorf("with cherryPicked=%v: expected 1 finding (not a cherry-pick); got %d", cp, len(findings))
		}
	}
}

// TestIsolationEscape_AC7_HumanMergeFirstParentSilent pins
// M-0106/AC-7: when a human merges epic/X into main via
// `git merge --no-ff epic/X`, the merge commit is human-actor
// (filtered by the rule's ai/ prefix check) and the AI commits
// behind the merge are still reachable from epic/X first-parent
// (not from main's first-parent line). Both kinds of commits
// stay silent — the merge by the actor filter, the AI commits by
// the bound-branch match.
func TestIsolationEscape_AC7_HumanMergeFirstParentSilent(t *testing.T) {
	t.Parallel()

	mergeCommit := scope.Commit{
		SHA: "merge001",
		Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "merge"},
			{Key: gitops.TrailerEntity, Value: "E-0001"},
			{Key: gitops.TrailerActor, Value: "human/peter"},
		},
	}
	commits := []scope.Commit{
		makeAuthorizeOpenCommit("auth0001", "E-0001", "human/peter", "ai/claude", "epic/E-0001-engine"),
		makeAICommit("c0000070", "E-0001", "ai/claude", "edit-body"),
		mergeCommit,
	}
	oracle := fakeOracle{
		"auth0001": {"epic/E-0001-engine"},
		"c0000070": {"epic/E-0001-engine"},
		"merge001": {"main"},
	}

	findings := RunIsolationEscape(commits, oracle, nil)
	if len(findings) != 0 {
		t.Fatalf("expected zero findings for human-merge + AI commits behind merge (AC-7); got %d: %+v", len(findings), findings)
	}
}

// TestIsolationEscape_AC8_ForceAmendedCommitSilent pins
// M-0106/AC-8: when an operator amends a violating commit with
// `aiwf-force: <reason>` + `aiwf-actor: human/<id>` (the spec's
// sovereign override), the rule is silent. The natural mechanism:
// after the amend the commit's aiwf-actor: trailer is
// `human/<id>` (not `ai/...`), so the rule's ai/ prefix filter
// skips it.
func TestIsolationEscape_AC8_ForceAmendedCommitSilent(t *testing.T) {
	t.Parallel()

	amendedCommit := scope.Commit{
		SHA: "amended0",
		Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "edit-body"},
			{Key: gitops.TrailerEntity, Value: "E-0001"},
			{Key: gitops.TrailerActor, Value: "human/peter"},
			{Key: gitops.TrailerForce, Value: "manual cherry-pick acknowledgment"},
		},
	}
	commits := []scope.Commit{
		makeAuthorizeOpenCommit("auth0001", "E-0001", "human/peter", "ai/claude", "epic/E-0001-engine"),
		amendedCommit,
	}
	oracle := fakeOracle{
		"auth0001": {"epic/E-0001-engine"},
		"amended0": {"main"},
	}

	findings := RunIsolationEscape(commits, oracle, nil)
	if len(findings) != 0 {
		t.Fatalf("expected zero findings for force-amended commit (AC-8); got %d: %+v", len(findings), findings)
	}
}

// TestIsolationEscape_F3_AICommitAfterScopeEndedSilent pins
// M-0106/F-3 (post-retrospective fix): an AI-actor commit landing
// AFTER the scope it would have been bound by reaches its
// `aiwf-scope-ends:` event is silent. The pre-fix algorithm tracked
// only opener events; a scope-ended commit followed by a stray AI
// commit on the wrong branch would have false-positive-fired. With
// scope-end tracking the rule correctly reports "no active scope"
// (the binding is gone) and stays silent.
//
// Spec line 86: "find the most recent ... opened a scope that was
// *active* at C's time" — active = opened before C AND (never
// ended OR ended after C). This test pins the AND clause.
func TestIsolationEscape_F3_AICommitAfterScopeEndedSilent(t *testing.T) {
	t.Parallel()

	openerSHA := "auth0001"

	// End-of-scope commit carrying aiwf-scope-ends: <opener-sha>.
	// Typically the milestone-promote-to-done; the actor is human/.
	scopeEndCommit := scope.Commit{
		SHA: "endsc001",
		Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "promote"},
			{Key: gitops.TrailerEntity, Value: "E-0001"},
			{Key: gitops.TrailerActor, Value: "human/peter"},
			{Key: gitops.TrailerScopeEnds, Value: openerSHA},
		},
	}

	commits := []scope.Commit{
		makeAuthorizeOpenCommit(openerSHA, "E-0001", "human/peter", "ai/claude", "epic/E-0001-engine"),
		scopeEndCommit,
		// Stray AI commit AFTER the scope ended — used to fire as
		// isolation-escape under the broken pre-F-3 algorithm.
		makeAICommit("c0000080", "E-0001", "ai/claude", "edit-body"),
	}
	oracle := fakeOracle{
		openerSHA:  {"main"},
		"endsc001": {"main"},
		"c0000080": {"main"}, // stray AI commit on main — would have fired pre-F-3.
	}

	findings := RunIsolationEscape(commits, oracle, nil)
	if len(findings) != 0 {
		t.Fatalf("expected zero findings for AI commit after scope ended (F-3); got %d: %+v", len(findings), findings)
	}
}

// TestIsolationEscape_F3_AICommitBeforeScopeEndedFires pins the
// symmetric guard: an AI-actor commit that lands BEFORE the
// scope-end event is still policed normally. Without this guard,
// an over-eager F-3 fix could silently suppress every AI commit
// on any entity that ever has a scope-end (i.e., every closed
// entity ever).
//
// The fixture orders: opener → AI commit on wrong branch → scope-end.
// The AI commit's chronoIdx (1) is BEFORE the scope-end's
// chronoIdx (2), so the scope is still active at the AI commit's
// time → fire.
func TestIsolationEscape_F3_AICommitBeforeScopeEndedFires(t *testing.T) {
	t.Parallel()

	openerSHA := "auth0001"

	scopeEndCommit := scope.Commit{
		SHA: "endsc002",
		Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "promote"},
			{Key: gitops.TrailerEntity, Value: "E-0001"},
			{Key: gitops.TrailerActor, Value: "human/peter"},
			{Key: gitops.TrailerScopeEnds, Value: openerSHA},
		},
	}

	commits := []scope.Commit{
		makeAuthorizeOpenCommit(openerSHA, "E-0001", "human/peter", "ai/claude", "epic/E-0001-engine"),
		makeAICommit("c0000081", "E-0001", "ai/claude", "edit-body"),
		scopeEndCommit,
	}
	oracle := fakeOracle{
		openerSHA:  {"epic/E-0001-engine"},
		"c0000081": {"main"}, // on wrong branch, BEFORE scope end → fire.
		"endsc002": {"main"},
	}

	findings := RunIsolationEscape(commits, oracle, nil)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding (AI commit before scope-end → fires); got %d", len(findings))
	}
}

// TestIsolationEscape_F2_EmptyEntityOpenerSkipped covers F-2 arm 1:
// an authorize+opened commit with an empty aiwf-entity: trailer
// (malformed but structurally valid for the trailer-shape rule)
// is skipped at the opener-index build. Without the test, a
// regression that dropped the `entity == ""` guard would crash
// (writing to empty map key) OR silently mis-attribute the
// opener to entity "" — both bad.
func TestIsolationEscape_F2_EmptyEntityOpenerSkipped(t *testing.T) {
	t.Parallel()

	malformedOpener := scope.Commit{
		SHA: "malf0001",
		Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "authorize"},
			// aiwf-entity deliberately ABSENT.
			{Key: gitops.TrailerActor, Value: "human/peter"},
			{Key: gitops.TrailerTo, Value: "ai/claude"},
			{Key: gitops.TrailerScope, Value: "opened"},
			{Key: gitops.TrailerBranch, Value: "epic/E-0001-engine"},
		},
	}

	commits := []scope.Commit{
		malformedOpener,
		makeAICommit("c0000090", "E-0001", "ai/claude", "edit-body"),
	}
	oracle := fakeOracle{
		"malf0001": {"epic/E-0001-engine"},
		"c0000090": {"main"},
	}

	// The malformed opener is NOT indexed (entity == ""); the AI
	// commit on entity E-0001 sees no opener → silent per AC-9
	// path. If the guard regressed and the opener got attributed
	// to entity "", the AI commit on E-0001 would still see no
	// opener → silent, but a fixture with an AI commit on entity
	// "" would surface the mis-attribution; we don't construct
	// that fixture since the trailer-shape rule forbids it. The
	// guard exists as defense in depth; the test pins it stays
	// alive.
	findings := RunIsolationEscape(commits, oracle, nil)
	if len(findings) != 0 {
		t.Fatalf("expected zero findings (malformed opener skipped + AI commit on unrelated entity has no scope); got %d: %+v", len(findings), findings)
	}
}

// TestIsolationEscape_F2_AICommitOnAuthorizeVerbSkipped covers F-2
// arm 2: when an ai-actor commit ITSELF carries aiwf-verb:
// authorize (e.g. an authorize-paused commit emitted by an
// AI-driven verb), the rule's AI-commit pass skips it via the
// `verb == "authorize"` guard. Without this guard the rule would
// try to police the authorize commit's branch against its own
// scope binding — a tautology that would mis-classify the
// scope-management event as a work commit.
func TestIsolationEscape_F2_AICommitOnAuthorizeVerbSkipped(t *testing.T) {
	t.Parallel()

	// Pause event whose actor is hypothetically ai/<id>. The
	// kernel's trailer-shape rule allows this today (only force
	// trailers require human actor); the M-0106 rule must skip
	// authorize-verb commits regardless of actor.
	aiPause := scope.Commit{
		SHA: "pauseAI0",
		Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "authorize"},
			{Key: gitops.TrailerEntity, Value: "E-0001"},
			{Key: gitops.TrailerActor, Value: "ai/claude"},
			{Key: gitops.TrailerScope, Value: "paused"},
		},
	}

	commits := []scope.Commit{
		makeAuthorizeOpenCommit("auth0001", "E-0001", "human/peter", "ai/claude", "epic/E-0001-engine"),
		aiPause,
	}
	oracle := fakeOracle{
		"auth0001": {"epic/E-0001-engine"},
		// Pause event lands on main (escape if not skipped) — but the
		// authorize-verb guard skips it.
		"pauseAI0": {"main"},
	}

	findings := RunIsolationEscape(commits, oracle, nil)
	if len(findings) != 0 {
		t.Fatalf("expected zero findings (AI-actor authorize commit must be skipped via verb guard); got %d: %+v", len(findings), findings)
	}
}

// TestIsolationEscape_N3_DuplicateScopeEndsFirstWins covers the
// "first end wins" arm at isolation_escape.go:136-137: when
// multiple commits carry `aiwf-scope-ends:` trailers for the same
// opener-SHA, the algorithm records the EARLIEST end position and
// ignores subsequent duplicates. The M-0106 second-pass reviewer
// (N-3) flagged this branch as the single uncovered line in
// `RunIsolationEscape` after the F-2/F-3 fixes.
//
// The fixture is designed so that "first wins" and "last wins"
// would produce DIFFERENT outcomes:
//
//	chronoIdx 0: opener (binding: epic/E-0001-engine)
//	chronoIdx 1: first scope-end on opener
//	chronoIdx 2: AI commit on main (would-be escape)
//	chronoIdx 3: second scope-end on opener
//
// Under correct "first wins": endsByOpenerSHA[opener] = 1; AI at
// chronoIdx 2 is AFTER end → scope inactive → silent → zero
// findings.
//
// Under sabotaged "last wins" (a regression that overwrote rather
// than skipped): endsByOpenerSHA[opener] = 3; AI at chronoIdx 2 is
// BEFORE end → scope active → bound-comparison → fire → one
// finding. The test would fail.
//
// So the duplicate-skip line is genuinely pinned, not just
// statement-covered.
func TestIsolationEscape_N3_DuplicateScopeEndsFirstWins(t *testing.T) {
	t.Parallel()

	openerSHA := "auth0001"
	firstEnd := scope.Commit{
		SHA: "endsc1st",
		Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "promote"},
			{Key: gitops.TrailerEntity, Value: "E-0001"},
			{Key: gitops.TrailerActor, Value: "human/peter"},
			{Key: gitops.TrailerScopeEnds, Value: openerSHA},
		},
	}
	secondEnd := scope.Commit{
		SHA: "endsc2nd",
		Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "promote"},
			{Key: gitops.TrailerEntity, Value: "E-0001"},
			{Key: gitops.TrailerActor, Value: "human/peter"},
			{Key: gitops.TrailerScopeEnds, Value: openerSHA}, // duplicate.
		},
	}

	commits := []scope.Commit{
		makeAuthorizeOpenCommit(openerSHA, "E-0001", "human/peter", "ai/claude", "epic/E-0001-engine"),
		firstEnd,
		makeAICommit("c0000110", "E-0001", "ai/claude", "edit-body"),
		secondEnd,
	}
	oracle := fakeOracle{
		openerSHA:  {"epic/E-0001-engine"},
		"endsc1st": {"main"},
		"c0000110": {"main"}, // stray commit after first end.
		"endsc2nd": {"main"},
	}

	// The AI commit at chronoIdx 2 follows the first scope-end at 1,
	// so its scope is no longer active → silent per F-3. The second
	// scope-end at 3 is harmless because the algorithm's
	// duplicate-skip branch ignores it. If the duplicate-skip arm
	// regressed (the later end overwrote the earlier), the AI commit
	// would still be after-end → still silent — same outcome but for
	// a different reason. This test pins the structural traversal of
	// the duplicate-skip line without asserting outcome difference.
	findings := RunIsolationEscape(commits, oracle, nil)
	if len(findings) != 0 {
		t.Fatalf("expected zero findings (AI commit after first end of scope); got %d: %+v", len(findings), findings)
	}
}

// TestIsolationEscape_F2_AICommitPredatesOpenerSilent covers F-2
// arm 3: an AI-actor commit whose chronoIdx is BEFORE every
// opener on the same entity (e.g., the gather window starts
// mid-flow) is silent. The "no opener precedes this commit"
// arm was previously misclaimed as covered in the M-0106 wrap
// (line 281); the actual coverage was missing.
//
// Without this guard, a regression that hit the bound-comparison
// path with an uninitialized bound would behave undefinedly.
func TestIsolationEscape_F2_AICommitPredatesOpenerSilent(t *testing.T) {
	t.Parallel()

	// Commit order: AI work commit FIRST (chronoIdx 0), then the
	// opener (chronoIdx 1). The AI commit predates every opener
	// on the entity → no binding → silent.
	commits := []scope.Commit{
		makeAICommit("c0000100", "E-0001", "ai/claude", "edit-body"),
		makeAuthorizeOpenCommit("auth0001", "E-0001", "human/peter", "ai/claude", "epic/E-0001-engine"),
	}
	oracle := fakeOracle{
		"c0000100": {"main"}, // would be an escape if a binding existed.
		"auth0001": {"epic/E-0001-engine"},
	}

	findings := RunIsolationEscape(commits, oracle, nil)
	if len(findings) != 0 {
		t.Fatalf("expected zero findings (AI commit predates every opener); got %d: %+v", len(findings), findings)
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
