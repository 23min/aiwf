package verb

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
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
	return editBodyExplicit(ctx, t, e, body, actor, reason)
}

// editBodyExplicit covers the M-058 path: caller supplies new body
// bytes (typically from `--body-file <path>` or stdin). The verb
// re-serializes the entity's existing frontmatter with the new body.
func editBodyExplicit(ctx context.Context, t *tree.Tree, e *entity.Entity, body []byte, actor, reason string) (*Result, error) {
	if err := validateUserBodyBytes(body); err != nil {
		return nil, fmt.Errorf("--body-file: %w", err)
	}

	// The frontmatter this write carries is re-serialized from a tree
	// loaded off disk, so a hand-edited field would ride into the commit
	// under `aiwf-verb: edit-body` and be attributed to a body edit
	// (G-0463). Both modes are body-only; both refuse the same way.
	diverged, err := workingFrontmatterDiverged(ctx, t.Root, e.Path)
	if err != nil {
		//coverage:ignore defensive: the loader read this same path moments ago to build e, so a failure here needs the file to vanish or git to break mid-verb
		return nil, err
	}
	if diverged {
		return nil, errFrontmatterChangedInWorkingCopy(e.ID)
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

	// Same-state convergence (M-0281/AC-8): writing content the entity already
	// carries lands a commit with an empty diff on every repeat (see
	// verb_result_noop_invariant.go for why aiwf does not reject one).
	// Converging is sound only once the requested content is what BOTH git and
	// the operator would see, which is why this settles nothing on its own and
	// defers to explicitBodySettled below.
	settled, err := explicitBodySettled(ctx, t, e, content)
	if err != nil {
		//coverage:ignore defensive: explicitBodySettled errors only from its own two annotated-unreachable arms (a git failure reading HEAD, or the loader's own file gone missing mid-verb), so this propagation is unreachable for the same reasons
		return nil, err
	}
	if settled {
		return &Result{
			NoOp:        true,
			NoOpMessage: fmt.Sprintf("%s: HEAD already carries this body; nothing to commit", e.ID),
		}, nil
	}

	proj := projectReplace(t, &modified, filepath.ToSlash(e.Path))
	if fs := projectionFindings(t, proj); check.HasErrors(fs) {
		return findings(fs), nil
	}

	// G-0184 verb-time scan: vet the new body bytes for malformed or
	// unallocated id-shaped tokens. Catches operator-supplied content
	// (--body-file / stdin) before the commit lands.
	if fs := check.ScanBodyProseID(body, e.ID, e.Path, check.BodyProseIDIndex(t)); check.HasErrors(fs) {
		return findings(fs), nil
	}

	result := plan(&Plan{
		Subject:  fmt.Sprintf("aiwf edit-body %s", e.ID),
		Body:     reason,
		Trailers: standardTrailers("edit-body", e.ID, actor),
		Ops:      []FileOp{{Type: OpWrite, Path: e.Path, Content: content, AdoptsWorkingCopy: true}},
	})
	result.Metadata = map[string]any{"entity_id": e.ID}
	return result, nil
}

// errFrontmatterChangedInWorkingCopy is the refusal both edit-body modes
// raise when the working copy's frontmatter no longer matches HEAD's. It
// names the verbs that own structured state, so the operator's next step
// is in the message rather than in the docs.
func errFrontmatterChangedInWorkingCopy(id string) error {
	return fmt.Errorf("%s: frontmatter changed in the working copy — `aiwf edit-body` is body-only by design; use `aiwf promote` / `aiwf rename` / `aiwf cancel` / `aiwf reallocate` for structured-state edits", id)
}

// workingFrontmatterDiverged reports whether the working copy at relPath
// carries frontmatter differing from HEAD's. An entity with no committed
// version has nothing to diverge from, so it reports false and explicit
// mode keeps working for a file that exists only in the working tree.
func workingFrontmatterDiverged(ctx context.Context, root, relPath string) (bool, error) {
	headBytes, err := gitops.ReadFromHEAD(ctx, root, filepath.ToSlash(relPath))
	if err != nil { //coverage:ignore defensive: ReadFromHEAD maps a missing path to (nil, nil); a non-nil error needs git absent or a broken workdir, matching the same arm in editBodyBless
		return false, fmt.Errorf("reading HEAD version of %s: %w", relPath, err)
	}
	if headBytes == nil {
		return false, nil
	}
	diskBytes, err := os.ReadFile(filepath.Join(root, relPath))
	if err != nil { //coverage:ignore defensive: the loader read this same path to build the entity moments earlier
		return false, fmt.Errorf("reading working copy of %s: %w", relPath, err)
	}
	return !entity.SameFrontmatterFields(headBytes, diskBytes), nil
}

// explicitBodySettled reports whether writing content would change nothing an
// operator or git can observe: the committed bytes at HEAD and the bytes on
// disk both already equal it.
//
// Both comparisons are load-bearing, and either one alone is wrong in a way
// that loses work:
//
//   - HEAD alone would report the body already matches while a dirty working
//     copy holds something else — stranding a revert (asking for the committed
//     content back is a legitimate declarative request) and stating something
//     false about the file the operator is looking at.
//   - Disk alone would converge when the working copy already carries the
//     requested content uncommitted, never landing the commit that was the
//     point of the call — the write-the-file-then-route-it-through-the-verb
//     flow this project's own guidance encourages.
//
// The comparison is on the SERIALIZED entity rather than the body bytes,
// because entity.Serialize re-canonicalizes frontmatter: a byte-identical body
// over non-canonical frontmatter still has a real write to make.
//
// An entity with no committed version yet (headBytes nil) is never settled, so
// explicit mode keeps working for a file that exists only in the working tree —
// which is exactly the case bless mode refuses and redirects here.
func explicitBodySettled(ctx context.Context, t *tree.Tree, e *entity.Entity, content []byte) (bool, error) {
	headBytes, err := gitops.ReadFromHEAD(ctx, t.Root, filepath.ToSlash(e.Path))
	if err != nil {
		//coverage:ignore defensive: ReadFromHEAD maps a missing path to (nil, nil); a non-nil error needs git absent or a broken workdir, matching the same arm in editBodyBless
		return false, fmt.Errorf("reading HEAD version of %s: %w", e.Path, err)
	}
	if headBytes == nil || !bytes.Equal(content, headBytes) {
		return false, nil
	}
	diskBytes, err := os.ReadFile(filepath.Join(t.Root, e.Path))
	if err != nil {
		//coverage:ignore defensive: the loader just read this same path to build e, so it is present and readable by the time this runs
		return false, fmt.Errorf("reading working copy of %s: %w", e.Path, err)
	}
	return bytes.Equal(content, diskBytes), nil
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
		return nil, errFrontmatterChangedInWorkingCopy(e.ID)
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

	// G-0184 verb-time scan: vet the working-copy body bytes for
	// malformed or unallocated id-shaped tokens. Bless mode commits
	// whatever the user edited, so the working-copy bytes are what
	// will land; scanning here catches a malformed id-shape edit
	// before the commit lands.
	if fs := check.ScanBodyProseID(workingBody, e.ID, e.Path, check.BodyProseIDIndex(t)); check.HasErrors(fs) {
		return findings(fs), nil
	}

	result := plan(&Plan{
		Subject:  fmt.Sprintf("aiwf edit-body %s", e.ID),
		Body:     reason,
		Trailers: standardTrailers("edit-body", e.ID, actor),
		Ops:      []FileOp{{Type: OpWrite, Path: e.Path, Content: workingBytes, AdoptsWorkingCopy: true}},
	})
	result.Metadata = map[string]any{"entity_id": e.ID}
	return result, nil
}
