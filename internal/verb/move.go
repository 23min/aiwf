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
// milestone, target epic missing or wrong kind. A milestone already
// under the target epic is not an error — it converges to a NoOp
// (M-0281/AC-3). Tree-level findings caused by the move (e.g. a
// depends_on cycle introduced by the new neighborhood) are returned in
// Result.Findings.
func Move(ctx context.Context, t *tree.Tree, id, newEpicID, actor string) (*Result, error) {
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
	// Resolve the target epic to canonical width for the comparison below.
	// Parsers accept narrower legacy spellings on input — ByID canonicalizes
	// both sides before matching — so `--epic E-01` names the same epic as a
	// stored `E-0001`. Canonicalizing here settles only the comparison: a
	// genuine move still stores the operator's spelling verbatim, matching
	// what Add writes for a parent supplied at creation.
	canonNew := entity.Canonicalize(newEpicID)

	source := filepath.ToSlash(e.Path)
	dest := filepath.ToSlash(filepath.Join(filepath.Dir(target.Path), filepath.Base(e.Path)))

	if claimErr := guardClaim(ctx, t.Root, id, source); claimErr != nil {
		return nil, claimErr
	}

	// Same-state convergence (M-0281/AC-3). A move's effect spans two surfaces
	// — the `parent:` field and the file's location under the epic's directory
	// — so both must already hold before there is nothing to relocate. move is
	// a field-mutation verb (no FSM transition), so this needs no ADR-0036
	// oracle changes.
	if entity.Canonicalize(e.Parent) == canonNew && source == dest {
		return &Result{NoOp: true, NoOpMessage: fmt.Sprintf("%s is already under epic %q; nothing to move", id, newEpicID)}, nil
	}

	modified := *e
	priorParent := e.Parent
	modified.Parent = newEpicID
	modified.Path = dest

	body, err := readBody(t.Root, e.Path)
	if err != nil {
		return nil, err
	}
	// ADR-0046: the milestone's own relative destinations resolve against
	// the directory it sits in, so moving it between epics changes what
	// they name. Recomputed against the destination before serialization,
	// so the file's single write carries both the new `parent:` and the
	// repaired links.
	moves := []EntityMove{{From: source, To: dest}}
	body = RewriteLinkDestinationsForMove(body, source, dest, moves)

	content, err := entity.Serialize(&modified, body)
	if err != nil {
		return nil, fmt.Errorf("serializing %s: %w", id, err)
	}

	proj := projectReplace(t, &modified, dest)
	if fs := projectionFindings(t, proj); check.HasErrors(fs) {
		return findings(fs), nil
	}

	// ADR-0033: every verb emitting an OpMove repairs the inbound links
	// pointing at what it moved. The moved milestone is excluded because
	// move already writes that file to update `parent:` and repair its own
	// outbound links; letting the helper emit a second write for the same
	// path would put two competing OpWrites in one plan.
	rewriteOps, err := planLinkRewriteWrites(t, moves, map[string]bool{e.Path: true})
	if err != nil { //coverage:ignore defensive: planLinkRewriteWrites only errors on a vanished file or an unserializable entity — neither reachable from a tree the loader just built
		return nil, err
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
		Ops: append([]FileOp{
			{Type: OpMove, Path: source, NewPath: dest},
			{Type: OpWrite, Path: dest, Content: content},
		}, rewriteOps...),
	})
	result.Metadata = map[string]any{"entity_id": canonID, "from": canonPrior, "to": canonNew}
	return result, nil
}
