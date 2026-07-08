package stresstest

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// ReportWriter appends JSONL events to an underlying writer, one
// Write call per event. Reuses internal/logger's O_APPEND +
// one-Write()-per-record discipline (ADR-0017 Decision #5) for the
// harness's own raw-report stream rather than inventing a second
// streaming primitive: POSIX guarantees a single write(2) under
// O_APPEND is atomic with respect to the file offset, so concurrent
// writers' events are never interleaved or torn.
type ReportWriter struct {
	w io.Writer
}

// newReportWriter wraps an arbitrary io.Writer. Unexported: tests use
// it directly to observe write-call counts against a fake; production
// callers use OpenReportWriter.
func newReportWriter(w io.Writer) *ReportWriter {
	return &ReportWriter{w: w}
}

// OpenReportWriter opens path for append, creating it if absent, and
// returns a ReportWriter backed by that file.
func OpenReportWriter(path string) (*ReportWriter, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("opening raw-report file %s: %w", path, err)
	}
	return newReportWriter(f), nil
}

// WriteEvent marshals event to JSON, appends a trailing newline, and
// writes the result as exactly one Write call.
func (rw *ReportWriter) WriteEvent(event any) error {
	b, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling raw-report event: %w", err)
	}
	b = append(b, '\n')
	if _, err := rw.w.Write(b); err != nil {
		return fmt.Errorf("writing raw-report event: %w", err)
	}
	return nil
}

// Close closes the underlying writer if it implements io.Closer.
func (rw *ReportWriter) Close() error {
	if c, ok := rw.w.(io.Closer); ok {
		return c.Close()
	}
	return nil
}
