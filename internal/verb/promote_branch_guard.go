package verb

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// requireExpectedBranchForActivatingTransition guards G-0269: an epic
// proposed -> active or milestone -> in_progress promote is a
// sovereign activating act that must land on ADR-0010's expected
// parent branch (trunk for an epic, the parent epic's ritual branch
// for a milestone) — never wherever a concurrent session happened to
// leave HEAD checked out between the operator's preflight and this
// commit. Reuses G-0270's branch-choreography model
// (expectedActivationBranch below mirrors internal/cli/check/
// provenance.go's expectedParentBranchesForPromote) as a synchronous
// pre-commit refusal rather than only a post-hoc check.RunPromoteOnWrongBranch
// finding: the sovereign act refuses and re-surfaces instead of
// silently landing on the wrong branch.
//
// Caller has already verified !force; this helper does not re-check
// that. --force is the explicit override (already human-only via the
// existing provenance coherence rule), matching
// requireHumanActorForSovereignAct's own contract — no new flag.
//
// Silent (no error) for any (kind, newStatus) outside the two
// activating transitions, and when the expected branch can't be
// resolved (no configured trunk name, or a milestone missing its
// parent epic) — an unresolvable expectation is not evidence of a
// violation, matching G-0270's own fail-shut posture.
func requireExpectedBranchForActivatingTransition(ctx context.Context, t *tree.Tree, e *entity.Entity, newStatus string) error {
	expected, ok := expectedActivationBranch(t, e, newStatus)
	if !ok {
		return nil
	}
	current, err := gitops.CurrentBranch(ctx, t.Root)
	if err != nil {
		//coverage:ignore defensive: CurrentBranch only errors on a git failure other than detached HEAD (handled as "", nil) — not reachable deterministically in-process
		return fmt.Errorf("aiwf promote %s %s: could not determine the current branch to verify it matches the expected parent branch %q: %w; check out %q and retry, or use `--force --reason \"...\"` to override", e.ID, newStatus, expected, err, expected)
	}
	if current == expected {
		return nil
	}
	return fmt.Errorf("aiwf promote %s %s: refusing to land on %q — this activation is expected on %q (a concurrent session checked out a different branch here? see G-0269); `git checkout %s` and retry, or use `--force --reason \"...\"` to override", e.ID, newStatus, currentBranchLabel(current), expected, expected)
}

// currentBranchLabel renders CurrentBranch's result for the refusal
// message — "" (detached HEAD) reads better as an explicit label than
// an empty pair of quotes.
func currentBranchLabel(current string) string {
	if current == "" {
		return "(detached HEAD)"
	}
	return current
}

// expectedActivationBranch returns the branch this (entity, newStatus)
// activating-promote must land on, per ADR-0010, and whether the
// expectation is resolvable at all. Mirrors internal/cli/check/
// provenance.go's expectedParentBranchesForPromote (the post-hoc
// check-rule's own derivation) so the pre-commit guard and the
// post-hoc finding agree on what "expected" means; internal/verb
// cannot import that CLI-layer function directly (layering direction:
// verb sits below cli), so the small derivation is duplicated in each
// package's own natural types rather than exported across the
// boundary for a single caller.
func expectedActivationBranch(t *tree.Tree, e *entity.Entity, newStatus string) (string, bool) {
	switch {
	case e.Kind == entity.KindEpic && newStatus == entity.StatusActive:
		cfg, err := config.Load(t.Root)
		if err != nil || cfg == nil {
			cfg = &config.Config{}
		}
		trunk := cfg.TrunkBranchShortName()
		return trunk, trunk != ""
	case e.Kind == entity.KindMilestone && newStatus == entity.StatusInProgress:
		if e.Parent == "" {
			return "", false
		}
		parent := t.ByID(e.Parent)
		if parent == nil || parent.Kind != entity.KindEpic {
			return "", false // parent lookup failed — fail-shut, not a violation
		}
		parentDir := filepath.Base(filepath.Dir(parent.Path))
		if parentDir == "" || parentDir == "." {
			return "", false
		}
		return "epic/" + parentDir, true
	default:
		return "", false
	}
}
