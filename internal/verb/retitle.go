package verb

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
)

// Retitle updates the frontmatter `title:` of an existing entity
// (top-level kind) or AC (composite id). For top-level entities whose
// slug still tracks the title, the on-disk slug is re-derived from the
// new title and the file is renamed atomically in the same commit
// (G-0108), so a title change does not leave a stale filename behind.
//
// A slug the operator set with `aiwf rename` is preserved instead: rename
// exists to choose a slug independently of the title, so a retitle that
// re-derived over it would leave rename's effect lasting only until the
// next retitle. Title and slug therefore agree by default and diverge
// only where the operator said so (ADR-0037).
//
// A canonical `# <ID> — <title>` body H1, if present, is rewritten to
// track the new title in the same commit (G-0083); bodies without a
// canonical H1 are left untouched, so an operator-shaped non-canonical
// heading is never silently clobbered. Use `aiwf rename` when you want a
// slug change without touching the title.
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
// new title (after trimming), or a title that slugifies to the empty
// string (e.g., punctuation-only). A new title equal to the current one
// is not an error — it converges to a NoOp (M-0281/AC-5). Tree-level
// findings caused by the projection are returned in Result.Findings.
//
// titleMaxLength caps the new title per `entities.title_max_length`
// (G-0102, kernel default 80). Title and slug share the same budget;
// retitle is also the natural verb to migrate existing entities
// whose pre-cap titles are over the cap (the operator picks the
// shorter form). Pass 0 from tests that don't care about cap policy.
func Retitle(ctx context.Context, t *tree.Tree, id, newTitle, actor, reason string, titleMaxLength int) (*Result, error) {
	if strings.TrimSpace(newTitle) == "" {
		return nil, fmt.Errorf("retitle: new title is empty")
	}
	if err := entity.ValidateTitle(newTitle, titleMaxLength); err != nil {
		return nil, err
	}
	if entity.IsCompositeID(id) {
		return retitleAC(ctx, t, id, newTitle, actor, reason)
	}
	e := t.ByID(id)
	if e == nil {
		return nil, fmt.Errorf("entity %q not found", id)
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
	source, dest, err := renamePaths(e, newSlug)
	if err != nil {
		// Reachable: the loader resolves an entity by its frontmatter id, not
		// by its filename, so a tree `aiwf check` calls clean can hold a file
		// or directory whose name renamePaths cannot rewrite.
		return nil, err
	}

	// The slug is retitle's to re-derive only while it still tracks the
	// title. One the operator set with `aiwf rename` is preserved: rename
	// exists to choose a slug independently of the title, so re-deriving over
	// that choice would make its effect last only until the next retitle.
	tracks, err := slugTracksTitle(e)
	if err != nil { //coverage:ignore defensive: renamePaths errors on the entity's path shape, not on the slug passed, and the call above already succeeded for this entity
		return nil, err
	}
	var slugNotices []check.Finding
	if tracks {
		if len(dropped) > 0 {
			slugNotices = append(slugNotices, slugDroppedFinding(id, newTitle, newSlug, dropped))
		}
	} else {
		dest = source
	}

	// The entity's own file, not `source`: for epic and contract that is a
	// directory, and the two surfaces this claim reads — the stored title
	// and the body H1 — both live in the file.
	if claimErr := guardClaim(ctx, t.Root, id, e.Path); claimErr != nil {
		return nil, claimErr
	}

	// Same-state convergence (M-0281/AC-5). Two surfaces must already read as
	// requested: the frontmatter title and a canonical `# <id> — <title>` body
	// H1. The filename is not a third. An unchanged title derives exactly the
	// slug this call would write, and a slug the operator set is preserved
	// above, so on both paths the destination already equals the source by the
	// time this runs — comparing them here would test a condition that cannot
	// be false.
	body, err := readBody(t.Root, e.Path)
	if err != nil {
		//coverage:ignore defensive: the loader read this same file to build e, so a failure here needs it to vanish mid-verb
		return nil, err
	}
	if e.Title == newTitle && entityH1MatchesTitle(body, e.ID, newTitle) {
		return &Result{NoOp: true, NoOpMessage: fmt.Sprintf("%s title is already %q; nothing to retitle", id, newTitle)}, nil
	}

	ops := make([]FileOp, 0, 2)
	contentPath := e.Path
	planned := []string{filepath.ToSlash(e.Path)}
	var moves []EntityMove
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
		moves = renameEntityMoves(t, e, source, dest)
	}

	// G-0083: keep a canonical `# <ID> — <title>` body H1 in sync with
	// the frontmatter title. Body H1 is optional (the BodyTemplate
	// scaffold doesn't produce one); when absent, rewriteEntityH1 is a
	// no-op. Non-canonical H1s (operator-shaped headings) are left
	// alone so an intentional divergence isn't silently clobbered.
	body = rewriteEntityH1(body, e.ID, newTitle)
	if len(moves) > 0 {
		// Fold the entity's own outgoing link rewrite into this same
		// body, rather than letting planLinkRewriteWrites below emit a
		// competing write for the same contentPath — a slug-changing
		// retitle of a dir-shaped kind (epic/contract) can link to one
		// of its own nested, co-moved entities.
		// Inbound only. For a flat-file kind the slug change renames the
		// file within its own directory, so no relative destination can
		// change meaning; for a dir-shaped kind the whole directory moves
		// and everything inside comes along, which is the case ADR-0046's
		// scope note excludes. Neither geometry wants an outbound
		// recompute, so both paths are the post-move one.
		body = RewriteLinkDestinationsForMove(body, contentPath, contentPath, moves)
	}
	content, err := entity.Serialize(&modified, body)
	if err != nil {
		return nil, fmt.Errorf("serializing %s: %w", id, err)
	}
	ops = append(ops, FileOp{Type: OpWrite, Path: contentPath, Content: content})

	if len(moves) > 0 {
		var dirShaped []string
		if e.Kind == entity.KindEpic || e.Kind == entity.KindContract {
			dirShaped = append(dirShaped, source)
		}
		rewriteOps, rwErr := planLinkRewriteWrites(t, moves, map[string]bool{e.Path: true}, dirShaped)
		if rwErr != nil { //coverage:ignore defensive: planLinkRewriteWrites only errors on a vanished file or an unserializable entity — neither reachable from a tree the loader just built
			return nil, rwErr
		}
		ops = append(ops, rewriteOps...)
	}

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
		Metadata: map[string]any{"entity_id": e.ID, "old_title": e.Title, "new_title": newTitle},
	}, nil
}

// slugTracksTitle reports whether e's on-disk slug is the one its current
// title derives — the condition under which the slug is retitle's to
// re-derive. A slug that does not track the title was chosen deliberately with
// `aiwf rename`, and retitle preserves it.
//
// The question is answered by rebuilding the path e's CURRENT title would
// produce and comparing it to the path on disk, so the slug is never parsed
// out of a filename a second time — renamePaths owns that decomposition for
// both dir-shaped and file-shaped kinds.
//
// A title that slugifies to nothing cannot be the source of the slug on disk,
// so the slug is the operator's either way.
func slugTracksTitle(e *entity.Entity) (bool, error) {
	derived := entity.Slugify(e.Title)
	if derived == "" {
		// A title that slugifies to nothing cannot be the source of the slug on
		// disk, so the slug is the operator's. add and retitle both reject such
		// a title, but the loader accepts one from a hand-authored or imported
		// file, which is what makes this arm reachable.
		return false, nil
	}
	source, dest, err := renamePaths(e, derived)
	if err != nil { //coverage:ignore defensive: same path-shape failure as the caller's own renamePaths call, which has already succeeded for this entity by the time this runs
		return false, err
	}
	return source == dest, nil
}

// entityH1MatchesTitle reports whether body's canonical `# <id> — <title>` H1
// already reads as rewriteEntityH1 would write it. A body with no canonical H1
// is already consistent: rewriteEntityH1 would not add one either.
//
// id must be the entity's own stored id, not the spelling the operator typed.
// Parsers accept narrower legacy widths on input, and a narrow spelling matches
// no H1 carrying the canonical one — which would make a stale heading look
// already-consistent and converge the caller over it.
func entityH1MatchesTitle(body []byte, id, newTitle string) bool {
	return bytes.Equal(body, rewriteEntityH1(body, id, newTitle))
}

// rewriteEntityH1 scans body for lines matching the canonical
// `# <id> — <anything>` H1 shape and rewrites them to carry newTitle.
// When no matching line exists, the body is returned unchanged — H1
// is optional in the kernel's body shape (BodyTemplate doesn't produce
// one), so most freshly-added entities have nothing to sync. Mirrors
// rewriteACHeading's pattern for top-level entity bodies (G-0083).
//
// The match is intentionally strict: only the canonical em-dash
// separator `# <id> — ` is recognized. Non-canonical headings (colon,
// hyphen, missing id, etc.) are operator-shaped hand edits and stay
// untouched so retitle never silently clobbers a deliberate
// divergence.
func rewriteEntityH1(body []byte, id, newTitle string) []byte {
	pattern := regexp.MustCompile(`(?m)^# ` + regexp.QuoteMeta(id) + ` — .*$`)
	replacement := fmt.Appendf(nil, "# %s — %s", id, newTitle)
	// ReplaceAllFunc, not ReplaceAll: the replacement carries the operator's
	// title, and ReplaceAll expands `$name` / `${name}` inside it. With no
	// capture groups in the pattern every such reference expands to the empty
	// string, so a title like `Cost is $1 per unit` would lose the `$1`.
	// rewriteACHeading takes the same precaution for the same reason.
	return pattern.ReplaceAllFunc(body, func([]byte) []byte { return replacement })
}

// retitleAC handles `aiwf retitle M-NNN/AC-N "<new-title>"`. Updates
// the AC's title in the milestone's frontmatter and rewrites the
// matching `### AC-<N>` body heading. One commit, no path change. The
// shape parallels rename's composite-id arm (`internal/verb/ac.go`'s
// renameAC) — both edit frontmatter title and body heading — but emits
// a `retitle` trailer so `aiwf history` distinguishes the two
// invocation paths.
func retitleAC(ctx context.Context, t *tree.Tree, compositeID, newTitle, actor, reason string) (*Result, error) {
	parent, ac, err := lookupAC(t, compositeID)
	if err != nil {
		return nil, err
	}
	if claimErr := guardClaim(ctx, t.Root, compositeID, parent.Path); claimErr != nil {
		return nil, claimErr
	}
	// Same-state convergence (M-0281/AC-5), matching the entity-level path
	// above — and, like it, spanning both surfaces the verb writes: the
	// frontmatter title and the `### AC-N — <title>` body heading.
	body, err := readBody(t.Root, parent.Path)
	if err != nil {
		//coverage:ignore defensive: lookupAC resolved the parent from the loaded tree, which read this same file, so a failure here needs it to vanish mid-verb
		return nil, err
	}
	if ac.Title == newTitle && acHeadingMatchesTitle(body, ac.ID, newTitle) {
		return &Result{NoOp: true, NoOpMessage: fmt.Sprintf("%s title is already %q; nothing to retitle", compositeID, newTitle)}, nil
	}
	modified, err := withACMutation(parent, ac.ID, func(updated *entity.AcceptanceCriterion) {
		updated.Title = newTitle
	})
	if err != nil {
		return nil, err
	}
	body = rewriteACHeading(body, ac.ID, newTitle)
	subject := fmt.Sprintf("aiwf retitle %s -> %q", compositeID, newTitle)
	return planEntityWrite(t, modified, parent.Path, body, entityWrite{
		subject:  subject,
		body:     reason,
		trailers: standardTrailers("retitle", compositeID, actor),
		metadata: map[string]any{"entity_id": compositeID, "old_title": ac.Title, "new_title": newTitle},
	})
}
