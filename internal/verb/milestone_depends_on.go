package verb

import (
	"context"
	"fmt"
	"slices"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
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

	// Same-state convergence (M-0281/AC-7): the list already reads exactly as
	// requested. Without this guard the verb wrote byte-identical content and
	// still landed a commit: Apply's empty-plan guard only refuses a plan with
	// ZERO Ops, and this plan has one write Op, while `git commit-tree` has no
	// same-tree refusal — so every re-run appended an empty-diff commit.
	//
	// Compared with slices.Equal, order included: `--on` is
	// replace-not-append, so a reordered list is a real change to the stored
	// sequence and still commits. Placed after the dep-resolution loop above,
	// so a bogus `--on` id is still refused rather than silently converged.
	if slices.Equal(e.DependsOn, modified.DependsOn) {
		return &Result{NoOp: true, NoOpMessage: dependsOnNoOpMessage(id, clearList)}, nil
	}

	body, err := readBody(t.Root, e.Path)
	if err != nil {
		return nil, err
	}
	canonID := entity.Canonicalize(id)
	subject := fmt.Sprintf("aiwf milestone depends-on %s", canonID)
	return planEntityWrite(t, &modified, e.Path, body, entityWrite{
		subject:  subject,
		body:     reason,
		trailers: standardTrailers("milestone-depends-on", canonID, actor),
		metadata: map[string]any{"entity_id": canonID, "depends_on": modified.DependsOn},
	})
}

// dependsOnNoOpMessage renders the same-state message for the two arms:
// re-declaring an identical list, and clearing an already-empty one.
func dependsOnNoOpMessage(id string, clearList bool) string {
	if clearList {
		return fmt.Sprintf("%s has no depends_on edges; nothing to clear", id)
	}
	return fmt.Sprintf("%s depends_on already reads exactly as requested; nothing to change", id)
}
