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
// since check.Finding carries no identity beyond its own fields.
func contractMutationGate(t *tree.Tree, current, next *aiwfyaml.Contracts, repoRoot string) []check.Finding {
	before := contractcheck.Run(t, current, repoRoot)
	after := contractcheck.Run(t, next, repoRoot)
	return diffIntroducedFindings(before, after)
}

// diffIntroducedFindings returns the findings in after that are not
// already accounted for by a matching occurrence in before — a
// multiset difference, kept as its own pure function so the diff
// algorithm is testable independent of contractcheck.Run.
func diffIntroducedFindings(before, after []check.Finding) []check.Finding {
	seen := make(map[check.Finding]int, len(before))
	for _, f := range before {
		seen[f]++
	}

	var introduced []check.Finding
	for _, f := range after {
		if seen[f] > 0 {
			seen[f]--
			continue
		}
		introduced = append(introduced, f)
	}
	return introduced
}
