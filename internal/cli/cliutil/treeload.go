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

// ConfiguredAreaMembers returns the consumer's declared workstream
// area tags from `aiwf.yaml: areas.members` (E-0043), or nil when no
// areas block is declared (or aiwf.yaml is absent/unreadable). This is
// the single source of truth the `aiwf add --area` write-time validation
// and the `--area` shell completion both read — the same declared set
// the M-0172 area-unknown check consults. Tolerant of a missing
// aiwf.yaml so dispatchers and completion funcs call it unconditionally.
func ConfiguredAreaMembers(rootDir string) []string {
	cfg, err := config.Load(rootDir)
	if err != nil || cfg == nil {
		return nil
	}
	return cfg.Areas.MemberNames()
}

// ConfiguredAreaMembersFull returns the consumer's declared workstream areas
// with their full label+location shape (E-0044, M-0179) — the member set the
// `aiwf rename-area` writer needs so it can preserve each member's `paths`
// across a rename. Returns nil when no areas block is declared (or aiwf.yaml is
// absent/unreadable). The name-only readers stay on ConfiguredAreaMembers /
// ConfiguredAreas (derived via MemberNames); only rename-area, which writes the
// block back, reads the full members.
func ConfiguredAreaMembersFull(rootDir string) []config.Member {
	cfg, err := config.Load(rootDir)
	if err != nil || cfg == nil {
		return nil
	}
	return cfg.Areas.Members
}

// ConfiguredAreaRequired returns the consumer's `aiwf.yaml: areas.required`
// flag (M-0178), or false when no areas block is declared (or aiwf.yaml is
// absent/unreadable). The single source of truth the `aiwf add` write-time
// refusal reads — the verb-time twin of the M-0178 area-required check.
// Tolerant of a missing aiwf.yaml so the dispatcher calls it unconditionally.
func ConfiguredAreaRequired(rootDir string) bool {
	cfg, err := config.Load(rootDir)
	if err != nil || cfg == nil {
		return false
	}
	return cfg.Areas.Required
}

// ConfiguredAreas returns the consumer's declared workstream areas from
// `aiwf.yaml: areas` (E-0043, M-0175): the member set and the optional
// `default:` display label for the untagged complement. Both are zero
// (nil, "") when no areas block is declared or aiwf.yaml is absent. The
// area-grouping renderers read this once and pass the pair through; an
// empty member set means flat (zero-migration) rendering.
func ConfiguredAreas(rootDir string) (members []string, defaultLabel string) {
	cfg, err := config.Load(rootDir)
	if err != nil || cfg == nil {
		return nil, ""
	}
	return cfg.Areas.MemberNames(), cfg.Areas.Default
}

// ConfiguredTrunkBranchShortName returns the consumer's trunk short
// name derived from `aiwf.yaml.allocate.trunk` via
// `Config.TrunkBranchShortName()`. Used by `aiwf authorize`'s
// AI-target preflight (M-0161/AC-1, G-0200) so the "main + ritual
// --branch" carve-out honors the configured trunk rather than the
// literal `"main"`. Tolerant of a missing aiwf.yaml — falls back to
// the kernel default trunk ref (`refs/remotes/origin/main`) →
// returns `"main"`, preserving backwards-compatibility for repos
// that never configured the value.
//
// Mirrors ConfiguredTitleMaxLength's shape: the CLI dispatcher calls
// this unconditionally and passes the value through as a verb-options
// primitive; the verb does not depend on config.Config directly.
func ConfiguredTrunkBranchShortName(rootDir string) string {
	cfg, err := config.Load(rootDir)
	if err != nil || cfg == nil {
		return (&config.Config{}).TrunkBranchShortName()
	}
	return cfg.TrunkBranchShortName()
}
