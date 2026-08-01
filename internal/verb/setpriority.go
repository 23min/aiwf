package verb

import (
	"context"
	"fmt"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// SetPriority points a single gap or decision at a closed-set priority
// level — or clears its priority tag — in one trailered commit (G-0078,
// E-0066, M-0262). It is the write-surface sibling of SetArea: where
// SetArea validates <member> against a config-declared set, SetPriority
// validates <level> against the fixed Go-hardcoded set
// (entity.IsAllowedPriorityLevel) — the same SSOT predicate the
// priority-valid check rule reads, so there is no parallel value check
// here.
//
// Two modes, dispatched on `clear`:
//
//   - clear == false: set the entity's priority to <level>. <level> must
//     be one of entity.AllowedPriorityLevels().
//   - clear == true: empty the priority field. `omitempty` on
//     entity.Priority drops the cleared key on serialize, so the on-disk
//     frontmatter returns to the unset shape byte-for-byte.
//
// Validation precedes any Plan, so a refusal writes nothing:
//   - an unknown id refuses;
//   - a target whose kind does not carry a priority (!CarriesOwnPriority)
//     refuses — priority is legal only on gap and decision;
//   - <level> and clear given together refuse (mutex);
//   - an out-of-range <level> refuses, naming the allowed set;
//
// A request that is already satisfied — already set to <level>, or
// --clear on an already-unset entity — is not a refusal: it converges to
// a NoOp at exit 0 and writes nothing (M-0281/AC-7).
//
// The commit carries `aiwf-verb: set-priority`, `aiwf-entity: <canonical
// id>`, and `aiwf-actor:`. The verb trailer suppresses the
// `provenance-untrailered-entity-commit` audit a hand-edit would trip —
// the whole point of the verb, for set, reset, AND clear.
//
// What undoes this? The same verb, total: a set (unset->set) reverses
// with --clear; a reset reverses with the prior level; a --clear reverses
// by setting the prior level. One verb owns one field with a complete
// reversal story.
func SetPriority(
	ctx context.Context,
	t *tree.Tree,
	id, level string,
	clearTag bool,
	actor string,
) (*Result, error) {
	e := t.ByID(id)
	if e == nil {
		return nil, fmt.Errorf("unknown id %q", id)
	}
	if !entity.CarriesOwnPriority(e.Kind) {
		return nil, fmt.Errorf(
			"%s (kind=%s) does not carry a priority; priority is legal only on gap and decision entities",
			id, e.Kind,
		)
	}

	if level != "" && clearTag {
		return nil, fmt.Errorf("--clear and <level> are mutually exclusive")
	}

	if !clearTag {
		if !entity.IsAllowedPriorityLevel(level) {
			return nil, fmt.Errorf("priority %q is not a recognized priority level; allowed: %s", level, strings.Join(entity.AllowedPriorityLevels(), ", "))
		}
	}

	if claimErr := guardClaim(ctx, t.Root, id, e.Path); claimErr != nil {
		return nil, claimErr
	}

	// Same-state convergence (M-0281/AC-7): the priority already reads as
	// requested, so a re-run converges to a NoOp at exit 0 rather than an error.
	if !clearTag && e.Priority == level {
		return &Result{NoOp: true, NoOpMessage: fmt.Sprintf("%s priority is already set to %q; nothing to change", id, level)}, nil
	}
	if clearTag && e.Priority == "" {
		return &Result{NoOp: true, NoOpMessage: fmt.Sprintf("%s priority is already unset; nothing to clear", id)}, nil
	}

	modified := *e
	if clearTag {
		modified.Priority = ""
	} else {
		modified.Priority = level
	}

	// Deliberately NOT routed through planEntityWrite: set-priority skips
	// the projection safety-net on purpose. The only checks that read the
	// priority field (priority-valid, priority-not-applicable) are already
	// preempted by this verb's own closed-set and carries-priority guards
	// above, and --clear can introduce no finding (there is no
	// priority-required check). Routing it through the projecting helper
	// would add a provably-inert double check.Run pass for no gain. See
	// the set-area sibling for the case where projection would actively
	// break --clear.
	body, err := readBody(t.Root, e.Path)
	if err != nil {
		return nil, err
	}
	content, err := entity.Serialize(&modified, body)
	if err != nil { //coverage:ignore yaml.Marshal of a loaded, valid Entity does not fail; defensive, mirrors the setarea serialize path
		return nil, fmt.Errorf("serializing %s after priority change: %w", e.ID, err)
	}

	canonID := entity.Canonicalize(id)
	subject := fmt.Sprintf("aiwf set-priority %s %s", canonID, level)
	if clearTag {
		subject = fmt.Sprintf("aiwf set-priority %s --clear", canonID)
	}
	result := plan(&Plan{
		Subject:  subject,
		Trailers: standardTrailers("set-priority", canonID, actor),
		Ops:      []FileOp{{Type: OpWrite, Path: e.Path, Content: content}},
	})
	result.Metadata = map[string]any{"entity_id": canonID, "priority": modified.Priority}
	return result, nil
}
