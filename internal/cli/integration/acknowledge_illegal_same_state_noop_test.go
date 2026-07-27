package integration

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// TestAcknowledgeIllegal_AlreadyAcknowledged_NoOp_ExitZeroNoCommit is the
// CLI-seam half of M-0281/AC-4: re-acknowledging a SHA that HEAD's history
// already acknowledges exits 0, surfaces the NoOp message, and — the
// correctness half — appends no duplicate empty audit commit. The commit count
// is compared across the second invocation, so a regression that re-lands the
// duplicate fails here even if the exit code stays 0.
func TestAcknowledgeIllegal_AlreadyAcknowledged_NoOp_ExitZeroNoCommit(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "Foo", "--actor", "human/test", "--root", root)
	target := headSHA(t, root)

	const reason = "squash-merge from a pre-audit era; intermediate steps lost"
	mustRun(t, "acknowledge", "illegal", target, "--reason", reason, "--actor", "human/test", "--root", root)

	countAfterFirst, err := commitCount(t, root)
	if err != nil {
		t.Fatalf("counting commits after the first ack: %v", err)
	}

	var rc int
	out := testutil.CaptureStdout(t, func() {
		rc = cli.Execute([]string{"acknowledge", "illegal", target, "--reason", reason, "--actor", "human/test", "--root", root})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("re-ack via CLI: rc=%d, want ExitOK (%d)", rc, cliutil.ExitOK)
	}
	if !strings.Contains(string(out), "already acknowledged") {
		t.Errorf("stdout = %q, want it to contain the NoOp message", out)
	}
	countAfterSecond, err := commitCount(t, root)
	if err != nil {
		t.Fatalf("counting commits after the re-ack: %v", err)
	}
	if countAfterSecond != countAfterFirst {
		t.Errorf("commit count = %d, want %d — the NoOp must not append a duplicate audit commit", countAfterSecond, countAfterFirst)
	}
}
