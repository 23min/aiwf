// Package testutil holds test-only helpers shared across the
// CLI test surface. These helpers are intentionally NOT under a
// `_test.go` file because they need to be imported by tests in
// other packages — _test.go files are only visible to their own
// package's tests.
//
// Production code must not import this package. The drift policy
// `PolicyTestUtilNotImportedFromProduction` under internal/policies/
// (M-0118/AC-7) is the chokepoint.
package testutil

import (
	"bytes"
	"io"
	"os"
	"testing"
)

// CaptureStdout replaces os.Stdout with a pipe for the duration of fn
// and returns whatever was written. Used by tests that drive verbs
// in-process (the verbs write to os.Stdout directly so the dispatcher
// tests need this to assert against output).
//
// Tests calling CaptureStdout cannot run under t.Parallel — os.Stdout
// is a process-level fd shared by every goroutine. The cmd/aiwf and
// internal/cli/integration test packages' setup_test.go skip-lists
// document which tests stay serial because they call CaptureStdout.
//
// Why this lives in a shared testutil package (M-0118/AC-7): the
// pre-M-0118 codebase had two parallel copies of this function — at
// cmd/aiwf/helpers_test.go and internal/cli/initcmd/helpers_test.go —
// because _test.go files cannot cross package boundaries. Sharing
// the implementation here is the only way to keep one canonical
// definition.
func CaptureStdout(t *testing.T, fn func()) []byte {
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

// CaptureStderr is the os.Stderr sibling of CaptureStdout. Used by
// tests that assert on verb-level error messages — verbs write usage
// errors to stderr while findings/data go to stdout, so separate
// capture is needed.
func CaptureStderr(t *testing.T, fn func()) []byte {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = orig }()

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

// CaptureRun redirects os.Stdout and os.Stderr around fn, returning
// the exit code and both captured streams. Used by tests that need
// both — typically `aiwf upgrade` where success/failure surface
// across both streams.
func CaptureRun(t *testing.T, fn func() int) (rc int, stdout, stderr string) {
	t.Helper()
	origOut, origErr := os.Stdout, os.Stderr
	or, ow, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	er, ew, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout, os.Stderr = ow, ew
	defer func() {
		os.Stdout, os.Stderr = origOut, origErr
	}()

	rc = fn()

	_ = ow.Close()
	_ = ew.Close()
	o, _ := io.ReadAll(or)
	e, _ := io.ReadAll(er)
	return rc, string(o), string(e)
}
