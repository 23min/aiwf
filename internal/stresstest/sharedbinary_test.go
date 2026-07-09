package stresstest

import (
	"context"
	"os"
	"runtime"
	"sync"
	"testing"
)

// sharedBinaryOnce/-Path/-Err coordinate a single BuildBinary call
// shared across every test in this package's test binary process.
// The AC-1..AC-5 real-subprocess scenarios all need a built aiwf
// binary; rebuilding it per test would multiply an already-slow `go
// build` across dozens of subtests for no benefit. Mirrors this
// repo's own sync.Once-shared-fixture convention (CLAUDE.md "Test
// discipline"). Do not mutate the file the returned path names.
var (
	sharedBinaryOnce sync.Once
	sharedBinaryPath string
	sharedBinaryErr  error
)

// sharedTestBinary returns the absolute path to a real aiwf binary
// built from this module, built once per test process.
func sharedTestBinary(t *testing.T) string {
	t.Helper()
	sharedBinaryOnce.Do(func() {
		dir, err := os.MkdirTemp("", "stresstest-shared-bin-")
		if err != nil {
			sharedBinaryErr = err
			return
		}
		sharedBinaryPath, sharedBinaryErr = BuildBinary(context.Background(), repoRootRelative, dir)
	})
	if sharedBinaryErr != nil {
		t.Fatalf("building shared test binary: %v", sharedBinaryErr)
	}
	return sharedBinaryPath
}

// skipIfUnsupported gates the real-subprocess scenario tests in this
// package: they need `go build` (slow; skip under -short) and are
// unix-only, matching aiwf itself.
func skipIfUnsupported(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping real-subprocess stress scenario (-short); requires go build")
	}
	if runtime.GOOS == "windows" {
		t.Skip("aiwf is unix-only; stress scenarios follow suit")
	}
}
