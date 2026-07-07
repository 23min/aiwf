package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

// failWriter fails the test the moment Write is called — used to prove
// a disabled logger performs zero I/O (AC-1: "no I/O ... beyond the
// closed-form Info call").
type failWriter struct{ t *testing.T }

func (w failWriter) Write(p []byte) (int, error) {
	w.t.Fatalf("Write called on a disabled logger's writer: %q", p)
	return 0, nil
}

func TestNew_Disabled_NoIO(t *testing.T) {
	t.Parallel()
	l := New(Config{}, failWriter{t})
	l.Info("should not be written")
	l.Error("should not be written either")
}

func TestNew_Disabled_HandlerNeverEnabled(t *testing.T) {
	t.Parallel()
	l := New(Config{}, failWriter{t})
	h := l.Handler()
	for _, level := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
		if h.Enabled(context.Background(), level) {
			t.Fatalf("Enabled(%v) = true on a disabled logger, want false", level)
		}
	}
}

func TestNew_Enabled_TextFormat(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	l := New(Config{Enabled: true, Level: slog.LevelInfo, Format: "text"}, &buf)
	l.Info("event.fired", "verb", "promote", "entity", "M-0090")

	out := buf.String()
	for _, want := range []string{"msg=event.fired", "verb=promote", "entity=M-0090", "level=INFO"} {
		if !strings.Contains(out, want) {
			t.Fatalf("text output %q does not contain %q", out, want)
		}
	}
}

func TestNew_Enabled_JSONFormat(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	l := New(Config{Enabled: true, Level: slog.LevelInfo, Format: "json"}, &buf)
	l.Info("event.fired", "verb", "promote", "entity", "M-0090")

	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("json output %q did not parse: %v", buf.String(), err)
	}
	if decoded["msg"] != "event.fired" {
		t.Fatalf("decoded[msg] = %v, want event.fired", decoded["msg"])
	}
	if decoded["verb"] != "promote" {
		t.Fatalf("decoded[verb] = %v, want promote", decoded["verb"])
	}
	if decoded["level"] != "INFO" {
		t.Fatalf("decoded[level] = %v, want INFO", decoded["level"])
	}
}

func TestNew_Enabled_LevelFiltersBelowThreshold(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	l := New(Config{Enabled: true, Level: slog.LevelWarn, Format: "text"}, &buf)

	l.Info("below threshold, must not appear")
	if buf.Len() != 0 {
		t.Fatalf("buffer = %q after Info() below the Warn threshold, want empty", buf.String())
	}

	l.Warn("at threshold, must appear")
	if !strings.Contains(buf.String(), "msg=\"at threshold, must appear\"") {
		t.Fatalf("buffer = %q, want it to contain the Warn record", buf.String())
	}
}
