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
		{Code: check.CodeStatusValid, Severity: check.SeverityError, Message: "bad status", EntityID: "E-0001"},
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
		if env.Status != "findings" || len(env.Findings) != 1 || env.Findings[0].Code != check.CodeStatusValid {
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
		out, errOut := captureStdStreams(t, func() { jsonFmt.emitSuccess("promoted E-0001", nil, nil) })
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
		out, errOut := captureStdStreams(t, func() { textFmt.emitSuccess("promoted E-0001", nil, nil) })
		if errOut != "" {
			t.Errorf("text success: stderr must be empty; got %q", errOut)
		}
		if out != "promoted E-0001\n" {
			t.Errorf("stdout = %q, want %q", out, "promoted E-0001\n")
		}
	})
}

// TestOutputFormat_CorrelationID pins M-0239/AC-1: when CorrelationID
// is set, every one of the three JSON envelope shapes (error, findings,
// success) carries it under metadata.correlation_id — so an operator
// can correlate any outcome, not just a clean success, with the
// invocation's diagnostic log lines. Text mode is untouched: it never
// prints metadata at all.
func TestOutputFormat_CorrelationID(t *testing.T) {
	withID := OutputFormat{Format: "json", CorrelationID: "run-abc123"}
	noID := OutputFormat{Format: "json"}
	findings := []check.Finding{
		{Code: check.CodeStatusValid, Severity: check.SeverityError, Message: "bad status", EntityID: "E-0001"},
	}

	decodeMetadata := func(t *testing.T, raw string) map[string]any {
		t.Helper()
		var env struct {
			Metadata map[string]any `json:"metadata"`
		}
		if err := json.Unmarshal([]byte(raw), &env); err != nil {
			t.Fatalf("stdout not JSON: %v\n%s", err, raw)
		}
		return env.Metadata
	}

	t.Run("emitErrorEnvelope carries correlation_id", func(t *testing.T) {
		out, _ := captureStdStreams(t, func() { withID.emitErrorEnvelope("aiwf promote", "", "boom") })
		md := decodeMetadata(t, out)
		if md["correlation_id"] != "run-abc123" {
			t.Errorf("metadata.correlation_id = %v, want %q", md["correlation_id"], "run-abc123")
		}
	})

	t.Run("emitFindings carries correlation_id", func(t *testing.T) {
		out, _ := captureStdStreams(t, func() { withID.emitFindings(findings) })
		md := decodeMetadata(t, out)
		if md["correlation_id"] != "run-abc123" {
			t.Errorf("metadata.correlation_id = %v, want %q", md["correlation_id"], "run-abc123")
		}
	})

	t.Run("emitSuccess carries correlation_id", func(t *testing.T) {
		out, _ := captureStdStreams(t, func() { withID.emitSuccess("promoted E-0001", nil, nil) })
		md := decodeMetadata(t, out)
		if md["correlation_id"] != "run-abc123" {
			t.Errorf("metadata.correlation_id = %v, want %q", md["correlation_id"], "run-abc123")
		}
	})

	t.Run("emitSuccess merges verb-supplied metadata with correlation_id", func(t *testing.T) {
		out, _ := captureStdStreams(t, func() {
			withID.emitSuccess("promoted E-0001", nil, map[string]any{"entity_id": "E-0001", "to": "active"})
		})
		md := decodeMetadata(t, out)
		if md["correlation_id"] != "run-abc123" {
			t.Errorf("metadata.correlation_id = %v, want %q", md["correlation_id"], "run-abc123")
		}
		if md["entity_id"] != "E-0001" {
			t.Errorf("metadata.entity_id = %v, want %q", md["entity_id"], "E-0001")
		}
		if md["to"] != "active" {
			t.Errorf("metadata.to = %v, want %q", md["to"], "active")
		}
	})

	t.Run("emitSuccess carries verb-supplied metadata even without a CorrelationID", func(t *testing.T) {
		out, _ := captureStdStreams(t, func() {
			noID.emitSuccess("promoted E-0001", nil, map[string]any{"entity_id": "E-0001"})
		})
		md := decodeMetadata(t, out)
		if md["entity_id"] != "E-0001" {
			t.Errorf("metadata.entity_id = %v, want %q", md["entity_id"], "E-0001")
		}
		if _, ok := md["correlation_id"]; ok {
			t.Errorf("metadata.correlation_id = %v, want absent (CorrelationID was never set)", md["correlation_id"])
		}
	})

	t.Run("emitSuccess omits metadata entirely when CorrelationID is empty", func(t *testing.T) {
		out, _ := captureStdStreams(t, func() { noID.emitSuccess("promoted E-0001", nil, nil) })
		var env struct {
			Metadata map[string]any `json:"metadata"`
		}
		if err := json.Unmarshal([]byte(out), &env); err != nil {
			t.Fatalf("stdout not JSON: %v\n%s", err, out)
		}
		if env.Metadata != nil {
			t.Errorf("metadata = %+v, want nil (omitempty) when no CorrelationID is set", env.Metadata)
		}
	})
}
