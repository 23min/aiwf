package integration

import (
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// TestAutoEnd_TerminalStatus_EndsAPausedScope is M-0325/AC-4.
//
// The automatic end fires where it already fires; what this pins is
// which scopes it covers. A paused scope on an entity that reaches a
// terminal status is finished, not suspended — nothing can resume it,
// because resuming is a transition on an entity no verb will act on
// again — yet the scope FSM permits `paused → ended` and, before this
// milestone, nothing fired it.
//
// The assertion reads the replayed state rather than the emitted
// trailer, so it holds whichever way the predicate that selects the
// scopes is written.
func TestAutoEnd_TerminalStatus_EndsAPausedScope(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)

	cases := []struct {
		name    string
		closing []string
	}{
		{"closed by promote", []string{"promote", "E-0001", "done"}},
		{"closed by cancel", []string{"cancel", "E-0001", "--reason", "abandoning the epic"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root, binDir := sovereignScopedRepo(t, [][]string{{"promote", "E-0001", "active"}})

			for _, args := range [][]string{
				{"authorize", "E-0001", "--pause", "holding while the plan is reworked"},
				// Both closing verbs refuse an epic with a non-terminal
				// child, so the milestone goes first either way.
				{"cancel", "M-0001", "--reason", "not doing this one"},
			} {
				if out, err := testutil.RunBin(t, root, binDir, nil, args...); err != nil {
					t.Fatalf("setup aiwf %v: %v\n%s", args, err, out)
				}
			}

			if _, paused := showScopes(t, root, binDir, "E-0001"); paused[0].State != "paused" {
				t.Fatalf("fixture scope is %q, want paused — the case is about the state the auto-end "+
					"skipped, so reaching it is a precondition", paused[0].State)
			}

			if out, err := testutil.RunBin(t, root, binDir, nil, tc.closing...); err != nil {
				t.Fatalf("aiwf %v: %v\n%s", tc.closing, err, out)
			}

			status, after := showScopes(t, root, binDir, "E-0001")
			if after[0].State != "ended" {
				t.Errorf("E-0001 reached %q with its scope still %q; a delegation left paused on a "+
					"closed entity is stranded — its own FSM permits the exit that nothing fires",
					status, after[0].State)
			}
		})
	}
}
