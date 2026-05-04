// Package trunk reads entity ids from the configured trunk ref's
// tree, applying the policy from id-allocation.md:
//
//   - No remotes configured → skip (the repo has no cross-branch
//     coordination surface; trunk-awareness is moot).
//   - Remote(s) configured AND trunk ref resolves → return the
//     (kind, id, path) triples found under work/ and docs/adr/ in
//     that ref's tree.
//   - Remote(s) configured AND trunk ref missing → hard error.
//     Whether the trunk was the default or explicitly set, the
//     consumer has remotes; an unresolvable trunk is a real
//     misconfiguration to fix, not something to silently work around.
//
// The package is read-only and has no per-process cache; callers
// invoke once per verb run.
package trunk

import (
	"context"
	"errors"
	"fmt"

	"github.com/23min/ai-workflow-v2/tools/internal/config"
	"github.com/23min/ai-workflow-v2/tools/internal/entity"
	"github.com/23min/ai-workflow-v2/tools/internal/gitops"
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

// Read returns the entity ids in cfg's configured trunk ref. It
// applies the no-remote skip and the missing-ref-with-remote hard
// error documented in the package comment.
//
// cfg may be nil — in which case the default trunk ref is used.
// Callers pass workdir as the consumer repo root.
//
// When workdir is not a git repository at all, Read returns
// Skipped silently. Tooling that legitimately runs outside a repo
// (test fixtures, exploratory invocations on plain directories) does
// not have a cross-branch surface to police, so the trunk-awareness
// layer has nothing to add.
func Read(ctx context.Context, workdir string, cfg *config.Config) (Result, error) {
	if !gitops.IsRepo(ctx, workdir) {
		return Result{Skipped: true}, nil
	}
	hasRemotes, err := gitops.HasRemotes(ctx, workdir)
	if err != nil {
		return Result{}, fmt.Errorf("checking git remotes: %w", err)
	}
	if !hasRemotes {
		return Result{Skipped: true}, nil
	}

	ref, _ := cfg.AllocateTrunkRef()
	paths, err := gitops.LsTreePaths(ctx, workdir, ref, "work/", "docs/adr/")
	if err != nil {
		if errors.Is(err, gitops.ErrRefNotFound) {
			return Result{}, fmt.Errorf("trunk ref %q does not resolve in this repo; fetch it or set allocate.trunk in aiwf.yaml", ref)
		}
		return Result{}, fmt.Errorf("reading trunk ref %q: %w", ref, err)
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
