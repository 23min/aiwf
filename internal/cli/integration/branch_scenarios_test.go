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
				env.MustRunBin("add", "epic", "--title", "Engine")
				env.MustRunGit("checkout", "-b", "epic/E-0001-engine")
				OpenBoundScope(t, env, "E-0001")
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
				env.MustRunGit("checkout", "-b", "epic/E-0001-engine")
				OpenBoundScope(t, env, "E-0001")
				// Escape: switch back to main, then have the AI
				// commit work on the still-scoped entity.
				env.MustRunGit("checkout", "main")
				AICommit(t, env, "E-0001", "## Goal\n\nAI-attributed body update on the WRONG branch.\n")
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
				env.MustRunGit("checkout", "-b", "epic/E-0001-engine")
				OpenBoundScope(t, env, "E-0001")
				env.MustRunGit("checkout", "main")
				AICommit(t, env, "E-0001", "## Goal\n\nAI body update; will be sovereign-amended.\n")
				// Legacy override: amend the violating commit so
				// the actor is human/ and a force-reason is
				// recorded. The rule's ai/ prefix filter skips it.
				ForceAmendHEAD(t, env, "manual sovereign override per M-0106 AC-8 legacy path")
			},
			Expect: Expectation{
				NoFindingWithCode: "isolation-escape",
			},
		},
	})
}
