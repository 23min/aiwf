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

	// Id comparisons run at canonical width. The grammar accepts narrower
	// legacy spellings (`M-002` for `M-0002`) and ByID canonicalizes before
	// matching, so a narrow spelling of the milestone's own id would otherwise
	// slip past the self-edge refusal that ByID then resolves.
	canonID := entity.Canonicalize(id)
	for _, dep := range deps {
		if entity.Canonicalize(dep) == canonID {
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
		// Stored as given, matching the verbatim convention Add documents
		// for this same field. Width normalization across the tree is
		// `aiwf rewidth`'s job, not a side effect of an edge declaration.
		modified.DependsOn = append([]string(nil), deps...)
	}

	if claimErr := guardClaim(ctx, t.Root, id, e.Path); claimErr != nil {
		return nil, claimErr
	}

	// Same-state convergence (M-0281/AC-7): the list already reads exactly as
	// requested. Without this guard a re-run writes byte-identical content and
	// still lands a commit with an empty diff (see
	// verb_result_noop_invariant.go for why aiwf does not reject one).
	//
	// Compared with slices.Equal, order included: `--on` is
	// replace-not-append, so a reordered list is a real change to the stored
	// sequence and still commits. Placed after the dep-resolution loop above,
	// so a bogus `--on` id is still refused rather than silently converged.
	// Both sides are normalized to canonical width for the comparison only, so
	// a narrow argument naming the stored entity converges instead of
	// re-writing the list to a different spelling of the same edges.
	if slices.Equal(canonicalIDs(e.DependsOn), canonicalIDs(modified.DependsOn)) {
		return &Result{NoOp: true, NoOpMessage: dependsOnNoOpMessage(id, clearList)}, nil
	}

	body, err := readBody(t.Root, e.Path)
	if err != nil {
		return nil, err
	}
	subject := fmt.Sprintf("aiwf milestone depends-on %s", canonID)
	return planEntityWrite(t, &modified, e.Path, body, entityWrite{
		subject:  subject,
		body:     reason,
		trailers: standardTrailers("milestone-depends-on", canonID, actor),
		metadata: map[string]any{"entity_id": canonID, "depends_on": modified.DependsOn},
	})
}

// canonicalIDs returns ids rewritten to canonical width, so a stored list
// written before the width migration compares equal to a freshly canonicalized
// one. Returns nil for an empty input, which keeps slices.Equal's nil/empty
// equivalence intact for the cleared-list arm.
func canonicalIDs(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = entity.Canonicalize(id)
	}
	return out
}

// dependsOnNoOpMessage renders the same-state message for the two arms:
// re-declaring an identical list, and clearing an already-empty one.
func dependsOnNoOpMessage(id string, clearList bool) string {
	if clearList {
		return fmt.Sprintf("%s has no depends_on edges; nothing to clear", id)
	}
	return fmt.Sprintf("%s depends_on already reads exactly as requested; nothing to change", id)
}
