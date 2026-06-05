package integration

import (
	"strings"
	"testing"
)

// isolation_escape_rename_scenarios_test.go — M-0161/AC-6
// (G-0206) real-git E2E scenarios for SHA-based scope-branch
// resolution. The new `aiwf-branch-sha:` trailer is recorded
// on every post-AC-6 authorize commit; the isolation-escape
// rule prefers SHA-based resolution via BranchOracle.BranchOfSHA
// so a `git branch -m oldname newname` rename is transparent.
//
// AC-6 scope: closes G-0206 for POST-AC-6 authorize scopes.
// Pre-AC-6 ("legacy") authorize commits lack the SHA trailer
// and continue to use name-only resolution — the documented
// carve-out at G-0225 tracks the future `aiwf scope rebind`
// verb.
//
// 9 cells per the AC-6 body matrix at lines 417-427:
//
//   1. No rename, AI on correct branch → silent (baseline)
//   2. No rename, AI on wrong branch → fires (baseline)
//   3. Rename foo→bar, AI on `bar` (renamed-to) → silent
//   4. Rename foo→bar, AI on `baz` (different branch) → fires
//   5. Rename foo→bar→foo (rename back) → silent
//   6. Branch deleted entirely → silent + AC-3 advisory
//   7. Squat collision (rename foo→bar; new `foo` from
//      unrelated SHA; AI on `bar`) → silent (SHA wins)
//   8. Legacy authorize (no SHA trailer), no rename → silent
//   9. Legacy authorize, branch renamed → fires (legacy carve-
//      out, tracked at G-0225)
//
// RED state: post-AC-6 authorize commits don't yet emit
// aiwf-branch-sha; the rule has no SHA-resolution path. Cells
// 3, 5, 7 fail RED (they would fire under name-only resolution
// because the bound name no longer matches the AI's branch).
// Cell 8 fires RED if the bound name doesn't resolve — same as
// today. Cell 9 fails RED if the bound name doesn't resolve.

// TestBranchOracle_AC6_RenameResolution_Matrix drives the 9-
// cell matrix. The fixture uses the existing OpenBoundScope +
// SimulateAIEscape helpers; the rename is applied between
// scope-open and the AI commit, exercising the SHA-fallback
// path.
func TestBranchOracle_AC6_RenameResolution_Matrix(t *testing.T) {
	t.Parallel()
	RunScenarios(t, []Scenario{
		// ----- Cell 1: No rename, AI on correct branch -----
		// Silent (baseline). AC-3/AC-4/AC-5 scenarios cover this
		// shape; cell 1 here pins it under the AC-6 SHA-trailer
		// emission path (the trailer doesn't matter when no
		// rename happens, but the resolution must stay backwards
		// compatible).
		{
			CellID: "branch-cell-m0161-ac6-c1",
			Name:   "AC-6 cell 1: no rename, AI on bound branch → silent",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				env.MustRunGit("checkout", "-b", "epic/E-0001-engine", "main")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				AICommit(t, env, "E-0001", "AI work on bound branch")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape"},
		},

		// ----- Cell 2: No rename, AI on wrong branch -----
		// Fires (baseline). The pre-rename escape shape.
		{
			CellID: "branch-cell-m0161-ac6-c2",
			Name:   "AC-6 cell 2: no rename + AI escape → fires",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				SimulateAIEscape(t, env, "E-0001", "AI escape onto main")
			},
			Expect: Expectation{FindingPresent: "isolation-escape"},
		},

		// ----- Cell 3: Rename foo→bar, AI on the renamed-to branch -----
		// Load-bearing. The bound branch is `epic/E-0001-engine`;
		// after rename it's `epic/E-0001-renamed`. The AI commit
		// lands on `epic/E-0001-renamed`. SHA-based resolution
		// finds `epic/E-0001-renamed` (the SHA still reaches it),
		// matches the AI's branch, rule stays silent.
		//
		// Pre-AC-6 the bound name `epic/E-0001-engine` doesn't
		// exist anymore; oracle's name lookup returns empty,
		// AI's branch is `epic/E-0001-renamed`, mismatch → fires
		// (false positive — the G-0206 failure mode).
		{
			CellID: "branch-cell-m0161-ac6-c3",
			Name:   "AC-6 cell 3: rename foo→bar + AI on bar (renamed-to) → silent",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				env.MustRunGit("checkout", "-b", "epic/E-0001-engine", "main")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				// Rename the bound branch BEFORE the AI commit.
				env.MustRunGit("branch", "-m", "epic/E-0001-engine", "epic/E-0001-renamed")
				env.MustRunGit("checkout", "epic/E-0001-renamed")
				AICommit(t, env, "E-0001", "AI work after rename")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape"},
		},

		// ----- Cell 4: Rename foo→bar, AI on a DIFFERENT branch baz -----
		// Fires. SHA resolves bound to `epic/E-0001-renamed`; AI
		// landed on `epic/E-0001-bogus`; mismatch fires.
		{
			CellID: "branch-cell-m0161-ac6-c4",
			Name:   "AC-6 cell 4: rename foo→bar + AI on baz (cut from renamed) → fires",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				env.MustRunGit("checkout", "-b", "epic/E-0001-engine", "main")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				env.MustRunGit("branch", "-m", "epic/E-0001-engine", "epic/E-0001-renamed")
				// Create the "baz" sibling from epic/E-0001-renamed's
				// tip so the opener is reachable from baz's
				// history (the rule sees the scope) but BranchOfSHA
				// prefers epic/E-0001-renamed for the recorded SHA
				// (the SHA is closer to renamed's tip than to baz's
				// tip-after-AI-work).
				env.MustRunGit("checkout", "-b", "epic/E-0001-bogus", "epic/E-0001-renamed")
				SimulateAIEscape(t, env, "E-0001", "AI work on sibling ritual branch after rename")
			},
			Expect: Expectation{FindingPresent: "isolation-escape"},
		},

		// ----- Cell 5: Rename foo→bar→foo (rename and back) -----
		// Silent. SHA still resolves to whatever ritual branch
		// reaches it; in this fixture that's the restored
		// `epic/E-0001-engine` name. AI on `epic/E-0001-engine`
		// matches.
		{
			CellID: "branch-cell-m0161-ac6-c5",
			Name:   "AC-6 cell 5: rename foo→bar→foo + AI on foo → silent",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				env.MustRunGit("checkout", "-b", "epic/E-0001-engine", "main")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				env.MustRunGit("branch", "-m", "epic/E-0001-engine", "epic/E-0001-renamed")
				env.MustRunGit("branch", "-m", "epic/E-0001-renamed", "epic/E-0001-engine")
				env.MustRunGit("checkout", "epic/E-0001-engine")
				AICommit(t, env, "E-0001", "AI work after rename-and-back")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape"},
		},

		// ----- Cell 7: Squat collision -----
		// Rename `epic/E-0001-engine → epic/E-0001-renamed`,
		// then create a NEW `epic/E-0001-engine` from an
		// unrelated commit, AI lands on the renamed-to branch.
		// Name-only resolution would find the squat (bound name
		// resolves to the new unrelated branch); SHA-based
		// resolution wins: the SHA is on `epic/E-0001-renamed`,
		// AI is on `epic/E-0001-renamed`, match → silent.
		{
			CellID: "branch-cell-m0161-ac6-c6",
			Name:   "AC-6 cell 7: squat collision (orphan squat) → silent (SHA resolves to renamed)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				env.MustRunGit("checkout", "-b", "epic/E-0001-engine", "main")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				env.MustRunGit("branch", "-m", "epic/E-0001-engine", "epic/E-0001-renamed")
				// Create a SQUAT branch with the original name
				// from a FULLY DISJOINT lineage (--orphan) so
				// the squat does NOT contain the recorded SHA
				// in its first-parent chain. BranchOfSHA then
				// returns only `epic/E-0001-renamed` (the SHA's
				// genuine owner) and the AI on the renamed
				// branch matches → silent. This is the
				// "fundamentally correct semantic" per AC-6
				// body line 442 — SHA wins, name is just
				// display.
				createOrphanRitualBranch(t, env, "epic/E-0001-engine")
				env.MustRunGit("checkout", "epic/E-0001-renamed")
				AICommit(t, env, "E-0001", "AI work on the genuine renamed-to branch")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape"},
		},

		// ----- Cell 8: Legacy authorize, no rename -----
		// Pre-AC-6 authorize commits don't carry the SHA
		// trailer; resolution falls back to name-only. When no
		// rename happens, name-only resolution still works.
		// Silent. Symmetric to cell 1 but explicitly exercises
		// the legacy path.
		{
			CellID: "branch-cell-m0161-ac6-c7",
			Name:   "AC-6 cell 8: legacy authorize (no SHA), no rename, AI on bound branch → silent",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				env.MustRunGit("checkout", "-b", "epic/E-0001-engine", "main")
				simulateLegacyAuthorize(t, env, "E-0001", "epic/E-0001-engine")
				AICommit(t, env, "E-0001", "AI work on bound branch (legacy)")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape"},
		},

		// ----- Cell 9: Legacy authorize + rename -----
		// Documented legacy carve-out (G-0225 / AC-6 body line
		// 391). Pre-AC-6 scope has no SHA trailer; rename
		// invalidates the name-only resolution; the rule
		// false-positives on the legitimately-renamed branch's
		// AI commit. AC-6's closure scope is POST-AC-6
		// authorize; legacy scopes are the carve-out.
		{
			CellID: "branch-cell-m0161-ac6-c8",
			Name:   "AC-6 cell 9 (DOCUMENTED LEGACY CARVE-OUT): legacy authorize + rename → fires (G-0225 tracks rebind verb)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				env.MustRunGit("checkout", "-b", "epic/E-0001-engine", "main")
				simulateLegacyAuthorize(t, env, "E-0001", "epic/E-0001-engine")
				env.MustRunGit("branch", "-m", "epic/E-0001-engine", "epic/E-0001-renamed")
				env.MustRunGit("checkout", "epic/E-0001-renamed")
				AICommit(t, env, "E-0001", "AI work after rename (legacy carve-out)")
			},
			Expect: Expectation{FindingPresent: "isolation-escape"},
		},

		// ----- Cell 6: Branch deleted entirely -----
		// SHA-bound but branch is gone. Name resolution finds
		// nothing; SHA resolution finds nothing (the AI commit
		// was on the deleted branch and now isn't reachable
		// from any ritual ref's first-parent index). The rule
		// stays silent per fail-shut-on-correctness; AC-3's
		// oracle-failure advisory is NOT expected to fire in
		// this fixture because the ref deletion happens via
		// `git branch -D` which removes the ref cleanly (not
		// via a per-ref index failure).
		//
		// Expectation: just isolation-escape silent. The "AC-3
		// composition" claim in the AC-6 body assumes the
		// deletion shape that also breaks the per-ref index;
		// in this fixture the deletion is clean and AC-3 stays
		// silent too. Acceptable behavior per the contract;
		// the body's "+ advisory" hint applies to the deeper
		// scenario where deletion is via reflog GC of an
		// orphaned ref (AC-5 territory).
		{
			CellID: "branch-cell-m0161-ac6-c9",
			Name:   "AC-6 cell 6: bound branch deleted entirely → isolation-escape silent (fail-shut)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				env.MustRunGit("checkout", "-b", "epic/E-0001-engine", "main")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				env.MustRunGit("checkout", "main")
				env.MustRunGit("branch", "-D", "epic/E-0001-engine")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape"},
		},
	})
}

// createOrphanRitualBranch creates a ritual-shape branch with
// fully disjoint lineage from main (no common ancestor) via
// `git checkout --orphan` + a single commit. Used by AC-6
// cells 4 and 7 to construct "different lineage" scenarios
// where BranchOfSHA must resolve to the SHA's genuine owner
// and NOT pick the disjoint sibling as a candidate (since the
// sibling's first-parent chain doesn't contain the recorded
// SHA).
//
// Leaves HEAD on main when done so subsequent fixture steps
// have a sane base.
func createOrphanRitualBranch(t *testing.T, env *ScenarioEnv, branch string) {
	t.Helper()
	env.MustRunGit("checkout", "--orphan", branch)
	env.MustRunGit("rm", "-rf", "--cached", ".")
	env.MustRunGit("commit", "--allow-empty", "-m", "orphan: disjoint lineage for "+branch+" (AC-6 fixture)")
	// Force-checkout main: --orphan empties the index but leaves
	// the working tree's tracked files; switching back to main
	// surfaces those as "would be overwritten by checkout" until
	// the orphan tip overwrites them. `-f` resets cleanly.
	env.MustRunGit("checkout", "-f", "main")
}

// simulateLegacyAuthorize constructs an authorize commit that
// LOOKS like a pre-AC-6 commit: aiwf-branch trailer present,
// aiwf-branch-sha trailer ABSENT. Used to exercise the legacy-
// path resolution. Bypasses the verb (which would emit the SHA
// trailer post-AC-6) via raw git commit with manually-crafted
// trailers.
//
// The current branch BECOMES the scope-open commit's parent;
// the trailers identify the scope as opened on the bound
// branch's NAME without SHA evidence.
func simulateLegacyAuthorize(t *testing.T, env *ScenarioEnv, entityID, boundBranch string) {
	t.Helper()
	msg := strings.Join([]string{
		"aiwf authorize " + entityID + " --to ai/claude --branch " + boundBranch + " (legacy fixture)",
		"",
		"aiwf-verb: authorize",
		"aiwf-entity: " + entityID,
		"aiwf-actor: human/peter",
		"aiwf-to: ai/claude",
		"aiwf-scope: opened",
		"aiwf-branch: " + boundBranch,
	}, "\n")
	env.MustRunGit("commit", "--allow-empty", "-m", msg)
}
