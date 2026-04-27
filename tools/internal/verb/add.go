package verb

import (
	"fmt"
	"os"
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
	// Contract: required machine-readable schema format
	// (e.g., openapi, json-schema, proto).
	Format string
	// Contract: required source path on disk to copy into the new
	// contract directory's schema/ subdir. Path is repo-root-relative
	// or absolute. The destination filename is the source's basename.
	ArtifactSource string
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

	// Compute file ops *before* projecting so contract artifact copies
	// are visible to validateProjection (which checks artifact existence).
	ops, err := buildAddOps(e, opts)
	if err != nil {
		return nil, err
	}

	planned := make([]string, 0, len(ops))
	for _, op := range ops {
		if op.Type == OpWrite {
			planned = append(planned, filepath.ToSlash(op.Path))
		}
	}
	proj := projectAdd(t, e, planned...)
	if fs := validateProjection(proj); check.HasErrors(fs) {
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
	case entity.KindContract:
		e.Format = opts.Format
		// Artifact path inside the contract dir; the basename comes
		// from the source's basename (per the schema/ convention).
		if opts.ArtifactSource != "" {
			e.Artifact = filepath.ToSlash(filepath.Join("schema", filepath.Base(opts.ArtifactSource)))
		}
	}
}

// buildAddOps composes the file operations needed to land the new
// entity. For most kinds it's a single OpWrite of contract.md or epic.md
// or the entity file. For contracts with an --artifact-source, it's
// two OpWrites: contract.md plus the artifact copied into schema/.
func buildAddOps(e *entity.Entity, opts AddOptions) ([]FileOp, error) {
	body := entity.BodyTemplate(e.Kind)
	mainContent, err := entity.Serialize(e, body)
	if err != nil {
		return nil, fmt.Errorf("serializing %s: %w", e.ID, err)
	}
	ops := []FileOp{{Type: OpWrite, Path: e.Path, Content: mainContent}}

	if e.Kind == entity.KindContract {
		if opts.Format == "" {
			return nil, fmt.Errorf("contract requires --format")
		}
		if opts.ArtifactSource == "" {
			return nil, fmt.Errorf("contract requires --artifact-source <path>")
		}
		artifactBytes, err := os.ReadFile(opts.ArtifactSource)
		if err != nil {
			return nil, fmt.Errorf("reading --artifact-source %q: %w", opts.ArtifactSource, err)
		}
		artifactDest := filepath.Join(filepath.Dir(e.Path), filepath.FromSlash(e.Artifact))
		ops = append(ops, FileOp{Type: OpWrite, Path: artifactDest, Content: artifactBytes})
	}

	return ops, nil
}
