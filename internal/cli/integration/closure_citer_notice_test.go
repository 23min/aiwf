package integration

import (
	"encoding/json"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// citerRow matches one line of the notice's list — an id, then the
// file:line it was found at. Matching the row's shape rather than the
// header's wording keeps the negative assertions binding when the
// sentence around them is reworded.
var citerRow = regexp.MustCompile(`(?m)^ +[A-Z]+-\d+ +\S+:\d+$`)

// seedCiterFixture builds a repo holding one gap that cites another in
// its body prose, and returns the root. The citing gap stays open so it
// is a record someone could still act on.
func seedCiterFixture(t *testing.T) string {
	t.Helper()
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--title", "The cited defect", "--actor", "human/test", "--root", root,
		"--body", "## What's missing\n\nThe thing that gets closed.\n\n## Why it matters\n\nIt is cited.\n")
	mustRun(t, "add", "gap", "--title", "The citing record", "--actor", "human/test", "--root", root,
		"--body", "## What's missing\n\nThis waits on G-0001 landing first.\n\n## Why it matters\n\nIt would go stale.\n")
	mustRun(t, "add", "gap", "--title", "The second citing record", "--actor", "human/test", "--root", root,
		"--body", "## What's missing\n\nAlso rests on G-0001.\n\n## Why it matters\n\nTwo citers read as a list.\n")
	return root
}

// TestClosureNotice_NamesLiveCitersOnTerminalPromote is the seam: a
// terminal promote through the real command prints the live records
// naming the entity it just closed. The helper is unit-tested in
// internal/check; what this pins is that the verb reaches it at all,
// which no test of the helper can show.
func TestClosureNotice_NamesLiveCitersOnTerminalPromote(t *testing.T) {
	// Serial by design, per this package's setup_test.go skip-list:
	// CaptureStdout swaps the process-global os.Stdout.
	root := seedCiterFixture(t)

	rc, stdout, _ := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{
			"promote", "G-0001", "addressed", "--by-commit", headSHA(t, root),
			"--actor", "human/test", "--root", root,
		})
	})
	if rc != 0 {
		t.Fatalf("promote exited %d, want 0\nstdout:\n%s", rc, stdout)
	}
	for _, want := range []string{"G-0002", "G-0003"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("closure printed no notice naming the citing record %s; stdout:\n%s", want, stdout)
		}
	}
}

// TestClosureNotice_SilentWhenNothingCitesIt keeps the notice from
// becoming furniture. Most closures name nobody, and a closure that
// prints a header with an empty list trains the reader to skip it.
func TestClosureNotice_SilentWhenNothingCitesIt(t *testing.T) {
	// Serial by design — see the sibling test.
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--title", "Cited by nobody", "--actor", "human/test", "--root", root,
		"--body", "## What's missing\n\nAlone.\n\n## Why it matters\n\nIt is not cited.\n")

	rc, stdout, _ := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{
			"promote", "G-0001", "addressed", "--by-commit", headSHA(t, root),
			"--actor", "human/test", "--root", root,
		})
	})
	if rc != 0 {
		t.Fatalf("promote exited %d, want 0\nstdout:\n%s", rc, stdout)
	}
	if row := citerRow.FindString(stdout); row != "" {
		t.Errorf("closure printed a citer row with nothing to report: %q\nstdout:\n%s", row, stdout)
	}
}

// TestClosureNotice_AbsentFromJSONOutput protects the envelope. The
// notice is a prompt for a person; `--format=json` speaks to a program,
// whose parse a loose line would break.
func TestClosureNotice_AbsentFromJSONOutput(t *testing.T) {
	// Serial by design — see the sibling test.
	root := seedCiterFixture(t)

	rc, stdout, _ := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{
			"promote", "G-0001", "addressed", "--by-commit", headSHA(t, root),
			"--actor", "human/test", "--root", root, "--format", "json",
		})
	})
	if rc != 0 {
		t.Fatalf("promote exited %d, want 0\nstdout:\n%s", rc, stdout)
	}
	// Structural, not a phrase grep: the contract is that stdout is one
	// JSON value and nothing else, so anything appended fails the decode
	// however the notice is worded.
	dec := json.NewDecoder(strings.NewReader(stdout))
	var envelope map[string]any
	if err := dec.Decode(&envelope); err != nil {
		t.Fatalf("stdout is not a JSON envelope: %v\nstdout:\n%s", err, stdout)
	}
	if _, err := dec.Token(); err != io.EOF {
		t.Errorf("stdout carried trailing output after the envelope (err=%v); stdout:\n%s", err, stdout)
	}
}

// TestClosureNotice_SilentOnNonTerminalPromote pins the guard that
// scopes the notice to closures. Without it every verb routed through
// the shared finish path — add, rename, retitle, move, reallocate,
// edit-body and the rest — would print it.
func TestClosureNotice_SilentOnNonTerminalPromote(t *testing.T) {
	// Serial by design — see the sibling test.
	root := seedCiterFixture(t)
	mustRun(t, "add", "adr", "--title", "A decision that stays live", "--actor", "human/test", "--root", root,
		"--body", "## Context\n\nThe subject.\n\n## Decision\n\nKeep it.\n")
	// The ADR must itself be cited, or dropping the guard would still
	// print nothing and this test would pass for the wrong reason.
	mustRun(t, "add", "gap", "--title", "Rests on the live decision", "--actor", "human/test", "--root", root,
		"--body", "## What's missing\n\nRests on ADR-0001 staying as it is.\n\n## Why it matters\n\nIt would go stale.\n")

	// proposed -> accepted is a real transition with outgoing edges, so
	// it moves the entity without closing it.
	rc, stdout, _ := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{
			"promote", "ADR-0001", "accepted",
			"--actor", "human/test", "--root", root,
		})
	})
	if rc != 0 {
		t.Fatalf("promote exited %d, want 0\nstdout:\n%s", rc, stdout)
	}
	if row := citerRow.FindString(stdout); row != "" {
		t.Errorf("a non-terminal promote printed a citer row: %q\nstdout:\n%s", row, stdout)
	}
}
