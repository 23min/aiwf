package verb

import (
	"context"
	"fmt"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// SetArea points a single entity at an existing declared area member —
// or clears its area tag — in one trailered commit (E-0044, M-0183). It
// is the membership-change sibling of RenameArea: where RenameArea
// rewrites the *vocabulary* (a member's name, carrying every referrer),
// SetArea changes one entity's *membership* against a fixed vocabulary.
//
// It is the guaranteed remediation for `areas.required` (M-0178): when
// the knob flags an untagged entity, `aiwf set-area <id> <member>` is
// the one-command unblock. It also owns the inverse — `aiwf set-area
// <id> --clear` untags an entity back to the untagged state (legitimate
// and never-flagged unless `areas.required` is set, under which an
// untagged entity is itself flagged by area-required), the clean
// correction for a mis-tag.
//
// Two modes, dispatched on `clear`:
//
//   - clear == false: set the entity's area to <member>. <member> must be
//     a declared member of `members` (the validated single source of
//     truth the CLI passes from config.Load); the verb never invents a
//     member (that is RenameArea's and config's job). With no areas block
//     declared, `members` is empty and every member is undeclared, so a
//     set refuses.
//   - clear == true: empty the area field. `omitempty` on entity.Area
//     drops the cleared key on serialize, so the on-disk frontmatter
//     returns to the untagged shape byte-for-byte.
//
// Validation precedes any Plan, so a refusal writes nothing:
//   - a composite/AC id or a milestone target refuses — area derives from
//     the parent epic; the message names the epic and the remediation
//     command;
//   - an unknown id refuses;
//   - <member> and clear given together refuse (mutex);
//   - a non-empty <member> not in `members` refuses, naming the declared
//     set;
//   - a no-op (already tagged <member>, or --clear on an already-untagged
//     entity) refuses.
//
// The commit carries `aiwf-verb: set-area`, `aiwf-entity: <canonical id>`,
// and `aiwf-actor:`. The verb trailer suppresses the
// `provenance-untrailered-entity-commit` audit a hand-edit would trip —
// the whole point of the verb, for tag, retag, AND untag.
//
// What undoes this? The same verb, total: a tag (untagged→tagged)
// reverses with `--clear`; a retag reverses with the prior member; a
// `--clear` reverses by setting the prior member. One verb owns one field
// with a complete reversal story.
func SetArea(
	ctx context.Context,
	t *tree.Tree,
	members []string,
	id, member string,
	clearTag bool,
	actor string,
) (*Result, error) {
	_ = ctx

	// Composite/AC ids resolve to their parent milestone; both a
	// composite id and a bare milestone derive their area from the parent
	// epic and so are refused with a remediation pointer at the epic.
	lookupID := id
	if entity.IsCompositeID(id) {
		lookupID = entity.CompositeRoot(id)
	}
	e := t.ByID(lookupID)
	if e == nil {
		return nil, fmt.Errorf("unknown id %q", id)
	}
	if entity.IsCompositeID(id) || !entity.CarriesOwnArea(e.Kind) {
		return nil, fmt.Errorf(
			"%s derives its area from parent epic %s; run: aiwf set-area %s %s",
			id, e.Parent, e.Parent, areaArgHint(member, clearTag),
		)
	}

	if member != "" && clearTag {
		return nil, fmt.Errorf("--clear and <member> are mutually exclusive")
	}

	if !clearTag {
		declared := false
		for _, m := range members {
			if m == member {
				declared = true
				break
			}
		}
		if !declared {
			return nil, fmt.Errorf("area %q is not a declared member; declared areas: %s", member, declaredList(members))
		}
	}

	// No-op refusals: nothing to change.
	if !clearTag && e.Area == member {
		return nil, fmt.Errorf("%s is already tagged %q; nothing to change", id, member)
	}
	if clearTag && e.Area == "" {
		return nil, fmt.Errorf("%s is already untagged; nothing to clear", id)
	}

	modified := *e
	if clearTag {
		modified.Area = ""
	} else {
		modified.Area = member
	}

	body, err := readBody(t.Root, e.Path)
	if err != nil {
		return nil, err
	}
	content, err := entity.Serialize(&modified, body)
	if err != nil { //coverage:ignore yaml.Marshal of a loaded, valid Entity does not fail; defensive, mirrors the renamearea serialize path
		return nil, fmt.Errorf("serializing %s after area change: %w", e.ID, err)
	}

	canonID := entity.Canonicalize(id)
	subject := fmt.Sprintf("aiwf set-area %s %s", canonID, member)
	if clearTag {
		subject = fmt.Sprintf("aiwf set-area %s --clear", canonID)
	}
	return plan(&Plan{
		Subject: subject,
		Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "set-area"},
			{Key: gitops.TrailerEntity, Value: canonID},
			{Key: gitops.TrailerActor, Value: actor},
		},
		Ops: []FileOp{{Type: OpWrite, Path: e.Path, Content: content}},
	}), nil
}

// areaArgHint renders the placeholder for the remediation command in a
// milestone/composite refusal: `--clear` when the caller asked to untag,
// otherwise `<member>` (or the supplied member when one was given).
func areaArgHint(member string, clearTag bool) string {
	if clearTag {
		return "--clear"
	}
	if member != "" {
		return member
	}
	return "<member>"
}
