package verb

import (
	"github.com/23min/aiwf/internal/aiwfyaml"
	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/contractcheck"
	"github.com/23min/aiwf/internal/tree"
)

// contractMutationGate is the shared validation gate for the four
// contract-mutating verbs (D-0041): it computes the findings a
// projected mutation introduces as a before/after diff of
// contractcheck.Run's output, rather than filtering by entity id — the
// shape ContractBind previously used, which cannot scope findings for
// verbs (RecipeInstall, RecipeRemove) that mutate the validators map
// and have no single bound entity id to filter on.
//
// A finding present in both the current and next runs is a
// pre-existing issue on an entry the mutation didn't touch and is
// excluded. A finding present only in the next run was introduced by
// the mutation and is returned. A finding present only in the current
// run was resolved by the mutation and is not reported — the gate
// reports only additions, never removals.
//
// The diff is multiset-based (a duplicate finding surviving unchanged
// from current to next is matched and excluded once per occurrence),
// keyed on identity fields (Code/Severity/Subcode/EntityID/Path)
// rather than the full struct: contractcheck.Run's Message embeds the
// finding's positional index within contracts.Entries, so removing or
// inserting an earlier entry shifts every later entry's index and
// changes its Message text even though nothing about that entry
// changed. Diffing on the full struct would then treat a merely
// reindexed finding as newly introduced.
func contractMutationGate(t *tree.Tree, current, next *aiwfyaml.Contracts, repoRoot string) []check.Finding {
	before := contractcheck.Run(t, current, repoRoot)
	after := contractcheck.Run(t, next, repoRoot)
	return diffIntroducedFindings(before, after)
}

// findingIdentity is the subset of check.Finding's fields that
// identify *what* a finding is about, excluding the derived/
// positional fields (Message, Line, Hint, Field) that can vary
// between two runs without the underlying issue actually changing.
type findingIdentity struct {
	Code     string
	Severity check.Severity
	EntityID string
	Subcode  string
	Path     string
}

func identityOf(f check.Finding) findingIdentity {
	return findingIdentity{Code: f.Code, Severity: f.Severity, EntityID: f.EntityID, Subcode: f.Subcode, Path: f.Path}
}

// diffIntroducedFindings returns the findings in after that are not
// already accounted for by a matching occurrence in before — a
// multiset difference over finding identity, kept as its own pure
// function so the diff algorithm is testable independent of
// contractcheck.Run.
func diffIntroducedFindings(before, after []check.Finding) []check.Finding {
	seen := make(map[findingIdentity]int, len(before))
	for i := range before {
		seen[identityOf(before[i])]++
	}

	var introduced []check.Finding
	for i := range after {
		id := identityOf(after[i])
		if seen[id] > 0 {
			seen[id]--
			continue
		}
		introduced = append(introduced, after[i])
	}
	return introduced
}
