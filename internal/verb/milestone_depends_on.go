package verb

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/23min/ai-workflow-v2/internal/check"
	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/gitops"
	"github.com/23min/ai-workflow-v2/internal/tree"
)

// MilestoneDependsOn writes the depends_on frontmatter array on a
// milestone. Closes the post-allocation half of G-072 (the create-time
// half is the --depends-on flag on `aiwf add milestone`).
//
// Two modes, dispatched on `clear`:
//
//   - clear == false: replace-not-append. The supplied `deps` list
//     becomes the milestone's depends_on. To add a single dependency
//     to an existing list, the caller passes the full updated list.
//   - clear == true: empty the list. `deps` must be empty (the mutex
//     is enforced by the dispatcher; this verb pins the contract).
//
// Both modes emit one OpWrite with `aiwf-verb: milestone-depends-on`
// trailers, producing the kernel's per-mutation atomicity guarantee.
//
// Each id in `deps` must resolve to an existing milestone; the verb
// refuses before the commit otherwise. Cycle detection stays at
// `aiwf check`'s layer — different concern, different chokepoint.
//
// Forward-compatibility note: the verb shape `aiwf milestone
// depends-on M-NNN --on <ids>` is a clean subset of the future
// `aiwf <kind> depends-on <id> --on <ids>` cross-kind generalisation
// (G-073). The verb-name segment "milestone" is the *kind*; the
// generalisation extends to other kinds without renaming this verb.
//
// reason is optional free-form prose; when non-empty it lands in the
// commit body so the rationale surfaces in `aiwf history`.
func MilestoneDependsOn(ctx context.Context, t *tree.Tree, id string, deps []string, clearList bool, actor, reason string) (*Result, error) {
	_ = ctx
	if entity.IsCompositeID(id) {
		return nil, fmt.Errorf("milestone depends-on does not accept composite ids; pass a milestone id (M-NNN)")
	}
	if clearList && len(deps) > 0 {
		return nil, fmt.Errorf("--clear and --on are mutually exclusive")
	}
	if !clearList && len(deps) == 0 {
		return nil, fmt.Errorf("milestone depends-on requires --on <id,id,...> or --clear")
	}

	e := t.ByID(id)
	if e == nil {
		return nil, fmt.Errorf("milestone %q not found", id)
	}
	if e.Kind != entity.KindMilestone {
		return nil, fmt.Errorf("%q is of kind %s, not milestone", id, e.Kind)
	}

	for _, dep := range deps {
		if dep == id {
			return nil, fmt.Errorf("--on %q is the milestone itself; a milestone cannot depend on itself", dep)
		}
		ref := t.ByID(dep)
		if ref == nil {
			return nil, fmt.Errorf("--on %q does not resolve to an existing entity", dep)
		}
		if ref.Kind != entity.KindMilestone {
			return nil, fmt.Errorf("--on %q is of kind %s, not milestone (depends_on edges are milestone→milestone only)", dep, ref.Kind)
		}
	}

	modified := *e
	if clearList {
		modified.DependsOn = nil
	} else {
		modified.DependsOn = append([]string(nil), deps...)
	}

	body, err := readBody(t.Root, e.Path)
	if err != nil {
		return nil, err
	}
	content, err := entity.Serialize(&modified, body)
	if err != nil {
		return nil, fmt.Errorf("serializing %s: %w", id, err)
	}

	proj := projectReplace(t, &modified, filepath.ToSlash(e.Path))
	if fs := projectionFindings(t, proj); check.HasErrors(fs) {
		return findings(fs), nil
	}

	subject := fmt.Sprintf("aiwf milestone depends-on %s", id)
	return plan(&Plan{
		Subject: subject,
		Body:    reason,
		Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "milestone-depends-on"},
			{Key: gitops.TrailerEntity, Value: id},
			{Key: gitops.TrailerActor, Value: actor},
		},
		Ops: []FileOp{{Type: OpWrite, Path: e.Path, Content: content}},
	}), nil
}
