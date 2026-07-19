package integration

import (
	"testing"
)

// promote_wrong_branch_scenarios_test.go — M-0161/AC-8
// (G-0209 partial-close) real-git E2E scenarios for the
// new promote-on-wrong-branch finding.
//
// AC-8 contract (per body):
//
//   - `aiwf promote E-NNNN active` (epic activation) must
//     land on trunk (per Config.TrunkBranchShortName()).
//   - `aiwf promote M-NNNN in_progress` (milestone activation)
//     must land on the parent epic's ritual branch
//     (epic/E-XXXX-<slug>).
//   - Non-activating promotes are out of the rule's domain
//     (active → done, in_progress → done, ADR proposed →
//     accepted, etc.) — silent regardless of branch.
//
// Sovereign overrides shared with AC-5 + AC-6:
//   - `aiwf acknowledge illegal <sha>` silences post-hoc.
//   - per-commit `aiwf-force: "..."` trailer silences via
//     `--force --reason` on the promote.
//
// RED state: the rule doesn't exist; all "wrong branch"
// scenarios fail to fire today.

// TestPromoteOnWrongBranch_AC8_Matrix drives the 9-cell AC-8
// matrix.
func TestPromoteOnWrongBranch_AC8_Matrix(t *testing.T) {
	t.Parallel()
	RunScenarios(t, []Scenario{
		// ----- Baseline cells -----

		// Cell 1: epic activating promote on trunk → silent.
		{
			CellID: "branch-cell-m0161-ac8-c1",
			Name:   "AC-8 cell 1: epic active on trunk → silent (baseline)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				env.MustRunBin("promote", "E-0001", "active")
			},
			Expect: Expectation{NoFindingWithCode: "promote-on-wrong-branch"},
		},

		// Cell 2: milestone activating promote on parent epic → silent.
		{
			CellID: "branch-cell-m0161-ac8-c2",
			Name:   "AC-8 cell 2: milestone in_progress on parent epic branch → silent (baseline)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				env.MustRunBin("promote", "E-0001", "active")
				env.MustRunBin("add", "milestone", "--epic", "E-0001", "--tdd", "none", "--title", "Bootstrap")
				// M-0268/AC-1: draft -> in_progress now refuses a
				// zero-AC milestone; seed one so this cell exercises
				// the branch guard, not the AC-completeness guard.
				env.MustRunBin("add", "ac", "M-0001", "--title", "Boots up")
				// Cut the epic branch and switch to it before promoting milestone.
				env.MustRunGit("checkout", "-b", "epic/E-0001-engine", "main")
				env.MustRunBin("promote", "M-0001", "in_progress")
			},
			Expect: Expectation{NoFindingWithCode: "promote-on-wrong-branch"},
		},

		// ----- Firing cells -----

		// Cell 3: epic activating promote on a ritual branch → fires.
		{
			CellID: "branch-cell-m0161-ac8-c3",
			Name:   "AC-8 cell 3: epic active on epic/E-X branch → fires",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				// Cut a ritual branch first, then promote from there — wrong order per ADR-0010.
				env.MustRunGit("checkout", "-b", "epic/E-0001-engine", "main")
				// G-0269: the pre-commit branch guard now refuses this
				// wrong-branch activation outright; --force constructs
				// the "already landed wrong" fixture this check-rule
				// cell needs to test detection against, then
				// StripForceTrailer removes the resulting aiwf-force
				// trailer (which would itself suppress this check
				// rule's finding) — the commit ends up shaped exactly
				// like a genuine, un-overridden race.
				env.MustRunBin("promote", "E-0001", "active", "--force", "--reason", "AC-8 fixture: construct a wrong-branch commit for the check rule to detect")
				StripForceTrailer(t, env)
			},
			Expect: Expectation{
				FindingPresent:         "promote-on-wrong-branch",
				FindingSeverity:        "warning",
				FindingHintContainsAll: []string{"aiwf acknowledge illegal"},
			},
		},

		// Cell 4: milestone activating promote on a milestone/ branch → fires.
		{
			CellID: "branch-cell-m0161-ac8-c4",
			Name:   "AC-8 cell 4: milestone in_progress on milestone/M-Y branch → fires",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				env.MustRunBin("promote", "E-0001", "active")
				env.MustRunBin("add", "milestone", "--epic", "E-0001", "--tdd", "none", "--title", "Bootstrap")
				// Land milestone in_progress on a sibling milestone
				// branch — wrong per ADR-0010 (should be parent epic).
				// --force + StripForceTrailer: see cell 3's comment.
				env.MustRunGit("checkout", "-b", "milestone/M-9999-other", "main")
				env.MustRunBin("promote", "M-0001", "in_progress", "--force", "--reason", "AC-8 fixture: construct a wrong-branch commit for the check rule to detect")
				StripForceTrailer(t, env)
			},
			Expect: Expectation{FindingPresent: "promote-on-wrong-branch"},
		},

		// Cell 5: milestone activating promote on trunk (skipping parent) → fires.
		{
			CellID: "branch-cell-m0161-ac8-c5",
			Name:   "AC-8 cell 5: milestone in_progress on trunk (skipped parent epic) → fires",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				env.MustRunBin("promote", "E-0001", "active")
				env.MustRunBin("add", "milestone", "--epic", "E-0001", "--tdd", "none", "--title", "Bootstrap")
				// HEAD stays on main; promote milestone here without
				// cutting epic branch first — wrong per ADR-0010.
				// --force + StripForceTrailer: see cell 3's comment.
				env.MustRunBin("promote", "M-0001", "in_progress", "--force", "--reason", "AC-8 fixture: construct a wrong-branch commit for the check rule to detect")
				StripForceTrailer(t, env)
			},
			Expect: Expectation{
				FindingPresent:         "promote-on-wrong-branch",
				FindingHintContainsAll: []string{"epic/E-0001-engine"}, // hint names the expected parent epic branch
			},
		},

		// ----- Out-of-domain cell -----

		// Cell 7: non-activating promote on wrong branch → silent.
		{
			CellID: "branch-cell-m0161-ac8-c6",
			Name:   "AC-8 cell 7: non-activating promote (epic active → done) on wrong branch → silent (out of domain)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				env.MustRunBin("promote", "E-0001", "active")
				env.MustRunGit("checkout", "-b", "epic/E-0001-engine", "main")
				// active → done is NOT an activating transition;
				// rule should stay silent regardless of branch.
				env.MustRunBin("promote", "E-0001", "done")
			},
			Expect: Expectation{NoFindingWithCode: "promote-on-wrong-branch"},
		},

		// ----- Sovereign override cells -----

		// Cell 8: wrong-branch promote + ack → silent.
		{
			CellID: "branch-cell-m0161-ac8-c7",
			Name:   "AC-8 cell 8: wrong-branch promote + aiwf acknowledge illegal → silent",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				env.MustRunGit("checkout", "-b", "epic/E-0001-engine", "main")
				// --force + StripForceTrailer: see cell 3's comment —
				// this cell tests the acknowledge-illegal override
				// specifically, so the constructed commit must carry
				// no aiwf-force trailer of its own.
				env.MustRunBin("promote", "E-0001", "active", "--force", "--reason", "AC-8 fixture: construct a wrong-branch commit for the check rule to detect")
				badSHA := StripForceTrailer(t, env)
				AcknowledgeIllegal(t, env, badSHA, "AC-8 fixture: sovereign override of wrong-branch promote")
			},
			Expect: Expectation{NoFindingWithCode: "promote-on-wrong-branch"},
		},

		// Cell 9: wrong-branch promote with aiwf-force trailer → silent (per-commit override).
		{
			CellID: "branch-cell-m0161-ac8-c8",
			Name:   "AC-8 cell 9: wrong-branch promote --force --reason → silent (per-commit override)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				env.MustRunGit("checkout", "-b", "epic/E-0001-engine", "main")
				env.MustRunBin("promote", "E-0001", "active", "--force", "--reason", "AC-8 sovereign per-commit override")
			},
			Expect: Expectation{NoFindingWithCode: "promote-on-wrong-branch"},
		},

		// Cell 6 (detached HEAD): composes with AC-7. The
		// promote verb may itself refuse on detached HEAD or
		// land an aiwf-force commit; either way the check-time
		// rule's behavior depends on the verb's outcome. Cell
		// deferred — the verb-side interaction is its own
		// surface (not core to AC-8's load-bearing claim).
		// The AC-9 catalog refactor can revisit if needed.
	})
}
