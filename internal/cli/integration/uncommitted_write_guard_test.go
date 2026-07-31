package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
)

// TestUncommittedWriteGuard_ReportsUsageNotInternal pins the exit code
// the commit-side write guard reports through the CLI (ADR-0038,
// M-0283/AC-1).
//
// Every other verb.Apply failure is infrastructure breaking, and the
// shared handler reports those as ExitInternal. This one is not: the
// operator has an uncommitted edit and a choice to make about it, and
// nothing in aiwf's own machinery has gone wrong. Scripts branch on the
// exit code, so the distinction has to survive the seam between the verb
// layer and the CLI rather than living only in the message.
func TestUncommittedWriteGuard_ReportsUsageNotInternal(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--root", root, "--actor", "human/test",
		"--title", "Some gap",
		"--body", "## What's missing\n\nFixture prose.\n\n## Why it matters\n\nFixture prose.\n")

	gapPath := filepath.Join(root, "work", "gaps", "G-0001-some-gap.md")
	raw, err := os.ReadFile(gapPath) //nolint:gosec // fixture path inside the test's own temp root
	if err != nil {
		t.Fatalf("reading the gap: %v", err)
	}
	dirty := string(raw) + "\nAn unblessed body edit.\n"
	if writeErr := os.WriteFile(gapPath, []byte(dirty), 0o600); writeErr != nil {
		t.Fatalf("dirtying the gap: %v", writeErr)
	}

	rc := cli.Execute([]string{"set-priority", "G-0001", "high", "--root", root, "--actor", "human/test"})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage (%d) — a working-copy refusal is not an internal failure",
			rc, cliutil.ExitUsage)
	}

	// The edit is still the operator's, uncommitted and intact.
	after, err := os.ReadFile(gapPath) //nolint:gosec // fixture path inside the test's own temp root
	if err != nil {
		t.Fatalf("re-reading the gap: %v", err)
	}
	if !strings.Contains(string(after), "An unblessed body edit.") {
		t.Errorf("the refusal discarded the working-copy edit:\n%s", after)
	}
}
