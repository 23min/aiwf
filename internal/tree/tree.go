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

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/trunk"
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
	// TrunkRef is the resolved trunk ref name (e.g.
	// "refs/remotes/origin/main") or empty when no trunk read
	// happened. The reallocate tiebreaker uses it as the second
	// argument to `git merge-base --is-ancestor` when two entities
	// collide on an id and the verb has to pick which side to
	// renumber. Populated alongside TrunkIDs by the cmd dispatcher;
	// empty in tests that don't set it (the verb falls back to
	// today's "ambiguous, pass a path" error in that case).
	TrunkRef string
	// TrunkRenames maps a pre-rename path on TrunkRef to the
	// corresponding post-rename path in the working tree, as detected
	// by `git diff -M` (gitops.RenamesFromRef). Used by the ids-unique
	// trunk-collision check to recognize that a branch-side slug
	// rename of an existing entity is the same entity moved, not a
	// duplicate id allocation (G-0109).
	//
	// Populated alongside TrunkIDs by the cmd dispatcher. Tests that
	// build trees in-memory leave it nil, in which case the rule
	// degrades to today's behavior (every different-path same-id pair
	// surfaces as a collision, modulo the archive-sweep exception).
	TrunkRenames map[string]string
	// Strays holds repo-relative file paths (forward-slash form) that
	// the loader walked under work/{epics,gaps,decisions,contracts}/
	// but could not classify as a recognized entity file via
	// entity.PathKind. The tree-discipline check (G40) reports each as
	// `unexpected-tree-file`; without that field the loader's silent
	// skip would let any LLM-written stray inside those subtrees
	// linger undetected.
	//
	// Scope is the four entity-bearing subdirs, not all of work/.
	// Files at the work/ root or under non-entity sibling dirs
	// (work/migration/, work/scratch/, etc.) are outside the loader's
	// walk roots and never appear here. docs/adr/ is walked but
	// conventionally permissive (READMEs, templates, etc.), so its
	// strays are not tracked. Files inside a contract's directory
	// (work/contracts/C-NNN-*/) are recorded here but filtered by the
	// check rule, since contracts legitimately carry schema/fixture
	// artifacts alongside contract.md.
	Strays []string
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

	walkRoots := []struct {
		sub         string
		trackStrays bool
	}{
		{filepath.Join("work", "epics"), true},
		{filepath.Join("work", "gaps"), true},
		{filepath.Join("work", "decisions"), true},
		{filepath.Join("work", "contracts"), true},
		{filepath.Join("docs", "adr"), false},
	}

	for _, wr := range walkRoots {
		sub := wr.sub
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
				if wr.trackStrays {
					tree.Strays = append(tree.Strays, filepath.ToSlash(relPath))
				}
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
	sort.Strings(tree.Strays)
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
		// Canonicalize both sides so a narrow `addressed_by: M-007`
		// referring to a tree storing `M-0007` (or vice versa) lands
		// on the same key — see the AC-2 lookup-seam rule. The
		// referrer's id is also canonicalized so the value-slice is
		// uniform regardless of on-disk width.
		target = entity.Canonicalize(target)
		referrer = entity.Canonicalize(referrer)
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
//
// The lookup canonicalizes id first so a narrow query resolves to
// referrers of the canonical form (the AC-2 lookup-seam rule); the
// reverse-ref map itself is keyed by canonical id (see
// buildReverseRefs).
func (t *Tree) ReferencedBy(id string) []string {
	return t.ReverseRefs[entity.Canonicalize(id)]
}

// ReachesScope reports whether `target` is within the scope-tree rooted
// at `scopeEntity`, per D-0006's three-edge reachability model. Unlike
// the (deprecated) full-graph Reaches, exactly three edges traverse:
//
//  1. parent forward    — target's parent chain reaches scopeEntity.
//  2. composite rollup  — an AC `M/AC-N` is reachable iff M is.
//  3. discovered_in rev — target's discovered_in points into the
//     scope subtree (one hop, then parent-climb).
//
// No governance edge (depends_on, addressed_by, relates_to, supersedes,
// superseded_by, linked_adrs) traverses — scope is a governance
// boundary, not the full reference grammar. The function reads
// e.Parent / e.DiscoveredIn directly rather than filtering ForwardRefs,
// so a future governance edge cannot silently re-broaden it.
func (t *Tree) ReachesScope(target, scopeEntity string) bool {
	target = compositeParentOrSame(target)
	scope := compositeParentOrSame(scopeEntity)
	// Edges 1 + 2: parent chain (composite already rolled up above).
	if t.parentChainReaches(target, scope) {
		return true
	}
	// Edge 3: one discovered_in hop, then parent-climb into the subtree.
	if e := t.ByID(target); e != nil && e.DiscoveredIn != "" {
		if t.parentChainReaches(compositeParentOrSame(e.DiscoveredIn), scope) {
			return true
		}
	}
	return false
}

// ReachesScopeAny reports whether any of `targets` is within the
// scope-tree rooted at `scopeEntity` (the creation-act variant: a new
// entity's proposed outbound references are evaluated as a set).
func (t *Tree) ReachesScopeAny(targets []string, scopeEntity string) bool {
	for _, target := range targets {
		if t.ReachesScope(target, scopeEntity) {
			return true
		}
	}
	return false
}

// parentChainReaches climbs `from`'s parent chain — canonicalizing each
// hop — and reports whether it reaches `to`. The visited guard
// terminates on a malformed parent cycle: the loader tolerates invalid
// parent edges (errors-are-findings) that the FSM/ref checks flag
// separately, so reachability must not assume an acyclic chain.
func (t *Tree) parentChainReaches(from, to string) bool {
	cur := from
	visited := map[string]bool{}
	for cur != "" && !visited[cur] {
		if cur == to {
			return true
		}
		visited[cur] = true
		e := t.ByID(cur)
		if e == nil {
			return false
		}
		cur = compositeParentOrSame(e.Parent)
	}
	return false
}

// compositeParentOrSame returns the parent id of a composite (e.g.
// "M-007/AC-1" → "M-007"); returns the input unchanged when it isn't
// a composite. The result is also passed through entity.Canonicalize
// so callers comparing two ids in the reach graph never compare a
// narrow form against a canonical form. The shared trim used by
// Reaches / ReachesAny.
func compositeParentOrSame(id string) string {
	if parent, _, ok := entity.ParseCompositeID(id); ok {
		return entity.Canonicalize(parent)
	}
	return entity.Canonicalize(id)
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
//
// Both the query id and each candidate entity's id are run through
// entity.Canonicalize before comparison, so a narrow legacy query
// (`E-22`) resolves the same canonical entity as `E-0022`. This is
// the AC-2 lookup-seam canonicalization: the grammar accepts both
// widths, the loader stores whatever shape was on disk, and the
// lookup compares canonical form.
func (t *Tree) ByID(id string) *entity.Entity {
	canon := entity.Canonicalize(id)
	for _, e := range t.Entities {
		if entity.Canonicalize(e.ID) == canon {
			return e
		}
	}
	return nil
}

// ByIDAll returns every entity matching the id, in tree-walk order.
// Used by `aiwf reallocate` to detect the duplicate-id case so the
// trunk-ancestry tiebreaker can run; ByID alone would silently pick
// one and obscure that there's a choice to make.
//
// Comparison goes through entity.Canonicalize on both sides — see
// the docstring on ByID for the AC-2 width-tolerance rule.
func (t *Tree) ByIDAll(id string) []*entity.Entity {
	canon := entity.Canonicalize(id)
	var out []*entity.Entity
	for _, e := range t.Entities {
		if entity.Canonicalize(e.ID) == canon {
			out = append(out, e)
		}
	}
	return out
}

// ByPriorID returns the entity whose `prior_ids` lineage list
// includes id, or nil if no entity claims that id as a prior. When
// multiple entities claim the same prior id (a hand-edit accident or
// the rare lineage-broken case), ByPriorID returns the first match
// in tree-walk order; iterate Entities to enumerate all.
//
// Comparison goes through entity.Canonicalize on both sides — see
// the docstring on ByID for the AC-2 width-tolerance rule.
func (t *Tree) ByPriorID(id string) *entity.Entity {
	canon := entity.Canonicalize(id)
	for _, e := range t.Entities {
		for _, p := range e.PriorIDs {
			if entity.Canonicalize(p) == canon {
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

// FilterByKindStatuses returns entities whose Kind matches k (when k
// is non-empty) and whose Status appears in statuses (when statuses
// is non-empty), sorted by ID ascending. Empty k means "any kind";
// empty statuses means "any status". Used by both `aiwf list --kind
// X --status Y` and `aiwf status`'s per-section slices so the two
// verbs cannot drift on the same query.
func (t *Tree) FilterByKindStatuses(k entity.Kind, statuses ...string) []*entity.Entity {
	var statusSet map[string]struct{}
	if len(statuses) > 0 {
		statusSet = make(map[string]struct{}, len(statuses))
		for _, s := range statuses {
			statusSet[s] = struct{}{}
		}
	}
	out := make([]*entity.Entity, 0)
	for _, e := range t.Entities {
		if k != "" && e.Kind != k {
			continue
		}
		if statusSet != nil {
			if _, ok := statusSet[e.Status]; !ok {
				continue
			}
		}
		out = append(out, e)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
