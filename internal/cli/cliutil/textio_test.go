package cliutil

import (
	"errors"
	"testing"
)

var errBoom = errors.New("boom")

// Serial: uses captureStdStreams (see setup_test.go's serial note).
func TestTextIO_Wrappers(t *testing.T) {
	t.Run("Errorf writes a formatted line to stderr, not stdout", func(t *testing.T) {
		out, errOut := captureStdStreams(t, func() { Errorf("aiwf %s: %v\n", "cancel", errBoom) })
		if out != "" {
			t.Errorf("stdout = %q, want empty", out)
		}
		if want := "aiwf cancel: boom\n"; errOut != want {
			t.Errorf("stderr = %q, want %q", errOut, want)
		}
	})

	t.Run("Errorln joins its args with spaces and a trailing newline on stderr", func(t *testing.T) {
		out, errOut := captureStdStreams(t, func() { Errorln("aiwf:", errBoom) })
		if out != "" {
			t.Errorf("stdout = %q, want empty", out)
		}
		if want := "aiwf: boom\n"; errOut != want {
			t.Errorf("stderr = %q, want %q", errOut, want)
		}
	})

	t.Run("Errorln with no args writes a bare newline to stderr", func(t *testing.T) {
		out, errOut := captureStdStreams(t, func() { Errorln() })
		if out != "" {
			t.Errorf("stdout = %q, want empty", out)
		}
		if want := "\n"; errOut != want {
			t.Errorf("stderr = %q, want %q", errOut, want)
		}
	})

	t.Run("Printf writes a formatted line to stdout, not stderr", func(t *testing.T) {
		out, errOut := captureStdStreams(t, func() { Printf("current:  %s\n", "v0.1.0") })
		if errOut != "" {
			t.Errorf("stderr = %q, want empty", errOut)
		}
		if want := "current:  v0.1.0\n"; out != want {
			t.Errorf("stdout = %q, want %q", out, want)
		}
	})

	t.Run("Println writes to stdout, not stderr", func(t *testing.T) {
		out, errOut := captureStdStreams(t, func() { Println("status:", "ok") })
		if errOut != "" {
			t.Errorf("stderr = %q, want empty", errOut)
		}
		if want := "status: ok\n"; out != want {
			t.Errorf("stdout = %q, want %q", out, want)
		}
	})

	t.Run("Print writes without adding a newline", func(t *testing.T) {
		out, errOut := captureStdStreams(t, func() { Print("no newline here") })
		if errOut != "" {
			t.Errorf("stderr = %q, want empty", errOut)
		}
		if want := "no newline here"; out != want {
			t.Errorf("stdout = %q, want %q", out, want)
		}
	})
}
