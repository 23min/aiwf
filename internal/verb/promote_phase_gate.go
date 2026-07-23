package verb

import (
	"context"
	"fmt"
	"strings"

	"github.com/23min/aiwf/internal/areamatch"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// requireDiffShapeForPhasePromote enforces red-first ordering on `--phase red`
// and `--phase green` promotes (M-0276, D-0047): it classifies the working-tree
// dirty paths against the configured tdd.test_paths globs and refuses a
// `--phase red` when any non-test path is already dirty (implementation before
// test) or nothing is dirty at all (a red phase with no failing test written),
// and a `--phase green` when no non-test (implementation) path is dirty (no
// implementation to have turned the test green). The check is stateless — it
// inspects the current diff only, with no red-time snapshot.
//
// Opt-in: a no-op unless tdd.test_paths is configured, and silent for any phase
// other than red or green. The caller has already verified !force; --force is
// the explicit human-only override (via the existing provenance rule), so this
// helper does not re-check it.
func requireDiffShapeForPhasePromote(ctx context.Context, t *tree.Tree, newPhase string) error {
	if newPhase != entity.TDDPhaseRed && newPhase != entity.TDDPhaseGreen {
		return nil
	}
	cfg, err := config.Load(t.Root)
	if err != nil || cfg == nil {
		cfg = &config.Config{}
	}
	globs := cfg.TDD.TestPaths
	if len(globs) == 0 {
		return nil // opt-in: gate inactive when no test-path globs are configured
	}
	dirty, err := gitops.DirtyPaths(ctx, t.Root)
	if err != nil {
		//coverage:ignore defensive: DirtyPaths only errors on a broken repo (e.g. unborn HEAD); unreachable here — promoting an AC requires a committed milestone, so HEAD exists
		return fmt.Errorf("aiwf promote --phase red: could not inspect the working tree for the red/green diff-shape gate: %w", err)
	}
	var testDirty int
	var nonTestDirty []string
	for _, p := range dirty {
		if pathMatchesAnyGlob(globs, p) {
			testDirty++
		} else {
			nonTestDirty = append(nonTestDirty, p)
		}
	}
	switch newPhase {
	case entity.TDDPhaseRed:
		if len(nonTestDirty) > 0 {
			return fmt.Errorf("aiwf promote --phase red: refusing — red-first requires the test to change before the implementation, but these non-test paths are already dirty: %s; write the failing test first, or use `--force --reason \"...\"` to override", strings.Join(nonTestDirty, ", "))
		}
		if testDirty == 0 {
			return fmt.Errorf("aiwf promote --phase red: refusing — a red phase records a failing test, but no test-path changes are present in the working tree; write the failing test first, or use `--force --reason \"...\"` to override")
		}
	case entity.TDDPhaseGreen:
		if len(nonTestDirty) == 0 {
			return fmt.Errorf("aiwf promote --phase green: refusing — a green phase records the implementation that turns the test green, but no non-test (implementation) changes are present in the working tree; write the implementation first, or use `--force --reason \"...\"` to override")
		}
	}
	return nil
}

// pathMatchesAnyGlob reports whether the repo-relative path matches any of the
// test-path globs (M-0276). The globs are Tier-1 validated at config load
// (M-0276/AC-1), so areamatch.Match cannot return a pattern error here; a
// theoretical bad pattern is treated as non-matching.
func pathMatchesAnyGlob(globs []string, path string) bool {
	for _, g := range globs {
		if ok, err := areamatch.Match(g, path); err == nil && ok {
			return true
		}
	}
	return false
}
