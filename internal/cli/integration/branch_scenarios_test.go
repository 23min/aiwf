package integration

import "testing"

// branch_scenarios_test.go — M-0159/AC-1: framework-exercise tests
// for the combinatorial real-git E2E scenario-table driver. Pins
// three representative shapes the framework must support:
//
//  1. Silent path (no finding asserted)
//  2. Fires path (specific finding code asserted)
//  3. Composed setup (multi-step fixture with a legacy override)
//
// The actual M-0106 isolation-escape path coverage (every
// behavioral AC of M-0106 as a real-git scenario) lands in AC-2;
// this test exists to lock the framework's API surface and prove
// the driver works end-to-end through the three core shapes.
//
// All three rows fail in the AC-1 red phase: RunScenarios panics
// with "not implemented" before the assertions can run. The
// GREEN phase replaces every stub in branch_scenarios_helpers_test.go
// with a real implementation.

// TestBranchScenarios_AC1_FrameworkExercise locks the M-0159/AC-1
// framework's API through three scenarios. Reads like prose: each
// Setup function is a narrative of "given X, when Y" expressed in
// kernel verbs and raw git, exactly as an operator would type at
// the shell.
//
// Three shapes pinned:
//
//   - **Silent**: AI commits on the bound branch — the rule
//     correctly does not fire (M-0106 AC-4).
//   - **Fires**: AI commits on main while bound to an epic
//     ritual — the rule fires isolation-escape (M-0106 AC-1).
//   - **Composed**: After an escape, the operator amends the
//     violating commit with human/ actor + aiwf-force — the
//     rule's actor-prefix filter falls through (M-0106 AC-8
//     legacy override).
//
// The framework supports both "silent" and "fires" assertions in
// the same Expectation struct because real scenarios need both
// shapes interchangeably; the AC-2 work that follows wires up the
// remaining M-0106 paths against the same table.
func TestBranchScenarios_AC1_FrameworkExercise(t *testing.T) {
	t.Parallel()
	RunScenarios(t, []Scenario{
		{
			Name: "AI commit on bound branch is silent (M-0106 AC-4)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				// aiwfx-start-epic step 7 pattern: authorize
				// from main with --branch naming the future
				// epic ritual. The opener lands on main with the
				// aiwf-branch: trailer pointing at the future ref.
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				// Step 8: cut the named branch from main's HEAD
				// (which is now the opener commit). The AI then
				// works on that branch — the verb's preflight
				// accepts since current ref matches the bound
				// ref.
				env.MustRunGit("checkout", "-b", "epic/E-0001-engine")
				AICommit(t, env, "E-0001", "## Goal\n\nAI-attributed body update on the bound branch.\n")
			},
			Expect: Expectation{
				NoFindingWithCode: "isolation-escape",
			},
		},
		{
			Name: "AI commit on main while bound to epic ritual fires isolation-escape (M-0106 AC-1)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				// Same step-7 pattern: opener on main, bound to
				// future epic/E-0001-engine.
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				// Real-world escape: AI subagent confused about
				// which branch it's on (G-0099 founding incident
				// shape) ran raw git on main — never cut the
				// ritual branch. AICommit-via-verb would be
				// refused by the M-0103-era preflight; the raw-
				// git path bypasses the verb entirely and lands
				// on main, which the isolation-escape rule
				// (M-0106) is the defense-in-depth for.
				SimulateAIEscape(t, env, "E-0001", "AI subagent body edit (raw git, on wrong branch)")
			},
			Expect: Expectation{
				FindingPresent: "isolation-escape",
			},
		},
		{
			Name: "Force-amend with human actor silences the escape (M-0106 AC-8 legacy override)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				// Same raw-git escape as the fires scenario.
				SimulateAIEscape(t, env, "E-0001", "AI subagent body edit (raw git, on wrong branch)")
				// Legacy override: operator amends the violating
				// commit so the actor is human/ and a force
				// reason is recorded. The rule's ai/ prefix
				// filter then skips the amended commit.
				ForceAmendHEAD(t, env, "manual sovereign override per M-0106 AC-8 legacy path")
			},
			Expect: Expectation{
				NoFindingWithCode: "isolation-escape",
			},
		},
	})
}
