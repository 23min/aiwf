package verb

import (
	"context"
	"fmt"

	"github.com/23min/aiwf/internal/branchparse"
	"github.com/23min/aiwf/internal/config"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/tree"
)

// refuseEpicCreationOnRitualBranch refuses `aiwf add epic` while the
// operator stands on a ritual branch (D-0074).
//
// An epic's activating promote must land on trunk per ADR-0010 Tier 1.
// An epic created on a ritual branch lives only there until that branch
// merges, so the promote refuses, and moving to trunk puts the entity
// out of view — the operator meets "entity not found", a message about
// a different problem (G-0616). Every exit from that state is poor:
// merging the branch to trunk drags unrelated in-flight work along,
// `--force` records a sovereign override for an accident, and
// cherry-picking leaves the entity on two branches. Refusing here costs
// nothing, because no entity exists yet.
//
// The predicate is the branch's *rung*, not inequality with trunk.
// Those differ: in a repo whose trunk is `master` while the configured
// trunk name is the default `main`, every branch is unequal to trunk,
// and a guard written that way refuses every epic creation in the repo.
// branchparse.RungOf classifies against ADR-0010's grammar and reports
// "" for such a branch, so only the ritual rungs — the branches that
// merge later, which is what strands the entity — reach the refusal.
//
// Scoped to epics. ADR-0010 Tier 2 sanctions gaps discovered during
// ritual work landing on the branch, so a guard across kinds would
// refuse a documented flow. Milestones activate on their parent epic's
// branch rather than trunk, which turns on whether trunk was merged
// into that branch — the wrap rituals' reconcile step, not a
// creation-time rule.
//
// Silent when the current branch cannot be read, matching the promote
// guard's posture that an unresolvable expectation is not evidence of a
// violation. opts.Force is the bypass, already sovereign and human-only
// on this verb.
func refuseEpicCreationOnRitualBranch(ctx context.Context, t *tree.Tree, kind entity.Kind, opts AddOptions) error {
	if kind != entity.KindEpic || opts.Force {
		return nil
	}
	cfg, err := config.Load(t.Root)
	if err != nil || cfg == nil {
		//coverage:ignore defensive: config.Load failing is already fatal to the surrounding verb, which loads the tree from the same root
		cfg = &config.Config{}
	}
	trunk := cfg.TrunkBranchShortName()
	current, err := gitops.CurrentBranch(ctx, t.Root)
	if err != nil {
		//coverage:ignore defensive: CurrentBranch errors only on a git failure other than detached HEAD, which the surrounding verb would already have hit
		return nil
	}
	switch branchparse.RungOf(current, trunk) {
	case "epic", "milestone", "patch":
	default:
		return nil
	}
	return fmt.Errorf("aiwf add epic: refusing to create on ritual branch %q — an epic is created on trunk so its activation can land there (ADR-0010 Tier 1); created here it exists only on this branch until it merges, and the activating promote would refuse. Create it from trunk, or use `--force --reason \"...\"` to create here anyway", current)
}
