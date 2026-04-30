package verb

import (
	"fmt"
	"path/filepath"

	"github.com/23min/ai-workflow-v2/tools/internal/check"
	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/gitops"
	"github.com/23min/ai-workflow-v2/tools/internal/tree"
)

// AddOptions carries the per-kind extra arguments to Add. Only the
// fields relevant to the kind are read; others are ignored.
type AddOptions struct {
	// Milestone: id of the parent epic. Required.
	EpicID string
	// Gap: optional reference to the milestone or epic where the gap
	// was discovered.
	DiscoveredIn string
	// Decision: optional list of entity ids the decision relates to.
	RelatesTo []string
}

// Add creates a new entity of the given kind. Allocates the next free
// id, builds the entity, projects it onto the tree, runs `aiwf check`
// against the projection, and either returns findings (no changes
// staged) or a Plan that the orchestrator applies.
//
// Returns a Go error only when arguments are malformed (missing
// required option, parent epic not found). Tree-integrity issues
// arising from the addition are returned as findings, not errors.
func Add(t *tree.Tree, kind entity.Kind, title, actor string, opts AddOptions) (*Result, error) {
	if title == "" {
		return nil, fmt.Errorf("--title is required")
	}
	id := entity.AllocateID(kind, t.Entities)
	slug := entity.Slugify(title)
	if slug == "" {
		return nil, fmt.Errorf("title %q produces an empty slug; try a different title", title)
	}

	path, err := newEntityPath(t, kind, id, slug, opts)
	if err != nil {
		return nil, err
	}

	e := &entity.Entity{
		Kind:   kind,
		ID:     id,
		Title:  title,
		Status: initialStatus(kind),
		Path:   path,
	}
	applyAddOpts(e, opts)

	ops, err := buildAddOps(e)
	if err != nil {
		return nil, err
	}

	proj := projectAdd(t, e)
	if fs := projectionFindings(t, proj); check.HasErrors(fs) {
		return findings(fs), nil
	}

	subject := fmt.Sprintf("aiwf add %s %s %q", kind, id, title)
	return plan(&Plan{
		Subject: subject,
		Trailers: []gitops.Trailer{
			{Key: "aiwf-verb", Value: "add"},
			{Key: "aiwf-entity", Value: id},
			{Key: "aiwf-actor", Value: actor},
		},
		Ops: ops,
	}), nil
}

// newEntityPath computes the relative path the new entity will live at.
func newEntityPath(t *tree.Tree, kind entity.Kind, id, slug string, opts AddOptions) (string, error) {
	switch kind {
	case entity.KindEpic:
		return filepath.Join("work", "epics", id+"-"+slug, "epic.md"), nil
	case entity.KindMilestone:
		if opts.EpicID == "" {
			return "", fmt.Errorf("milestone requires --epic <epic-id>")
		}
		epic := t.ByID(opts.EpicID)
		if epic == nil {
			return "", fmt.Errorf("--epic %q does not exist", opts.EpicID)
		}
		if epic.Kind != entity.KindEpic {
			return "", fmt.Errorf("--epic %q is not an epic (it's a %s)", opts.EpicID, epic.Kind)
		}
		epicDir := filepath.Dir(epic.Path)
		return filepath.Join(epicDir, id+"-"+slug+".md"), nil
	case entity.KindADR:
		return filepath.Join("docs", "adr", id+"-"+slug+".md"), nil
	case entity.KindGap:
		return filepath.Join("work", "gaps", id+"-"+slug+".md"), nil
	case entity.KindDecision:
		return filepath.Join("work", "decisions", id+"-"+slug+".md"), nil
	case entity.KindContract:
		return filepath.Join("work", "contracts", id+"-"+slug, "contract.md"), nil
	}
	return "", fmt.Errorf("unsupported kind %q", kind)
}

// applyAddOpts copies kind-specific options from opts onto the entity.
func applyAddOpts(e *entity.Entity, opts AddOptions) {
	switch e.Kind {
	case entity.KindMilestone:
		e.Parent = opts.EpicID
	case entity.KindGap:
		if opts.DiscoveredIn != "" {
			e.DiscoveredIn = opts.DiscoveredIn
		}
	case entity.KindDecision:
		if len(opts.RelatesTo) > 0 {
			e.RelatesTo = append([]string(nil), opts.RelatesTo...)
		}
	}
}

// buildAddOps composes the file operations needed to land the new
// entity: a single OpWrite of the entity file with serialized
// frontmatter and the kind's body template.
func buildAddOps(e *entity.Entity) ([]FileOp, error) {
	body := entity.BodyTemplate(e.Kind)
	content, err := entity.Serialize(e, body)
	if err != nil {
		return nil, fmt.Errorf("serializing %s: %w", e.ID, err)
	}
	return []FileOp{{Type: OpWrite, Path: e.Path, Content: content}}, nil
}
