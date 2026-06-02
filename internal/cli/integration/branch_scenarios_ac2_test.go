package integration

import "testing"

// branch_scenarios_ac2_test.go — M-0159/AC-2: real-git E2E coverage
// of M-0106 isolation-escape paths that AC-1's framework exercise
// doesn't already pin. Consumes AC-1's Scenario / Expectation /
// RunScenarios framework + the branch-choreography helpers; adds
// scenarios for AC-2 / AC-5 / AC-7 / AC-9 / AC-10 / AC-11 /
// AC-12 + F-3 (both arms) + F-2 predates-opener + T-6 +
// Followup-#58 (filter specificity) + UnknownBranchSilent.
//
// Out of AC-2 scope — documented decisions, NOT silently
// uncovered. Each entry names the M-0106 unit test it corresponds
// to and the structural reason E2E is impractical / impossible /
// redundant:
//
//   - AC-3 worktree-branch mismatch (M-0106 unit at
//     isolation_escape_test.go:127-175). The unit test itself is a
//     documentation pin: "the M-0106 rule does not — and by design
//     cannot — distinguish a worktree-induced escape from a
//     `git checkout main`-induced escape." Algorithmic coverage of
//     the "AI commit on the wrong branch" path comes from AC-1's
//     fires scenario. Real-git E2E from a sibling worktree fails a
//     different way: `aiwf check` runs from the main worktree
//     where HEAD lives, and `readProvenanceCommits` walks
//     `git log HEAD` (no `--all`), so the sibling worktree's
//     escape commit is unreachable from the main worktree's check.
//     The fixture cannot reach the rule.
//
//   - AC-5 paused scope as "pause suppresses" (M-0106 unit at
//     isolation_escape_test.go:355-399). The unit test's docstring
//     line 364 is honest: "The pause event does NOT change the
//     binding ... if the commit rides the bound branch, the rule
//     has no opinion about pause." The unit silence is from
//     bound-match (rides bound), NOT from any pause-state
//     suppression — the rule's algorithm has no paused-state code
//     path. AC-2 retains a scenario that mirrors the unit test
//     accurately (commits on the BOUND branch with a pause event
//     interposed; pinned silence = "pause doesn't trigger spurious
//     findings on the bound-match path"), NOT the original false
//     "rule reads paused-state and suppresses" framing.
//
//   - AC-6 cherry-pick re-author silent (M-0106 unit at
//     isolation_escape_test.go:559-588) — BLOCKED by M-0159/AC-6
//     (G-0202 cherry-pick gather-side not yet implemented). The
//     production state today has cherryPicked=nil; the rule's
//     suppression arm cannot fire. Land scenario when AC-6 closes.
//
//   - AC-6 non-cherry-pick still fires (M-0106 unit at
//     isolation_escape_test.go:590-607) — lower-bound complement
//     to AC-6 silence. Asserts that under production state
//     (cherryPicked=nil), the rule fires on the wrong-branch AI
//     commit. THIS IS ALREADY PINNED by
//     TestBranchScenarios_AC1_FrameworkExercise's "AI commit on
//     main ... fires" scenario in branch_scenarios_test.go, which
//     exercises exactly that production state. No duplicate row
//     needed.
//
//   - F-2 malformed opener variants (M-0106 units at
//     isolation_escape_test.go:799-806 and 837-848) — the verb
//     refuses commits with missing aiwf-entity: trailers; the
//     fixture requires raw-git fabrication of an
//     aiwf-verb:authorize commit with a malformed trailer set, a
//     shape no real-world workflow produces. F-2 "predates opener"
//     (a third variant at line 1020) is in AC-2 scope: it uses
//     normal commits with ordering manipulation.
//
//   - LegacyPreM0102 (M-0106 unit at
//     isolation_escape_test.go:306-332). The pre-M-0102 shape is
//     an authorize commit WITHOUT aiwf-branch: trailer. Per
//     internal/verb/authorize.go:281-347, the post-M-0102 verb
//     ALWAYS stamps the trailer when the ai/* preflight accepts
//     (implicit-from-current promotes opts.Branch to
//     opts.CurrentBranch at line 345; --branch explicit always
//     stamps). The only post-M-0102 shape without the trailer is
//     a sovereign-override --force --reason — a deliberately
//     different code path. Pre-M-0102's shape is not reproducible
//     via the current verb; E2E coverage would require raw-git
//     fabrication. Unit test pins the rule's behavior on that
//     input; AC-2 cannot add real-git coverage.
//
//   - N-3 duplicate scope-ends, NilOracleSilent, AC-13 typed code
//     — hard / impossible / compile-time-only. Unit tests
//     sufficient.
//
// Phase: RED. Setup functions call new helpers (PauseScope,
// EndScope, HumanCommit) whose green-phase bodies are panic stubs.
// Scenarios that depend on those stubs panic at runtime; scenarios
// using only AC-1 helpers pass-or-fail based on live kernel
// behavior — each is genuinely new real-git coverage of an
// M-0106 path that was previously only fake-oracle unit-tested.

// TestBranchScenarios_AC2_M0106Paths drives 12 real-git scenarios
// against `aiwf check --format=json`'s envelope. Each scenario row
// reads as a narrative: kernel verbs + raw git ops in Setup, an
// envelope-shaped assertion in Expect.
//
// The scenarios fall into three groups:
//
//  1. Behavioral fires/silent (AC-2, AC-5-revised, AC-7, AC-9,
//     AC-10, F-3 both arms, F-2 predates, T-6, UnknownBranch) —
//     the core rule's per-path coverage in real-git.
//  2. Finding-shape assertions (AC-11 severity, AC-12 hint) —
//     pinning observable envelope fields the rule promises.
//  3. Filter specificity (HumanCommit on wrong branch silent;
//     follow-up #58 from AC-1 reviewer) — mechanizes the claim
//     that the rule's actor filter is specifically `ai/`, not
//     "anyone with a role."
func TestBranchScenarios_AC2_M0106Paths(t *testing.T) {
	t.Parallel()
	RunScenarios(t, []Scenario{
		// M-0106 AC-2: AI commit on a DIFFERENT ritual branch than
		// the scope's bound branch fires. The escape commit's
		// actualBranches (from the oracle) is a ritual ref distinct
		// from the bound one, so the bound-vs-actual comparison
		// surfaces the mismatch. Distinct from AC-1's "on main"
		// shape (where the actual is the trunk) — here the operator
		// genuinely cut a ritual branch but the wrong one.
		{
			Name: "AI commit on different ritual branch fires (M-0106 AC-2)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				env.MustRunBin("add", "epic", "--title", "Other")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				env.MustRunGit("checkout", "-b", "epic/E-0002-other")
				SimulateAIEscape(t, env, "E-0001", "AI body edit on wrong ritual branch")
			},
			Expect: Expectation{FindingPresent: "isolation-escape"},
		},

		// M-0106 AC-5 (revised): a paused scope WITH the AI commit
		// riding the BOUND branch is silent. The rule's algorithm
		// has no paused-state code path — silence here is from
		// bound-match (rides bound), and the assertion pins
		// "interposing a pause event does not trigger spurious
		// findings on the bound-match path." A future buggy
		// addition like "fire on every AI commit during paused
		// scope" would break this scenario.
		{
			Name: "AI commit on bound branch with paused scope interposed is silent (M-0106 AC-5 revised — pause is a no-op for bound-match)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				env.MustRunGit("checkout", "-b", "epic/E-0001-engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				PauseScope(t, env, "E-0001", "blocked on design decision")
				// Raw-git commit on the BOUND branch — the
				// VERB path (AICommit) refuses during paused
				// scope per the M-0103-era preflight enforcement
				// of "no active scope → refuse" (paused ≠
				// active). For the rule's AC-5 silence claim to
				// be testable end-to-end, we need an AI-trailer
				// commit to actually exist in history during the
				// paused window — only raw-git bypass produces
				// it. The commit's bound (from trailer) matches
				// its actual branch (oracle index) → rule silent
				// via bound-match, with the pause event present
				// but structurally invisible.
				SimulateAIEscape(t, env, "E-0001", "raw-git AI body edit on bound branch during paused scope")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape"},
		},

		// M-0106 AC-7: a human merge that brings AI commits onto an
		// integration branch via second-parent ancestry is silent.
		// The oracle's first-parent walk doesn't include the AI
		// commits on main's path; their actualBranches still match
		// the bound ritual branch, so the rule stays silent.
		// `git merge --no-ff` keeps the merge commit explicit so
		// first-parent semantics are observable.
		{
			Name: "Human merge bringing AI commits via second-parent is silent (M-0106 AC-7)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				env.MustRunGit("checkout", "-b", "epic/E-0001-engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				AICommit(t, env, "E-0001", "## Goal\n\nAI body update on bound branch.\n")
				env.MustRunGit("checkout", "main")
				env.MustRunGit("merge", "--no-ff", "epic/E-0001-engine", "-m", "human merge epic into main")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape"},
		},

		// M-0106 AC-9: no scope opened on the entity. The rule has
		// no opener to bind against → silent (no policing).
		// Important to pin: a regression that fired on any AI
		// commit (regardless of scope) would surface here.
		{
			Name: "AI commit with no scope opened is silent (M-0106 AC-9)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				// Deliberately skip authorize — no scope opened.
				SimulateAIEscape(t, env, "E-0001", "AI body edit with no active scope")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape"},
		},

		// M-0106 AC-10: N violating AI commits produce EXACTLY N
		// isolation-escape findings — one per commit, no aggregate,
		// no duplicates. Pinned by FindingCount=3 in the
		// Expectation; the per-commit cardinality is the AC's
		// load-bearing claim.
		{
			Name: "Three violating AI commits produce three isolation-escape findings (M-0106 AC-10)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				SimulateAIEscape(t, env, "E-0001", "AI body edit 1 of 3")
				SimulateAIEscape(t, env, "E-0001", "AI body edit 2 of 3")
				SimulateAIEscape(t, env, "E-0001", "AI body edit 3 of 3")
			},
			Expect: Expectation{
				FindingPresent: "isolation-escape",
				FindingCount:   3,
			},
		},

		// M-0106 AC-11: isolation-escape findings are SeverityWarning
		// (not SeverityError). Pinned via the FindingSeverity field
		// on Expectation. A future tightening (D-NNN flipping to
		// error after the false-positive rate is known) would need
		// to be deliberate; this assertion is the single place that
		// breaks at that decision.
		{
			Name: "isolation-escape finding has severity warning (M-0106 AC-11)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				SimulateAIEscape(t, env, "E-0001", "AI body edit fires isolation-escape")
			},
			Expect: Expectation{
				FindingPresent:  "isolation-escape",
				FindingSeverity: "warning",
			},
		},

		// M-0106 AC-12: the hint text names both sovereign-override
		// paths (cherry-pick and force-amend) so an operator who
		// hits the finding sees a single place that lists the
		// legitimate ways out. The substrings "cherry-pick" and
		// "force" must BOTH appear in the same hint — pinned via
		// FindingHintContainsAll. The unit test
		// TestIsolationEscape_AC12_HintTextNamesAllOverridePaths
		// pins the strict anchor set (originally 4 markers;
		// M-0159/AC-9 added `aiwf acknowledge-illegal` for 5 total);
		// this E2E scenario is intentionally weaker — pins
		// presence-in-envelope, not exact wording. The strict text
		// remains pinned by the unit.
		{
			Name: "isolation-escape hint names cherry-pick AND force override paths (M-0106 AC-12)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				SimulateAIEscape(t, env, "E-0001", "AI body edit fires for hint assertion")
			},
			Expect: Expectation{
				FindingPresent:         "isolation-escape",
				FindingHintContainsAll: []string{"cherry-pick", "force"},
			},
		},

		// M-0106 F-3 negative arm: an AI commit landing AFTER the
		// scope's terminal-end event (the `aiwf-scope-ends:
		// <opener-sha>` trailer on the parent's terminal-promote
		// commit) is silent. The rule tracks scope-end positions
		// per opener and treats the scope as inactive at any
		// chronoIdx beyond the end. Pins the F-3 retrospective fix
		// from M-0106 cycle 3.
		{
			Name: "AI commit after scope-ended is silent (M-0106 F-3 negative arm)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				EndScope(t, env, "E-0001")
				SimulateAIEscape(t, env, "E-0001", "AI body edit after scope ended")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape"},
		},

		// M-0106 F-3 positive arm: an AI commit landing BEFORE the
		// scope's terminal-end event still fires — the rule must
		// not over-eagerly suppress every AI commit. The symmetric
		// guard against an F-3 fix that silently suppresses
		// regardless of timing.
		{
			Name: "AI commit before scope-ended still fires (M-0106 F-3 positive arm)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				// Escape lands BEFORE scope-end — scope is still
				// active at the escape's chronoIdx.
				SimulateAIEscape(t, env, "E-0001", "AI body edit before scope ended")
				EndScope(t, env, "E-0001")
			},
			Expect: Expectation{FindingPresent: "isolation-escape"},
		},

		// M-0106 F-2 predates-opener silent: an AI commit whose
		// chronoIdx is BEFORE every opener on the same entity is
		// silent. The rule's per-commit walk finds no preceding
		// opener at the escape's chronoIdx → AC-9 path. Distinct
		// from AC-9 (no-scope-ever-opened) because the scope IS
		// eventually opened; the rule must still respect commit
		// ordering.
		{
			Name: "AI commit predating opener is silent (M-0106 F-2 ordering)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				// Escape FIRST (no opener exists yet at this
				// chronoIdx)...
				SimulateAIEscape(t, env, "E-0001", "AI body edit predating any opener")
				// ...then authorize lands AFTER the escape.
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape"},
		},

		// M-0106 T-6: the bound branch in the opener's aiwf-branch:
		// trailer names a ref that doesn't exist in the oracle's
		// index (operator typo, stale ref, future ritual that was
		// never cut). The AI commit lands on a real ritual branch;
		// the rule fires because actualBranches doesn't include the
		// typo'd bound ref. Pins the operator-typo surfacing path.
		//
		// Note: the verb may accept --branch values that don't yet
		// resolve (M-0104/AC-4 carve-out for the step-7 future-ref
		// pattern); the trailer is stamped even when the ref doesn't
		// resolve at authorize-time.
		{
			Name: "Bound branch absent from oracle (operator typo) fires (M-0106 T-6)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-9999-typo")
				env.MustRunGit("checkout", "-b", "epic/E-0001-engine")
				SimulateAIEscape(t, env, "E-0001", "AI body edit on real ritual branch; scope bound to typo'd ref")
			},
			Expect: Expectation{FindingPresent: "isolation-escape"},
		},

		// Follow-up #58 from AC-1 reviewer: the rule's actor-prefix
		// filter is specifically `ai/`, not "anyone with a role."
		// A HUMAN-actor commit on the wrong branch is silent — the
		// filter skips it even though every other condition (bound
		// branch mismatch, active scope on entity) is met.
		// Without this scenario, mutating the filter from `ai/` to
		// `human/` or removing the filter entirely would not be
		// caught by any other AC-2 row.
		{
			Name: "Human-actor commit on wrong branch is silent (M-0106 filter specificity #58)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				HumanCommit(t, env, "E-0001", "## Goal\n\nHuman-attributed body update on wrong branch.\n")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape"},
		},

		// M-0106 UnknownBranchSilent: AI commit lands on a branch
		// the oracle doesn't classify (non-ritual shape like
		// `feature/foo`). The rule treats unknown-branch commits as
		// silent — the kernel cannot confidently police what it
		// can't classify.
		{
			Name: "AI commit on non-ritual branch is silent (oracle unknown) (M-0106 UnknownBranch)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				env.MustRunGit("checkout", "-b", "feature/random-experiment")
				SimulateAIEscape(t, env, "E-0001", "AI body edit on non-ritual branch")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape"},
		},
	})
}
