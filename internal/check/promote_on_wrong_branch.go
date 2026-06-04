package check

import (
	"fmt"
	"slices"
	"strings"

	codespkg "github.com/23min/aiwf/internal/codes"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/scope"
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
//   - `aiwf acknowledge-illegal <sha> --reason "..."` silences
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
// RunProvenance convention). oracle supplies per-commit branch
// info; a nil oracle silences the rule (graceful degradation
// matching the M-0106 isolation-escape pattern).
//
// ackedSHAs honors M-0159/AC-3 acknowledgments via the shared
// per-SHA exemption.
func RunPromoteOnWrongBranch(commits []scope.Commit, expectedBranches map[string]string, oracle BranchOracle, ackedSHAs map[string]bool) []Finding {
	if oracle == nil {
		return nil
	}
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
		expected, hasExpectation := expectedBranches[entityID]
		if !hasExpectation || expected == "" {
			continue // No expectation — gap entity, non-ritual kind, or parent lookup failed (fail-shut).
		}
		actualBranches := oracle.FirstParentBranches(c.SHA)
		if len(actualBranches) == 0 {
			continue // Unknown branch — do not fire on a commit the oracle can't classify (matches isolation-escape's fail-shut).
		}
		if slices.Contains(actualBranches, expected) {
			continue // Correct branch — silent.
		}
		findings = append(findings, Finding{
			Code:     CodePromoteOnWrongBranch.ID,
			Severity: SeverityWarning,
			Message: fmt.Sprintf(
				"commit %s: aiwf promote %s -> %s landed on %q, not the expected parent branch %q",
				shortHash(c.SHA), entityID, targetStatus, strings.Join(actualBranches, ","), expected,
			),
			Hint: fmt.Sprintf(
				"the ADR-0010 branch model requires sovereign activating-promote commits on the parent branch (%q) BEFORE the ritual branch is cut. If the order was deliberate (re-activating from a ritual branch, or rebuilding a historical scope), silence with `aiwf acknowledge-illegal %s --reason \"...\"`; or amend the promote commit with `git commit --amend --trailer 'aiwf-force: <reason>'` as a sovereign per-commit override.",
				expected, shortHash(c.SHA),
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
