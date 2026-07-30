package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/entityview"
)

// TestPromote_SameStatus_NoOp_ExitZeroNoCommit is the CLI-seam half of
// M-0281/AC-1: a verb-layer NoOp must surface through the promote command as
// exit 0 with no new commit. Re-promoting an already-active epic to `active`
// is an FSM-illegal transition (active carries no self-edge), so exit 0 here
// can only come from the NoOp guard — and the unchanged history length proves
// no second commit landed.
func TestPromote_SameStatus_NoOp_ExitZeroNoCommit(t *testing.T) {
	// Serial by design, per this package's setup_test.go skip-list: the NoOp
	// assertion below goes through testutil.CaptureStdout, which swaps the
	// process-global os.Stdout. Declaring t.Parallel() here races every
	// concurrent reader of that fd (cobra's OutOrStdout, cliutil.Println).
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "Foo", "--actor", "human/test", "--root", root)
	mustRun(t, "promote", "--actor", "human/test", "--root", root, "E-0001", "active")

	var rc int
	out := testutil.CaptureStdout(t, func() {
		rc = cli.Execute([]string{"promote", "--actor", "human/test", "--root", root, "E-0001", "active"})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("same-status promote via CLI: rc=%d, want ExitOK (%d)", rc, cliutil.ExitOK)
	}
	// The NoOpMessage must reach stdout via emitSuccess — not just a bare
	// exit 0 — so the operator sees why nothing happened.
	if !strings.Contains(string(out), "E-0001 is already active") {
		t.Errorf("stdout = %q, want it to contain the NoOp message %q", out, "E-0001 is already active")
	}

	events, err := entityview.ReadHistory(context.Background(), root, "E-0001")
	if err != nil {
		t.Fatalf("readHistory: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("history has %d events, want 2 (add, promote) — the NoOp must not commit:\n%+v", len(events), events)
	}
}
