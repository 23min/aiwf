package stresstest

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// countingWriter records how many times Write is called and the bytes
// passed each time, so tests can assert "exactly one Write() call per
// event" directly rather than only checking the end-state file
// content, which wouldn't distinguish one write from two.
type countingWriter struct {
	calls [][]byte
}

func (c *countingWriter) Write(p []byte) (int, error) {
	cp := make([]byte, len(p))
	copy(cp, p)
	c.calls = append(c.calls, cp)
	return len(p), nil
}

// erroringWriter always fails, so tests can exercise WriteEvent's
// underlying-Write failure path.
type erroringWriter struct{}

func (erroringWriter) Write([]byte) (int, error) {
	return 0, errors.New("simulated write failure")
}

func TestReportWriter_WriteEvent_SingleWriteCall(t *testing.T) {
	t.Parallel()
	cw := &countingWriter{}
	rw := newReportWriter(cw)

	if err := rw.WriteEvent(map[string]string{"scenario": "placeholder", "result": "pass"}); err != nil {
		t.Fatalf("WriteEvent: %v", err)
	}

	if len(cw.calls) != 1 {
		t.Fatalf("expected exactly 1 Write call, got %d: %v", len(cw.calls), cw.calls)
	}

	line := cw.calls[0]
	if line[len(line)-1] != '\n' {
		t.Fatalf("expected event to end in a newline, got %q", line)
	}
	var decoded map[string]string
	if err := json.Unmarshal(line[:len(line)-1], &decoded); err != nil {
		t.Fatalf("event line is not valid JSON: %v\n%s", err, line)
	}
	if decoded["scenario"] != "placeholder" || decoded["result"] != "pass" {
		t.Fatalf("unexpected decoded event: %+v", decoded)
	}
}

func TestReportWriter_WriteEvent_MultipleEventsEachOwnWriteCall(t *testing.T) {
	t.Parallel()
	cw := &countingWriter{}
	rw := newReportWriter(cw)

	for i := 0; i < 3; i++ {
		if err := rw.WriteEvent(map[string]int{"n": i}); err != nil {
			t.Fatalf("WriteEvent %d: %v", i, err)
		}
	}

	if len(cw.calls) != 3 {
		t.Fatalf("expected 3 Write calls (one per event), got %d", len(cw.calls))
	}
}

func TestReportWriter_WriteEvent_MarshalErrorWritesNothing(t *testing.T) {
	t.Parallel()
	cw := &countingWriter{}
	rw := newReportWriter(cw)

	// Channels are not JSON-marshalable; confirm the failure surfaces
	// as an error and never reaches the underlying Write call.
	if err := rw.WriteEvent(make(chan int)); err == nil {
		t.Fatal("expected WriteEvent to fail marshaling a channel value")
	}
	if len(cw.calls) != 0 {
		t.Fatalf("expected no Write calls on marshal failure, got %d", len(cw.calls))
	}
}

func TestReportWriter_Close_NoopWhenWriterIsNotCloser(t *testing.T) {
	t.Parallel()
	rw := newReportWriter(&countingWriter{})
	if err := rw.Close(); err != nil {
		t.Fatalf("Close on a non-Closer writer should be a no-op, got: %v", err)
	}
}

func TestOpenReportWriter_AppendsToExistingContent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "report.jsonl")

	if err := os.WriteFile(path, []byte(`{"pre":"existing"}`+"\n"), 0o600); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	rw, err := OpenReportWriter(path)
	if err != nil {
		t.Fatalf("OpenReportWriter: %v", err)
	}
	if writeErr := rw.WriteEvent(map[string]string{"new": "event"}); writeErr != nil {
		t.Fatalf("WriteEvent: %v", writeErr)
	}
	if closeErr := rw.Close(); closeErr != nil {
		t.Fatalf("Close: %v", closeErr)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read report file: %v", err)
	}
	want := "{\"pre\":\"existing\"}\n{\"new\":\"event\"}\n"
	if string(got) != want {
		t.Fatalf("report file = %q, want %q (pre-existing content must survive — append, not truncate)", got, want)
	}
}

func TestOpenReportWriter_ErrorsWhenParentDirMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "no-such-subdir", "report.jsonl")

	if _, err := OpenReportWriter(path); err == nil {
		t.Fatal("expected OpenReportWriter to fail when the parent directory doesn't exist")
	}
}

func TestReportWriter_WriteEvent_UnderlyingWriteErrorPropagates(t *testing.T) {
	t.Parallel()
	rw := newReportWriter(erroringWriter{})

	if err := rw.WriteEvent(map[string]string{"a": "b"}); err == nil {
		t.Fatal("expected WriteEvent to propagate the underlying writer's error")
	}
}
