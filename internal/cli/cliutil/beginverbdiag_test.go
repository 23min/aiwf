package cliutil

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// readOneDiagRecord reads the single JSON diagnostic record BeginVerbDiag's
// finish closure wrote to an explicit AIWF_LOG_FILE destination.
func readOneDiagRecord(t *testing.T, path string) map[string]any {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	var rec map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(content), &rec); err != nil {
		t.Fatalf("record %q did not parse as JSON: %v", content, err)
	}
	return rec
}

// enabledDiagEnv is the fake environment that turns the diagnostic
// logger on and points it at a JSON file the test can read back.
func enabledDiagEnv(path string) func(string) string {
	return fakeGetenv(map[string]string{
		"AIWF_LOG":        "info",
		"AIWF_LOG_FILE":   path,
		"AIWF_LOG_FORMAT": "json",
	})
}

func TestBeginVerbDiag_EnabledEmitsCompletedWithFinalCodeAndSHA(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "diag.log")
	finish := beginVerbDiag(t.TempDir(), enabledDiagEnv(path),
		"promote", "M-0001", "human/peter", "corr-123")

	// Pointer capture: the verb assigns code/sha mid-body, after the
	// defer was registered. finish dereferences the pointers at call
	// time, so it reports the final assigned values.
	code := ExitOK
	sha := "deadbeef"
	finish(&code, &sha)

	rec := readOneDiagRecord(t, path)
	for field, want := range map[string]string{
		"msg":    "verb.completed",
		"sha":    "deadbeef",
		"verb":   "promote",
		"entity": "M-0001",
		"actor":  "human/peter",
		"run_id": "corr-123",
	} {
		if rec[field] != want {
			t.Errorf("record[%q] = %v, want %q", field, rec[field], want)
		}
	}
}

func TestBeginVerbDiag_EmptyCorrelationID_GeneratesRunID(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "diag.log")
	finish := beginVerbDiag(t.TempDir(), enabledDiagEnv(path),
		"promote", "M-0001", "human/peter", "")

	code := ExitOK
	sha := ""
	finish(&code, &sha)

	rec := readOneDiagRecord(t, path)
	runID, ok := rec["run_id"].(string)
	if !ok || len(runID) != 16 {
		// logger.NewRunID() is 16 hex chars from 8 random bytes.
		t.Errorf("run_id = %v (len %d), want a 16-char generated id", rec["run_id"], len(runID))
	}
}

func TestBeginVerbDiag_NonOK_EmitsFailedWithErrorClass(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "diag.log")
	finish := beginVerbDiag(t.TempDir(), enabledDiagEnv(path),
		"promote", "M-0001", "human/peter", "corr-123")

	code := ExitInternal
	sha := ""
	finish(&code, &sha)

	rec := readOneDiagRecord(t, path)
	if rec["msg"] != "verb.failed" {
		t.Errorf("msg = %v, want %q", rec["msg"], "verb.failed")
	}
	if rec["error_class"] != "internal" {
		t.Errorf("error_class = %v, want %q", rec["error_class"], "internal")
	}
	if got, ok := rec["exit_code"].(float64); !ok || int(got) != ExitInternal {
		t.Errorf("exit_code = %v, want %d", rec["exit_code"], ExitInternal)
	}
}

func TestBeginVerbDiag_Disabled_NoOutputNoPanic(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "diag.log")
	// fakeGetenv(nil): AIWF_LOG unset → the logger resolves to disabled,
	// exercising the false arm of the Enabled guard (WithVerb skipped)
	// and EmitVerbOutcome's no-op path.
	finish := beginVerbDiag(t.TempDir(), fakeGetenv(nil),
		"promote", "M-0001", "human/peter", "corr-123")

	code := ExitOK
	sha := "deadbeef"
	finish(&code, &sha) // must not panic

	if content, err := os.ReadFile(path); err == nil && len(content) > 0 {
		t.Errorf("disabled logger wrote %q, want no output", content)
	}
}
