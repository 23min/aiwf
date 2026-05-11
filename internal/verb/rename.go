package verb

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

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
			Ops: []FileOp{{Type: OpMove, Path: source, NewPath: dest}},
		},
	}, nil
}

// renamePaths returns the (source, dest) paths to pass to git mv. For
// directory-based kinds (epic, contract), the source is the entity's
// containing directory and the dest is the dir's new name. For
// file-based kinds, the source is the entity file itself.
func renamePaths(e *entity.Entity, newSlug string) (source, dest string, err error) {
	switch e.Kind {
	case entity.KindEpic, entity.KindContract:
		// Containing directory moves; the file inside keeps its name.
		dir := filepath.Dir(e.Path)
		parent, oldName := filepath.Split(dir)
		newName, err := substituteSlug(oldName, newSlug)
		if err != nil {
			return "", "", err
		}
		// strip trailing separator from parent
		parent = strings.TrimRight(parent, "/")
		return dir, filepath.Join(parent, newName), nil
	default:
		// File renames: the .md basename gets a new slug.
		dir, oldName := filepath.Split(e.Path)
		newName, err := substituteSlug(strings.TrimSuffix(oldName, ".md"), newSlug)
		if err != nil {
			return "", "", err
		}
		dir = strings.TrimRight(dir, "/")
		return e.Path, filepath.Join(dir, newName+".md"), nil
	}
}

// substituteSlug replaces the slug portion of a name like "E-19-old-slug"
// with newSlug, yielding "E-19-new-slug". Returns an error when the
// name does not contain a recognizable id-prefix.
func substituteSlug(name, newSlug string) (string, error) {
	// Find the first hyphen after the digits run that follows the
	// kind prefix. We don't need to know the kind here: the convention
	// is "<letters>-<digits>-<rest>", so split after the second hyphen.
	first := strings.IndexByte(name, '-')
	if first < 0 {
		return "", fmt.Errorf("name %q has no id prefix to keep", name)
	}
	second := strings.IndexByte(name[first+1:], '-')
	if second < 0 {
		// "E-01" with no slug — append the new slug.
		return name + "-" + newSlug, nil
	}
	idPart := name[:first+1+second]
	return idPart + "-" + newSlug, nil
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
