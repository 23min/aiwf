package branch

import (
	"sort"

	"github.com/23min/aiwf/internal/workflows/spec"
)

// Rules returns the layer-4 branch-choreography cells, sorted by cell
// id for deterministic output (M-0158/AC-7). The closed set comprises:
//
//   - 12 corner-case cells `branch-cell-1` through `branch-cell-12`
//     from E-0030 §"Corner cases" (Cycle 2).
//   - 4 override-surface cells `branch-cell-override-<mechanism>`
//     from E-0030 §"Sovereign override surface" (Cycle 3).
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
		// branch-cell-3 — Corner case 3: AI authorize on epic/E-NN-X
		// without --branch, ritual shape matches. Preflight accepts;
		// trailer records current branch. Test:
		// TestAuthorize_Open_AITarget_ImplicitFromCurrent_AcceptsAndEmitsTrailer.
		{
			ID:            "branch-cell-3",
			Verb:          "authorize",
			Preconditions: []spec.Predicate{{Subject: "target-agent-role", Op: "==", Value: "ai"}, {Subject: "ritual-branch-context-present", Op: "==", Value: "true"}},
			Outcome:       spec.OutcomeLegal,
			Sources:       spec.RuleSource{Decision: "ADR-0010"},
		},
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
		// branch-cell-5 — Corner case 5: AI commit on bound branch.
		// Finding silent. Test: TestIsolationEscape_AC4_AICommitOnBoundBranchSilent
		// + TestRunProvenanceCheck_IsolationEscape_SilentOnBoundBranchCommit.
		{
			ID:            "branch-cell-5",
			Preconditions: []spec.Predicate{{Subject: "commit-actor-role", Op: "==", Value: "ai"}, {Subject: "scope-binding-branch", Op: "==", Value: "commit-branch"}},
			Outcome:       spec.OutcomeLegal,
			Sources:       spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-6 — Corner case 6: AI commit on bound branch
		// while scope paused. Finding silent (paused doesn't change
		// binding). Test: TestIsolationEscape_AC5_AICommitOnBoundBranchPausedScopeSilent.
		{
			ID:            "branch-cell-6",
			Preconditions: []spec.Predicate{{Subject: "commit-actor-role", Op: "==", Value: "ai"}, {Subject: "scope-state", Op: "==", Value: "paused"}, {Subject: "scope-binding-branch", Op: "==", Value: "commit-branch"}},
			Outcome:       spec.OutcomeLegal,
			Sources:       spec.RuleSource{Decision: "ADR-0010"},
		},
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
		// branch-cell-8 — Corner case 8: Human cherry-pick of ai/X
		// commit. Finding silent (committer ≠ actor + marker = sovereign
		// re-author). Test: TestIsolationEscape_AC6_CherryPickReAuthorSilent.
		{
			ID:            "branch-cell-8",
			Preconditions: []spec.Predicate{{Subject: "commit-actor-role", Op: "==", Value: "ai"}, {Subject: "committer-differs-from-actor", Op: "==", Value: "true"}, {Subject: "cherry-pick-marker-present", Op: "==", Value: "true"}},
			Outcome:       spec.OutcomeLegal,
			Sources:       spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-9 — Corner case 9: Human merge of epic/E-NN-X
		// into main via --no-ff. Finding silent on the merge (merge
		// commit is human-actor; AI commits behind merge are still
		// reachable from epic branch first-parent, not main's).
		// Test: TestIsolationEscape_AC7_HumanMergeFirstParentSilent.
		{
			ID:            "branch-cell-9",
			Preconditions: []spec.Predicate{{Subject: "commit-verb", Op: "==", Value: "merge"}, {Subject: "commit-actor-role", Op: "==", Value: "human"}},
			Outcome:       spec.OutcomeLegal,
			Sources:       spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-10 — Corner case 10: Sovereign --force amend.
		// Finding silent (aiwf-force trailer + human actor = gated
		// override). Test: TestIsolationEscape_AC8_ForceAmendedCommitSilent.
		{
			ID:            "branch-cell-10",
			Preconditions: []spec.Predicate{{Subject: "aiwf-force-trailer-present", Op: "==", Value: "true"}, {Subject: "commit-actor-role", Op: "==", Value: "human"}},
			Outcome:       spec.OutcomeLegal,
			Sources:       spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-11 — Corner case 11: AI commit on entity with
		// no scope opened. Finding silent (no scope, no binding).
		// Test: TestIsolationEscape_AC9_NoScopeOpenedSilent.
		{
			ID:            "branch-cell-11",
			Preconditions: []spec.Predicate{{Subject: "commit-actor-role", Op: "==", Value: "ai"}, {Subject: "active-scope-on-entity", Op: "==", Value: "false"}},
			Outcome:       spec.OutcomeLegal,
			Sources:       spec.RuleSource{Decision: "ADR-0010"},
		},
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
		// branch-cell-override-cherry-pick — M-0106 cherry-pick
		// suppression: a `git cherry-pick -x` re-author by a human
		// (committer ≠ original ai actor + cherry-pick marker in
		// body) is recognized as sovereign re-author; the
		// isolation-escape finding is silent. Test:
		// TestIsolationEscape_AC6_CherryPickReAuthorSilent.
		// Same shape as branch-cell-8 (the corner case) but registered
		// here as the explicit override-surface entry per the spec body.
		{
			ID:            "branch-cell-override-cherry-pick",
			Preconditions: []spec.Predicate{{Subject: "commit-actor-role", Op: "==", Value: "ai"}, {Subject: "committer-differs-from-actor", Op: "==", Value: "true"}, {Subject: "cherry-pick-marker-present", Op: "==", Value: "true"}},
			Outcome:       spec.OutcomeLegal,
			Sources:       spec.RuleSource{Decision: "ADR-0010"},
		},
		// branch-cell-override-force-amend — M-0106 aiwf-force
		// amend override: amending the violating commit with
		// aiwf-force: <reason> trailer + flipped human/ actor
		// suppresses the isolation-escape finding. Gated by the
		// existing trailer-shape rule (aiwf-force requires human/
		// actor). Test: TestIsolationEscape_AC8_ForceAmendedCommitSilent.
		// Same shape as branch-cell-10 (corner case) registered here
		// as the explicit override entry per the spec body.
		{
			ID:            "branch-cell-override-force-amend",
			Preconditions: []spec.Predicate{{Subject: "aiwf-force-trailer-present", Op: "==", Value: "true"}, {Subject: "commit-actor-role", Op: "==", Value: "human"}},
			Outcome:       spec.OutcomeLegal,
			Sources:       spec.RuleSource{Decision: "ADR-0010"},
		},
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
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}
