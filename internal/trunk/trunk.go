// Package trunk reads entity ids from the configured trunk ref's
// tree, applying the policy from id-allocation.md:
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
// The package is read-only and has no per-process cache; callers
// invoke once per verb run.
package trunk

import (
	"context"
	"errors"
	"fmt"

	"github.com/23min/ai-workflow-v2/internal/config"
	"github.com/23min/ai-workflow-v2/internal/entity"
	"github.com/23min/ai-workflow-v2/internal/gitops"
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
	return Result{IDs: ids}, nil
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
