package initcmd

import (
	"bytes"
	"io"
	"os"
	"testing"
)

// captureStdout replaces os.Stdout with a pipe for the duration of fn
// and returns whatever was written. Local copy of the cmd/aiwf helper;
// used only by rituals_test.go which asserts against printed messages.
func captureStdout(t *testing.T, fn func()) []byte {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	done := make(chan []byte, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.Bytes()
	}()

	fn()
	_ = w.Close()
	return <-done
}
