package promote_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/cli/promote"
)

// M-0253/AC-1 backfill: promote.Run carries the largest concentration
// of entity-lifecycle guards branch-coverage-audit flags in this
// milestone's wave-2 scope — its own --phase/positional-status mutex,
// --force/--audit-only gating, and resolver-flag (--by/--by-commit/
// --superseded-by) mutex checks, on top of the generic
// ResolveRoot/ResolveActor/tree.Load guard shape shared with every
// other entity-lifecycle verb. This file drives each flagged guard
// directly. The ResolveRoot and tree.Load "fatal IO error" branches
// are `//coverage:ignore`d in promote.go itself, mirroring the
// established internal/cli/archive and wave-1
// internal/cli/add/internal/cli/editbody precedent — those errors are
// not deterministically reproducible in a unit-test harness.

// runArgs bundles promote.Run's many positional parameters with
// zero-value defaults so each test below only overrides what it needs
// to reach its target branch.
type runArgs struct {
	args         []string
	actor        string
	principal    string
	root         string
	reason       string
	phase        string
	tests        string
	by           string
	byCommit     string
	supersededBy string
	force        bool
	auditOnly    bool
	out          cliutil.OutputFormat
}

func (a runArgs) run() int {
	return promote.Run(a.args, a.actor, a.principal, a.root, a.reason,
		a.phase, a.tests, a.by, a.byCommit, a.supersededBy, a.force, a.auditOnly, a.out)
}

// TestRun_PhaseMutexWithPositionalStatus covers the `--phase` +
// positional new-status mutex: both supplied at once is a usage error
// regardless of id shape, checked before any root/tree work.
func TestRun_PhaseMutexWithPositionalStatus(t *testing.T) {
	t.Parallel()
	rc := runArgs{args: []string{"E-0001", "active"}, phase: "green"}.run()
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRun_PhaseRequiresCompositeID covers the `--phase` composite-id
// guard: `--phase` on a top-level (non-composite) id is a usage error.
func TestRun_PhaseRequiresCompositeID(t *testing.T) {
	t.Parallel()
	rc := runArgs{args: []string{"E-0001"}, phase: "green"}.run()
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRun_ForceAndAuditOnlyMutex covers the --force/--audit-only
// mutex: --force makes a transition, --audit-only records one that
// already happened, so both together is a usage error.
func TestRun_ForceAndAuditOnlyMutex(t *testing.T) {
	t.Parallel()
	rc := runArgs{args: []string{"E-0001", "active"}, force: true, auditOnly: true}.run()
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRun_ForceRequiresReason covers the --force/--audit-only
// --reason gate: either flag set with an empty (or whitespace-only)
// --reason is a usage error. Table-driven across both flags: --force
// alone doesn't reach the `gateFlag = "--audit-only"` reassignment
// (G-0411, found during M-0253's wrap review — this line predates the
// diff-scoped audit's base so branch-coverage-audit never flagged it),
// so the --audit-only case is needed to exercise it.
func TestRun_ForceRequiresReason(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		force     bool
		auditOnly bool
	}{
		{name: "force", force: true},
		{name: "audit-only", auditOnly: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rc := runArgs{
				args: []string{"E-0001", "active"}, reason: "  ",
				force: tc.force, auditOnly: tc.auditOnly,
			}.run()
			if rc != cliutil.ExitUsage {
				t.Errorf("rc = %d, want ExitUsage", rc)
			}
		})
	}
}

// TestRun_ResolverFlagsNotAllowedWithAuditOnly covers the resolver-flag
// (--by/--by-commit/--superseded-by) + --audit-only mutex: audit-only
// records an existing transition, so a resolver-flag value implying a
// mutation is a usage error.
func TestRun_ResolverFlagsNotAllowedWithAuditOnly(t *testing.T) {
	t.Parallel()
	rc := runArgs{
		args: []string{"E-0001", "active"}, auditOnly: true,
		reason: "manual flip from earlier", by: "G-0001",
	}.run()
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRun_ResolverFlagsNotValidInPhaseMode covers the resolver-flag +
// --phase mutex: resolver fields apply to entity status, not AC
// phase, so combining them is a usage error.
func TestRun_ResolverFlagsNotValidInPhaseMode(t *testing.T) {
	t.Parallel()
	rc := runArgs{args: []string{"M-0001/AC-1"}, phase: "green", by: "G-0001"}.run()
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRun_ResolveActorFailure covers Run's cliutil.ResolveActor guard
// using M-0252's BrokenGitIdentity fixture. Serial: BrokenGitIdentity
// uses t.Setenv, which panics under t.Parallel.
func TestRun_ResolveActorFailure(t *testing.T) {
	testutil.BrokenGitIdentity(t)
	root := t.TempDir()
	rc := runArgs{args: []string{"E-0001", "active"}, root: root}.run()
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRun_PhaseAuditOnlyRejectsTests covers phase-mode audit-only's
// own --tests guard, reached past a successful
// root/actor/lock/tree-load sequence: audit-only records an existing
// transition, so a --tests value (implying a test cycle ran) is a
// usage error.
func TestRun_PhaseAuditOnlyRejectsTests(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	rc := runArgs{
		args: []string{"M-0001/AC-1"}, actor: "human/test", root: root,
		reason: "already advanced by hand", phase: "green",
		tests: "pass=1 fail=0 skip=0 total=1", auditOnly: true,
	}.run()
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRun_NonHumanForceIsRefusedAtTheDispatcher covers the
// sovereign-force pre-check Run makes right after the prelude
// (M-0293/AC-3). The guard's own arms — message, exit class, actor
// shapes — are covered in internal/cli/cliutil; what this pins is that
// the dispatcher calls it at all, and that a human is not caught by it.
//
// The human case is the discriminator. Both invocations name an entity
// no tree here contains, so both fail; only the non-human one may fail
// with the legality exit the coherence refusal carries.
func TestRun_NonHumanForceIsRefusedAtTheDispatcher(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	base := runArgs{
		args: []string{"E-0001", "active"}, root: root,
		reason: "an agent reaching for a sovereign act", force: true,
	}

	nonHuman := base
	nonHuman.actor = "ai/claude"
	if rc := nonHuman.run(); rc != cliutil.ExitFindings {
		t.Errorf("non-human --force: rc = %d, want ExitFindings (%d) — the dispatcher never "+
			"consulted the sovereign-force guard", rc, cliutil.ExitFindings)
	}

	human := base
	human.actor = "human/test"
	if rc := human.run(); rc == cliutil.ExitFindings {
		t.Error("human --force reached the same exit as the non-human one, so the assertion " +
			"above proves nothing about the guard — both may simply be failing on the missing entity")
	}
}
