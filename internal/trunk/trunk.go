// Package trunk reads entity ids visible in git refs for the
// allocator's cross-branch view. Its primary surface, Read, scans the
// configured trunk ref's tree, applying the policy from
// id-allocation.md:
//
//   - Not a git repository at all → skip. Tooling that legitimately
//     runs outside a repo (test fixtures, exploratory invocations on
//     plain directories) has no cross-branch surface to police.
//
//   - Configured trunk ref resolves → return the (kind, id, path)
//     triples found under work/ and docs/adr/ in that ref's tree.
//
//   - Trunk ref missing AND was explicitly set in aiwf.yaml → hard
//     error. The user named a specific ref; if it doesn't exist
//     they should fix the typo or fetch it. Silent degradation
//     would defeat their explicit intent.
//
//   - Trunk ref missing AND was the default (no allocate.trunk in
//     aiwf.yaml) AND no refs/remotes/* tracking refs exist → skip.
//     Covers "no remote configured" (sandbox repos), "remote
//     configured but never fetched" (transient setup), and "freshly
//     cloned an empty bare" (canonical first-push setup, where the
//     bare has no branches so the clone has none either). None has
//     anything to collide with.
//
//   - Trunk ref missing AND was the default AND tracking refs DO
//     exist → hard error. The remote has been populated; an
//     unresolvable default trunk is a real misconfiguration (the
//     team's trunk is named something other than main, or the
//     consumer hasn't fetched it). Setting allocate.trunk in
//     aiwf.yaml fixes it.
//
// LocalRefIDs is the second surface: it widens that view to every
// local branch ref (refs/heads/*) for the allocator only — never the
// ids-unique check — so a sibling worktree's freshly-committed id is
// seen at allocation time (M-0212).
//
// The package is read-only and has no per-process cache; callers
// invoke once per verb run.
package trunk

import (
	"context"
	"errors"
	"fmt"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
)

// ID names one entity that exists in the trunk ref's tree, by kind,
// id string, and the repo-relative path the trunk has it at.
type ID struct {
	Kind entity.Kind
	ID   string
	Path string
}

// Result carries the trunk ids visible to the allocator and the
// cross-tree check. Skipped is true when the repo has no remotes and
// the trunk read was deliberately bypassed.
type Result struct {
	IDs     []ID
	Skipped bool
}

// Read returns the entity ids in cfg's configured trunk ref,
// following the policy in this package's doc comment.
//
// cfg may be nil — in which case the default trunk ref is used.
// Callers pass workdir as the consumer repo root.
func Read(ctx context.Context, workdir string, cfg *config.Config) (Result, error) {
	if !gitops.IsRepo(ctx, workdir) {
		return Result{Skipped: true}, nil
	}
	ref, explicit := cfg.AllocateTrunkRef()
	paths, err := gitops.LsTreePaths(ctx, workdir, ref, "work/", "docs/adr/")
	if err != nil {
		if !errors.Is(err, gitops.ErrRefNotFound) {
			return Result{}, fmt.Errorf("reading trunk ref %q: %w", ref, err)
		}
		if explicit {
			return Result{}, fmt.Errorf("trunk ref %q does not resolve in this repo; fetch it or fix allocate.trunk in aiwf.yaml", ref)
		}
		// Default trunk missing. Skip when no tracking refs exist
		// (empty remote, never-fetched, or no remote at all); error
		// when tracking refs are present (the team's trunk is named
		// something other than main and allocate.trunk needs to be
		// set).
		hasTracking, hErr := gitops.HasAnyRemoteTrackingRefs(ctx, workdir)
		if hErr != nil {
			return Result{}, fmt.Errorf("checking remote-tracking refs: %w", hErr)
		}
		if !hasTracking {
			return Result{Skipped: true}, nil
		}
		return Result{}, fmt.Errorf("trunk ref %q does not resolve in this repo; fetch it or set allocate.trunk in aiwf.yaml", ref)
	}

	return Result{IDs: idsFromPaths(paths)}, nil
}

// idsFromPaths converts repo-relative entity paths (as returned by
// gitops.LsTreePaths) into typed ID triples, dropping any path that
// does not classify as a recognized entity file. Shared by Read (the
// trunk ref) and LocalRefIDs (every local branch ref).
func idsFromPaths(paths []string) []ID {
	ids := make([]ID, 0, len(paths))
	for _, p := range paths {
		k, ok := entity.PathKind(p)
		if !ok {
			continue
		}
		id, ok := entity.IDFromPath(p, k)
		if !ok {
			continue
		}
		ids = append(ids, ID{Kind: k, ID: id, Path: p})
	}
	return ids
}

// LocalRefIDs returns the entity id strings reachable from every
// local branch ref (refs/heads/*) in workdir's repository. It is the
// allocator's broadened cross-branch view (M-0212): a sibling git
// worktree's freshly-committed entity already lives in the shared
// local refs, so unioning these ids into the allocator's max keeps
// the next allocation from colliding with it (G-0272 class 1).
//
// Best-effort and read-only: it never returns an error. On any odd
// repo state — not a git repository, no local branches, or a ref that
// lists but fails to read — it degrades to the ids it could collect
// (down to none), never blocking or failing the caller's allocation
// (M-0212/AC-2). The scan narrows the collision window; it is not a
// correctness gate, and the irreducible cross-machine race stays
// `aiwf reallocate`'s to cure.
//
// Unlike Read, the result feeds the allocator ONLY — never the
// ids-unique trunk-collision check, which keeps its working-tree-vs-
// trunk basis. Folding every sibling branch into the uniqueness
// comparison would false-flag the same entity present on two branches
// (e.g. a feature branch forked from main); E-0052's decision is to
// take only the prevention half here.
//
// Cost is one `git ls-tree` per local branch — O(local branches)
// subprocesses per call. Trivial at the solo / handful-of-branches
// scale this targets; it grows linearly, so a repo carrying hundreds
// of stale local branches would pay for them on every allocation.
func LocalRefIDs(ctx context.Context, workdir string) []string {
	if !gitops.IsRepo(ctx, workdir) {
		return nil
	}
	refs, err := gitops.LocalBranchRefs(ctx, workdir)
	if err != nil { //coverage:ignore not portably triggerable: once IsRepo passed, `git for-each-ref` returns 0 even for a repo with broken refs (it warns and skips them); a non-zero exit needs a git-level failure (missing binary) that the unit harness cannot stage. Degrade to local-only.
		return nil
	}
	var ids []string
	for _, ref := range refs {
		paths, err := gitops.LsTreePaths(ctx, workdir, ref, "work/", "docs/adr/")
		if err != nil {
			// An individual ref that lists but won't read (corrupt or
			// raced away mid-scan) is skipped, not fatal — degrade to
			// the rest (M-0212/AC-2).
			continue
		}
		for _, id := range idsFromPaths(paths) {
			ids = append(ids, id.ID)
		}
	}
	return ids
}

// IDStrings returns just the id strings from r, in the order they
// appear in r.IDs. Convenience for AllocateID, which only needs the
// id values.
func (r Result) IDStrings() []string {
	if len(r.IDs) == 0 {
		return nil
	}
	out := make([]string, len(r.IDs))
	for i, id := range r.IDs {
		out[i] = id.ID
	}
	return out
}
