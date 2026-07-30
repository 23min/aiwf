package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/entity"
)

// TestEditBody_ExplicitIdenticalContent_NoOp_ExitZeroNoCommit is the CLI-seam
// half of M-0281/AC-8: handed a --body-file whose content is already committed
// and already on disk, the command exits 0 with the NoOp message and appends no
// commit. The commit count is compared across the invocation, so the empty-diff
// commit this exists to stop fails the test even if the exit code looks right.
func TestEditBody_ExplicitIdenticalContent_NoOp_ExitZeroNoCommit(t *testing.T) {
	// Serial by design, per this package's setup_test.go skip-list: the NoOp
	// assertion below goes through testutil.CaptureStdout, which swaps the
	// process-global os.Stdout. Declaring t.Parallel() here races every
	// concurrent reader of that fd (cobra's OutOrStdout, cliutil.Println).
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "Foundations", "--actor", "human/test", "--root", root)

	epicPath := filepath.Join(root, "work", "epics", "E-0001-foundations", "epic.md")
	raw, err := os.ReadFile(epicPath) //nolint:gosec // fixture path inside the test's own temp root
	if err != nil {
		t.Fatalf("reading the epic: %v", err)
	}
	_, body, ok := entity.Split(raw)
	if !ok {
		t.Fatalf("epic file has no frontmatter:\n%s", raw)
	}
	bodyFile := filepath.Join(t.TempDir(), "body.md")
	if writeErr := os.WriteFile(bodyFile, body, 0o600); writeErr != nil {
		t.Fatalf("writing the body fixture: %v", writeErr)
	}

	before, err := commitCount(t, root)
	if err != nil {
		t.Fatalf("counting commits before the edit: %v", err)
	}

	var rc int
	out := testutil.CaptureStdout(t, func() {
		rc = cli.Execute([]string{"edit-body", "E-0001", "--body-file", bodyFile, "--actor", "human/test", "--root", root})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("edit-body with identical content: rc=%d, want ExitOK (%d)", rc, cliutil.ExitOK)
	}
	if !strings.Contains(string(out), "already carries this body") {
		t.Errorf("stdout = %q, want it to contain the NoOp message", out)
	}
	after, err := commitCount(t, root)
	if err != nil {
		t.Fatalf("counting commits after the edit: %v", err)
	}
	if after != before {
		t.Errorf("commit count = %d, want %d — the NoOp must not land an empty-diff commit", after, before)
	}
}
