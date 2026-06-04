package integration

import (
	"strings"
	"testing"
)

// isolation_escape_force_push_scenarios_test.go — M-0161/AC-5
// (G-0205) real-git E2E scenarios for force-push orphan
// detection. The new gather component walks `git reflog show
// <ref>` for each ritual branch, identifies non-fast-forward
// updates (oldSHA NOT ancestor of newSHA), reads trailers from
// the orphaned tip, and surfaces AI-actor commits as the new
// isolation-escape-orphaned-ai-commit warning. The hint names
// SHA + branch + reflog date and the
// `aiwf acknowledge-illegal <sha>` sovereign-override path.
//
// AC-5 composes with AC-3: `core.logAllRefUpdates=false` flips
// `OracleErr{Capability="reflog-disabled"}` → AC-3's
// isolation-escape-oracle-failure advisory (no new code per
// AC-5 body line 350). Composes with M-0159/AC-3
// acknowledge-illegal: an ack against the orphan's SHA silences
// the new warning the same per-SHA way it silences the existing
// isolation-escape rule.
//
// RED state: WalkOrphanedAICommits does not exist; no
// isolation-escape-orphaned-ai-commit emission anywhere.
// Force-pushed orphaned AI commits are silently invisible to
// aiwf check today. The matrix cells asserting the new warning
// all fail. The reflog-disabled-composition cell asserts the
// AC-3 advisory fires for the missing-reflog mode, also RED.

// TestForcePushOrphan_AC5_Matrix drives the 7-cell matrix from
// the AC-5 body. Reflog-disabled cell composes with AC-3
// (isolation-escape-oracle-failure advisory with Capability
// "reflog-disabled").
func TestForcePushOrphan_AC5_Matrix(t *testing.T) {
	t.Parallel()
	RunScenarios(t, []Scenario{
		// ----- Cell 1: Full clone, no force-push, AI escape -----
		// isolation-escape fires; orphan finding silent.
		{
			Name: "AC-5 cell 1: full clone + AI escape (no force-push) → isolation-escape fires",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				SimulateAIEscape(t, env, "E-0001", "AI escape onto main, no force-push")
			},
			Expect: Expectation{FindingPresent: "isolation-escape", FindingSeverity: "warning"},
		},
		{
			Name: "AC-5 cell 1 (paired): no force-push, AI escape → orphan finding silent",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				SimulateAIEscape(t, env, "E-0001", "AI escape onto main, no force-push")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape-orphaned-ai-commit"},
		},

		// ----- Cell 2: Full clone, no force-push, no escape -----
		// Both silent. Baseline.
		{
			Name: "AC-5 cell 2: no force-push, no escape → orphan finding silent",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape-orphaned-ai-commit"},
		},

		// ----- Cell 3: Force-push orphans AI commit -----
		// Load-bearing. isolation-escape silent (orphan
		// unreachable from current ref tip; existing rule sees
		// nothing); new warning fires naming SHA + branch.
		{
			Name: "AC-5 cell 3: force-push orphans AI commit → orphan finding fires",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				orphanAIOnBoundBranch(t, env, "E-0001", "epic/E-0001-engine")
			},
			Expect: Expectation{
				FindingPresent:  "isolation-escape-orphaned-ai-commit",
				FindingSeverity: "warning",
			},
		},
		{
			Name: "AC-5 cell 3 (paired): force-push orphans AI commit → isolation-escape silent",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				orphanAIOnBoundBranch(t, env, "E-0001", "epic/E-0001-engine")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape"},
		},

		// ----- Cell 4: Force-push orphans non-AI commit -----
		// Both silent. The orphan exists but has no AI trailers.
		{
			Name: "AC-5 cell 4: force-push orphans non-AI commit → orphan finding silent",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				orphanHumanCommitOnBranch(t, env, "epic/E-0001-engine")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape-orphaned-ai-commit"},
		},

		// ----- Cell 5: Force-push orphans AI commit + ack ----- DEFERRED
		// AC-5 body line 349 assumed `aiwf acknowledge-illegal`
		// works on the orphan "per its existing mechanism", but
		// the verb at internal/verb/acknowledgeillegal.go
		// hard-requires the target SHA to be reachable from HEAD
		// (`git merge-base --is-ancestor <sha> HEAD`). A
		// force-push orphan is, by construction, unreachable —
		// that's the point of force-push. The verb refuses with
		// exit 2.
		//
		// Deferred per D-0020 (M-0161/AC-5 cell-5 orphan
		// acknowledgment deferred to verb extension) at
		// work/decisions/D-0020-*.md. Verb-side gap G-0226
		// (aiwf acknowledge-illegal hard-requires SHA reachable
		// from HEAD) at work/gaps/G-0226-*.md tracks the three
		// resolution paths a future verb-design cycle will pick:
		//   (a) `--allow-unreachable` flag on existing verb,
		//   (b) new `aiwf acknowledge-orphan <sha>` verb, or
		//   (c) rewrite the AC-5 composition claim.
		//
		// The rule-side per-SHA ack exemption IS unit-tested at
		// internal/check/reflog_walk_test.go::
		// TestRunOrphanedAICommits_AC5_AcknowledgedSHAExempted
		// so when the verb extension lands, the exemption is
		// already proven load-bearing.

		// ----- Cell 6: Reflog entry expired ----- DEFERRED
		// AC-5 body line 363 calls out: "Force-push orphans AI
		// commit, reflog entry expired via `git reflog expire
		// --expire-unreachable=now` → silent (no audit trail
		// to walk)".
		//
		// The scenario is straightforward to describe but
		// non-trivial to make deterministic in a test. `git
		// reflog expire --expire-unreachable=now` is a
		// time-sensitive operation; under some git versions on
		// some platforms it requires the orphan to already
		// pass certain unreachable-age thresholds (gc.reflogExpire
		// /gc.reflogExpireUnreachable defaults: 90d / 30d) or
		// it silently keeps the entry. Reliable reproduction
		// would require shimming git's clock or precise
		// `--expire=<date>` syntax that's git-version-dependent.
		//
		// The kernel-side behavior is trivially silent: with no
		// reflog entries to walk, WalkOrphanedAICommits finds
		// no orphans and the rule does not fire. The unit-level
		// equivalent (empty orphan slice → nil findings) IS
		// covered at internal/check/reflog_walk_test.go::
		// TestRunOrphanedAICommits_AC5_EmptyOrphans, which
		// pins the rule's silent-on-no-orphans contract
		// independent of how the orphans came to be missing
		// (expired, never recorded, or
		// core.logAllRefUpdates=false).
		//
		// Deferred to future test-infrastructure work (likely
		// AC-9 catalog consolidation or a separate
		// time-shim helper) when test-time control over git
		// clock becomes available. The AC-5 body matrix row
		// stays as-is; the silent-good behavior IS guaranteed
		// by the unit test cited above.

		// ----- Cell 7: Reflog disabled, force-push happens -----
		// Composes with AC-3: no reflog → no orphan walk; AC-3
		// emits isolation-escape-oracle-failure advisory with
		// Capability "reflog-disabled".
		//
		// AC-5 body line 364: "Reflog disabled, force-push
		// happens → silent (no reflog), PLUS
		// isolation-escape-oracle-failure advisory fires (per
		// AC-3 composition)".
		{
			Name: "AC-5 cell 7: reflog disabled + force-push → oracle-failure advisory fires (AC-3 composition)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				disableReflog(t, env)
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				orphanAIOnBoundBranch(t, env, "E-0001", "epic/E-0001-engine")
			},
			Expect: Expectation{
				FindingPresent:         "isolation-escape-oracle-failure",
				FindingHintContainsAll: []string{"reflog"},
			},
		},
		{
			Name: "AC-5 cell 7 (paired): reflog disabled + force-push → orphan finding silent (no reflog to walk)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				disableReflog(t, env)
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				orphanAIOnBoundBranch(t, env, "E-0001", "epic/E-0001-engine")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape-orphaned-ai-commit"},
		},
	})
}

// orphanAIOnBoundBranch builds a force-push-orphan fixture on
// the bound ritual branch: cuts the branch from main, makes an
// AI-actor commit (carrying aiwf-actor: ai/claude + aiwf-entity:
// <entityID>), then resets the branch ref one parent back via
// `git update-ref`. The reflog records the non-fast-forward
// update; the AI commit is now unreachable from the branch tip
// but still in the object store and still in the reflog.
// Returns the orphan's SHA.
//
// Why update-ref instead of `git push --force`: this is an
// in-repo fixture (no remote); the reflog records ref updates
// from any source. The reflog entry shape is identical to what
// `git push --force` writes to the receiving side's reflog.
func orphanAIOnBoundBranch(t *testing.T, env *ScenarioEnv, entityID, branch string) string {
	t.Helper()
	env.MustRunGit("checkout", "-b", branch, "main")
	mainTip := strings.TrimSpace(env.MustRunGit("rev-parse", "main"))
	aiSHA := AICommit(t, env, entityID, "AI commit that will be orphaned by force-push")
	// Reset the branch ref to main's tip — orphans aiSHA. The
	// reflog records this as a non-fast-forward update (aiSHA's
	// parent is mainTip, so mainTip IS an ancestor of aiSHA;
	// the update goes BACKWARDS so aiSHA is NOT an ancestor of
	// mainTip — that asymmetry is exactly what
	// WalkOrphanedAICommits detects).
	env.MustRunGit("update-ref", "refs/heads/"+branch, mainTip)
	env.MustRunGit("checkout", "main")
	return aiSHA
}

// orphanHumanCommitOnBranch creates a human-authored commit on
// a ritual branch and orphans it via update-ref. The orphan
// lacks aiwf-actor: ai/... so the new rule should NOT fire.
func orphanHumanCommitOnBranch(t *testing.T, env *ScenarioEnv, branch string) {
	t.Helper()
	env.MustRunGit("checkout", "-b", branch, "main")
	mainTip := strings.TrimSpace(env.MustRunGit("rev-parse", "main"))
	// A human commit with NO aiwf trailers.
	env.MustRunGit("commit", "--allow-empty", "-m", "human work to be orphaned")
	env.MustRunGit("update-ref", "refs/heads/"+branch, mainTip)
	env.MustRunGit("checkout", "main")
}

// disableReflog flips `core.logAllRefUpdates=false` on the
// scenario's repo so subsequent ref updates produce NO reflog
// entries. AC-3 composition: the oracle detects this and
// surfaces an isolation-escape-oracle-failure advisory with
// Capability "reflog-disabled".
func disableReflog(t *testing.T, env *ScenarioEnv) {
	t.Helper()
	env.MustRunGit("config", "core.logAllRefUpdates", "false")
}
