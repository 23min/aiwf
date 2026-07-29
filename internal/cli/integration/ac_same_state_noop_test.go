package integration

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// TestCompositeSameState_NoOp_ExitZeroNoCommit is the CLI-seam half of
// M-0281/AC-9: promoting an AC to the status it already holds, and cancelling
// one already at a terminal status, each surface through their command as exit 0
// with the NoOp message and no new commit. Both previously refused — promote via
// the FSM at exit 1, cancel via a bespoke error at exit 2 — so exit 0 here can
// only come from the convergence guards.
func TestCompositeSameState_NoOp_ExitZeroNoCommit(t *testing.T) {
	// Serial by design, per this package's setup_test.go skip-list: the NoOp
	// assertion below goes through testutil.CaptureStdout, which swaps the
	// process-global os.Stdout. Declaring t.Parallel() here races every
	// concurrent reader of that fd (cobra's OutOrStdout, cliutil.Println).
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--title", "Cache", "--epic", "E-0001", "--tdd", "none",
		"--actor", "human/test", "--root", root)
	mustRun(t, "add", "ac", "M-0001", "--title", "warms on boot", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "ac", "M-0001", "--title", "second criterion", "--actor", "human/test", "--root", root)

	// AC-1 to `met`, AC-2 to the terminal `deferred` — the status whose cancel
	// used to write an edge the FSM does not contain.
	mustRun(t, "promote", "M-0001/AC-1", "met", "--actor", "human/test", "--root", root)
	mustRun(t, "promote", "M-0001/AC-2", "deferred", "--actor", "human/test", "--root", root)

	cases := []struct {
		name    string
		args    []string
		wantMsg string
	}{
		{
			name:    "promote an AC to the status it already holds",
			args:    []string{"promote", "M-0001/AC-1", "met", "--actor", "human/test", "--root", root},
			wantMsg: "is already met",
		},
		{
			name:    "cancel an AC already at a terminal status",
			args:    []string{"cancel", "M-0001/AC-2", "--reason", "probe", "--actor", "human/test", "--root", root},
			wantMsg: "already at terminal status",
		},
	}
	for _, tc := range cases {
		// Serial by design: both cases drive the same repo and assert the
		// commit count, so a parallel run would race on the shared count.
		t.Run(tc.name, func(t *testing.T) {
			before, err := commitCount(t, root)
			if err != nil {
				t.Fatalf("counting commits before: %v", err)
			}
			var rc int
			out := testutil.CaptureStdout(t, func() {
				rc = cli.Execute(tc.args)
			})
			if rc != cliutil.ExitOK {
				t.Fatalf("%s: rc=%d, want ExitOK (%d)", tc.name, rc, cliutil.ExitOK)
			}
			if !strings.Contains(string(out), tc.wantMsg) {
				t.Errorf("stdout = %q, want it to contain %q", out, tc.wantMsg)
			}
			after, err := commitCount(t, root)
			if err != nil {
				t.Fatalf("counting commits after: %v", err)
			}
			if after != before {
				t.Errorf("commit count = %d, want %d — the NoOp must append no commit", after, before)
			}
		})
	}
}
