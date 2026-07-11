package check

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
)

// TestNewCmd_FlagShape pins the check verb's flag surface so a
// future migration can't silently drop or rename a flag without
// the test failing. The completion drift test in cmd/aiwf/
// catches the same regression at the binary level.
func TestNewCmd_FlagShape(t *testing.T) {
	t.Parallel()
	cmd := NewCmd("")
	if cmd.Use != "check" {
		t.Errorf("Use = %q, want check", cmd.Use)
	}
	expected := []string{"root", "format", "pretty", "since", "shape-only", "fast", "verbose"}
	for _, name := range expected {
		if cmd.Flags().Lookup(name) == nil {
			t.Errorf("flag %q missing", name)
		}
	}
}

// TestRun_BadFormat pins the format-validation guard at the top of
// Run. A non-{text,json} value returns ExitUsage immediately
// without loading the tree.
func TestRun_BadFormat(t *testing.T) {
	t.Parallel()
	code := Run("", "yaml", false, "", false, false, false, nil, "")
	if code != cliutil.ExitUsage {
		t.Errorf("Run with --format=yaml: got %d, want %d", code, cliutil.ExitUsage)
	}
}

// TestRunDiag_FallsBackWhenCorrelationIDEmpty pins Run's own
// `if runID == "" { runID = logger.NewRunID() }` diagnostic-logging
// fallback: an empty correlationID (unreachable through cli.Execute,
// since NewRootCmd always mints a real one) still produces a non-empty
// run_id in the diagnostic record. --shape-only keeps this fast and
// avoids needing a real aiwf tree at root — the diagLog block runs
// (and this test's assertion fires) before shapeOnly's own outcome is
// even known.
func TestRunDiag_FallsBackWhenCorrelationIDEmpty(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(t.TempDir(), "diag.log")
	t.Setenv("AIWF_LOG", "info")
	t.Setenv("AIWF_LOG_FORMAT", "json")
	t.Setenv("AIWF_LOG_FILE", logPath)

	_ = Run(root, "text", false, "", true, false, false, nil, "")

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
