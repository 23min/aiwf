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

// TestRenameRetitle_SameValue_NoOp_ExitZeroNoCommit is the CLI-seam half of
// M-0281/AC-5: renaming to the current slug and retitling to the current title
// each surface through their command as exit 0 with the NoOp message and no new
// commit. Both previously refused, so exit 0 can only come from the NoOp guards.
// One repo serves both verbs; the history length is checked after each so a
// stray commit from either is attributed correctly.
func TestRenameRetitle_SameValue_NoOp_ExitZeroNoCommit(t *testing.T) {
	t.Parallel()
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root)

	historyLen := func() int {
		t.Helper()
		events, err := entityview.ReadHistory(context.Background(), root, "E-0001")
		if err != nil {
			t.Fatalf("readHistory: %v", err)
		}
		return len(events)
	}
	afterAdd := historyLen()

	cases := []struct {
		name    string
		args    []string
		wantMsg string
	}{
		{
			name:    "rename to the current slug",
			args:    []string{"rename", "--actor", "human/test", "--root", root, "E-0001", "foundations"},
			wantMsg: "already named",
		},
		{
			name:    "retitle to the current title",
			args:    []string{"retitle", "--actor", "human/test", "--root", root, "E-0001", "Foundations"},
			wantMsg: "title is already",
		},
	}
	for _, tc := range cases {
		// Serial by design: both cases drive the same repo and assert the
		// history length, so a parallel run would race on the shared count.
		t.Run(tc.name, func(t *testing.T) {
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
			if got := historyLen(); got != afterAdd {
				t.Errorf("history has %d events, want %d — the NoOp must not commit", got, afterAdd)
			}
		})
	}
}
