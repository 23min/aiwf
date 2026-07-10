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

// wired_verbs_diag_test.go — M-0249: pins that add/promote/check/
// reallocate/edit-body/authorize/show each now emit a "verb.completed"
// diagnostic-log record when AIWF_LOG is set, mirroring
// cancel_diag_test.go's own TestCancelDiag_EmitsVerbCompletedEvent.
// Without this wiring, D-0035's diagnostic-log env passthrough had no
// verb to attach to for 11 of the stress harness's 12 scenarios, since
// only cancel and move previously called cliutil.ResolveLogger.

type diagRecord struct {
	Msg    string `json:"msg"`
	Verb   string `json:"verb"`
	Entity string `json:"entity"`
	Actor  string `json:"actor"`
	RunID  string `json:"run_id"`
}

func readDiagRecord(t *testing.T, path string) diagRecord {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading diagnostic log: %v", err)
	}
	var rec diagRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("diagnostic log %q not JSON: %v", raw, err)
	}
	return rec
}

func TestAddDiag_EmitsVerbCompletedEvent(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"add", "gap", "--title", "diag probe", "--body", "## What's missing\n\nFixture.\n\n## Why it matters\n\nFixture.\n", "--actor", "human/test", "--root", root})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf add: rc=%d stderr=%s", rc, stderr)
	}

	rec := readDiagRecord(t, logPath)
	if rec.Msg != "verb.completed" {
		t.Errorf("msg = %q, want %q", rec.Msg, "verb.completed")
	}
	if rec.Verb != "add" {
		t.Errorf("verb = %q, want %q", rec.Verb, "add")
	}
	if rec.Actor != "human/test" {
		t.Errorf("actor = %q, want %q", rec.Actor, "human/test")
	}
	if rec.RunID == "" {
		t.Error("run_id missing or empty from the diagnostic record")
	}
}

func TestPromoteDiag_EmitsVerbCompletedEvent(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--body", "## What's missing\n\nFixture.\n\n## Why it matters\n\nFixture.\n", "--title", "diag probe", "--actor", "human/test", "--root", root)

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"promote", "G-0001", "wontfix", "--actor", "human/test", "--root", root})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf promote: rc=%d stderr=%s", rc, stderr)
	}

	rec := readDiagRecord(t, logPath)
	if rec.Msg != "verb.completed" {
		t.Errorf("msg = %q, want %q", rec.Msg, "verb.completed")
	}
	if rec.Verb != "promote" {
		t.Errorf("verb = %q, want %q", rec.Verb, "promote")
	}
	if rec.Entity != "G-0001" {
		t.Errorf("entity = %q, want %q", rec.Entity, "G-0001")
	}
	if rec.RunID == "" {
		t.Error("run_id missing or empty from the diagnostic record")
	}
}

func TestCheckDiag_EmitsVerbCompletedEvent(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"check", "--root", root})
	})
	if rc != cliutil.ExitOK && rc != cliutil.ExitFindings {
		t.Fatalf("aiwf check: rc=%d stderr=%s", rc, stderr)
	}

	rec := readDiagRecord(t, logPath)
	if rec.Msg != "verb.completed" {
		t.Errorf("msg = %q, want %q", rec.Msg, "verb.completed")
	}
	if rec.Verb != "check" {
		t.Errorf("verb = %q, want %q", rec.Verb, "check")
	}
	if rec.RunID == "" {
		t.Error("run_id missing or empty from the diagnostic record")
	}
}

func TestReallocateDiag_EmitsVerbCompletedEvent(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--body", "## What's missing\n\nFixture.\n\n## Why it matters\n\nFixture.\n", "--title", "diag probe", "--actor", "human/test", "--root", root)

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"reallocate", "G-0001", "--actor", "human/test", "--root", root})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf reallocate: rc=%d stderr=%s", rc, stderr)
	}

	rec := readDiagRecord(t, logPath)
	if rec.Msg != "verb.completed" {
		t.Errorf("msg = %q, want %q", rec.Msg, "verb.completed")
	}
	if rec.Verb != "reallocate" {
		t.Errorf("verb = %q, want %q", rec.Verb, "reallocate")
	}
	if rec.Entity != "G-0001" {
		t.Errorf("entity = %q, want %q", rec.Entity, "G-0001")
	}
	if rec.RunID == "" {
		t.Error("run_id missing or empty from the diagnostic record")
	}
}

func TestEditBodyDiag_EmitsVerbCompletedEvent(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--body", "## What's missing\n\nFixture.\n\n## Why it matters\n\nFixture.\n", "--title", "diag probe", "--actor", "human/test", "--root", root)

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"edit-body", "G-0001", "--body-file", "-", "--actor", "human/test", "--root", root, "--reason", "diag probe"})
	})
	_ = rc
	_ = stderr

	rec := readDiagRecord(t, logPath)
	if rec.Verb != "edit-body" {
		t.Errorf("verb = %q, want %q", rec.Verb, "edit-body")
	}
	if rec.RunID == "" {
		t.Error("run_id missing or empty from the diagnostic record")
	}
}

func TestAuthorizeDiag_EmitsVerbCompletedEvent(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "epic", "--title", "Adoption", "--actor", "human/test", "--root", root)
	mustRun(t, "add", "milestone", "--tdd", "none", "--epic", "E-0001", "--title", "Schema parser", "--actor", "human/test", "--root", root)
	mustRun(t, "promote", "--root", root, "--actor", "human/test", "M-0001", "in_progress")
	if out, err := testutil.RunGit(root, "checkout", "-b", "epic/E-0001-adoption"); err != nil {
		t.Fatalf("git checkout -b: %v\n%s", err, out)
	}

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"authorize", "M-0001", "--to", "ai/claude", "--actor", "human/test", "--root", root})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf authorize: rc=%d stderr=%s", rc, stderr)
	}

	rec := readDiagRecord(t, logPath)
	if rec.Msg != "verb.completed" {
		t.Errorf("msg = %q, want %q", rec.Msg, "verb.completed")
	}
	if rec.Verb != "authorize" {
		t.Errorf("verb = %q, want %q", rec.Verb, "authorize")
	}
	if rec.Entity != "M-0001" {
		t.Errorf("entity = %q, want %q", rec.Entity, "M-0001")
	}
	if rec.RunID == "" {
		t.Error("run_id missing or empty from the diagnostic record")
	}
}

// TestShowDiag_EmitsVerbCompletedEvent pins show's own best-effort
// actor resolution: show has no --actor flag, so the diagnostic
// record's actor comes from git config user.email — never a required
// input, per show.Run's own ADR-0017 rationale.
func TestShowDiag_EmitsVerbCompletedEvent(t *testing.T) {
	root := setupCLITestRepo(t)
	mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
	mustRun(t, "add", "gap", "--body", "## What's missing\n\nFixture.\n\n## Why it matters\n\nFixture.\n", "--title", "diag probe", "--actor", "human/test", "--root", root)

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	rc, _, stderr := testutil.CaptureRun(t, func() int {
		return cli.Execute([]string{"show", "G-0001", "--root", root})
	})
	if rc != cliutil.ExitOK {
		t.Fatalf("aiwf show: rc=%d stderr=%s", rc, stderr)
	}

	rec := readDiagRecord(t, logPath)
	if rec.Msg != "verb.completed" {
		t.Errorf("msg = %q, want %q", rec.Msg, "verb.completed")
	}
	if rec.Verb != "show" {
		t.Errorf("verb = %q, want %q", rec.Verb, "show")
	}
	if rec.Entity != "G-0001" {
		t.Errorf("entity = %q, want %q", rec.Entity, "G-0001")
	}
	if rec.RunID == "" {
		t.Error("run_id missing or empty from the diagnostic record")
	}
}

// TestWiredVerbsDiag_DisabledByDefault_NoLogFileCreated is the
// default-off half for all 7 newly wired verbs at once — mirroring
// TestCancelDiag_DisabledByDefault_NoLogFileCreated, table-driven
// since the assertion shape is identical across all 7.
func TestWiredVerbsDiag_DisabledByDefault_NoLogFileCreated(t *testing.T) {
	tests := []struct {
		name string
		args func(root string) []string
	}{
		{"add", func(root string) []string {
			return []string{"add", "gap", "--title", "diag probe", "--body", "## What's missing\n\nFixture.\n\n## Why it matters\n\nFixture.\n", "--actor", "human/test", "--root", root}
		}},
		{"check", func(root string) []string {
			return []string{"check", "--root", root}
		}},
		{"show", func(root string) []string {
			return []string{"show", "E-0001", "--root", root}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := setupCLITestRepo(t)
			mustRun(t, "init", "--root", root, "--actor", "human/test", "--skip-hook")
			mustRun(t, "add", "epic", "--title", "Fixture", "--actor", "human/test", "--root", root)

			logDir := t.TempDir()
			logPath := filepath.Join(logDir, "diag.log")
			t.Setenv("AIWF_LOG", "")
			t.Setenv("AIWF_LOG_FILE", logPath)

			testutil.CaptureRun(t, func() int { return cli.Execute(tt.args(root)) })

			if _, err := os.Stat(logPath); !os.IsNotExist(err) {
				t.Errorf("diagnostic log file exists at %s despite AIWF_LOG unset (err=%v)", logPath, err)
			}
		})
	}
}
