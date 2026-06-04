package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// isolation_escape_oracle_scenarios_test.go — M-0161/AC-3 (G-0203)
// real-git E2E scenarios for the BranchOracle typed-error +
// per-ref-fault-tolerance contract.
//
// AC-3 contract (per body + D-0019):
//
//   - `isolation-escape` (M-0106, warning) fires on actual AI-
//     actor escapes; fails-shut on rule correctness — does not
//     fire on commits whose branch resolution lost coverage
//     from a per-ref failure.
//   - `isolation-escape-oracle-failure` (new, warning-advisory)
//     fires once per failed ref at oracle construction time,
//     naming the ref and the underlying failure mode in the
//     hint. Surfaces partial-coverage states mechanically so
//     operators see what dropped.
//   - Fail-shut on correctness + fail-open on coverage:
//     - Healthy refs continue to populate the per-SHA index.
//     - A single corrupt ref does NOT disable the rule for the
//       whole repo.
//     - Commits whose branch can only be resolved via the
//       failed ref's index get no finding (no false positives).
//
// The 7-cell matrix below mirrors the AC-3 body's table; the
// 8th scenario exercises the AC body's "Sovereign-override
// path stays clean" assertion. Cell 6 (empty repo) is deferred:
// the ScenarioEnv framework bootstraps main + origin/main; a
// truly-empty repo would require a parallel env not yet
// modeled. The unit-level guard at
// internal/cli/check/isolation_escape_oracle.go:54-56 pins
// the empty-repo silent path until then.
//
// RED state: `isolation-escape-oracle-failure` is not emitted
// by any code path today; scenarios asserting its presence fail
// at envelope-inspection time. The cells that assert silence on
// isolation-escape under partial coverage also fail today —
// pre-AC-3 the whole oracle returns nil on per-ref failure, so
// isolation-escape silent under "one ref corrupted + healthy
// escape" is a false-positive silent: the rule should fire on
// the healthy-ref escape but currently misses it because the
// oracle aborted construction. Post-AC-3 the rule fires
// correctly on the healthy side.

// TestBranchOracle_AC3_OracleErrors_Matrix drives the 7-cell
// matrix from the AC-3 body. Each cell registers as a real-git
// E2E scenario under the M-0159 framework.
func TestBranchOracle_AC3_OracleErrors_Matrix(t *testing.T) {
	t.Parallel()
	RunScenarios(t, []Scenario{
		// ----- Cell 1: All refs healthy, no AI work -----
		// Both findings silent. Baseline silent-good.
		{
			Name: "AC-3 cell 1: all refs healthy, no AI commits → isolation-escape silent",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				// No authorize, no AI commits — purely the env
				// baseline + one entity-add commit on main.
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape"},
		},
		{
			Name: "AC-3 cell 1 (paired): all refs healthy, no AI commits → oracle-failure silent",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape-oracle-failure"},
		},

		// ----- Cell 2: All refs healthy, escape present -----
		// `isolation-escape` fires (warning per M-0106);
		// oracle-failure stays silent.
		{
			Name: "AC-3 cell 2: all refs healthy + AI escape → isolation-escape fires",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				SimulateAIEscape(t, env, "E-0001", "AI escape onto main")
			},
			Expect: Expectation{FindingPresent: "isolation-escape", FindingSeverity: "warning"},
		},
		{
			Name: "AC-3 cell 2 (paired): all refs healthy + AI escape → oracle-failure silent",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				SimulateAIEscape(t, env, "E-0001", "AI escape onto main")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape-oracle-failure"},
		},

		// ----- Cell 3: One ritual ref corrupted, no escape -----
		// isolation-escape silent (no AI escape); oracle-failure
		// fires advisory naming the corrupt ref.
		{
			Name: "AC-3 cell 3: one ritual ref corrupted, no AI escape → isolation-escape silent",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				corruptUnusedRitualRef(t, env, "epic/E-9999-stale")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape"},
		},
		{
			Name: "AC-3 cell 3 (paired): one ritual ref corrupted, no AI escape → oracle-failure fires advisory",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				corruptUnusedRitualRef(t, env, "epic/E-9999-stale")
			},
			Expect: Expectation{
				FindingPresent:         "isolation-escape-oracle-failure",
				FindingSeverity:        "warning",
				FindingHintContainsAll: []string{"epic/E-9999-stale"},
			},
		},

		// ----- Cell 4: One ritual ref corrupted, escape on healthy ref -----
		// Load-bearing case. isolation-escape STILL fires (the
		// escape lands on a healthy ref the oracle indexed
		// cleanly); oracle-failure ALSO fires for the corrupt
		// sibling. Pre-AC-3 the whole oracle was nil → silent
		// miss of the real escape. Post-AC-3 the rule polices
		// the healthy ref normally.
		{
			Name: "AC-3 cell 4: one ritual ref corrupted + escape on healthy ref → isolation-escape fires",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				SimulateAIEscape(t, env, "E-0001", "AI escape onto main (healthy ref)")
				corruptUnusedRitualRef(t, env, "epic/E-9999-stale")
			},
			Expect: Expectation{FindingPresent: "isolation-escape", FindingSeverity: "warning"},
		},
		{
			Name: "AC-3 cell 4 (paired): one ritual ref corrupted + escape on healthy ref → oracle-failure fires",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				SimulateAIEscape(t, env, "E-0001", "AI escape onto main (healthy ref)")
				corruptUnusedRitualRef(t, env, "epic/E-9999-stale")
			},
			Expect: Expectation{
				FindingPresent:         "isolation-escape-oracle-failure",
				FindingSeverity:        "warning",
				FindingHintContainsAll: []string{"epic/E-9999-stale"},
			},
		},

		// ----- Cell 5: All ritual refs corrupted -----
		// isolation-escape silent (no healthy ref to fire
		// against); oracle-failure fires per corrupted ref.
		// "All ritual refs" in this fixture = the one
		// epic/E-9999-stale we created; main is not a ritual ref
		// in the rule's filter perspective (main is the trunk
		// the oracle treats specially per the existing filter
		// at internal/cli/check/isolation_escape_oracle.go:105).
		{
			Name: "AC-3 cell 5: all ritual refs corrupted → isolation-escape silent",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				corruptUnusedRitualRef(t, env, "epic/E-9999-stale")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape"},
		},
		{
			Name: "AC-3 cell 5 (paired): all ritual refs corrupted → oracle-failure fires advisory",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				corruptUnusedRitualRef(t, env, "epic/E-9999-stale")
			},
			Expect: Expectation{FindingPresent: "isolation-escape-oracle-failure", FindingSeverity: "warning"},
		},

		// ----- Cell 6: Empty repo ----- DEFERRED
		// See file header. Unit-level guard pins this until the
		// framework grows a raw-repo env.

		// ----- Cell 7: Repo with only non-ritual refs -----
		// Non-ritual refs (`feature/foo`) are filtered before
		// per-ref indexing; both findings silent.
		{
			Name: "AC-3 cell 7: repo with only non-ritual refs → isolation-escape silent",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunGit("branch", "feature/foo", "main")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape"},
		},
		{
			Name: "AC-3 cell 7 (paired): repo with only non-ritual refs → oracle-failure silent",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunGit("branch", "feature/foo", "main")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape-oracle-failure"},
		},
	})
}

// TestBranchOracle_AC3_SovereignOverride_StaysClean exercises
// the AC-3 body's "Sovereign-override path stays clean"
// assertion: an acknowledge-illegal commit silences
// isolation-escape on a specific SHA; oracle-failure rides an
// independent codepath and continues to fire for unrelated ref
// failures.
//
// Pins: the typed-error split does not break the existing
// per-SHA closed-set scoping the M-0159/AC-3 lift established.
func TestBranchOracle_AC3_SovereignOverride_StaysClean(t *testing.T) {
	t.Parallel()
	RunScenarios(t, []Scenario{
		{
			Name: "AC-3 sovereign: acknowledged escape silences isolation-escape + unrelated corrupt ref",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				escapeSHA := SimulateAIEscape(t, env, "E-0001", "escape acknowledged below")
				AcknowledgeIllegal(t, env, escapeSHA, "AC-3 sovereign-override fixture")
				corruptUnusedRitualRef(t, env, "epic/E-9999-stale")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape"},
		},
		{
			Name: "AC-3 sovereign (paired): acknowledged escape + unrelated corrupt ref → oracle-failure fires",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				escapeSHA := SimulateAIEscape(t, env, "E-0001", "escape acknowledged below")
				AcknowledgeIllegal(t, env, escapeSHA, "AC-3 sovereign-override fixture")
				corruptUnusedRitualRef(t, env, "epic/E-9999-stale")
			},
			Expect: Expectation{
				FindingPresent:         "isolation-escape-oracle-failure",
				FindingHintContainsAll: []string{"epic/E-9999-stale"},
			},
		},
	})
}

// corruptUnusedRitualRef creates a fresh ritual-shape ref at
// the current main tip with a real commit, then overwrites the
// loose object file for the commit's tip so the per-ref
// first-parent walk fails while for-each-ref still emits the
// ref name.
//
// The ref is "unused" in the sense that no aiwf-scope or
// AI-commit fixture references it; its only role is to be a
// failed-walk candidate during oracle construction.
//
// Git stores loose objects mode 0o444; the chmod is required
// before the overwrite. Stale.md is removed from the worktree
// so subsequent fixture commits don't pick it up.
func corruptUnusedRitualRef(t *testing.T, env *ScenarioEnv, ref string) {
	t.Helper()
	env.MustRunGit("checkout", "-b", ref, "main")
	staleRel := "stale-" + sanitizeRefForFilename(ref) + ".md"
	if err := os.WriteFile(filepath.Join(env.Root, staleRel), []byte("stale tip; will be corrupted\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", staleRel, err)
	}
	env.MustRunGit("add", staleRel)
	env.MustRunGit("-c", "user.email=t@t", "-c", "user.name=t", "commit", "-m", "stale tip; will be corrupted")
	tip := strings.TrimSpace(env.MustRunGit("rev-parse", "HEAD"))

	// Switch off the stale ref BEFORE corrupting so HEAD doesn't
	// point at the broken object (subsequent git invocations
	// would otherwise fail on HEAD resolution).
	env.MustRunGit("checkout", "main")

	objPath := filepath.Join(env.Root, ".git", "objects", tip[:2], tip[2:])
	// Git writes loose objects mode 0o444 (read-only). The chmod
	// is required for the test to overwrite the object file in
	// place; without it os.WriteFile fails with EACCES.
	if err := os.Chmod(objPath, 0o644); err != nil {
		t.Fatalf("chmod object %s: %v", objPath, err)
	}
	if err := os.WriteFile(objPath, []byte("garbage-not-zlib\n"), 0o644); err != nil {
		t.Fatalf("corrupt object %s: %v", objPath, err)
	}
	_ = os.Remove(filepath.Join(env.Root, staleRel))
}

// sanitizeRefForFilename replaces ref characters that are not
// valid in filenames (notably '/') with '-' so multi-ref test
// scenarios don't clobber each other's worktree files.
func sanitizeRefForFilename(ref string) string {
	return strings.ReplaceAll(ref, "/", "-")
}
