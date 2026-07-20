package verb

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// Rename changes the slug portion of an entity's file or directory
// path. The id is preserved (per the design's "ids are immortal"
// invariant); the title in frontmatter is unchanged. Hand-edit the
// title in markdown if you want it to track the new slug.
//
// For epic and contract (directory-based kinds), the directory itself
// is moved; nested files (milestones under an epic, the schema/
// subdir under a contract) move with it. For file-based kinds, the
// single file moves.
//
// For composite ids (M-NNN/AC-N), Rename dispatches to renameAC: the
// second argument is interpreted as a new title (not a slug), the
// AC's frontmatter title is updated, and the matching `### AC-<N>`
// body heading is rewritten in place. No path change.
//
// Returns a Go error for "couldn't even start": id not found, slug
// produces an invalid path, source path missing on disk. Tree-level
// findings caused by the move are returned in Result.Findings.
// slugMaxLength caps the rewritten slug per
// `entities.title_max_length` (G-0102, kernel default 80). Title and
// slug share the same length budget so on-disk filenames and
// frontmatter titles stay in sync. Pass 0 from tests that don't care
// about cap policy.
func Rename(ctx context.Context, t *tree.Tree, id, newSlug, actor string, slugMaxLength int) (*Result, error) {
	_ = ctx
	if entity.IsCompositeID(id) {
		return renameAC(t, id, newSlug, actor)
	}
	e := t.ByID(id)
	if e == nil {
		return nil, fmt.Errorf("entity %q not found", id)
	}
	cleanSlug, dropped := entity.SlugifyDetailed(newSlug)
	if cleanSlug == "" {
		return nil, fmt.Errorf("new slug %q is empty after normalization", newSlug)
	}
	if err := entity.ValidateSlug(cleanSlug, slugMaxLength); err != nil {
		return nil, err
	}
	var slugNotices []check.Finding
	if len(dropped) > 0 {
		slugNotices = append(slugNotices, slugDroppedFinding(id, newSlug, cleanSlug, dropped))
	}

	source, dest, err := renamePaths(e, cleanSlug)
	if err != nil {
		return nil, err
	}
	if source == dest {
		return nil, fmt.Errorf("new slug %q matches the current slug; nothing to rename", cleanSlug)
	}

	// Update the entity's path so checks see the projected location.
	modified := *e
	modified.Path = newEntityPathAfterRename(e, source, dest)

	// Enumerate the planned file destinations so checks that consult
	// disk see the moved files at their new locations.
	planned, err := plannedDestinations(t.Root, source, dest, modified.Path)
	if err != nil {
		return nil, err
	}

	proj := projectReplace(t, &modified, planned...)
	if introduced := projectionFindings(t, proj); check.HasErrors(introduced) {
		return findings(introduced), nil
	}

	moves := renameEntityMoves(t, e, source, dest)
	rewriteOps, err := planLinkRewriteWrites(t, moves, nil)
	if err != nil { //coverage:ignore defensive: planLinkRewriteWrites only errors on a vanished file or an unserializable entity — neither reachable from a tree the loader just built
		return nil, err
	}
	ops := append([]FileOp{{Type: OpMove, Path: source, NewPath: dest}}, rewriteOps...)

	canonID := entity.Canonicalize(id)
	subject := fmt.Sprintf("aiwf rename %s slug -> %s", canonID, cleanSlug)
	return &Result{
		Findings: slugNotices,
		Plan: &Plan{
			Subject: subject,
			Trailers: []gitops.Trailer{
				{Key: gitops.TrailerVerb, Value: "rename"},
				// Canonical width per AC-1 in M-081.
				{Key: gitops.TrailerEntity, Value: canonID},
				{Key: gitops.TrailerActor, Value: actor},
			},
			Ops: ops,
		},
		Metadata: map[string]any{"entity_id": canonID, "new_slug": cleanSlug},
	}, nil
}

// renameEntityMoves computes the per-file EntityMoves produced by
// moving source -> dest for entity e: e's own move for a file-based
// kind, or one move per entity nested inside source for a directory-
// shaped kind (epic, contract) — mirrors archiveEntityMoves'
// directory-expansion pattern (internal/verb/archive.go), since a
// directory rename carries its nested milestones/schema along with it.
func renameEntityMoves(tr *tree.Tree, e *entity.Entity, source, dest string) []EntityMove {
	switch e.Kind {
	case entity.KindEpic, entity.KindContract:
		var out []EntityMove
		for _, other := range tr.Entities {
			if !pathInside(other.Path, source) {
				continue
			}
			out = append(out, EntityMove{
				From: other.Path,
				To:   newEntityPathAfterRename(other, source, dest),
			})
		}
		return out
	default:
		return []EntityMove{{From: e.Path, To: dest}}
	}
}

// renamePaths returns the (source, dest) paths to pass to git mv. For
// directory-based kinds (epic, contract), the source is the entity's
// containing directory and the dest is the dir's new name. For
// file-based kinds, the source is the entity file itself. The id
// prefix is kept; the slug is replaced (a slug-less id gains one by
// appending — see substituteNamePart's substituteSlugMode).
func renamePaths(e *entity.Entity, newSlug string) (source, dest string, err error) {
	return rewriteEntityName(e, func(name string) (string, error) {
		return substituteNamePart(name, newSlug, substituteSlugMode)
	})
}

// newEntityPathAfterRename derives the new entity file path given the
// old entity, the source dir/file being moved, and the destination
// dir/file. For dir-based kinds the entity file (epic.md / contract.md)
// keeps its name inside the renamed dir.
func newEntityPathAfterRename(e *entity.Entity, source, dest string) string {
	if source == e.Path {
		return dest
	}
	// dir rename: e.Path was source/<basename>, now dest/<basename>.
	rel, _ := filepath.Rel(source, e.Path)
	return filepath.Join(dest, rel)
}

// plannedDestinations enumerates the new paths every file currently
// under source will occupy after `git mv source dest`. For a
// single-file rename, that's just dest. For a directory rename, it's
// dest plus dest-relative versions of every file inside source.
func plannedDestinations(root, source, dest, primaryDest string) ([]string, error) {
	// primaryDest is the entity's own new path; always include it.
	planned := []string{filepath.ToSlash(primaryDest)}

	full := filepath.Join(root, source)
	info, err := os.Stat(full)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", source, err)
	}
	if !info.IsDir() {
		return planned, nil
	}

	walkErr := filepath.WalkDir(full, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(full, p)
		if relErr != nil {
			return relErr
		}
		planned = append(planned, filepath.ToSlash(filepath.Join(dest, rel)))
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("walking %s: %w", source, walkErr)
	}
	return planned, nil
}
