// Package tree loads aiwf entities from a consumer repository's directory
// tree into an in-memory model.
//
// The loader is deliberately tolerant: per-file parse errors are collected
// and returned as LoadErrors alongside the (possibly partial) tree, not
// folded into a single failure. This matches the framework's
// "errors are findings" principle — `aiwf check` reports inconsistent
// state, it does not refuse to start.
package tree

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/23min/ai-workflow-v2/tools/internal/entity"
)

// Tree is the in-memory representation of every aiwf entity discovered
// in a consumer repository.
type Tree struct {
	// Root is the absolute path to the consumer repo root the tree was
	// loaded from.
	Root string
	// Entities holds every successfully-parsed entity. Order is the
	// order encountered during the directory walk, which is stable
	// across runs but not otherwise specified.
	Entities []*entity.Entity
	// Stubs holds entities whose source file failed to parse. Each
	// carries only id (derived from the path), kind, and path; body
	// fields are zero. Stubs are deliberately not in Entities so that
	// frontmatter-shape, status-valid, and similar body-level checks
	// don't emit spurious findings. They exist so reference resolution
	// can still locate a target by id, preventing one file's parse
	// failure from cascading into "unresolved reference" findings on
	// every entity that links to it. The original parse failure is
	// reported as a load-error finding.
	Stubs []*entity.Entity
	// PlannedFiles records repo-relative file paths (forward-slash form)
	// that a verb plans to write but hasn't yet. Used by checks that
	// otherwise consult disk so that validate-then-write verbs can
	// validate the projected world, including files about to be created.
	// Loaded trees leave this nil.
	PlannedFiles map[string]struct{}
}

// HasPlannedFile reports whether path (forward-slash, repo-relative)
// appears in PlannedFiles. Safe to call when PlannedFiles is nil.
func (t *Tree) HasPlannedFile(path string) bool {
	if t.PlannedFiles == nil {
		return false
	}
	_, ok := t.PlannedFiles[path]
	return ok
}

// LoadError is a per-file error encountered during loading. The loader
// collects these instead of aborting; checks surface them as findings.
type LoadError struct {
	Path string
	Err  error
}

func (e *LoadError) Error() string { return fmt.Sprintf("%s: %v", e.Path, e.Err) }
func (e *LoadError) Unwrap() error { return e.Err }

// Load walks the consumer repo's entity-bearing directories and parses
// every recognized entity file. Per-file errors are returned in the
// LoadError slice; the (*Tree) is always populated with whatever could
// be parsed, even if some files failed.
//
// The third return is reserved for fatal errors that prevent the walk
// from completing (e.g., a permission error on a parent directory).
// A missing entity-bearing directory is not fatal — fresh repos may
// not yet have one.
func Load(ctx context.Context, root string) (*Tree, []LoadError, error) {
	tree := &Tree{Root: root}
	var loadErrs []LoadError

	walkRoots := []string{
		filepath.Join("work", "epics"),
		filepath.Join("work", "gaps"),
		filepath.Join("work", "decisions"),
		filepath.Join("work", "contracts"),
		filepath.Join("docs", "adr"),
	}

	for _, sub := range walkRoots {
		if err := ctx.Err(); err != nil {
			return tree, loadErrs, err
		}
		walkRoot := filepath.Join(root, sub)
		stat, err := os.Stat(walkRoot)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return tree, loadErrs, fmt.Errorf("statting %s: %w", walkRoot, err)
		}
		if !stat.IsDir() {
			loadErrs = append(loadErrs, LoadError{
				Path: sub,
				Err:  fmt.Errorf("expected a directory, found a file"),
			})
			continue
		}

		walkErr := filepath.WalkDir(walkRoot, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			relPath, relErr := filepath.Rel(root, path)
			if relErr != nil {
				loadErrs = append(loadErrs, LoadError{Path: path, Err: relErr})
				return nil
			}
			kind, ok := entity.PathKind(relPath)
			if !ok {
				return nil
			}

			content, readErr := os.ReadFile(path)
			if readErr != nil {
				loadErrs = append(loadErrs, LoadError{Path: relPath, Err: fmt.Errorf("reading file: %w", readErr)})
				registerStub(tree, relPath, kind)
				return nil
			}

			e, parseErr := entity.Parse(relPath, content)
			if parseErr != nil {
				loadErrs = append(loadErrs, LoadError{Path: relPath, Err: parseErr})
				registerStub(tree, relPath, kind)
				return nil
			}
			e.Kind = kind
			tree.Entities = append(tree.Entities, e)
			return nil
		})
		if walkErr != nil {
			return tree, loadErrs, fmt.Errorf("walking %s: %w", walkRoot, walkErr)
		}
	}

	return tree, loadErrs, nil
}

// registerStub appends a path-derived stub entity to tree.Stubs so that
// reference resolution can still find a target by id when the source
// file failed to load (read or parse failure). No stub is registered if
// the path does not yield a valid id for the kind.
func registerStub(t *Tree, relPath string, kind entity.Kind) {
	id, ok := entity.IDFromPath(relPath, kind)
	if !ok {
		return
	}
	t.Stubs = append(t.Stubs, &entity.Entity{
		ID:   id,
		Kind: kind,
		Path: relPath,
	})
}

// ByID returns the first entity matching the id, or nil if absent.
// In a tree with duplicate ids (which the ids-unique check reports),
// ByID returns one; iterate Entities to enumerate all.
func (t *Tree) ByID(id string) *entity.Entity {
	for _, e := range t.Entities {
		if e.ID == id {
			return e
		}
	}
	return nil
}

// ByKind returns every entity of the given kind, in tree-walk order.
func (t *Tree) ByKind(k entity.Kind) []*entity.Entity {
	out := make([]*entity.Entity, 0)
	for _, e := range t.Entities {
		if e.Kind == k {
			out = append(out, e)
		}
	}
	return out
}
