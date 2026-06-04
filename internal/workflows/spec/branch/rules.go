package branch

import (
	"sort"

	"github.com/23min/aiwf/internal/workflows/spec"
)

// Rules returns the layer-4 branch-choreography cells, sorted by cell
// id for deterministic output (M-0158/AC-7). The closed set comprises:
//
//   - 5 illegal-outcome corner-case cells with mechanical weight
//     (`branch-cell-1`, `-2`, `-4`, `-7`, `-12`) from E-0030
//     §"Corner cases" (Cycle 2). Cells 3, 5, 6, 8, 9, 10, 11 were
//     dropped per M-0162/AC-1 (M-0161/AC-9 §"Part 1"): the 5
//     legal-non-override cells (3/5/6/9/11) carried no mechanical
//     weight, and cells 8/10 duplicated override-surface entries.
//   - 2 standalone override cells `branch-cell-override-preflight`
//     and `branch-cell-override-f-nnnn-waiver` from E-0030
//     §"Sovereign override surface" (Cycle 3). The
//     `branch-cell-override-cherry-pick` and
//     `branch-cell-override-force-amend` entries were dropped per
//     M-0162/AC-1 as semantic duplicates of corner-case cells 8/10
//     (themselves dropped); the override mechanisms remain
//     present in the kernel — the catalog redundancy is what was
//     redundant.
//
// Top-level integration: consumers union with `spec.Rules()` at the
// call site (the parent package cannot import this sub-package without
// a cycle; see package doc).
func Rules() []spec.Rule {
	out := []spec.Rule{
		// branch-cell-1 — Corner case 1: AI-actor authorize on main,
		// no --branch. Preflight refuses with branch-context-required.
		// Tests: TestAuthorize_Open_AITarget_NoBranch_NoRitualCurrent_Refuses
		// (verb), TestRunAuthorize_AITarget_OnNonRitualBranch_NoBranch_Refuses
		// (CLI seam).
		{
			ID:                "branch-cell-1",
			Verb:              "authorize",
			Preconditions:     []spec.Predicate{{Subject: "target-agent-role", Op: "==", Value: "ai"}, {Subject: "ritual-branch-context-present", Op: "==", Value: "false"}, {Subject: "force", Op: "==", Value: "false"}},
			Outcome:           spec.OutcomeIllegal,
			ExpectedErrorCode: "branch-context-required",
			RejectionLayer:    spec.RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-2 — Corner case 2: AI authorize --branch <typo>.
		// Preflight refuses with branch-not-found. Test:
		// TestAuthorize_Open_AITarget_BranchMissing_Refuses + CLI seam.
		{
			ID:                "branch-cell-2",
			Verb:              "authorize",
			Preconditions:     []spec.Predicate{{Subject: "target-agent-role", Op: "==", Value: "ai"}, {Subject: "branch-flag-resolves", Op: "==", Value: "false"}, {Subject: "force", Op: "==", Value: "false"}},
			Outcome:           spec.OutcomeIllegal,
			ExpectedErrorCode: "branch-not-found",
			RejectionLayer:    spec.RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-3 dropped per M-0162/AC-1 (legal-non-override,
		// documentation-only): the implicit-from-current preflight
		// accept path is exercised by every legitimate ritual
		// authorize commit and pinned by branch-cell-1's negative
		// counterpart (a refusal absent branch context implies
		// presence of branch context is the legitimate path). The
		// dedicated cell carried no separate mechanical assertion.
		// branch-cell-4 — Corner case 4: AI commit on main while
		// scope binds epic/E-NN-X. Finding fires (check-time,
		// isolation-escape). Tests: TestIsolationEscape_AC1_AICommitOnMainFires
		// + TestRunProvenanceCheck_IsolationEscape_FiresOnViolatingCommit.
		{
			ID:                "branch-cell-4",
			Preconditions:     []spec.Predicate{{Subject: "commit-actor-role", Op: "==", Value: "ai"}, {Subject: "scope-binding-branch", Op: "!=", Value: "commit-branch"}, {Subject: "commit-branch", Op: "==", Value: "main"}},
			Outcome:           spec.OutcomeIllegal,
			ExpectedErrorCode: "isolation-escape",
			RejectionLayer:    spec.RejectionLayerCheckTime,
			BlockingStrict:    false, // warning severity at first land (M-0106 spec)
			Sources:           spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-5 and branch-cell-6 dropped per M-0162/AC-1
		// (legal-non-override, documentation-only): the bound-
		// branch silence path is the rule's default outcome —
		// branch-cell-4 / branch-cell-7's illegal-outcome cells
		// are the discriminators. A dedicated "this is silent
		// because nothing fires" cell carried no mechanical
		// weight.
		// branch-cell-7 — Corner case 7: AI commit on epic/E-NN-Y
		// while bound to epic/E-NN-X. Different epic branch → fires.
		// Test: TestIsolationEscape_AC2_AICommitOnDifferentRitualBranchFires.
		{
			ID:                "branch-cell-7",
			Preconditions:     []spec.Predicate{{Subject: "commit-actor-role", Op: "==", Value: "ai"}, {Subject: "scope-binding-branch", Op: "!=", Value: "commit-branch"}, {Subject: "commit-branch-shape", Op: "==", Value: "ritual"}},
			Outcome:           spec.OutcomeIllegal,
			ExpectedErrorCode: "isolation-escape",
			RejectionLayer:    spec.RejectionLayerCheckTime,
			BlockingStrict:    false,
			Sources:           spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-8 dropped per M-0162/AC-1 (legal-AND-override
		// duplicate): the cherry-pick re-author silence is the same
		// shape as branch-cell-override-cherry-pick was, registering
		// the same preconditions twice. Both are dropped here; the
		// kernel's cherry-pick re-author detection remains intact —
		// only the redundant catalog entries are removed.
		// branch-cell-9 dropped per M-0162/AC-1 (legal-non-override,
		// documentation-only): human-actor merge commits are silent
		// because the isolation-escape rule keys on commit-actor-
		// role=ai. The legal outcome here is the rule's default
		// behavior, not a discriminator. branch-cell-4 and
		// branch-cell-7 carry the discriminating illegal outcomes
		// the rule actually checks.
		// branch-cell-10 dropped per M-0162/AC-1 (legal-AND-override
		// duplicate): same shape as branch-cell-override-force-amend
		// was, registering the same preconditions twice. Both
		// dropped; the aiwf-force trailer override mechanism in the
		// kernel remains intact.
		// branch-cell-11 dropped per M-0162/AC-1 (legal-non-override,
		// documentation-only): AI commits without an open scope are
		// silent because the rule requires a binding to evaluate
		// branch-mismatch. The cell encoded "no rule applies = legal"
		// as a positive cell with no mechanical assertion.
		// branch-cell-12 — Corner case 12: Worktree-vs-branch mismatch
		// (subagent did git checkout main from inside its assigned
		// worktree). Finding fires — same code path as branch-cell-4
		// because the rule's detection is branch-identity, not path-
		// based. Test: TestIsolationEscape_AC3_WorktreeBranchMismatchFires.
		{
			ID:                "branch-cell-12",
			Preconditions:     []spec.Predicate{{Subject: "commit-actor-role", Op: "==", Value: "ai"}, {Subject: "scope-binding-branch", Op: "!=", Value: "commit-branch"}, {Subject: "worktree-path-mismatches-branch", Op: "==", Value: "true"}},
			Outcome:           spec.OutcomeIllegal,
			ExpectedErrorCode: "isolation-escape",
			RejectionLayer:    spec.RejectionLayerCheckTime,
			BlockingStrict:    false,
			Sources:           spec.RuleSource{Decision: "ADR-0010"},
		},

		// Override cells per E-0030 §"Sovereign override surface".
		// Each names a kernel-readable override mechanism that
		// suppresses the otherwise-illegal cell's outcome. All are
		// Legal cells in the spec table — the override IS the
		// gated, audited acceptance.

		// branch-cell-override-preflight — M-0103 preflight override:
		// `aiwf authorize <id> --to ai/<x> --force --reason "..."`
		// bypasses the branch-context-required and branch-not-found
		// refusals. Gated by the trailer-shape rule (--force requires
		// human/ actor + non-empty --reason). Test:
		// TestAuthorize_Open_AITarget_ForceReasonBypassesPreflight +
		// CLI seam.
		{
			ID:            "branch-cell-override-preflight",
			Verb:          "authorize",
			Preconditions: []spec.Predicate{{Subject: "target-agent-role", Op: "==", Value: "ai"}, {Subject: "force", Op: "==", Value: "true"}, {Subject: "actor-role", Op: "==", Value: "human"}, {Subject: "reason", Op: "non-empty", Value: ""}},
			Outcome:       spec.OutcomeLegal,
			Sources:       spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-override-cherry-pick and
		// branch-cell-override-force-amend dropped per M-0162/AC-1
		// (semantic duplicates of corner-case cells 8 and 10
		// respectively — both also dropped in this AC). The kernel's
		// cherry-pick suppression and aiwf-force trailer override
		// mechanisms remain implemented in the rules engine; the
		// catalog redundancy is what was redundant.
		// branch-cell-override-f-nnnn-waiver — At-check F-NNNN waiver
		// per ADR-0003: `aiwf promote F-NNNN waived --force --reason
		// "..."` records a finding-waiver as a sovereign act. The
		// waiver itself is the override. Gated by the F-NNNN
		// AC-closure rule and the standard trailer-shape rule.
		// Behavioral tests live in the F-NNNN milestone family
		// (outside E-0030 scope); this cell registers the override
		// surface in the spec table for catalog completeness.
		//
		// Kind is `finding` per ADR-0003 §"Decision" — the ADR
		// declares finding as the seventh entity kind, stored at
		// work/findings/F-NNNN-*.md. The entity kind itself isn't
		// implemented in the PoC yet (entity.AllKinds returns only
		// the six existing kinds); this cell forward-declares the
		// correct surface name so consumers reading the spec table
		// see the right Kind value without needing the entity-side
		// implementation. Pinned by
		// internal/policies/m0159_ac8_kind_correction_test.go.
		{
			ID:            "branch-cell-override-f-nnnn-waiver",
			Kind:          "finding",
			Verb:          "promote",
			Preconditions: []spec.Predicate{{Subject: "self.status", Op: "==", Value: "waived"}, {Subject: "force", Op: "==", Value: "true"}, {Subject: "actor-role", Op: "==", Value: "human"}, {Subject: "reason", Op: "non-empty", Value: ""}},
			Outcome:       spec.OutcomeLegal,
			Sources:       spec.RuleSource{Decision: "ADR-0003"},
		},
		// branch-cell-id-rename-untrailered — M-0160/AC-4: an
		// inline `git mv` rename of an id-bearing entity file (the
		// CLAUDE.md §"Id-collision resolution at merge time"
		// operator-discipline failure mode) commits without an
		// `aiwf-verb` trailer in the rename-class closed set
		// (retitle / rename / reallocate / archive / move). The
		// kernel rule id-rename-untrailered fires at check time.
		// Tests:
		// TestIDRenameUntrailered_TypedCodeClassIsBranchChoreography
		// (unit) +
		// TestIDRenameUntrailered_AC4_InlineGitMvFiresFinding
		// (integration via the M-0159 RunScenarios framework).
		{
			ID:                "branch-cell-id-rename-untrailered",
			Preconditions:     []spec.Predicate{{Subject: "rename-class-verb-trailer", Op: "==", Value: "absent"}, {Subject: "renamed-file", Op: "==", Value: "id-bearing-entity"}},
			Outcome:           spec.OutcomeIllegal,
			ExpectedErrorCode: "id-rename-untrailered",
			RejectionLayer:    spec.RejectionLayerCheckTime,
			BlockingStrict:    false, // warning severity at first land (M-0160/AC-4 design)
			Sources:           spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-isolation-escape-oracle-failure — M-0161/AC-3
		// (G-0203) + D-0019: the BranchOracle accumulates per-ref
		// construction failures (a single ritual ref's first-parent
		// walk fails while sibling refs succeed) and surfaces them
		// as isolation-escape-oracle-failure advisory findings. The
		// fail-shut-on-correctness contract means the
		// isolation-escape rule does NOT fire on commits whose
		// branch resolution lost coverage through the failed ref;
		// the advisory exists so operators see partial-coverage
		// mechanically.
		//
		// Tests:
		// TestNewGitBranchOracle_AC3_PerRefTolerance_OneCorruptedRef
		// (unit, internal/cli/check/) +
		// TestBranchOracle_AC3_OracleErrors_Matrix
		// (integration, internal/cli/integration/).
		//
		// AC-9 (G-0210) consolidates the 7-scenario matrix from the
		// AC-3 body into a fuller cell set; this single cell satisfies
		// the M-0158/AC-6 ClassBranchChoreography drift bidirectional
		// invariant in the interim.
		{
			ID:                "branch-cell-isolation-escape-oracle-failure",
			Preconditions:     []spec.Predicate{{Subject: "oracle-per-ref-resolution-failed", Op: "==", Value: "true"}},
			Outcome:           spec.OutcomeIllegal,
			ExpectedErrorCode: "isolation-escape-oracle-failure",
			RejectionLayer:    spec.RejectionLayerCheckTime,
			BlockingStrict:    false, // advisory severity per AC-3 body / M-0125 ratchet
			Sources:           spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-isolation-escape-shallow-clone — M-0161/AC-4
		// (G-0204): the oracle detects shallow state at construction
		// time via `git rev-parse --is-shallow-repository`. On
		// shallow the per-SHA map is left EMPTY (fail-shut on
		// correctness — a half-walked first-parent index would
		// produce silent false-negatives for commits beyond the
		// shallow boundary) and a typed OracleErr with
		// Capability="shallow-clone" surfaces as the
		// isolation-escape-shallow-clone warning, hint naming
		// `git fetch --unshallow` as the remediation. Separate code
		// (not the AC-3 isolation-escape-oracle-failure advisory)
		// per AC-4 body line 292 — total-coverage failure is louder
		// than per-ref partial.
		//
		// Tests:
		// TestNewGitBranchOracle_AC4_ShallowDetection_EmptyMapPlusTypedError
		// (unit) +
		// TestBranchOracle_AC4_ShallowClone_Matrix
		// (integration; 6 cells + sovereign override).
		//
		// AC-9 (G-0210) consolidates the matrix into a fuller cell
		// set; this single cell satisfies the M-0158/AC-6 drift
		// invariant in the interim.
		{
			ID:                "branch-cell-isolation-escape-shallow-clone",
			Preconditions:     []spec.Predicate{{Subject: "repository-is-shallow", Op: "==", Value: "true"}},
			Outcome:           spec.OutcomeIllegal,
			ExpectedErrorCode: "isolation-escape-shallow-clone",
			RejectionLayer:    spec.RejectionLayerCheckTime,
			BlockingStrict:    false, // warning severity per AC-4 body / M-0125 ratchet
			Sources:           spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-isolation-escape-orphaned-ai-commit —
		// M-0161/AC-5 (G-0205): WalkOrphanedAICommits walks
		// each ritual ref's reflog for non-fast-forward updates
		// (oldSHA NOT ancestor of newSHA) and surfaces AI-actor
		// commits orphaned by the update as
		// isolation-escape-orphaned-ai-commit warnings, hint
		// naming `aiwf acknowledge-illegal` for deliberate
		// sovereign cleanup. Composes with AC-3 acknowledge-
		// illegal via the shared per-SHA ackedSHAs map.
		//
		// Reflog-disabled (core.logAllRefUpdates=false) is the
		// missing-coverage mode: it surfaces via AC-3's
		// isolation-escape-oracle-failure advisory with
		// Capability "reflog-disabled" (no separate code per
		// AC-5 body line 350).
		//
		// Tests:
		// TestForcePushOrphan_AC5_Matrix (integration; 6 cells
		// + reflog-disabled-AC-3-composition; cell 5 ack
		// composition deferred — see test file comment).
		//
		// AC-9 (G-0210) consolidates the matrix into a fuller
		// cell set; this single cell satisfies the M-0158/AC-6
		// drift invariant in the interim.
		{
			ID:                "branch-cell-isolation-escape-orphaned-ai-commit",
			Preconditions:     []spec.Predicate{{Subject: "force-push-orphans-ai-commit", Op: "==", Value: "true"}},
			Outcome:           spec.OutcomeIllegal,
			ExpectedErrorCode: "isolation-escape-orphaned-ai-commit",
			RejectionLayer:    spec.RejectionLayerCheckTime,
			BlockingStrict:    false, // warning severity per AC-5 body / M-0125 ratchet
			Sources:           spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-isolation-escape-rename-survival —
		// M-0161/AC-6 (G-0206): the SHA-based scope-branch
		// resolution. `aiwf authorize --branch <name>` records
		// `aiwf-branch-sha: <sha>` (the bound branch's tip SHA
		// at scope-open). BranchOracle.BranchOfSHA(sha) resolves
		// the current ritual branch where the SHA appears
		// closest to the tip, preferring ritual-shape branches
		// over trunk. A `git branch -m` rename is transparent
		// to the rule.
		//
		// Closure scope (honest): POST-AC-6 authorize scopes
		// only. Pre-AC-6 ("legacy") authorize commits lack the
		// SHA trailer and continue to use name-only resolution
		// per the documented carve-out at G-0225 (future
		// `aiwf scope rebind` verb).
		//
		// Tests: TestBranchOracle_AC6_RenameResolution_Matrix
		// (integration; 9 cells covering rename, squat collision
		// via orphan lineage, branch deletion, legacy carve-out).
		//
		// AC-9 (G-0210) consolidates the matrix; this single
		// cell satisfies M-0158/AC-6 ClassBranchChoreography
		// drift in the interim.
		//
		// Note: this cell registers a LEGAL outcome (silent on
		// the rename-survival path) — distinct from the other
		// AC-3/4/5 cells which register illegal+finding-code.
		// AC-6 doesn't introduce a new code; the closure is via
		// avoiding a false-positive on the existing
		// isolation-escape code. The drift policy
		// (M-0158/AC-6) checks ClassBranchChoreography codes
		// against ExpectedErrorCode references; isolation-escape
		// already has a cell (branch-cell-4), so this cell
		// stays Legal and documents the SHA-survival contract.
		{
			ID:            "branch-cell-isolation-escape-rename-survival",
			Preconditions: []spec.Predicate{{Subject: "aiwf-branch-sha-trailer-present", Op: "==", Value: "true"}, {Subject: "branch-renamed-since-scope-open", Op: "==", Value: "true"}, {Subject: "sha-resolves-via-oracle", Op: "==", Value: "true"}},
			Outcome:       spec.OutcomeLegal,
			Sources:       spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-detached-head-preflight — M-0161/AC-7
		// (G-0207): the authorize verb refines its rung-pair /
		// branch-context refusal text to explicitly name
		// "detached HEAD has no ritual context" when the current
		// checkout is detached. No new code introduced; the
		// existing `rung-pair-illegal` (current=non-ritual,
		// target=epic|milestone|patch) and `branch-context-
		// required` refusal paths continue but with refined text.
		// The doctor verb additionally surfaces a `head:
		// detached-head: advisory ...` line on detached HEAD so
		// operators discover the state proactively.
		//
		// Tests:
		// TestDetachedHEAD_AC7_PreflightRefusesWithRefinedMessage
		// (integration; substring match on stderr per AC-7 body
		// line 498 substring-exception),
		// TestDetachedHEAD_AC7_PreflightForceReasonBypasses,
		// TestDetachedHEAD_AC7_CheckSucceedsNoFalseFindings,
		// TestDetachedHEAD_AC7_DoctorSurfacesAdvisory,
		// TestDetachedHEAD_AC7_DoctorSilentOnAttachedHEAD.
		//
		// The cell registers as an Illegal cell pointing at
		// rung-pair-illegal (the code that actually fires for
		// detached HEAD on the typical AI-target path). The
		// refined message text is what AC-7 ships; the code
		// identity is unchanged.
		{
			ID:                "branch-cell-detached-head-preflight",
			Verb:              "authorize",
			Preconditions:     []spec.Predicate{{Subject: "target-agent-role", Op: "==", Value: "ai"}, {Subject: "head-detached", Op: "==", Value: "true"}, {Subject: "force", Op: "==", Value: "false"}},
			Outcome:           spec.OutcomeIllegal,
			ExpectedErrorCode: "rung-pair-illegal",
			RejectionLayer:    spec.RejectionLayerVerbTime,
			BlockingStrict:    true,
			Sources:           spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-promote-on-wrong-branch — M-0161/AC-8
		// (G-0209 partial-close): the new check-time finding
		// fires when an activating-promote commit (epic→active,
		// milestone→in_progress) lands on a branch other than
		// the entity's expected parent branch per ADR-0010.
		// Composes with AC-1 (trunk name), AC-3 (BranchOracle
		// + fail-shut), and acknowledge-illegal / aiwf-force
		// per-commit overrides.
		//
		// Honest closure scope: partially closes G-0209 — only
		// the promote-side ordering. The authorize-side
		// implicit-current path (operator on epic/E-NN
		// authorizes E-NN scope WITHOUT --branch) rides
		// M-0103/M-0105's existing carve-outs that are
		// load-bearing for legitimate ritual flows; AC-8
		// deliberately leaves that residual case as operator-
		// discipline per the AC-8 body line 524-526.
		//
		// Tests:
		// TestPromoteOnWrongBranch_AC8_Matrix
		// (integration; 9 cells: 2 silent baselines, 4 firing
		// cases, 1 non-activating silent, 2 sovereign overrides).
		// Plus unit-level coverage at
		// internal/check/promote_on_wrong_branch_test.go.
		//
		// AC-9 (G-0210) consolidates the matrix.
		{
			ID:                "branch-cell-promote-on-wrong-branch",
			Preconditions:     []spec.Predicate{{Subject: "activating-promote-on-wrong-branch", Op: "==", Value: "true"}},
			Outcome:           spec.OutcomeIllegal,
			ExpectedErrorCode: "promote-on-wrong-branch",
			RejectionLayer:    spec.RejectionLayerCheckTime,
			BlockingStrict:    false, // warning severity per AC-8 body / M-0125 ratchet
			Sources:           spec.RuleSource{Decision: "ADR-0010"},
		},
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}
