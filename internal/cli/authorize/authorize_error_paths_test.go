package authorize_test

import (
	"strings"
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
		end    bool
	}{
		{name: "none selected"},
		{name: "to and pause both set", to: "ai/claude", pause: "blocked"},
		// --end is a bool rather than a reason-carrying string, so it
		// reaches the counter by a different route than its three
		// siblings; a counter that missed it would open a scope and
		// silently discard the end the operator also asked for.
		{name: "to and end both set", to: "ai/claude", end: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rc := authorize.Run(authorize.Options{ID: "E-0001", To: tc.to, Pause: tc.pause, Resume: tc.resume, End: tc.end})
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

// TestRun_ScopeRequiresEnd covers the --scope gate: every other mode
// either creates a scope or re-derives its target from the FSM, so a
// --scope they ignored would read to the operator as having selected
// one.
func TestRun_ScopeRequiresEnd(t *testing.T) {
	t.Parallel()
	rc := authorize.Run(authorize.Options{ID: "E-0001", To: "ai/claude", ScopeSHA: "1a2b3c4", Reason: "delegate"})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRun_ScopePassedEmpty_Refuses covers the distinction cobra erases:
// an operator who passed `--scope "$SCOPE"` with the variable unset
// arrives with the same empty string as one who passed no --scope at
// all. Falling back to the sole-candidate default there would resolve a
// target they did not name and then end it irreversibly.
func TestRun_ScopePassedEmpty_Refuses(t *testing.T) {
	t.Parallel()
	rc := authorize.Run(authorize.Options{
		ID: "E-0001", Root: t.TempDir(), End: true, Reason: "the selector named nothing",
		ScopeSHA: "   ", ScopeSHASet: true,
	})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRun_BranchWithEndRejected pins what the --branch gate rewrite was
// made for. The gate reads "--branch without --to" rather than
// enumerating the modes that reject it, so --end is refused rather than
// silently dropping the flag; no other test passes Branch alongside End.
func TestRun_BranchWithEndRejected(t *testing.T) {
	t.Parallel()
	rc := authorize.Run(authorize.Options{ID: "E-0001", End: true, Reason: "ending", Branch: "epic/E-0001-eng"})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
}

// TestRun_EndDispatchesToTheEndMode covers the dispatch arm that maps
// --end onto verb.AuthorizeEnd and forwards --scope.
//
// The entity deliberately does not exist, so the verb refuses right
// after Run hands off: what this pins is which mode Run handed off in.
// The exit code cannot say — FinishVerb reports an uncoded refusal as
// ExitUsage, the same code the flag gates return — so the refusal text
// is the discriminator. A dispatch that fell through would leave
// vOpts.Mode at its zero value, AuthorizeOpen, whose own missing-agent
// refusal fires before the tree is consulted and never mentions the id.
//
// Serial: CaptureRun redirects the process's stderr, which cannot be
// shared with a parallel test.
func TestRun_EndDispatchesToTheEndMode(t *testing.T) {
	root := mustNewGitRepo(t)
	mustGit(t, root, "commit", "--allow-empty", "-m", "init")

	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return authorize.Run(authorize.Options{ID: "E-0001", Root: root, End: true, Reason: "no such entity"})
	})
	if rc == cliutil.ExitOK {
		t.Fatalf("rc = ExitOK; the entity does not exist, so the verb must refuse")
	}
	if !strings.Contains(stderr, `entity "E-0001" not found`) {
		t.Errorf("refusal was %q; want the tree lookup that only runs once Run has dispatched a mode "+
			"and called the verb", strings.TrimSpace(stderr))
	}
	if strings.Contains(stderr, "--to") {
		t.Errorf("refusal names --to, so the dispatch fell through to the open mode: %q", strings.TrimSpace(stderr))
	}
}
