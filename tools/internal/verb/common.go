package verb

import (
	"github.com/23min/ai-workflow-v2/tools/internal/check"
	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
)

// initialStatus is the status `aiwf add` assigns to a freshly-created
// entity. Each kind starts at the leftmost state of its FSM.
func initialStatus(k entity.Kind) string {
	switch k {
	case entity.KindEpic:
		return "proposed"
	case entity.KindMilestone:
		return "draft"
	case entity.KindADR:
		return "proposed"
	case entity.KindGap:
		return "open"
	case entity.KindDecision:
		return "proposed"
	case entity.KindContract:
		return "draft"
	}
	return ""
}

// projectAdd returns a new tree value that includes e alongside all of
// t's existing entities. plannedPaths lists repo-relative
// (forward-slash) paths that the verb plans to write but hasn't yet,
// so checks like contract-artifact-exists can treat them as present.
// The original tree is not mutated.
func projectAdd(t *tree.Tree, e *entity.Entity, plannedPaths ...string) *tree.Tree {
	proj := *t
	proj.Entities = make([]*entity.Entity, len(t.Entities), len(t.Entities)+1)
	copy(proj.Entities, t.Entities)
	proj.Entities = append(proj.Entities, e)
	proj.PlannedFiles = withPlanned(t.PlannedFiles, plannedPaths)
	return &proj
}

// projectReplace returns a new tree value where the entity matching
// modified.ID is replaced with modified. If the id is not present,
// projectReplace returns the original tree unchanged.
func projectReplace(t *tree.Tree, modified *entity.Entity, plannedPaths ...string) *tree.Tree {
	proj := *t
	proj.Entities = make([]*entity.Entity, len(t.Entities))
	for i, e := range t.Entities {
		if e.ID == modified.ID {
			proj.Entities[i] = modified
			continue
		}
		proj.Entities[i] = e
	}
	proj.PlannedFiles = withPlanned(t.PlannedFiles, plannedPaths)
	return &proj
}

// withPlanned merges existing planned paths with new ones into a fresh
// map. Returns nil only when both inputs are empty.
func withPlanned(existing map[string]struct{}, additions []string) map[string]struct{} {
	if len(existing) == 0 && len(additions) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(existing)+len(additions))
	for k := range existing {
		out[k] = struct{}{}
	}
	for _, p := range additions {
		out[p] = struct{}{}
	}
	return out
}

// validateProjection runs all checks against an in-memory projected
// tree. There are no load errors to surface (the projection lives
// only in memory).
func validateProjection(t *tree.Tree) []check.Finding {
	return check.Run(t, nil)
}
