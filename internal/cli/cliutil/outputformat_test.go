package cliutil

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/23min/aiwf/internal/check"
)

// captureStdStreams runs fn with os.Stdout/os.Stderr redirected to pipes
// and returns what each received. SERIAL by construction (it mutates the
// process-global std streams) — see setup_test.go's serial note.
func captureStdStreams(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()
	origOut, origErr := os.Stdout, os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout, os.Stderr = wOut, wErr
	fn()
	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout, os.Stderr = origOut, origErr
	var bo, be bytes.Buffer
	_, _ = io.Copy(&bo, rOut)
	_, _ = io.Copy(&be, rErr)
	return bo.String(), be.String()
}

// TestOutputFormat_EmitHelpers exercises every branch of the three M-0143
// envelope-emit helpers (emitErrorEnvelope / emitFindings / emitSuccess)
// in both text and JSON modes — the branch-coverage chokepoint for the
// D-0013 wiring, independent of which verbs happen to reach each path at
// the binary seam. Text mode must write to stderr/stdout exactly as
// before the milestone; JSON mode must write a single clean envelope to
// stdout and nothing to stderr.
func TestOutputFormat_EmitHelpers(t *testing.T) {
	jsonFmt := OutputFormat{Format: "json"}
	textFmt := OutputFormat{Format: "text"}

	t.Run("emitErrorEnvelope json carries code", func(t *testing.T) {
		out, errOut := captureStdStreams(t, func() {
			jsonFmt.emitErrorEnvelope("aiwf promote", "fsm-transition-illegal", "boom")
		})
		if errOut != "" {
			t.Errorf("JSON error: stderr must be empty; got %q", errOut)
		}
		var env struct {
			Status string `json:"status"`
			Error  *struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(out), &env); err != nil {
			t.Fatalf("stdout not JSON: %v\n%s", err, out)
		}
		if env.Status != "error" || env.Error == nil || env.Error.Code != "fsm-transition-illegal" || env.Error.Message != "boom" {
			t.Errorf("unexpected envelope: status=%q error=%+v", env.Status, env.Error)
		}
	})

	t.Run("emitErrorEnvelope text writes label+message to stderr", func(t *testing.T) {
		out, errOut := captureStdStreams(t, func() {
			textFmt.emitErrorEnvelope("aiwf promote", "fsm-transition-illegal", "boom")
		})
		if out != "" {
			t.Errorf("text error: stdout must be empty; got %q", out)
		}
		if errOut != "aiwf promote: boom\n" {
			t.Errorf("stderr = %q, want %q", errOut, "aiwf promote: boom\n")
		}
	})

	errFindings := []check.Finding{
		{Code: "status-valid", Severity: check.SeverityError, Message: "bad status", EntityID: "E-0001"},
	}

	t.Run("emitFindings json carries findings", func(t *testing.T) {
		out, errOut := captureStdStreams(t, func() { jsonFmt.emitFindings(errFindings) })
		if errOut != "" {
			t.Errorf("JSON findings: stderr must be empty; got %q", errOut)
		}
		var env struct {
			Status   string          `json:"status"`
			Findings []check.Finding `json:"findings"`
		}
		if err := json.Unmarshal([]byte(out), &env); err != nil {
			t.Fatalf("stdout not JSON: %v\n%s", err, out)
		}
		if env.Status != "findings" || len(env.Findings) != 1 || env.Findings[0].Code != "status-valid" {
			t.Errorf("unexpected envelope: status=%q findings=%+v", env.Status, env.Findings)
		}
	})

	t.Run("emitFindings text writes to stderr", func(t *testing.T) {
		out, errOut := captureStdStreams(t, func() { textFmt.emitFindings(errFindings) })
		if out != "" {
			t.Errorf("text findings: stdout must be empty; got %q", out)
		}
		if errOut == "" {
			t.Error("text findings: expected per-instance rendering on stderr")
		}
	})

	t.Run("emitSuccess json carries result.subject", func(t *testing.T) {
		out, errOut := captureStdStreams(t, func() { jsonFmt.emitSuccess("promoted E-0001", nil) })
		if errOut != "" {
			t.Errorf("JSON success: stderr must be empty; got %q", errOut)
		}
		var env struct {
			Status string `json:"status"`
			Result *struct {
				Subject string `json:"subject"`
			} `json:"result"`
			Error any `json:"error"`
		}
		if err := json.Unmarshal([]byte(out), &env); err != nil {
			t.Fatalf("stdout not JSON: %v\n%s", err, out)
		}
		if env.Status != "ok" || env.Result == nil || env.Result.Subject != "promoted E-0001" || env.Error != nil {
			t.Errorf("unexpected envelope: status=%q result=%+v error=%v", env.Status, env.Result, env.Error)
		}
	})

	t.Run("emitSuccess text prints subject to stdout", func(t *testing.T) {
		out, errOut := captureStdStreams(t, func() { textFmt.emitSuccess("promoted E-0001", nil) })
		if errOut != "" {
			t.Errorf("text success: stderr must be empty; got %q", errOut)
		}
		if out != "promoted E-0001\n" {
			t.Errorf("stdout = %q, want %q", out, "promoted E-0001\n")
		}
	})
}
