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
	"sort"

	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/trunk"
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
	// ReverseRefs maps each entity id (and each composite id mentioned
	// as a reference target) to the ids of entities that reference it.
	// Built from entity.ForwardRefs at Load time; consumed by aiwf show's
	// referenced_by field, by aiwf check audits ("ADR is unreferenced"),
	// and by the I2.5 provenance scope-reachability check.
	//
	// Composite-id targets roll up to their parent: a gap with
	// `addressed_by: M-007/AC-1` appears in the AC's referrer list AND
	// in M-007's referrer list. Each value-slice is sorted ascending
	// and de-duplicated for stable output.
	ReverseRefs map[string][]string
	// TrunkIDs is the entity-id set observed in the configured trunk
	// ref's tree, used by AllocateID (so a new id can't collide with
	// trunk) and by the ids-unique check (so a working-tree id that
	// also exists at a different path on trunk surfaces as a finding
	// before push).
	//
	// Tree.Load does not populate this field — the cmd dispatcher reads
	// the trunk via the trunk package once per verb run and assigns
	// here so the verb's projection check sees the same trunk view.
	// Tests that build trees in-memory leave TrunkIDs nil, in which
	// case the allocator and the check rule degrade to working-tree-
	// only behavior (the previous default).
	TrunkIDs []trunk.ID
}

// TrunkIDStrings returns the id strings from TrunkIDs. Convenience
// for AllocateID, which only needs id values; the full trunk.ID is
// kept on the tree so the ids-unique check can include the trunk-side
// path in its finding message.
func (t *Tree) TrunkIDStrings() []string {
	if len(t.TrunkIDs) == 0 {
		return nil
	}
	out := make([]string, len(t.TrunkIDs))
	for i, x := range t.TrunkIDs {
		out[i] = x.ID
	}
	return out
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

	tree.ReverseRefs = buildReverseRefs(tree.Entities)
	return tree, loadErrs, nil
}

// buildReverseRefs inverts each entity's outbound references into a
// map keyed by referenced id. Composite-id targets (M-NNN/AC-N) appear
// under both the composite key AND the parent's bare key — a gap with
// `addressed_by: M-007/AC-1` shows up in the AC's referrers and in
// M-007's referrers. Each value-slice is sorted ascending and de-
// duplicated for stable output across runs.
//
// An empty entity slice yields a non-nil empty map; callers can always
// range or index without a nil check.
func buildReverseRefs(entities []*entity.Entity) map[string][]string {
	rev := make(map[string]map[string]struct{})
	add := func(target, referrer string) {
		if rev[target] == nil {
			rev[target] = make(map[string]struct{})
		}
		rev[target][referrer] = struct{}{}
	}
	for _, e := range entities {
		for _, ref := range entity.ForwardRefs(e) {
			add(ref.Target, e.ID)
			if parent, _, ok := entity.ParseCompositeID(ref.Target); ok {
				add(parent, e.ID)
			}
		}
	}
	out := make(map[string][]string, len(rev))
	for target, referrers := range rev {
		ids := make([]string, 0, len(referrers))
		for id := range referrers {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		out[target] = ids
	}
	return out
}

// ReferencedBy returns the ids of entities that reference id, in
// sorted order. Returns nil when no entity references id.
func (t *Tree) ReferencedBy(id string) []string {
	return t.ReverseRefs[id]
}

// Reaches reports whether `from` can reach `to` by walking forward
// references (parent, depends_on, addressed_by, relates_to,
// supersedes, discovered_in, linked_adrs, etc.) through the tree.
// Self-loop returns true (an entity reaches itself trivially).
//
// Composite ids are resolved to their parent before traversal: an
// AC walks under the milestone's reference graph, and reaching the
// milestone counts as reaching one of its ACs. This matches the
// scope-reachability rule in docs/pocv3/design/provenance-model.md
// §"Scope check": "addressed_by: M-007/AC-1" makes the gap reach
// M-007 (and therefore anything M-007 reaches via parent etc.).
//
// The walk is bounded by the existing entity set; an unresolved id
// (referenced but not in the tree) is a dead end. Used by the I2.5
// allow-rule (verb.Allow) to gate non-human-actor verbs against an
// active scope's scope-entity.
func (t *Tree) Reaches(from, to string) bool {
	from = compositeParentOrSame(from)
	to = compositeParentOrSame(to)
	if from == to {
		return true
	}
	visited := map[string]bool{from: true}
	queue := []string{from}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		e := t.ByID(cur)
		if e == nil {
			continue
		}
		for _, ref := range entity.ForwardRefs(e) {
			target := compositeParentOrSame(ref.Target)
			if target == to {
				return true
			}
			if !visited[target] {
				visited[target] = true
				queue = append(queue, target)
			}
		}
	}
	return false
}

// ReachesAny reports whether any of `froms` reaches `to`. Used by
// the allow-rule's creation-act branch: a new entity's outbound
// references are evaluated as a set against the scope-entity.
func (t *Tree) ReachesAny(froms []string, to string) bool {
	for _, from := range froms {
		if t.Reaches(from, to) {
			return true
		}
	}
	return false
}

// compositeParentOrSame returns the parent id of a composite (e.g.
// "M-007/AC-1" → "M-007"); returns the input unchanged when it isn't
// a composite. The shared trim used by Reaches / ReachesAny.
func compositeParentOrSame(id string) string {
	if parent, _, ok := entity.ParseCompositeID(id); ok {
		return parent
	}
	return id
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

// ByPriorID returns the entity whose `prior_ids` lineage list
// includes id, or nil if no entity claims that id as a prior. When
// multiple entities claim the same prior id (a hand-edit accident or
// the rare lineage-broken case), ByPriorID returns the first match
// in tree-walk order; iterate Entities to enumerate all.
func (t *Tree) ByPriorID(id string) *entity.Entity {
	for _, e := range t.Entities {
		for _, p := range e.PriorIDs {
			if p == id {
				return e
			}
		}
	}
	return nil
}

// ResolveByCurrentOrPriorID resolves id to an entity by trying
// ByID first (current id), then ByPriorID (lineage match). Returns
// nil when neither matches. Used by `aiwf history` so a query for
// an old id transparently resolves to the current entity, then the
// caller walks the chain via the entity's PriorIDs slice.
func (t *Tree) ResolveByCurrentOrPriorID(id string) *entity.Entity {
	if e := t.ByID(id); e != nil {
		return e
	}
	return t.ByPriorID(id)
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
