package show_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/show"
)

// TestRunDiag_FallsBackWhenCorrelationIDEmpty pins Run's own
// `if runID == "" { runID = logger.NewRunID() }` diagnostic-logging
// fallback: an empty correlationID (unreachable through cli.Execute,
// since NewRootCmd always mints a real one) still produces a non-empty
// run_id in the diagnostic record. The diagLog block runs (and this
// test's assertion fires) before the entity lookup even happens, so a
// nonexistent id and an empty root are enough to reach it.
func TestRunDiag_FallsBackWhenCorrelationIDEmpty(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	_ = show.Run("G-0001", root, "text", "", false, 10, "")

	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading diagnostic log: %v", err)
	}
	var rec struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("diagnostic log %q not JSON: %v", raw, err)
	}
	if rec.RunID == "" {
		t.Error("run_id empty even though correlationID was passed as \"\"; the fallback mint did not fire")
	}
}
