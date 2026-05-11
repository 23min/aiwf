package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/trunk"
)

// loadTreeWithTrunk loads the consumer repo's entity tree and stamps
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
func loadTreeWithTrunk(ctx context.Context, rootDir string) (*tree.Tree, []tree.LoadError, error) {
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
	}
	return tr, loadErrs, nil
}

// configuredTitleMaxLength returns the consumer's
// `entities.title_max_length` from aiwf.yaml, or the kernel default
// when absent (G-0102). Tolerant of a missing aiwf.yaml — the kernel
// default applies in that case too, so the verb dispatchers in
// cmd/aiwf can call this unconditionally without a precondition
// check.
func configuredTitleMaxLength(rootDir string) int {
	cfg, err := config.Load(rootDir)
	if err != nil || cfg == nil {
		return config.DefaultEntityTitleMaxLength
	}
	return cfg.EntityTitleMaxLength()
}
