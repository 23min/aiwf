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

// TestCancel_AlreadyTerminal_NoOp_ExitZeroNoCommit is the CLI-seam half of
// M-0281/AC-2: a verb-layer NoOp from cancel must surface through the cancel
// command as exit 0 with no new commit and the state-naming message. Re-cancelling
// an already-cancelled epic previously refused (exit 1, a coded FSM error), so
// exit 0 here can only come from the NoOp guard — and the unchanged history
// length proves no second commit landed.
func TestCancel_AlreadyTerminal_NoOp_ExitZeroNoCommit(t *testing.T) {
	// Serial by design, per this package's setup_test.go skip-list: the NoOp
	// assertion below goes through testutil.CaptureStdout, which swaps the
	// process-global os.Stdout. Declaring t.Parallel() here races every
	// concurrent reader of that fd (cobra's OutOrStdout, cliutil.Println).
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "Doomed", "--actor", "human/test", "--root", root)
	mustRun(t, "cancel", "--actor", "human/test", "--root", root, "E-0001")

	var rc int
	out := testutil.CaptureStdout(t, func() {
		rc = cli.Execute([]string{"cancel", "--actor", "human/test", "--root", root, "E-0001"})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("cancel of an already-cancelled epic via CLI: rc=%d, want ExitOK (%d)", rc, cliutil.ExitOK)
	}
	if !strings.Contains(string(out), "already at terminal status") {
		t.Errorf("stdout = %q, want it to contain the NoOp message", out)
	}

	events, err := entityview.ReadHistory(context.Background(), root, "E-0001")
	if err != nil {
		t.Fatalf("readHistory: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("history has %d events, want 2 (add, cancel) — the NoOp must not commit:\n%+v", len(events), events)
	}
}
