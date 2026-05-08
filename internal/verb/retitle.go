package verb

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/23min/ai-workflow-v2/internal/check"
	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/tree"
)

// Retitle updates the frontmatter `title:` of an existing entity
// (top-level kind) or AC (composite id). Title only — no body changes
// for top-level entities, no slug renames (those go through
// `aiwf rename`). Closes G-065.
//
// For composite ids (M-NNN/AC-N), Retitle dispatches to retitleAC,
// which updates the AC's title in the parent milestone's acs[] array
// AND regenerates the matching `### AC-<N> — <title>` body heading.
// Both changes land in one atomic commit per kernel rule.
//
// reason is optional free-form prose; when non-empty it lands in the
// commit body so the rationale surfaces in `aiwf history`.
//
// Returns a Go error for "couldn't even start": id not found, empty
// new title (after trimming), no-op (current title equals new title).
// Tree-level findings caused by the projection are returned in
// Result.Findings.
func Retitle(ctx context.Context, t *tree.Tree, id, newTitle, actor, reason string) (*Result, error) {
	_ = ctx
	if strings.TrimSpace(newTitle) == "" {
		return nil, fmt.Errorf("retitle: new title is empty")
	}
	if entity.IsCompositeID(id) {
		return retitleAC(t, id, newTitle, actor, reason)
	}
	e := t.ByID(id)
	if e == nil {
		return nil, fmt.Errorf("entity %q not found", id)
	}
	if e.Title == newTitle {
		return nil, fmt.Errorf("%s title already %q", id, newTitle)
	}

	modified := *e
	modified.Title = newTitle

	body, err := readBody(t.Root, e.Path)
	if err != nil {
		return nil, err
	}
	content, err := entity.Serialize(&modified, body)
	if err != nil {
		return nil, fmt.Errorf("serializing %s: %w", id, err)
	}

	proj := projectReplace(t, &modified, filepath.ToSlash(e.Path))
	if fs := projectionFindings(t, proj); check.HasErrors(fs) {
		return findings(fs), nil
	}

	subject := fmt.Sprintf("aiwf retitle %s -> %q", id, newTitle)
	return plan(&Plan{
		Subject:  subject,
		Body:     reason,
		Trailers: standardTrailers("retitle", id, actor),
		Ops:      []FileOp{{Type: OpWrite, Path: e.Path, Content: content}},
	}), nil
}

// retitleAC handles `aiwf retitle M-NNN/AC-N "<new-title>"`. Updates
// the AC's title in the milestone's frontmatter and rewrites the
// matching `### AC-<N>` body heading. One commit, no path change. The
// shape parallels rename's composite-id arm (`internal/verb/ac.go`'s
// renameAC) — both edit frontmatter title and body heading — but emits
// a `retitle` trailer so `aiwf history` distinguishes the two
// invocation paths.
func retitleAC(t *tree.Tree, compositeID, newTitle, actor, reason string) (*Result, error) {
	parent, ac, err := lookupAC(t, compositeID)
	if err != nil {
		return nil, err
	}
	if ac.Title == newTitle {
		return nil, fmt.Errorf("%s title already %q", compositeID, newTitle)
	}
	modified, err := withACMutation(parent, ac.ID, func(updated *entity.AcceptanceCriterion) {
		updated.Title = newTitle
	})
	if err != nil {
		return nil, err
	}
	body, err := readBody(t.Root, parent.Path)
	if err != nil {
		return nil, err
	}
	body = rewriteACHeading(body, ac.ID, newTitle)
	content, err := entity.Serialize(modified, body)
	if err != nil {
		return nil, fmt.Errorf("serializing %s: %w", parent.ID, err)
	}
	proj := projectReplace(t, modified, filepath.ToSlash(parent.Path))
	if fs := projectionFindings(t, proj); check.HasErrors(fs) {
		return findings(fs), nil
	}
	subject := fmt.Sprintf("aiwf retitle %s -> %q", compositeID, newTitle)
	return plan(&Plan{
		Subject:  subject,
		Body:     reason,
		Trailers: standardTrailers("retitle", compositeID, actor),
		Ops:      []FileOp{{Type: OpWrite, Path: parent.Path, Content: content}},
	}), nil
}
