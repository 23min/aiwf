package verb

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// Move relocates a milestone from its current epic to a different epic.
// The id is preserved (so references in other entities still resolve);
// only the file's location on disk and the milestone's `parent:`
// frontmatter field change. One commit per move with trailers
// `aiwf-verb: move`, `aiwf-entity: <M-id>`, `aiwf-prior-parent: <old-epic>`,
// `aiwf-actor: …` so `aiwf history` can answer "where did this milestone
// come from?" from either the milestone's or the old epic's perspective.
//
// Returns a Go error for "couldn't even start": id not found, kind not
// milestone, target epic missing or wrong kind, milestone already under
// the target epic. Tree-level findings caused by the move (e.g. a
// depends_on cycle introduced by the new neighborhood) are returned in
// Result.Findings.
func Move(ctx context.Context, t *tree.Tree, id, newEpicID, actor string) (*Result, error) {
	_ = ctx
	e := t.ByID(id)
	if e == nil {
		return nil, fmt.Errorf("entity %q not found", id)
	}
	if e.Kind != entity.KindMilestone {
		return nil, fmt.Errorf("only milestones can be moved (entity %q is a %s)", id, e.Kind)
	}
	if newEpicID == "" {
		return nil, fmt.Errorf("--epic <epic-id> is required")
	}
	target := t.ByID(newEpicID)
	if target == nil {
		return nil, fmt.Errorf("target epic %q does not exist", newEpicID)
	}
	if target.Kind != entity.KindEpic {
		return nil, fmt.Errorf("--epic %q is not an epic (it's a %s)", newEpicID, target.Kind)
	}
	// Resolve the target epic to canonical width once, and use it for both
	// the comparison below and the value written to `parent:`. Parsers accept
	// narrower legacy spellings on input — ByID canonicalizes both sides
	// before matching — so `--epic E-01` names the same epic as a stored
	// `E-0001`. Comparing the raw argument missed that convergence, and the
	// miss then wrote the operator's spelling into the frontmatter, degrading
	// a canonical id to legacy width against ADR-0008.
	canonNew := entity.Canonicalize(newEpicID)

	// Same-state convergence (M-0281/AC-3): the milestone is already under
	// the requested epic — there's nothing to relocate — so a re-run
	// converges to a NoOp at exit 0 rather than an error. move is a
	// field-mutation verb (no FSM transition), so this needs no ADR-0036
	// oracle changes.
	if entity.Canonicalize(e.Parent) == canonNew {
		return &Result{NoOp: true, NoOpMessage: fmt.Sprintf("milestone %q is already under epic %q; nothing to move", id, newEpicID)}, nil
	}

	source := filepath.ToSlash(e.Path)
	dest := filepath.ToSlash(filepath.Join(filepath.Dir(target.Path), filepath.Base(e.Path)))

	modified := *e
	priorParent := e.Parent
	modified.Parent = canonNew
	modified.Path = dest

	body, err := readBody(t.Root, e.Path)
	if err != nil {
		return nil, err
	}
	content, err := entity.Serialize(&modified, body)
	if err != nil {
		return nil, fmt.Errorf("serializing %s: %w", id, err)
	}

	proj := projectReplace(t, &modified, dest)
	if fs := projectionFindings(t, proj); check.HasErrors(fs) {
		return findings(fs), nil
	}

	// Canonical width per AC-1 in M-081. canonNew is resolved above, where the
	// same-state comparison needs it.
	canonID := entity.Canonicalize(id)
	canonPrior := entity.Canonicalize(priorParent)
	subject := fmt.Sprintf("aiwf move %s %s -> %s", canonID, canonPrior, canonNew)
	result := plan(&Plan{
		Subject: subject,
		Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "move"},
			{Key: gitops.TrailerEntity, Value: canonID},
			{Key: gitops.TrailerPriorParent, Value: canonPrior},
			{Key: gitops.TrailerActor, Value: actor},
		},
		Ops: []FileOp{
			{Type: OpMove, Path: source, NewPath: dest},
			{Type: OpWrite, Path: dest, Content: content},
		},
	})
	result.Metadata = map[string]any{"entity_id": canonID, "from": canonPrior, "to": canonNew}
	return result, nil
}
