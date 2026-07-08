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

// TestStatuslineScaffoldDiag_EmitsVerbCompletedEvent pins M-0238/AC-1+AC-2
// for the statusline scaffold flow (`aiwf update --statusline`): a
// successful scaffold with AIWF_LOG=info fires a "verb.completed" event.
// --scope project confines the write under root's own .claude/, never
// the real $HOME.
func TestStatuslineScaffoldDiag_EmitsVerbCompletedEvent(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, stdout, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"update", "--root", root, "--scope", "project", "--statusline", "--allow-untagged-statusline"})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf update --statusline: rc=%d stdout=%s stderr=%s", rc, stdout, stderr)
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading diagnostic log: %v", err)
	}
	var rec struct {
		Msg    string `json:"msg"`
		Verb   string `json:"verb"`
		Entity string `json:"entity"`
	}
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("diagnostic log %q not JSON: %v", raw, err)
	}
	if rec.Msg != "verb.completed" {
		t.Errorf("msg = %q, want %q", rec.Msg, "verb.completed")
	}
	if rec.Verb != "statusline-scaffold" {
		t.Errorf("verb = %q, want %q", rec.Verb, "statusline-scaffold")
	}
	if rec.Entity != "project" {
		t.Errorf("entity = %q, want %q (the --scope value, not a filesystem path)", rec.Entity, "project")
	}
}

// TestStatuslineScaffoldDiag_FailedRunEmitsNoEvent: a scaffold write
// that fails (a read-only .claude/ directory) must not emit
// "verb.completed" even with AIWF_LOG=info set.
func TestStatuslineScaffoldDiag_FailedRunEmitsNoEvent(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	claudeDir := filepath.Join(root, ".claude")
	if err := os.Chmod(claudeDir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(claudeDir, 0o755) })

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, _, _ := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"update", "--root", root, "--scope", "project", "--statusline", "--allow-untagged-statusline"})
	})
	if rc == cliutil.ExitOK {
		t.Fatalf("aiwf update --statusline against a read-only .claude dir: rc=ExitOK, want a failure code")
	}

	if raw, err := os.ReadFile(logPath); err == nil {
		t.Errorf("diagnostic log %q written despite a failed scaffold", raw)
	} else if !os.IsNotExist(err) {
		t.Errorf("reading diagnostic log: %v", err)
	}
}

// TestStatuslineScaffoldDiag_DisabledByDefault_NoLogFileCreated pins
// the default-off half: without AIWF_LOG set, a successful scaffold
// creates no diagnostic log.
func TestStatuslineScaffoldDiag_DisabledByDefault_NoLogFileCreated(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, stdout, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"update", "--root", root, "--scope", "project", "--statusline", "--allow-untagged-statusline"})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf update --statusline: rc=%d stdout=%s stderr=%s", rc, stdout, stderr)
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Errorf("diagnostic log file exists at %s despite AIWF_LOG unset (err=%v)", logPath, err)
	}
}

// TestStatuslineRemoveDiag_EmitsVerbCompletedEvent pins the remove
// flow (`aiwf update --remove`): a successful removal with
// AIWF_LOG=info fires its own "verb.completed" event.
func TestStatuslineRemoveDiag_EmitsVerbCompletedEvent(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "update", "--root", root, "--scope", "project", "--statusline", "--allow-untagged-statusline")

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, stdout, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"update", "--root", root, "--scope", "project", "--remove"})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf update --remove: rc=%d stdout=%s stderr=%s", rc, stdout, stderr)
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading diagnostic log: %v", err)
	}
	var rec struct {
		Msg    string `json:"msg"`
		Verb   string `json:"verb"`
		Entity string `json:"entity"`
	}
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("diagnostic log %q not JSON: %v", raw, err)
	}
	if rec.Msg != "verb.completed" {
		t.Errorf("msg = %q, want %q", rec.Msg, "verb.completed")
	}
	if rec.Verb != "statusline-remove" {
		t.Errorf("verb = %q, want %q", rec.Verb, "statusline-remove")
	}
	if rec.Entity != "project" {
		t.Errorf("entity = %q, want %q (the --scope value, not a filesystem path)", rec.Entity, "project")
	}
}

// TestStatuslineRemoveDiag_DisabledByDefault_NoLogFileCreated pins the
// default-off half of the remove flow: without AIWF_LOG set, a
// successful removal creates no diagnostic log.
func TestStatuslineRemoveDiag_DisabledByDefault_NoLogFileCreated(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "update", "--root", root, "--scope", "project", "--statusline", "--allow-untagged-statusline")

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, stdout, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"update", "--root", root, "--scope", "project", "--remove"})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf update --remove: rc=%d stdout=%s stderr=%s", rc, stdout, stderr)
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Errorf("diagnostic log file exists at %s despite AIWF_LOG unset (err=%v)", logPath, err)
	}
}

// TestStatuslineRemoveDiag_RefusalEmitsNoEvent: a refused removal
// (foreign, non-aiwf-authored artifact) must not emit "verb.completed".
func TestStatuslineRemoveDiag_RefusalEmitsNoEvent(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	// A foreign script with no aiwf version marker.
	scriptPath := filepath.Join(root, ".claude", "statusline.sh")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\necho not-aiwf\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, _, _ := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"update", "--root", root, "--scope", "project", "--remove"})
	})
	if rc == cliutil.ExitOK {
		t.Fatalf("aiwf update --remove on a foreign script: rc=ExitOK, want a refusal code")
	}

	if raw, err := os.ReadFile(logPath); err == nil {
		t.Errorf("diagnostic log %q written despite a refused removal", raw)
	} else if !os.IsNotExist(err) {
		t.Errorf("reading diagnostic log: %v", err)
	}
}
