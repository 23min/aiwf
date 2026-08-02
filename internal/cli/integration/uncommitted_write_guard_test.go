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

// TestUncommittedWriteGuard_NestedPathReportsUsageNotInternal reaches the
// commit-side guard through a route the claim-side one does not cover.
//
// A verb whose own target is mid-edit is refused in its prelude, so that
// refusal travels as an ordinary verb error. The nested case cannot be:
// renaming an epic moves a directory, carrying every entity beneath it,
// and no verb names those — so the refusal comes from verb.Apply and has
// to survive the seam into the CLI as a usage exit rather than joining
// the internal-failure class every other Apply error reports as.
func TestUncommittedWriteGuard_NestedPathReportsUsageNotInternal(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--root", root, "--actor", "human/test", "--title", "Platform")
	mustRun(t, "add", "milestone", "--root", root, "--actor", "human/test",
		"--epic", "E-0001", "--tdd", "none", "--title", "Cache")

	nested := filepath.Join(root, "work", "epics", "E-0001-platform", "M-0001-cache.md")
	raw, err := os.ReadFile(nested) //nolint:gosec // fixture path inside the test's own temp root
	if err != nil {
		t.Fatalf("reading the milestone: %v", err)
	}
	dirty := string(raw) + "\nAn unblessed body edit on a nested entity.\n"
	if writeErr := os.WriteFile(nested, []byte(dirty), 0o600); writeErr != nil {
		t.Fatalf("dirtying the milestone: %v", writeErr)
	}

	// The epic's own file is clean, so the claim-side guard has nothing to
	// refuse; the directory move is what carries the nested edit.
	rc := cli.Execute([]string{"rename", "E-0001", "renamed-platform", "--root", root, "--actor", "human/test"})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage (%d) — a nested working-copy refusal is not an internal failure",
			rc, cliutil.ExitUsage)
	}

	after, err := os.ReadFile(nested) //nolint:gosec // fixture path inside the test's own temp root
	if err != nil {
		t.Fatalf("re-reading the milestone: %v", err)
	}
	if !strings.Contains(string(after), "An unblessed body edit on a nested entity.") {
		t.Error("the refusal destroyed the operator's edit")
	}
}

// TestCarriedSymlinkGuard_ReportsUsageNotInternal pins the same
// distinction for the symlink refusal. It reaches the shared handler by
// its own error type, so the mapping is a second arm rather than a reuse
// of the first, and a script branching on the exit code would see an
// internal failure if it were dropped.
func TestCarriedSymlinkGuard_ReportsUsageNotInternal(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--root", root, "--actor", "human/test",
		"--title", "Alpha epic",
		"--body", "## Goal\n\nFixture.\n\n## Scope\n\nFixture.\n\n## Out of scope\n\nFixture.\n")
	mustRun(t, "add", "milestone", "--root", root, "--actor", "human/test",
		"--epic", "E-0001", "--tdd", "none", "--title", "First milestone",
		"--body", "## Goal\n\nFixture.\n\n## Acceptance criteria\n\nFixture.\n")

	epicDir := filepath.Join(root, "work", "epics", "E-0001-alpha-epic")
	if err := os.Symlink("M-0001-first-milestone.md", filepath.Join(epicDir, "latest.md")); err != nil {
		t.Skipf("symlinks unavailable on this platform: %v", err)
	}
	commitAll(t, root)

	rc := cli.Execute([]string{"rename", "E-0001", "renamed-slug", "--root", root, "--actor", "human/test"})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage (%d) — a carried symlink is the operator's to resolve",
			rc, cliutil.ExitUsage)
	}
}
