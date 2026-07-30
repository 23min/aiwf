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

// TestMove_ToCurrentParent_NoOp_ExitZeroNoCommit is the CLI-seam half of
// M-0281/AC-3: moving a milestone to the epic it is already under surfaces
// through the move command as exit 0 with no new commit and the state-naming
// message. It previously refused (exit 2), so exit 0 here can only come from
// the NoOp guard, and the unchanged history length proves no commit landed.
func TestMove_ToCurrentParent_NoOp_ExitZeroNoCommit(t *testing.T) {
	// Serial by design, per this package's setup_test.go skip-list: the NoOp
	// assertion below goes through testutil.CaptureStdout, which swaps the
	// process-global os.Stdout. Declaring t.Parallel() here races every
	// concurrent reader of that fd (cobra's OutOrStdout, cliutil.Println).
	root := setupCLITestRepo(t)
	acBody := acBodyFixturePath(t, root)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "Platform", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--epic", "E-0001", "--tdd", "none", "--title", "Cache", "--actor", "human/test", "--root", root)
	// An AC keeps the milestone shape realistic; not required for the move.
	mustRun(t, "add", "ac", "M-0001", "--title", "does the thing", "--body-file", acBody, "--actor", "human/test", "--root", root)

	var rc int
	out := testutil.CaptureStdout(t, func() {
		rc = cli.Execute([]string{"move", "--actor", "human/test", "--root", root, "M-0001", "--epic", "E-0001"})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("move to current parent via CLI: rc=%d, want ExitOK (%d)", rc, cliutil.ExitOK)
	}
	if !strings.Contains(string(out), "already under epic") {
		t.Errorf("stdout = %q, want it to contain the NoOp message", out)
	}

	events, err := entityview.ReadHistory(context.Background(), root, "M-0001")
	if err != nil {
		t.Fatalf("readHistory: %v", err)
	}
	// add milestone + add ac = 2 events; the NoOp move must add none.
	if len(events) != 2 {
		t.Fatalf("history has %d events, want 2 (add, add ac) — the NoOp must not commit:\n%+v", len(events), events)
	}
}
