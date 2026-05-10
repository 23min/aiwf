package verb

import (
	"bytes"
	"context"
	"fmt"
	"os"
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
// M-058 introduced the explicit-content path; M-060 added bless mode.
// The verb has two modes, dispatched on body:
//
//   - body == nil: bless mode (M-060). Read the working-copy bytes
//     and HEAD bytes, refuse if there is no diff (no changes to
//     commit), refuse if the diff includes frontmatter changes
//     (point at promote/rename/cancel/reallocate), commit the
//     working-copy bytes verbatim with edit-body trailers. This is
//     the natural human workflow: edit the file in $EDITOR, then
//     bless the change with a verb route.
//
//   - body != nil: explicit-content mode (M-058). The supplied bytes
//     replace the body; the verb re-serializes the existing entity
//     frontmatter with the new body and writes the result. This is
//     the AI/script workflow — the body content was drafted
//     elsewhere and is supplied via `--body-file <path>` or stdin.
//
// Both modes refuse leading-`---` content via validateUserBodyBytes,
// refuse composite ids, return one OpWrite, and emit the same
// trailer set (`aiwf-verb edit-body`, `aiwf-entity`, `aiwf-actor`).
//
// reason is optional free-form prose; when non-empty it lands in the
// commit body so future readers can see *why* the body was rewritten,
// not just *what* changed.
//
// Returns a Go error for "couldn't even start": id not found,
// composite id, body validation failure, no-diff in bless mode,
// frontmatter-changed in bless mode. Tree-level findings caused by
// the projection are returned in Result.Findings.
func EditBody(ctx context.Context, t *tree.Tree, id string, body []byte, actor, reason string) (*Result, error) {
	if entity.IsCompositeID(id) {
		return nil, fmt.Errorf("aiwf edit-body does not yet support composite ids (M-NNN/AC-N); edit the parent milestone's body instead")
	}
	e := t.ByID(id)
	if e == nil {
		return nil, fmt.Errorf("entity %q not found", id)
	}
	if body == nil {
		return editBodyBless(ctx, t, e, actor, reason)
	}
	return editBodyExplicit(t, e, body, actor, reason)
}

// editBodyExplicit covers the M-058 path: caller supplies new body
// bytes (typically from `--body-file <path>` or stdin). The verb
// re-serializes the entity's existing frontmatter with the new body.
func editBodyExplicit(t *tree.Tree, e *entity.Entity, body []byte, actor, reason string) (*Result, error) {
	if err := validateUserBodyBytes(body); err != nil {
		return nil, fmt.Errorf("--body-file: %w", err)
	}

	// Re-serialize the existing entity (no frontmatter mutation) with
	// the new body. Same atomic-write shape as promote/cancel: one
	// OpWrite on the entity file produces one git commit, so the
	// per-mutation atomicity guarantee holds.
	modified := *e
	content, err := entity.Serialize(&modified, body)
	if err != nil {
		return nil, fmt.Errorf("serializing %s: %w", e.ID, err)
	}

	proj := projectReplace(t, &modified, filepath.ToSlash(e.Path))
	if fs := projectionFindings(t, proj); check.HasErrors(fs) {
		return findings(fs), nil
	}

	return plan(&Plan{
		Subject:  fmt.Sprintf("aiwf edit-body %s", e.ID),
		Body:     reason,
		Trailers: editBodyTrailers(e.ID, actor),
		Ops:      []FileOp{{Type: OpWrite, Path: e.Path, Content: content}},
	}), nil
}

// editBodyBless covers the M-060 path: the user already edited the
// entity file in their editor. The verb commits whatever changed
// against HEAD, refusing if the diff is empty or if the frontmatter
// was touched. Body content from the working copy is validated by
// the same shared rule the explicit path uses (no leading `---`).
//
// The committed bytes are the working-copy bytes verbatim — bless
// mode does not re-serialize through entity.Serialize, so YAML key
// order, comments, and whitespace formatting from the user's edit
// are preserved exactly.
func editBodyBless(ctx context.Context, t *tree.Tree, e *entity.Entity, actor, reason string) (*Result, error) {
	workingPath := filepath.Join(t.Root, e.Path)
	workingBytes, err := os.ReadFile(workingPath)
	if err != nil {
		return nil, fmt.Errorf("reading working copy of %s: %w", e.Path, err)
	}
	headBytes, err := gitops.ReadFromHEAD(ctx, t.Root, filepath.ToSlash(e.Path))
	if err != nil {
		return nil, fmt.Errorf("reading HEAD version of %s: %w", e.Path, err)
	}
	if headBytes == nil {
		return nil, fmt.Errorf("%s has no committed version yet — bless mode applies to existing entities only; use `aiwf add` for new entities, or supply `--body-file <path>` to set the body explicitly", e.ID)
	}
	if bytes.Equal(workingBytes, headBytes) {
		return nil, fmt.Errorf("%s: no changes to commit — bless mode commits a working-copy edit; edit the file first or supply `--body-file <path>`", e.ID)
	}

	workingFM, workingBody, ok := entity.Split(workingBytes)
	if !ok {
		return nil, fmt.Errorf("%s working copy lacks a frontmatter delimiter; cannot bless without an anchor", e.Path)
	}
	headFM, _, ok := entity.Split(headBytes)
	if !ok {
		return nil, fmt.Errorf("%s HEAD version lacks a frontmatter delimiter; the file was committed without one — fix the HEAD version with a structured-state verb first", e.Path)
	}
	if !bytes.Equal(workingFM, headFM) {
		return nil, fmt.Errorf("%s: frontmatter changed in the working copy — `aiwf edit-body` is body-only by design; use `aiwf promote` / `aiwf rename` / `aiwf cancel` / `aiwf reallocate` for structured-state edits", e.ID)
	}
	if err := validateUserBodyBytes(workingBody); err != nil {
		return nil, fmt.Errorf("on-disk body of %s: %w", e.Path, err)
	}

	// Projection check uses *e (no in-memory frontmatter mutation —
	// frontmatter is unchanged by contract). Disk-reading validators
	// (acsBodyCoherence) read the working-copy bytes that the user
	// just edited, so a malformed AC-heading rewrite surfaces here
	// before we commit.
	modified := *e
	proj := projectReplace(t, &modified, filepath.ToSlash(e.Path))
	if fs := projectionFindings(t, proj); check.HasErrors(fs) {
		return findings(fs), nil
	}

	return plan(&Plan{
		Subject:  fmt.Sprintf("aiwf edit-body %s", e.ID),
		Body:     reason,
		Trailers: editBodyTrailers(e.ID, actor),
		Ops:      []FileOp{{Type: OpWrite, Path: e.Path, Content: workingBytes}},
	}), nil
}

// editBodyTrailers builds the standard trailer triple for edit-body
// commits. Centralized so the explicit and bless paths emit
// identical trailers — `aiwf history <id>` cannot tell them apart,
// which is the right outcome (both are "the body was edited").
func editBodyTrailers(id, actor string) []gitops.Trailer {
	return []gitops.Trailer{
		{Key: gitops.TrailerVerb, Value: "edit-body"},
		// Canonical width per AC-1 in M-081.
		{Key: gitops.TrailerEntity, Value: entity.Canonicalize(id)},
		{Key: gitops.TrailerActor, Value: actor},
	}
}
