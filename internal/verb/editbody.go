package verb

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/23min/ai-workflow-v2/internal/check"
	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/gitops"
	"github.com/23min/ai-workflow-v2/internal/tree"
)

// EditBody replaces the markdown body of an existing entity file. The
// frontmatter is left untouched — that stays the domain of the
// structured-state verbs (promote, rename, cancel, reallocate).
//
// M-058 introduces this verb so the post-creation body-edit case has
// a kernel route. Before M-058 the only way to update body prose was
// a plain `git commit`, which the aiwf-add skill carve-out tolerated
// but `aiwf check` flagged via `provenance-untrailered-entity-commit`
// — a long-standing skill/check policy contradiction (G-052). With
// EditBody in place, every entity-file mutation goes through a verb
// route and the carve-out can be removed.
//
// body must contain markdown body content only. Content that begins
// with a YAML frontmatter delimiter (`---\n`) is refused via the
// shared validateUserBodyBytes helper — concatenating it with the
// verb's serialized frontmatter would produce a double-block file
// the loader can't parse.
//
// Composite ids (M-NNN/AC-N) are refused with a clear message.
// AC body sections live inside the parent milestone's body and need
// a sub-section editor; that is deliberately deferred.
//
// reason is optional free-form prose; when non-empty it lands in the
// commit body so future readers can see *why* the body was rewritten,
// not just *what* changed.
//
// Returns a Go error for "couldn't even start": id not found,
// composite id, body validation failure. Tree-level findings caused
// by the projection are returned in Result.Findings.
func EditBody(ctx context.Context, t *tree.Tree, id string, body []byte, actor, reason string) (*Result, error) {
	_ = ctx
	if entity.IsCompositeID(id) {
		return nil, fmt.Errorf("aiwf edit-body does not yet support composite ids (M-NNN/AC-N); edit the parent milestone's body instead")
	}
	e := t.ByID(id)
	if e == nil {
		return nil, fmt.Errorf("entity %q not found", id)
	}
	if err := validateUserBodyBytes(body); err != nil {
		return nil, fmt.Errorf("--body-file: %w", err)
	}

	// Re-serialize the existing entity (no frontmatter mutation) with
	// the new body. Same atomic-write shape as promote/cancel: one
	// OpWrite on the entity file produces one git commit, so the
	// per-mutation atomicity guarantee holds (AC-2).
	modified := *e
	content, err := entity.Serialize(&modified, body)
	if err != nil {
		return nil, fmt.Errorf("serializing %s: %w", id, err)
	}

	proj := projectReplace(t, &modified, filepath.ToSlash(e.Path))
	if fs := projectionFindings(t, proj); check.HasErrors(fs) {
		return findings(fs), nil
	}

	subject := fmt.Sprintf("aiwf edit-body %s", id)
	return plan(&Plan{
		Subject: subject,
		Body:    reason,
		Trailers: []gitops.Trailer{
			{Key: gitops.TrailerVerb, Value: "edit-body"},
			{Key: gitops.TrailerEntity, Value: id},
			{Key: gitops.TrailerActor, Value: actor},
		},
		Ops: []FileOp{{Type: OpWrite, Path: e.Path, Content: content}},
	}), nil
}
