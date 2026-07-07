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

// TestMoveDiag_EmitsVerbCompletedEvent pins M-0238/AC-1+AC-2 for
// `aiwf move`: a successful run with AIWF_LOG=info fires a
// "verb.completed" event with verb/entity/actor bound, independent of
// the verb's own stderr/exit-code behavior.
func TestMoveDiag_EmitsVerbCompletedEvent(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "Source epic", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "epic", "--title", "Target epic", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--epic", "E-0001", "--tdd", "none", "--title", "Child", "--actor", "human/test", "--root", root)

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"move", "M-0001", "--epic", "E-0002", "--actor", "human/test", "--root", root})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf move: rc=%d stderr=%s", rc, stderr)
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
	if rec.Verb != "move" {
		t.Errorf("verb = %q, want %q", rec.Verb, "move")
	}
	if rec.Entity != "M-0001" {
		t.Errorf("entity = %q, want %q", rec.Entity, "M-0001")
	}
	if rec.Actor != "human/test" {
		t.Errorf("actor = %q, want %q", rec.Actor, "human/test")
	}
}

// TestMoveDiag_FailedRunEmitsNoEvent: a move that never reaches a
// successful outcome (missing --epic, caught before any tree work)
// must not emit "verb.completed" even with AIWF_LOG=info set.
func TestMoveDiag_FailedRunEmitsNoEvent(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "Source epic", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--epic", "E-0001", "--tdd", "none", "--title", "Child", "--actor", "human/test", "--root", root)

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, _, _ := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"move", "M-0001", "--epic", "E-9999", "--actor", "human/test", "--root", root})
	})
	if rc == cliutil.ExitOK {
		t.Fatalf("aiwf move to a nonexistent epic: rc=ExitOK, want a failure code")
	}

	if raw, err := os.ReadFile(logPath); err == nil {
		t.Errorf("diagnostic log %q written despite a failed run", raw)
	} else if !os.IsNotExist(err) {
		t.Errorf("reading diagnostic log: %v", err)
	}
}

// TestMoveDiag_DisabledByDefault_NoLogFileCreated pins the default-off
// half: without AIWF_LOG set, `aiwf move` creates no diagnostic log.
func TestMoveDiag_DisabledByDefault_NoLogFileCreated(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "Source epic", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "epic", "--title", "Target epic", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--epic", "E-0001", "--tdd", "none", "--title", "Child", "--actor", "human/test", "--root", root)

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "")
	t.Setenv("AIWF_LOG_FILE", logPath)

	mustRun(t, "move", "M-0001", "--epic", "E-0002", "--actor", "human/test", "--root", root)

	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Errorf("diagnostic log file exists at %s despite AIWF_LOG unset (err=%v)", logPath, err)
	}
}
