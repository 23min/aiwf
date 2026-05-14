package main

import (
	"bytes"
	"io"
	"os"
	"testing"
)

// captureStdout replaces os.Stdout with a pipe for the duration of fn
// and returns whatever was written. Shared across init/history test
// files; the verbs write to os.Stdout directly so the run-dispatcher
// tests need this to assert against output.
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
