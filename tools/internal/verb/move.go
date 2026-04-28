package verb

import (
	"fmt"
	"path/filepath"

	"github.com/23min/ai-workflow-v2/tools/internal/check"
	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/gitops"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
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
func Move(t *tree.Tree, id, newEpicID, actor string) (*Result, error) {
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
	if e.Parent == newEpicID {
		return nil, fmt.Errorf("milestone %q is already under epic %q; nothing to move", id, newEpicID)
	}

	source := filepath.ToSlash(e.Path)
	dest := filepath.ToSlash(filepath.Join(filepath.Dir(target.Path), filepath.Base(e.Path)))

	modified := *e
	priorParent := e.Parent
	modified.Parent = newEpicID
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

	subject := fmt.Sprintf("aiwf move %s %s -> %s", id, priorParent, newEpicID)
	return plan(&Plan{
		Subject: subject,
		Trailers: []gitops.Trailer{
			{Key: "aiwf-verb", Value: "move"},
			{Key: "aiwf-entity", Value: id},
			{Key: "aiwf-prior-parent", Value: priorParent},
			{Key: "aiwf-actor", Value: actor},
		},
		Ops: []FileOp{
			{Type: OpMove, Path: source, NewPath: dest},
			{Type: OpWrite, Path: dest, Content: content},
		},
	}), nil
}
