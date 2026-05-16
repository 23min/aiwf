package cliutil

import (
	"context"
	"errors"
	"fmt"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/trunk"
)

// LoadTreeWithTrunk loads the consumer repo's entity tree and stamps
// the configured trunk ref's ids onto Tree.TrunkIDs, so the allocator
// (entity.AllocateID) and the cross-tree ids-unique check both see
// trunk in their view.
//
// Behavior matches package trunk: when the repo has no remotes the
// trunk read is silently skipped; when remotes exist but the
// configured trunk ref does not resolve the call returns an error
// and the operator sees a clear message.
//
// A missing aiwf.yaml is not fatal — the trunk read uses the default
// trunk ref in that case.
func LoadTreeWithTrunk(ctx context.Context, rootDir string) (*tree.Tree, []tree.LoadError, error) {
	tr, loadErrs, err := tree.Load(ctx, rootDir)
	if err != nil {
		return tr, loadErrs, err
	}
	cfg, err := config.Load(rootDir)
	if err != nil && !errors.Is(err, config.ErrNotFound) {
		return tr, loadErrs, fmt.Errorf("loading aiwf.yaml: %w", err)
	}
	res, err := trunk.Read(ctx, rootDir, cfg)
	if err != nil {
		return tr, loadErrs, err
	}
	tr.TrunkIDs = res.IDs
	if !res.Skipped {
		ref, _ := cfg.AllocateTrunkRef()
		tr.TrunkRef = ref
		// G-0109: hand the ids-unique trunk-collision check the set
		// of renames git detects between trunk and the working tree,
		// so a feature-branch slug rename of an existing entity is
		// treated as the same entity moved rather than a duplicate id
		// allocation. RenamesFromRef returns nil (no map) when the
		// trunk ref doesn't resolve — but res.Skipped already covers
		// the no-remotes case, so reaching here implies the ref
		// resolved.
		renames, err := gitops.RenamesFromRef(ctx, rootDir, ref)
		if err != nil {
			return tr, loadErrs, err
		}
		tr.TrunkRenames = renames
	}
	return tr, loadErrs, nil
}

// ConfiguredTitleMaxLength returns the consumer's
// `entities.title_max_length` from aiwf.yaml, or the kernel default
// when absent (G-0102). Tolerant of a missing aiwf.yaml — the kernel
// default applies in that case too, so the verb dispatchers in
// cmd/aiwf can call this unconditionally without a precondition
// check.
func ConfiguredTitleMaxLength(rootDir string) int {
	cfg, err := config.Load(rootDir)
	if err != nil || cfg == nil {
		return config.DefaultEntityTitleMaxLength
	}
	return cfg.EntityTitleMaxLength()
}
