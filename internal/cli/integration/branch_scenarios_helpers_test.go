package integration

import "testing"

// branch_scenarios_helpers_test.go — M-0159/AC-1 framework: types,
// driver, and branch-choreography helpers for the combinatorial
// real-git E2E test surface against the branch-choreography rule
// set (M-0102..M-0106, M-0136). Consumed by branch_scenarios_test.go
// and (in subsequent ACs) by per-rule test files that drive the
// table over their own scenario rows.
//
// Phase: RED. All bodies panic with "not implemented (M-0159/AC-1
// red phase)". A test invoking RunScenarios fails at runtime with
// the panic; the GREEN phase replaces the stubs with real
// implementations. Existing integration tests in this package are
// unaffected (the types compile clean and the stubs are not on any
// other test's execution path).

// Scenario is one row in the branch-choreography scenario table.
// Each row sets up its own fresh real-git fixture (Setup), then the
// driver runs `aiwf check --format=json` and asserts Expect against
// the resulting envelope.
//
// Setup is imperative — Go code that calls verb + git subprocesses
// via the ScenarioEnv helpers. There is no separate "Steps" slice
// or DSL: the kernel's verbs ARE the DSL, and Go's control flow
// keeps each scenario as readable as the equivalent narrative.
type Scenario struct {
	// Name is the t.Run subtest name. Should describe the
	// observable claim ("AI commit on bound branch is silent",
	// not "happy path A").
	Name string

	// Setup runs the scenario's preparation: create entities, set
	// up branches, open scopes, make commits. Mutates the env's
	// real-git repo via env.MustRunBin / env.MustRunGit, or via
	// the typed helpers (OpenBoundScope, AICommit, etc.).
	Setup func(t *testing.T, env *ScenarioEnv)

	// Expect describes the assertions the driver runs against
	// `aiwf check`'s envelope after Setup returns.
	Expect Expectation
}

// Expectation describes one or more assertions to run against
// `aiwf check --format=json`'s envelope. All set fields are
// asserted; unset fields are not checked. A scenario can both
// require a finding's presence (FindingPresent) and the absence
// of another (NoFindingWithCode) — but not the same code.
type Expectation struct {
	// NoFindingWithCode asserts no finding in the envelope has
	// this code. Used for "silent" paths (the bound-branch
	// commit, the cherry-pick, the force-amend override).
	NoFindingWithCode string

	// FindingPresent asserts at least one finding in the envelope
	// has this code. Used for "fires" paths (the escape, the
	// worktree mismatch).
	FindingPresent string
}

// ScenarioEnv is the per-scenario real-git state: a fresh temp
// repo with `aiwf init` already run, plus the directory housing
// the built aiwf binary. Constructed by the driver per scenario;
// not shared across scenarios.
type ScenarioEnv struct {
	T      *testing.T
	Root   string // working repo root
	BinDir string // directory containing aiwf binary (for PATH composition)
}

// MustRunBin invokes the aiwf binary inside the scenario's repo,
// fatal'ing the test on non-zero exit. Returns stdout for callers
// that need to parse output. Wraps testutil.RunBin with the env's
// root/binDir.
func (e *ScenarioEnv) MustRunBin(args ...string) string {
	panic("not implemented (M-0159/AC-1 red phase)")
}

// MustRunGit invokes git inside the scenario's repo, fatal'ing on
// non-zero exit. Returns stdout. Wraps testutil.RunGit.
func (e *ScenarioEnv) MustRunGit(args ...string) string {
	panic("not implemented (M-0159/AC-1 red phase)")
}

// RunScenarios is the table driver. Each row runs as a t.Run
// subtest with t.Parallel; the driver builds a fresh ScenarioEnv
// per row, calls Setup, then runs `aiwf check --format=json` and
// asserts Expect.
func RunScenarios(t *testing.T, scenarios []Scenario) {
	t.Helper()
	panic("not implemented (M-0159/AC-1 red phase)")
}

// Branch-choreography helpers consumed by scenarios.

// OpenBoundScope runs `aiwf authorize <entityID> --to ai/claude`
// from the current branch, opening a scope whose aiwf-branch:
// trailer captures the current ref (per M-0102's implicit-from-
// current path). Returns the authorize commit's SHA so subsequent
// scenarios can correlate AI commits to this scope.
//
// Caller is responsible for being on the correct branch when this
// is called — the implicit-current logic captures whatever ref
// HEAD currently points to.
func OpenBoundScope(t *testing.T, env *ScenarioEnv, entityID string) string {
	t.Helper()
	panic("not implemented (M-0159/AC-1 red phase)")
}

// AICommit runs `aiwf edit-body <entityID> --body-file -` with
// `--actor ai/claude --principal human/peter`, replacing the
// entity's body with bodyText. The resulting commit carries
// aiwf-actor: ai/claude (the trailer the isolation-escape rule's
// filter inspects) and aiwf-entity: <entityID>. Returns the
// commit SHA.
//
// Used as the canonical "AI does work on a scoped entity" shape
// in scenarios that exercise the isolation-escape rule's per-
// entity AI-actor check. The entity must exist (call
// env.MustRunBin("add", ...) in Setup first).
func AICommit(t *testing.T, env *ScenarioEnv, entityID, bodyText string) string {
	t.Helper()
	panic("not implemented (M-0159/AC-1 red phase)")
}

// ForceAmendHEAD runs git commit --amend on the HEAD commit,
// rewriting the commit message so the aiwf-actor: trailer reads
// "human/peter" (flipped from whatever it was) and a new
// `aiwf-force: <reason>` trailer is appended. The standard
// aiwf-entity / aiwf-verb / aiwf-principal trailers are preserved.
// Returns the new HEAD SHA (the amend rewrites the SHA).
//
// Pins the legacy M-0106/AC-8 sovereign-override mechanism: the
// rule's `ai/` actor-prefix filter sees `human/...` and skips the
// commit. The aiwf-force trailer is informational only (the
// kernel's enforcement is the actor flip, not the trailer).
func ForceAmendHEAD(t *testing.T, env *ScenarioEnv, reason string) string {
	t.Helper()
	panic("not implemented (M-0159/AC-1 red phase)")
}
