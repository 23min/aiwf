package verb

import (
	"context"
	"fmt"
	"strings"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
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
//   - a no-op (already set to <level>, or --clear on an already-unset
//     entity) refuses.
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
	_ = ctx

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

	// No-op refusals: nothing to change.
	if !clearTag && e.Priority == level {
		return nil, fmt.Errorf("%s priority is already set to %q; nothing to change", id, level)
	}
	if clearTag && e.Priority == "" {
		return nil, fmt.Errorf("%s priority is already unset; nothing to clear", id)
	}

	modified := *e
	if clearTag {
		modified.Priority = ""
	} else {
		modified.Priority = level
	}

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
		Subject: subject,
		Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "set-priority"},
			{Key: gitops.TrailerEntity, Value: canonID},
			{Key: gitops.TrailerActor, Value: actor},
		},
		Ops: []FileOp{{Type: OpWrite, Path: e.Path, Content: content}},
	})
	result.Metadata = map[string]any{"entity_id": canonID, "priority": modified.Priority}
	return result, nil
}
