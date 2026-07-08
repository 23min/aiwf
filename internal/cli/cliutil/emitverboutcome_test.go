package cliutil_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/logger"
)

func decodedRecord(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("output %q did not parse as JSON: %v", buf.String(), err)
	}
	return decoded
}

func boundTestLogger(buf *bytes.Buffer) *slog.Logger {
	base := logger.New(logger.Config{Enabled: true, Level: slog.LevelInfo, Format: "json"}, buf)
	return logger.WithVerb(base, "cancel", "G-0001", "human/peter", "run-test")
}

func TestEmitVerbOutcome_OKWithSHA_EmitsCompletedWithSHA(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	cliutil.EmitVerbOutcome(boundTestLogger(&buf), "verb", cliutil.ExitOK, "deadbeef")

	rec := decodedRecord(t, &buf)
	if rec["msg"] != "verb.completed" {
		t.Errorf("msg = %v, want %q", rec["msg"], "verb.completed")
	}
	if rec["sha"] != "deadbeef" {
		t.Errorf("sha = %v, want %q", rec["sha"], "deadbeef")
	}
}

func TestEmitVerbOutcome_OKWithoutSHA_EmitsCompletedNoSHAField(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	cliutil.EmitVerbOutcome(boundTestLogger(&buf), "install", cliutil.ExitOK, "")

	rec := decodedRecord(t, &buf)
	if rec["msg"] != "install.completed" {
		t.Errorf("msg = %v, want %q", rec["msg"], "install.completed")
	}
	if _, ok := rec["sha"]; ok {
		t.Errorf("sha field present = %v, want omitted when empty", rec["sha"])
	}
}

func TestEmitVerbOutcome_NonOK_EmitsFailedWithErrorClass(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		code      int
		wantClass string
	}{
		{"findings", cliutil.ExitFindings, "findings"},
		{"usage", cliutil.ExitUsage, "usage"},
		{"internal", cliutil.ExitInternal, "internal"},
		{"unrecognized code", 99, "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			cliutil.EmitVerbOutcome(boundTestLogger(&buf), "verb", tc.code, "")

			rec := decodedRecord(t, &buf)
			if rec["msg"] != "verb.failed" {
				t.Errorf("msg = %v, want %q", rec["msg"], "verb.failed")
			}
			if rec["error_class"] != tc.wantClass {
				t.Errorf("error_class = %v, want %q", rec["error_class"], tc.wantClass)
			}
			if got, ok := rec["exit_code"].(float64); !ok || int(got) != tc.code {
				t.Errorf("exit_code = %v, want %d", rec["exit_code"], tc.code)
			}
		})
	}
}

func TestEmitVerbOutcome_Disabled_NoIO(t *testing.T) {
	t.Parallel()
	discard := logger.New(logger.Config{}, nil)
	bound := logger.WithVerb(discard, "cancel", "G-0001", "human/peter", "run-test")
	// Must not panic or write anywhere when the logger is disabled.
	cliutil.EmitVerbOutcome(bound, "verb", cliutil.ExitOK, "deadbeef")
	cliutil.EmitVerbOutcome(bound, "verb", cliutil.ExitInternal, "")
}
