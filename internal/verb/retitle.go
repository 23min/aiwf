package verb

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// Retitle updates the frontmatter `title:` of an existing entity
// (top-level kind) or AC (composite id). For top-level entities, the
// on-disk slug is also re-derived from the new title and the file is
// renamed atomically in the same commit (G-0108) — so frontmatter
// title and filesystem slug never drift apart. Use `aiwf rename` when
// you want a slug change without touching the title.
//
// For composite ids (M-NNN/AC-N), Retitle dispatches to retitleAC,
// which updates the AC's title in the parent milestone's acs[] array
// AND regenerates the matching `### AC-<N> — <title>` body heading.
// Both changes land in one atomic commit per kernel rule. ACs have no
// slug, so no rename happens on the composite path.
//
// reason is optional free-form prose; when non-empty it lands in the
// commit body so the rationale surfaces in `aiwf history`.
//
// Returns a Go error for "couldn't even start": id not found, empty
// new title (after trimming), no-op (current title equals new title),
// or a title that slugifies to the empty string (e.g., punctuation-
// only). Tree-level findings caused by the projection are returned in
// Result.Findings.
//
// titleMaxLength caps the new title per `entities.title_max_length`
// (G-0102, kernel default 80). Title and slug share the same budget;
// retitle is also the natural verb to migrate existing entities
// whose pre-cap titles are over the cap (the operator picks the
// shorter form). Pass 0 from tests that don't care about cap policy.
func Retitle(ctx context.Context, t *tree.Tree, id, newTitle, actor, reason string, titleMaxLength int) (*Result, error) {
	_ = ctx
	if strings.TrimSpace(newTitle) == "" {
		return nil, fmt.Errorf("retitle: new title is empty")
	}
	if err := entity.ValidateTitle(newTitle, titleMaxLength); err != nil {
		return nil, err
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

	// G-0108: derive the new slug from the new title and prepare the
	// rename in the same commit. SlugifyDetailed mirrors what `aiwf
	// rename` accepts, so the resulting on-disk shape is identical.
	newSlug, dropped := entity.SlugifyDetailed(newTitle)
	if newSlug == "" {
		return nil, fmt.Errorf("retitle: new title %q produces an empty slug after normalization; pick a title with at least one alphanumeric character or use `aiwf rename` with an explicit slug", newTitle)
	}
	var slugNotices []check.Finding
	if len(dropped) > 0 {
		slugNotices = append(slugNotices, slugDroppedFinding(id, newTitle, newSlug, dropped))
	}

	source, dest, err := renamePaths(e, newSlug)
	if err != nil {
		return nil, err
	}

	ops := make([]FileOp, 0, 2)
	contentPath := e.Path
	planned := []string{filepath.ToSlash(e.Path)}
	if source != dest {
		// Slug also changed. Move first, then overwrite the moved file
		// with the title-updated content — the apply layer runs all
		// OpMoves before any OpWrite (verb.Apply phases), so the write
		// lands at the destination after the rename.
		modified.Path = newEntityPathAfterRename(e, source, dest)
		contentPath = modified.Path
		ops = append(ops, FileOp{Type: OpMove, Path: source, NewPath: dest})

		planned, err = plannedDestinations(t.Root, source, dest, modified.Path)
		if err != nil {
			return nil, err
		}
	}

	body, err := readBody(t.Root, e.Path)
	if err != nil {
		return nil, err
	}
	content, err := entity.Serialize(&modified, body)
	if err != nil {
		return nil, fmt.Errorf("serializing %s: %w", id, err)
	}
	ops = append(ops, FileOp{Type: OpWrite, Path: contentPath, Content: content})

	proj := projectReplace(t, &modified, planned...)
	if fs := projectionFindings(t, proj); check.HasErrors(fs) {
		return findings(fs), nil
	}

	subject := fmt.Sprintf("aiwf retitle %s -> %q", id, newTitle)
	return &Result{
		Findings: slugNotices,
		Plan: &Plan{
			Subject:  subject,
			Body:     reason,
			Trailers: standardTrailers("retitle", id, actor),
			Ops:      ops,
		},
	}, nil
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
