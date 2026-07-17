package list_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/list"
)

// list_diag_test.go — M-0249 follow-up: pins Run's own diagnostic-
// logging fallbacks. cli.Execute always mints a real correlation id
// (NewRootCmd), so the `runID == ""` fallback is unreachable through
// the CLI surface — only a direct Run call (a bare empty
// correlationID) exercises it. Mirrors check.Run's/show.Run's own
// established pattern.

func readRunID(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading diagnostic log: %v", err)
	}
	var rec struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("diagnostic log %q not JSON: %v", raw, err)
	}
	return rec.RunID
}

// TestRunDiag_FallsBackWhenCorrelationIDEmpty pins Run's own
// `if runID == "" { runID = logger.NewRunID() }` fallback.
func TestRunDiag_FallsBackWhenCorrelationIDEmpty(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	_ = list.Run(root, "", "", "", "", "", false, "text", false, false, "")

	if got := readRunID(t, logPath); got == "" {
		t.Error("run_id empty even though correlationID was passed as \"\"; the fallback mint did not fire")
	}
}

// TestRunDiag_ActorResolutionFailureStillEmitsEvent pins list's own
// best-effort actor rationale (no --actor flag, mirroring
// check.Run's/show.Run's identical pattern): when git config
// user.email is also unavailable, the verb still completes and still
// logs, with an empty actor field. Cannot use t.Parallel(): t.Setenv
// panics under parallel.
func TestRunDiag_ActorResolutionFailureStillEmitsEvent(t *testing.T) {
	root := t.TempDir()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	// Intentionally no .gitconfig at home — git config user.email fails.

	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	_ = list.Run(root, "", "", "", "", "", false, "text", false, false, "")

	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading diagnostic log: %v", err)
	}
	var rec struct {
		Actor string `json:"actor"`
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("diagnostic log %q not JSON: %v", raw, err)
	}
	if rec.Actor != "" {
		t.Errorf("actor = %q, want empty (git config user.email was deliberately unavailable)", rec.Actor)
	}
	if rec.RunID == "" {
		t.Error("run_id missing or empty from the diagnostic record")
	}
}
