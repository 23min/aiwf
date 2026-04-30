package verb

import (
	"path/filepath"
	"strings"

	"github.com/23min/ai-workflow-v2/tools/internal/check"
	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
)

// pathInside reports whether the repo-relative path p is the directory
// dir or lives somewhere underneath it. Comparison is forward-slash so
// callers don't need to normalize.
func pathInside(p, dir string) bool {
	p = filepath.ToSlash(p)
	dir = filepath.ToSlash(dir)
	if p == dir {
		return true
	}
	return strings.HasPrefix(p, dir+"/")
}

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
		return "proposed"
	}
	return ""
}

// projectAdd returns a new tree value that includes e alongside all of
// t's existing entities. plannedPaths lists repo-relative
// (forward-slash) paths that the verb plans to write but hasn't yet,
// so disk-consulting checks can treat them as present. The original
// tree is not mutated.
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

// projectionFindings returns the findings introduced by going from
// `original` to `projected`: any finding present on `projected` whose
// equivalent does not appear on `original` is considered "introduced
// by this verb." Pre-existing tree problems unrelated to the verb's
// change do not block it; the user can see them via `aiwf check`.
//
// Equivalence is by code + subcode + path + entity-id + message.
// That's strict enough that "same kind of problem on a different
// entity" is treated as a new finding (which is the right call:
// adding an entity that triggers a new ids-unique conflict, even when
// the tree already had unrelated ids-unique conflicts, is still the
// verb's responsibility).
func projectionFindings(original, projected *tree.Tree) []check.Finding {
	pre := check.Run(original, nil)
	post := check.Run(projected, nil)
	seen := make(map[string]bool, len(pre))
	for i := range pre {
		seen[findingKey(&pre[i])] = true
	}
	var introduced []check.Finding
	for i := range post {
		if !seen[findingKey(&post[i])] {
			introduced = append(introduced, post[i])
		}
	}
	return introduced
}

func findingKey(f *check.Finding) string {
	return f.Code + "|" + f.Subcode + "|" + f.Path + "|" + f.EntityID + "|" + f.Message
}
