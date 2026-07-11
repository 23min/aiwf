package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestRun_ErrorsOnWrongArgCount pins the usage branch: no repo root,
// or more than one argument, both refuse before ever touching
// internal/repolock.
func TestRun_ErrorsOnWrongArgCount(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
	}{
		{name: "no args", args: nil},
		{name: "two args", args: []string{"a", "b"}},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			stdout, stderr := newPipeFiles(t)
			code := run(tc.args, stdout, stderr, devNull(t))
			if code != 2 {
				t.Fatalf("run() = %d, want 2 (usage error)", code)
			}
		})
	}
}

// TestRun_ErrorsWhenAcquireFails pins the acquire-error branch: a
// nonexistent repo root can't resolve a lockfile path.
func TestRun_ErrorsWhenAcquireFails(t *testing.T) {
	t.Parallel()
	stdout, stderr := newPipeFiles(t)
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	code := run([]string{missing}, stdout, stderr, devNull(t))
	if code != 1 {
		t.Fatalf("run() = %d, want 1 (acquire error)", code)
	}
}

// TestRun_AcquiresPrintsReadyThenBlocksUntilStdinCloses pins the
// success path: the lock is acquired, "ACQUIRED" is printed, and the
// process blocks reading stdin until it's closed (standing in for
// being killed, without actually sending a signal in this unit test —
// the real-process kill behavior is covered by
// TestLockKillScenario_RealBinary in the stresstest package).
func TestRun_AcquiresPrintsReadyThenBlocksUntilStdinCloses(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	stdoutR, stdoutW := newPipe(t)
	_, stderrW := newPipe(t)
	stdinR, stdinW := newPipe(t)

	done := make(chan int, 1)
	go func() {
		done <- run([]string{dir}, stdoutW, stderrW, stdinR)
	}()

	buf := make([]byte, len("ACQUIRED\n"))
	if _, err := readFull(stdoutR, buf); err != nil {
		t.Fatalf("reading ready signal: %v", err)
	}
	if string(buf) != "ACQUIRED\n" {
		t.Fatalf("ready signal = %q, want %q", buf, "ACQUIRED\n")
	}

	select {
	case code := <-done:
		t.Fatalf("run returned (%d) before stdin closed — it did not block", code)
	case <-time.After(50 * time.Millisecond):
	}

	if err := stdinW.Close(); err != nil {
		t.Fatalf("closing stdin: %v", err)
	}
	select {
	case code := <-done:
		if code != 0 {
			t.Fatalf("run() = %d, want 0", code)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("run did not return after stdin closed")
	}
}

func newPipeFiles(t *testing.T) (stdout, stderr *os.File) {
	t.Helper()
	_, w1 := newPipe(t)
	_, w2 := newPipe(t)
	return w1, w2
}

func newPipe(t *testing.T) (r, w *os.File) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	t.Cleanup(func() {
		_ = r.Close()
		_ = w.Close()
	})
	return r, w
}

func devNull(t *testing.T) *os.File {
	t.Helper()
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open %s: %v", os.DevNull, err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

func readFull(r *os.File, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := r.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}
