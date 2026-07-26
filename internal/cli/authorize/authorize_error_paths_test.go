package authorize_test

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/authorize"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// M-0255/AC-1 backfill: authorize.Run's ResolveRoot and tree.Load
// guards, plus the LoadEntityScopes guard, are `//coverage:ignore`d in
// authorize.go itself, mirroring the established internal/cli/archive
// and this milestone's internal/cli/status precedent. The remaining
// flagged branches — the mode-selection mutex, the --reason/--pause
// exclusivity gate, the --force gates, and actor resolution — get
// real tests below.

// TestRun_ModeMutex covers the exactly-one-of --to/--pause/--resume
// guard: zero and two-or-more selected modes are both usage errors,
// checked before any root/tree work.
func TestRun_ModeMutex(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		to     string
		pause  string
		resume string
	}{
		{name: "none selected"},
		{name: "to and pause both set", to: "ai/claude", pause: "blocked"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rc := authorize.Run(authorize.Options{ID: "E-0001", To: tc.to, Pause: tc.pause, Resume: tc.resume})
			if rc != cliutil.ExitUsage {
				t.Errorf("rc = %d, want ExitUsage", rc)
			}
		})
	}
}

// TestRun_ReasonNotUsedWithPauseOrResume covers the --reason/--pause
// exclusivity gate: --pause's argument is itself the reason, so a
// separate --reason is a usage error.
func TestRun_ReasonNotUsedWithPauseOrResume(t *testing.T) {
	t.Parallel()
	rc := authorize.Run(authorize.Options{ID: "E-0001", Pause: "blocked", Reason: "also a reason"})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRun_ForceRequiresTo covers the --force gate: --force only
// applies to --to (overriding the terminal-scope-entity refusal).
func TestRun_ForceRequiresTo(t *testing.T) {
	t.Parallel()
	rc := authorize.Run(authorize.Options{ID: "E-0001", Pause: "blocked", Force: true})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRun_ForceRequiresReason covers --force's own --reason gate: a
// whitespace-only reason is rejected the same as an empty one.
func TestRun_ForceRequiresReason(t *testing.T) {
	t.Parallel()
	rc := authorize.Run(authorize.Options{ID: "E-0001", To: "ai/claude", Reason: "   ", Force: true})
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
	rc := authorize.Run(authorize.Options{ID: "E-0001", Root: root, To: "ai/claude", Reason: "delegate"})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}
