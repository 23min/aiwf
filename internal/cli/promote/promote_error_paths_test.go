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
// --reason is a usage error.
func TestRun_ForceRequiresReason(t *testing.T) {
	t.Parallel()
	rc := runArgs{args: []string{"E-0001", "active"}, force: true, reason: "  "}.run()
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
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
