package integration

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// M-0255/AC-1 backfill: TestAuditOnly_CancelG24Recovery (auditonly_cmd_test.go)
// already exercises the [audit-only: ...] chip end-to-end, but via
// testutil.RunBin — a separately-compiled binary run as a subprocess,
// invisible to `go test`'s own -coverprofile instrumentation. The two
// tests below drive the same shape of scenario through the in-process
// cli.Execute dispatcher instead, so `aiwf history`'s text-format
// [reason: ...] and [audit-only: ...] chip lines actually show up in
// coverage.

// TestRun_HistoryTextRendersReasonChip covers the text-format
// [reason: ...] chip: only specific verb shapes stamp the
// aiwf-reason: trailer HistoryEvent.Reason is read from — plain
// `cancel --reason` lands its text in the commit body instead. `aiwf
// authorize --pause "<reason>"` is the simplest trailer-stamping
// shape (the pause argument IS the reason, unconditionally).
func TestRun_HistoryTextRendersReasonChip(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := cli.Execute([]string{"add", "epic", "--title", "Foo", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("add epic: %d", rc)
	}
	if rc := cli.Execute([]string{"authorize", "E-0001", "--to", "ai/claude", "--reason", "delegate", "--branch", "epic/E-0001-foo", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("authorize --to: %d", rc)
	}
	if rc := cli.Execute([]string{"authorize", "E-0001", "--pause", "no longer needed", "--actor", "human/test", "--root", root}); rc != cliutil.ExitOK {
		t.Fatalf("authorize --pause: %d", rc)
	}

	captured := testutil.CaptureStdout(t, func() {
		if rc := cli.Execute([]string{"history", "--root", root, "E-0001"}); rc != cliutil.ExitOK {
			t.Fatalf("history: %d", rc)
		}
	})
	out := string(captured)
	if !strings.Contains(out, "[reason: no longer needed]") {
		t.Errorf("expected a [reason: ...] chip; got:\n%s", out)
	}
}

// TestRun_HistoryTextRendersAuditOnlyChip covers the text-format
// [audit-only: ...] chip via the in-process dispatcher: a gap is
// flipped to a terminal status by a manual (untrailered) commit, then
// `cancel --audit-only --reason "..."` records the recovery. Mirrors
// TestAuditOnly_CancelG24Recovery's scenario (auditonly_cmd_test.go)
// but through cli.Execute so it's coverage-visible.
func TestRun_HistoryTextRendersAuditOnlyChip(t *testing.T) {
	root := setupCLITestRepo(t)
	if rc := cli.Execute([]string{"init", "--root", root, "--actor", "human/test", "--skip-hook"}); rc != cliutil.ExitOK {
		t.Fatalf("init: %d", rc)
	}
	if rc := cli.Execute([]string{
		"add", "gap", "--title", "Validators leak temp files", "--actor", "human/test", "--root", root,
		"--body", "## What's missing\n\nFixture prose.\n\n## Why it matters\n\nFixture prose.\n",
	}); rc != cliutil.ExitOK {
		t.Fatalf("add gap: %d", rc)
	}

	gapRel := mustFindFile(t, root, "G-0001-")
	manualFlipStatus(t, root+"/"+gapRel, "open", "wontfix")
	if err := osExec(t, root, "git", "add", gapRel); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := osExec(t, root, "git", "commit", "-q", "-m", "manually mark G-0001 wontfix"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	if rc := cli.Execute([]string{
		"cancel", "G-0001", "--audit-only", "--reason", "manual flip from earlier",
		"--actor", "human/test", "--root", root,
	}); rc != cliutil.ExitOK {
		t.Fatalf("cancel --audit-only: %d", rc)
	}

	captured := testutil.CaptureStdout(t, func() {
		if rc := cli.Execute([]string{"history", "--root", root, "G-0001"}); rc != cliutil.ExitOK {
			t.Fatalf("history: %d", rc)
		}
	})
	out := string(captured)
	if !strings.Contains(out, "[audit-only: manual flip from earlier]") {
		t.Errorf("expected an [audit-only: ...] chip; got:\n%s", out)
	}
}
