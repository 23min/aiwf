package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// isolation_escape_shallow_scenarios_test.go — M-0161/AC-4
// (G-0204) real-git E2E scenarios for the shallow-clone
// detection contract: newGitBranchOracle detects shallow state
// via `git rev-parse --is-shallow-repository`; on shallow the
// per-SHA map is empty and OracleErrors carries a typed
// "shallow-clone" entry; RunProvenanceCheck emits the new
// isolation-escape-shallow-clone finding at warning severity
// with remediation hint naming `git fetch --unshallow`.
//
// AC-4 composes with AC-3: shallow rides the typed OracleErr
// slice, but surfaces as a SEPARATE finding code per the AC-4
// body line 292 ("warning severity, not advisory — total
// coverage failure is louder than per-ref partial-failure
// advisory"). This is the deliberate exception to D-0019
// Alternative D's "ride the typed slice" rule.
//
// Fail-shut on correctness: even `git clone --depth=N` for N>1
// (commits-within-window case) is treated as fail-shut — the
// rule does not fire on shallow because shallow is shallow,
// regardless of depth. The remediation hint carries the operator
// to unshallow.
//
// Fixture mechanic: write a SHA into .git/shallow to flip the
// shallow flag. Faster + more deterministic than spinning up a
// `git clone --depth=N` from a richer source.
//
// RED state: isolation-escape-shallow-clone is not emitted by
// any code path today; scenarios asserting its presence fail.
// The "shallow + AI escape" cell additionally fails because the
// pre-AC-4 oracle walks rev-list normally on shallow repos,
// populates branchesBySHA from the truncated walk, and the
// isolation-escape rule fires on the visible escape (the AC-4
// contract requires fail-shut silent here).

// TestBranchOracle_AC4_ShallowClone_Matrix drives the 6-cell
// matrix from the AC-4 body. Cell 6 (unshallow) is implicit —
// the non-shallow scenarios already cover the shallow-flag-
// cleared state.
func TestBranchOracle_AC4_ShallowClone_Matrix(t *testing.T) {
	t.Parallel()
	RunScenarios(t, []Scenario{
		// ----- Cell 1: Full clone, no escape -----
		// Both silent. Baseline.
		{
			Name: "AC-4 cell 1: full clone, no AI commits → isolation-escape silent",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape"},
		},
		{
			Name: "AC-4 cell 1 (paired): full clone, no AI commits → shallow-clone silent",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape-shallow-clone"},
		},

		// ----- Cell 2: Full clone, escape present -----
		// isolation-escape fires; shallow-clone silent.
		{
			Name: "AC-4 cell 2: full clone + AI escape → isolation-escape fires",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				SimulateAIEscape(t, env, "E-0001", "AI escape onto main, full clone")
			},
			Expect: Expectation{FindingPresent: "isolation-escape", FindingSeverity: "warning"},
		},
		{
			Name: "AC-4 cell 2 (paired): full clone + AI escape → shallow-clone silent",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				SimulateAIEscape(t, env, "E-0001", "AI escape onto main, full clone")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape-shallow-clone"},
		},

		// ----- Cell 3: Shallow clone, AI escape -----
		// Load-bearing case. isolation-escape SILENT (fail-shut on
		// shallow regardless of whether the escape is within the
		// shallow window); isolation-escape-shallow-clone fires
		// warning with the remediation hint.
		{
			Name: "AC-4 cell 3: shallow clone + AI escape → isolation-escape silent (fail-shut)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				SimulateAIEscape(t, env, "E-0001", "AI escape beyond shallow boundary")
				flipShallow(t, env)
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape"},
		},
		{
			Name: "AC-4 cell 3 (paired): shallow clone + AI escape → shallow-clone fires warning",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				SimulateAIEscape(t, env, "E-0001", "AI escape beyond shallow boundary")
				flipShallow(t, env)
			},
			Expect: Expectation{
				FindingPresent:         "isolation-escape-shallow-clone",
				FindingSeverity:        "warning",
				FindingHintContainsAll: []string{"unshallow"},
			},
		},

		// ----- Cell 4: Shallow clone, no escape -----
		// shallow-clone STILL fires (coverage incomplete regardless
		// of whether anything illegal is present); isolation-escape
		// silent (no commit to fire on, and the rule's fail-shut
		// would silence it anyway).
		{
			Name: "AC-4 cell 4: shallow clone, no AI escape → isolation-escape silent",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				flipShallow(t, env)
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape"},
		},
		{
			Name: "AC-4 cell 4 (paired): shallow clone, no AI escape → shallow-clone fires",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				flipShallow(t, env)
			},
			Expect: Expectation{
				FindingPresent:         "isolation-escape-shallow-clone",
				FindingSeverity:        "warning",
				FindingHintContainsAll: []string{"unshallow"},
			},
		},

		// ----- Cell 5: depth=N within-window case ------
		// AC-4 body explicitly carves this out: even when the
		// escape would be visible in the shallow window, the
		// oracle fails shut. Functionally identical to cell 3
		// in our fixture (we can't distinguish depth-window
		// semantics through .git/shallow alone); the cell pins
		// the contract behaviorally.
		{
			Name: "AC-4 cell 5: shallow depth=N + escape within window → isolation-escape STILL silent",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				SimulateAIEscape(t, env, "E-0001", "AI escape inside shallow depth=N window")
				// Same shallow flip — git treats any depth as shallow
				// per is-shallow-repository.
				flipShallow(t, env)
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape"},
		},

		// ----- Cell 6: Unshallow -----
		// Same fixture as cell 2 (full clone + escape) — pins that
		// the shallow-flag-cleared path returns to normal coverage.
		// Symmetric to cell 2; explicit naming for catalog clarity.
		{
			Name: "AC-4 cell 6: unshallow → isolation-escape works again (same as full-clone path)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				SimulateAIEscape(t, env, "E-0001", "AI escape after unshallow")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape-shallow-clone"},
		},
	})
}

// TestBranchOracle_AC4_SovereignOverride_ShallowFiresAnyway pins
// the AC-4 sovereign-override scenario: shallow + AI escape with
// `aiwf-force: "..."` → isolation-escape silent (structurally
// invisible inside the shallow boundary; override is moot),
// shallow-clone STILL fires (orthogonal to per-commit override —
// operator must unshallow to see the picture).
func TestBranchOracle_AC4_SovereignOverride_ShallowFiresAnyway(t *testing.T) {
	t.Parallel()
	RunScenarios(t, []Scenario{
		{
			Name: "AC-4 sovereign: shallow + force-amended escape → isolation-escape silent",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				SimulateAIEscape(t, env, "E-0001", "escape to be force-amended")
				ForceAmendHEAD(t, env, "AC-4 sovereign override")
				flipShallow(t, env)
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape"},
		},
		{
			Name: "AC-4 sovereign (paired): shallow + force-amend → shallow-clone STILL fires",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				SimulateAIEscape(t, env, "E-0001", "escape to be force-amended")
				ForceAmendHEAD(t, env, "AC-4 sovereign override")
				flipShallow(t, env)
			},
			Expect: Expectation{
				FindingPresent:         "isolation-escape-shallow-clone",
				FindingSeverity:        "warning",
				FindingHintContainsAll: []string{"unshallow"},
			},
		},
	})
}

// flipShallow writes a SHA into the repo's .git/shallow file,
// causing `git rev-parse --is-shallow-repository` to return
// true. Uses HEAD's SHA so the content is a valid commit-object
// id in the repo (some git versions validate ref-existence at
// shallow-file parse time).
func flipShallow(t *testing.T, env *ScenarioEnv) {
	t.Helper()
	headSHA := strings.TrimSpace(env.MustRunGit("rev-parse", "HEAD"))
	shallowPath := filepath.Join(env.Root, ".git", "shallow")
	if err := os.WriteFile(shallowPath, []byte(headSHA+"\n"), 0o644); err != nil {
		t.Fatalf("flipShallow: write %s: %v", shallowPath, err)
	}
}
