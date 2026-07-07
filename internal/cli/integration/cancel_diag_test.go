package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli"
	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// TestCancelDiag_EmitsVerbCompletedEvent pins M-0238/AC-1+AC-2: a
// successful `aiwf cancel` run with AIWF_LOG=info fires a
// "verb.completed" diagnostic event through the WithVerb-bound logger,
// with the verb/entity/actor fields bound — independent of (and never
// affecting) the verb's own stderr/exit-code behavior.
func TestCancelDiag_EmitsVerbCompletedEvent(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--body", "## What's missing\n\nFixture prose for test setup; not the subject under test.\n\n## Why it matters\n\nFixture prose for test setup; not the subject under test.\n", "--title", "Stale probe", "--actor", "human/test", "--root", root)

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"cancel", "G-0001", "--reason", "no longer needed", "--actor", "human/test", "--root", root})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf cancel: rc=%d stderr=%s", rc, stderr)
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading diagnostic log: %v", err)
	}
	var rec struct {
		Msg    string `json:"msg"`
		Verb   string `json:"verb"`
		Entity string `json:"entity"`
		Actor  string `json:"actor"`
	}
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("diagnostic log %q not JSON: %v", raw, err)
	}
	if rec.Msg != "verb.completed" {
		t.Errorf("msg = %q, want %q", rec.Msg, "verb.completed")
	}
	if rec.Verb != "cancel" {
		t.Errorf("verb = %q, want %q", rec.Verb, "cancel")
	}
	if rec.Entity != "G-0001" {
		t.Errorf("entity = %q, want %q", rec.Entity, "G-0001")
	}
	if rec.Actor != "human/test" {
		t.Errorf("actor = %q, want %q", rec.Actor, "human/test")
	}
}

// TestCancelDiag_FailedRunEmitsNoEvent pins the code == ExitOK guard:
// a cancel that never reaches a successful outcome (here, a nonexistent
// entity) must not emit "verb.completed" even with AIWF_LOG=info set.
func TestCancelDiag_FailedRunEmitsNoEvent(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, _, _ := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"cancel", "G-9999", "--reason", "does not exist", "--actor", "human/test", "--root", root})
	})
	if rc == cliutil.ExitOK {
		t.Fatalf("aiwf cancel on a nonexistent entity: rc=ExitOK, want a failure code")
	}

	if raw, err := os.ReadFile(logPath); err == nil {
		t.Errorf("diagnostic log %q written despite a failed run", raw)
	} else if !os.IsNotExist(err) {
		t.Errorf("reading diagnostic log: %v", err)
	}
}

// TestCancelDiag_DisabledByDefault_NoLogFileCreated pins the
// default-off half of the same AC: without AIWF_LOG set, `aiwf cancel`
// behaves identically and creates no diagnostic log file at all.
func TestCancelDiag_DisabledByDefault_NoLogFileCreated(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--body", "## What's missing\n\nFixture prose for test setup; not the subject under test.\n\n## Why it matters\n\nFixture prose for test setup; not the subject under test.\n", "--title", "Stale probe", "--actor", "human/test", "--root", root)

	logDir := t.TempDir()
	logPath := filepath.Join(logDir, "diag.log")
	t.Setenv("AIWF_LOG", "")
	t.Setenv("AIWF_LOG_FILE", logPath)

	mustRun(t, "cancel", "G-0001", "--reason", "no longer needed", "--actor", "human/test", "--root", root)

	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Errorf("diagnostic log file exists at %s despite AIWF_LOG unset (err=%v)", logPath, err)
	}
}
