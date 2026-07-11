package check

import (
	"fmt"
	"strings"

	codespkg "github.com/23min/aiwf/internal/codes"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/scope"
	"github.com/23min/aiwf/internal/tree"
)

// promote_on_wrong_branch.go — M-0161/AC-8 (G-0209 partial-
// close): the new kernel finding promote-on-wrong-branch.
//
// Per ADR-0010, sovereign activating-promote acts must land on
// the parent branch BEFORE the ritual branch is cut:
//
//   - `aiwf promote E-NNNN active` (epic activation): expected
//     branch is trunk (M-0161/AC-1's Config.TrunkBranchShortName()).
//   - `aiwf promote M-NNNN in_progress` (milestone activation):
//     expected branch is the parent epic's ritual branch
//     (epic/E-XXXX-<slug>).
//   - `aiwf promote G-NNNN active` (gap activation): no branch
//     expectation; gaps don't have ritual-branch-cut semantics.
//
// Non-activating promotes (active → done, in_progress → done,
// ADR proposed → accepted, etc.) are out of the rule's domain.
//
// AC-8 partially closes G-0209: only the promote-side ordering
// is covered. The authorize-side implicit-current path
// (operator on epic/E-NN authorizes E-NN scope without
// --branch) rides M-0103/M-0105's existing carve-outs that are
// load-bearing for legitimate ritual flows. That residual case
// is tracked as operator-discipline per the AC-8 body.

// CodePromoteOnWrongBranch is the warning finding code that
// fires when an activating-promote commit lands on a branch
// other than the entity's expected parent branch (M-0161/AC-8 /
// G-0209). One finding per violating commit; per-SHA dedup is
// not applied because each commit is a distinct activation
// event.
//
// Severity is warning per the M-0125 ratchet pattern. The AC-8
// body's "future D-NNN may tighten to error after one epic of
// usage" path is consistent with M-0106's same trajectory.
//
// Composes with AC-3 fail-shut: if the parent-branch expectation
// can't be computed (parent lookup failed, tree truncated,
// non-ritual entity kind), the rule stays silent rather than
// firing a false positive.
//
// Override paths (shared with AC-5 and AC-6):
//   - `aiwf acknowledge illegal <sha> --reason "..."` silences
//     post-hoc via the shared ackedSHAs map.
//   - `aiwf-force: <reason>` trailer on the promote commit
//     itself silences per-commit (existing override pattern).
var CodePromoteOnWrongBranch = codespkg.Code{ID: "promote-on-wrong-branch", Class: codespkg.ClassBranchChoreography}

// RunPromoteOnWrongBranch applies the AC-8 rule to a commit
// history. expectedBranches maps entity ids to the expected
// parent-branch short name; empty/missing values silence the
// rule for that entity (fail-shut on correctness).
//
// commits must be ordered oldest-first (matches the
// RunProvenance convention). Unlike the original M-0161/AC-8
// design, this rule no longer asks a [BranchOracle] "what
// branches is this commit reachable from" — that question
// requires enumerating and name-filtering local branches, which
// is exactly what silently missed G-0270's incident (the
// activation commit landed on a non-ritual-shaped branch, so the
// oracle's branch enumeration never indexed it, and the rule
// stayed silent on an "unknown branch"). Instead it asks the
// narrower, branch-name-independent question "is this commit
// correctly reachable from the expected branch," via dag,
// branchTips, and trunkShort. Only a nil dag fail-shuts the whole
// rule (no ancestry data at all to test against).
//
// The correctness test is NOT a single ancestor check — plain
// full-ancestry (any path) can't distinguish two very different
// milestone histories once the epic's ritual branch has been
// merged and deleted (its normal end-of-life, per
// aiwfx-wrap-epic): a milestone activation commit correctly made
// on that branch becomes, after the merge, a full ancestor of
// trunk's tip too — arriving via the merge commit's non-first
// parent, never itself on trunk's own lineage. A milestone
// activation that instead skipped the ritual branch and landed
// directly on trunk is ALSO a full ancestor of trunk's tip, but
// IS on trunk's own first-parent chain. The same fixture-verified
// pattern occurs for epics too (aiwfx-wrap-epic's own "promote E-NN
// active" step, run from the epic's own branch just before it is
// merged and deleted — confirmed against this repo's own history).
// One check handles both kinds uniformly:
//
//   - Correct if the commit is an ancestor of expected's own tip
//     (branchTips[expected]) — for an epic this already IS trunk's
//     tip (expected == trunkShort), so this alone covers both
//     "made directly on trunk" and "made on a branch since merged
//     into trunk"; for a milestone this covers the still-live,
//     in-flight epic branch.
//   - Otherwise, for a milestone whose epic branch has since been
//     merged and deleted (absent from branchTips), still correct
//     if the commit is an ancestor of trunk's tip while NOT itself
//     on trunk's own first-parent chain — i.e. it arrived via a
//     merge, not by landing directly on trunk (which would mean
//     the milestone skipped its epic branch entirely).
//
// dag is the shared in-memory commit DAG (check.BuildCommitDAG),
// built once per check invocation. branchTips maps a branch's
// short name (as expectedBranches' values name them — the
// configured trunk short name for epics, "epic/<slug>" for
// milestones) to that branch's current LOCAL tip SHA; an absent
// entry (the branch doesn't exist locally — never cut, or cut and
// since deleted) is not ambiguous on its own — see the two-step
// check above for how it resolves. Comparing against the local
// branch (not a remote-tracking ref) is deliberate: a legitimate,
// correct, not-yet-pushed activation commit is an ancestor of
// local trunk immediately, but would not yet be an ancestor of a
// remote-tracking ref. trunkShort is the configured trunk short
// name, used to resolve trunk's own tip from branchTips
// independently of expected (which for a milestone names the epic
// branch, not trunk).
//
// ackedSHAs honors M-0159/AC-3 acknowledgments via the shared
// per-SHA exemption.
//
// t is the current entity tree, consulted to resolve a commit's
// aiwf-entity: trailer forward through any reallocation before
// the expectedBranches lookup (G-0308). expectedBranches is keyed
// by the *current* tree's ids; a commit that predates a
// reallocate carries the freed id verbatim in its trailer, and
// that id may since have been reclaimed by an unrelated entity.
// Resolving the trailer id through the rename chain (in-window
// aiwf-prior-entity: trailers) and then prior_ids frontmatter
// (out-of-window reallocations) — the same two-step RunProvenance
// uses for authorization-out-of-scope — finds the expectation for
// the commit's *actual* entity instead of the id's current
// claimant. A nil t skips only the prior_ids fallback; in-window
// rename-chain resolution (walkRenameChain) still applies since it
// reads solely from commits.
func RunPromoteOnWrongBranch(commits []scope.Commit, expectedBranches map[string]string, dag *CommitDAG, branchTips map[string]string, trunkShort string, ackedSHAs map[string]bool, t *tree.Tree) []Finding {
	if dag == nil {
		return nil
	}
	// Computed once, reused per commit: the set of SHAs on trunk's
	// OWN first-parent lineage. Membership distinguishes "landed
	// directly on trunk" from "arrived via a merged-in side branch"
	// (see the doc comment above) — the same distinction the old
	// BranchOracle's FirstParentBranches captured implicitly, kept
	// here since full ancestry alone can't tell the two apart.
	var onTrunkFirstParent map[string]bool
	if trunkTip := branchTips[trunkShort]; trunkTip != "" {
		onTrunkFirstParent = make(map[string]bool, len(dag.parents))
		for _, sha := range dag.FirstParentChain(trunkTip) {
			onTrunkFirstParent[sha] = true
		}
	}
	renameChain := buildRenameChain(commits)
	var findings []Finding
	for i := range commits {
		c := &commits[i]
		idx := indexCommitTrailersForProvenance(c.Trailers)
		if idx[gitops.TrailerVerb] != "promote" {
			continue
		}
		entityID := idx[gitops.TrailerEntity]
		targetStatus := idx[gitops.TrailerTo]
		if !isActivatingPromoteTransition(entityID, targetStatus) {
			continue
		}
		// Per-commit force suppresses (existing override
		// pattern shared with M-0106).
		if idx[gitops.TrailerForce] != "" {
			continue
		}
		if ackedSHAs[c.SHA] {
			continue
		}
		resolvedID := resolveViaPriorIDs(walkRenameChain(entityID, renameChain), t)
		expected, hasExpectation := expectedBranches[resolvedID]
		if !hasExpectation || expected == "" {
			continue // No expectation — gap entity, non-ritual kind, or parent lookup failed (fail-shut).
		}
		// For an epic, expected == trunkShort, so this alone already
		// covers "made directly on trunk" and "made on a branch
		// since merged into trunk" — both are full ancestors of
		// trunk's own tip.
		if tip := branchTips[expected]; tip != "" && dag.isAncestor(c.SHA, tip) {
			continue // On the expected branch (or trunk) — correct, silent.
		}
		// The expected branch is gone (never cut, or cut and since
		// merged+deleted). For a milestone whose epic branch was
		// properly merged, the commit is still a full ancestor of
		// trunk's tip — just not on trunk's own first-parent chain,
		// since it arrived via the merge's non-first parent. A
		// milestone that instead skipped its epic branch and landed
		// directly on trunk IS on trunk's own first-parent chain, so
		// this does not silence that case (still fires below).
		if trunkTip := branchTips[trunkShort]; trunkTip != "" && dag.isAncestor(c.SHA, trunkTip) && !onTrunkFirstParent[c.SHA] {
			continue // Reachable from trunk only via a merged-in side branch — correct, silent.
		}
		findings = append(findings, Finding{
			Code:     CodePromoteOnWrongBranch.ID,
			Severity: SeverityWarning,
			Message: fmt.Sprintf(
				"commit %s: aiwf promote %s -> %s is not reachable from the expected parent branch %q",
				shortHash(c.SHA), entityID, targetStatus, expected,
			),
			Hint: fmt.Sprintf(
				"the ADR-0010 branch model requires sovereign activating-promote commits on the parent branch (%q) BEFORE the ritual branch is cut. Find the commit's actual branch with `git branch --contains %s`. If the order was deliberate (re-activating from a ritual branch, or rebuilding a historical scope), silence with `aiwf acknowledge illegal %s --reason \"...\"`; or amend the promote commit with `git commit --amend --trailer 'aiwf-force: <reason>'` as a sovereign per-commit override.",
				expected, shortHash(c.SHA), shortHash(c.SHA),
			),
			EntityID: entityID,
		})
	}
	return findings
}

// isActivatingPromoteTransition reports whether the (entity id,
// target status) pair represents an activating ritual-step
// transition that AC-8 polices. Epic and milestone are the
// only entity kinds with branch-cut semantics today; gaps,
// ADRs, decisions, and contracts have no ritual-branch
// expectations and are out of the rule's domain.
//
// The entity id's leading prefix character disambiguates the
// kind without requiring a tree lookup (the rule stays pure
// per the BranchOracle pattern). E-NNNN → epic; M-NNNN →
// milestone.
func isActivatingPromoteTransition(entityID, targetStatus string) bool {
	if strings.HasPrefix(entityID, "E-") {
		return targetStatus == entity.StatusActive
	}
	if strings.HasPrefix(entityID, "M-") {
		return targetStatus == entity.StatusInProgress
	}
	return false
}
