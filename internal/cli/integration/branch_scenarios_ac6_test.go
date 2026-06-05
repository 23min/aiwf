package integration

import (
	"testing"
)

// branch_scenarios_ac6_test.go — M-0159/AC-6: real-git E2E
// coverage of the cherry-pick gather-side. Closes G-0202 — the
// parked gather-side derivation that left
// `internal/cli/check/provenance.go:67` passing `nil` for
// cherryPicked. With nil, the M-0106 isolation-escape rule's
// suppression arm at internal/check/isolation_escape.go:258
// could not fire end-to-end; sovereign human re-authors via
// `git cherry-pick -x` of an AI commit landed on a non-bound
// branch were spuriously flagged as escapes.
//
// AC-6's load-bearing claim per the milestone spec body:
//
//	"Cherry-pick gather-side implemented in the CLI: real
//	 (cherry picked from commit <sha>) markers in commit bodies
//	 populate the cherryPicked map. Real-git E2E: git cherry-pick
//	 -x of an isolation-escape commit → check silent."
//
// The rule's docstring at internal/check/isolation_escape.go:67-78
// pins the both-signals-required contract: the gather adds a SHA
// to the cherryPicked map iff
//
//	(a) the commit body carries the `(cherry picked from commit <sha>)`
//	    marker that `git cherry-pick -x` writes by default, AND
//	(b) the commit's committer email differs from its author email
//	    (i.e., git author/committer identity gap — what
//	    `git cherry-pick -x` produces when a different identity
//	    re-authors the original)
//
// Both together are what a sovereign human re-author actually
// looks like. Either alone is insufficient: a fabricated marker
// (no real cherry-pick) lacks the gap; an amended commit (gap
// without -x) lacks the marker. The negative-control scenarios
// below exercise each insufficient-signal arm.
//
// Scenario groups:
//
//  1. Silencing happy path — AI commits on bound branch B (legit,
//     does not fire); human cherry-picks `-x` to non-bound branch
//     A with distinct committer email → cherry-pick silent on A.
//     The AC's core claim end-to-end.
//
//  2. Marker-only negative control — same-identity cherry-pick
//     `-x` (committer == author; e.g., the AI cherry-picks its
//     own commit). The marker is present but the gap is absent.
//     The cherry-pick STILL fires under the both-signals contract.
//
//  3. Gap-only negative control — cherry-pick WITHOUT `-x`
//     (distinct committer, no marker emitted). The gap is present
//     but the marker is absent. The cherry-pick STILL fires.
//
//  4. Baseline positive control — AI commit on non-bound branch
//     with no cherry-pick at all. The rule's primary fire path
//     reaches the fixture; pins that the gather→consumer wiring
//     does not over-exempt. (M-0106's own AC-1 scenarios cover
//     this shape in branch_scenarios_test.go; included here for
//     self-contained discrimination evidence inside the AC-6
//     file.)

// TestBranchScenarios_AC6_CherryPickGatherSide drives the four
// scenario groups described in the file header. Uses the real
// `aiwf check --format=json` against fixtures fabricated via the
// CherryPick helper (which composes `git cherry-pick [-x]` with
// optional `-c user.email=` committer override).
//
// RUNTIME-RED today: scenario 1 (silencing happy path) FAILS
// against current production because provenance.go:67 passes
// `nil` for cherryPicked; the rule's suppression arm cannot fire;
// the cherry-pick is spuriously flagged. After GREEN lands the
// gather-side `WalkCherryPicks` and wires it through, scenario 1
// passes. Scenarios 2 + 3 (negative controls) and 4 (baseline)
// pass in BOTH states — RED and GREEN — because their fixtures
// don't depend on the suppression arm.
func TestBranchScenarios_AC6_CherryPickGatherSide(t *testing.T) {
	t.Parallel()
	RunScenarios(t, []Scenario{
		// Fixture topology pattern (used by all four scenarios
		// below): the bound scope's opener is created on main
		// FIRST (the aiwfx-start-epic step-7 pattern OpenBoundScope
		// already supports), THEN the ritual branch is cut from
		// main. This keeps the opener reachable from BOTH main
		// AND epic/E-0001-engine — load-bearing because when the
		// scenarios cherry-pick from the ritual branch back to
		// main, the cherry-pick commit's preserved
		// aiwf-authorized-by: trailer must resolve to a commit
		// reachable from HEAD when aiwf check runs (otherwise
		// provenance-authorization-missing fires AND the
		// isolation-escape rule cannot determine the bound branch
		// to compare against, silently masking the AC's claim).
		//
		// Topology:
		//
		//   main:                 A → B (epic-add) → C (scope-open)
		//                                                     ↘
		//   epic/E-0001-engine:                                 → D (AI commit)
		//
		//   after cherry-pick:    A → B → C → E (cherry-pick of D)  (main)
		//                                  ↘
		//                                   → D                     (epic/E-0001-engine)
		//
		// Reachable from main's HEAD (E): E, C, B, A. Scope opener
		// C reachable. Cherry-pick E's authorized-by → C resolves.

		// AC-6 Group 1: silencing happy path. RED today, GREEN
		// after gather-side WalkCherryPicks lands and provenance.go
		// passes the real map instead of nil.
		{
			CellID: "branch-cell-m0159-ac6-c1",
			Name:   "AI commit cherry-picked -x by human (distinct committer) to non-bound branch is silent (M-0159/AC-6: G-0202 happy path)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				// Open the bound scope on main FIRST so the opener
				// commit (C) lives on main's history — reachable
				// from any branch cut from main below.
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				// Now cut the ritual branch from main; HEAD = C.
				env.MustRunGit("checkout", "-b", "epic/E-0001-engine")
				// AI commits on the bound branch (D). Legit —
				// doesn't fire the rule on the original commit.
				// Its git author identity (preserved by cherry-pick
				// below) is the env's default "peter@example.com".
				aiSHA := AICommit(t, env, "E-0001", "AI work on epic body — original on bound branch")
				// Switch to main so the cherry-pick lands on a
				// non-bound branch.
				env.MustRunGit("checkout", "main")
				// Human cherry-picks -x with a distinct committer
				// identity. Result on main: git author preserved
				// (peter@example.com), git committer = the human
				// override (gap present), body marker present.
				// Both signals → cherryPicked map should contain
				// the cherry-pick SHA → suppressed.
				_ = CherryPick(t, env, aiSHA,
					"human-cherry-picker@example.com", "Human Cherry Picker",
					true /* withMarker */)
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape"},
		},

		// AC-6 Group 2: marker-only negative control. The marker
		// is present (`-x` was used) but the committer equals the
		// author (no gap). Under the both-signals contract, this
		// commit is NOT in the cherryPicked map; the rule fires.
		// A regression that suppressed on marker alone (skipping
		// the gap check) would spuriously pass scenario 1 AND
		// silently over-suppress here.
		{
			CellID: "branch-cell-m0159-ac6-c2",
			Name:   "Cherry-pick -x by same identity (committer == author, no gap) still fires (M-0159/AC-6: marker-only negative)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				env.MustRunGit("checkout", "-b", "epic/E-0001-engine")
				aiSHA := AICommit(t, env, "E-0001", "AI work on epic body — original on bound branch")
				env.MustRunGit("checkout", "main")
				// Empty committerEmail → cherry-pick uses the env's
				// default user.email (peter@example.com). Git
				// author preserved (also peter@example.com).
				// committer == author → no gap → both-signals
				// contract requires BOTH; this commit does NOT
				// qualify even with the marker.
				_ = CherryPick(t, env, aiSHA,
					"", "",
					true /* withMarker */)
			},
			Expect: Expectation{
				FindingPresent:  "isolation-escape",
				FindingSeverity: "warning",
			},
		},

		// AC-6 Group 3: gap-only negative control. The committer
		// identity is distinct (gap present) but `-x` was NOT used
		// so git emitted no body marker. Under the both-signals
		// contract, this commit is NOT in the cherryPicked map; the
		// rule fires. A regression that suppressed on gap alone
		// (skipping the marker check) would spuriously pass scenario
		// 1 AND silently over-suppress here.
		{
			CellID: "branch-cell-m0159-ac6-c3",
			Name:   "Cherry-pick WITHOUT -x by distinct committer (gap, no marker) still fires (M-0159/AC-6: gap-only negative)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				env.MustRunGit("checkout", "-b", "epic/E-0001-engine")
				aiSHA := AICommit(t, env, "E-0001", "AI work on epic body — original on bound branch")
				env.MustRunGit("checkout", "main")
				// Distinct committerEmail → gap present (set via
				// GIT_COMMITTER_EMAIL env var inside the helper,
				// which overrides TestMain's process-wide default).
				// withMarker false → no `(cherry picked from commit
				// <sha>)` line in the body. The gap alone is
				// insufficient.
				_ = CherryPick(t, env, aiSHA,
					"human-cherry-picker@example.com", "Human Cherry Picker",
					false /* withMarker */)
			},
			Expect: Expectation{
				FindingPresent:  "isolation-escape",
				FindingSeverity: "warning",
			},
		},

		// AC-6 Group 4: baseline positive control. AI commit on
		// non-bound branch with no cherry-pick at all. Pins that
		// the rule's primary fire path reaches the fixture; pins
		// that a gather→consumer regression which spuriously
		// exempted EVERY commit (e.g., gather returned a map
		// containing every SHA in history) would surface here.
		// M-0106's own AC-1 scenarios cover this shape elsewhere;
		// the duplication here keeps the AC-6 file self-contained.
		//
		// HEAD stays on main throughout — no branch switch — so
		// the opener (on main) is trivially reachable; the AI
		// escape commit lands on main, and main != the bound
		// branch name "epic/E-0001-engine" → isolation-escape
		// fires.
		{
			CellID: "branch-cell-m0159-ac6-c4",
			Name:   "AI commit on non-bound branch without cherry-pick fires (M-0159/AC-6: baseline positive control)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				SimulateAIEscape(t, env, "E-0001",
					"AI body edit escaping the bound branch — no cherry-pick involved")
			},
			Expect: Expectation{
				FindingPresent:  "isolation-escape",
				FindingSeverity: "warning",
			},
		},
	})
}
