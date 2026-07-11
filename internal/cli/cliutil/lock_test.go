package cliutil_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/repolock"
)

// lock_test.go pins G-0391: AcquireRepoLock's refusal paths must honor
// --format=json (an OutputFormat envelope), not just the plain-text
// stderr message every mutating verb previously got regardless of
// requested format.

// TestAcquireRepoLock_JSONEnvelopeOnBusy holds the repo lock itself
// (a zero-timeout Acquire against the same dir always returns
// ErrBusy while a lock is held), then confirms AcquireRepoLock's
// busy path emits a JSON error envelope on stdout when asked, rather
// than a bare stderr line with empty stdout.
func TestAcquireRepoLock_JSONEnvelopeOnBusy(t *testing.T) {
	dir := t.TempDir()
	lock, err := repolock.Acquire(dir, 0)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer func() { _ = lock.Release() }()

	var rc int
	captured := testutil.CaptureStdout(t, func() {
		_, rc = cliutil.AcquireRepoLock(dir, "aiwf test", cliutil.OutputFormat{Format: "json"})
	})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}

	var env struct {
		Tool   string `json:"tool"`
		Status string `json:"status"`
		Error  struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if jsonErr := json.Unmarshal(captured, &env); jsonErr != nil {
		t.Fatalf("stdout did not parse as a JSON envelope: %v\n%s", jsonErr, captured)
	}
	if env.Tool != "aiwf" || env.Status != "error" {
		t.Errorf("envelope tool/status = %q/%q, want aiwf/error", env.Tool, env.Status)
	}
	if !strings.Contains(env.Error.Message, "another aiwf process is running") {
		t.Errorf("error.message = %q, want it to name the busy-lock condition", env.Error.Message)
	}
}

// TestAcquireRepoLock_TextModeUnchanged pins the pre-existing text-mode
// shape (label: message, to stderr) so the JSON-awareness fix doesn't
// silently reword the default (non-JSON) path.
func TestAcquireRepoLock_TextModeUnchanged(t *testing.T) {
	dir := t.TempDir()
	lock, err := repolock.Acquire(dir, 0)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer func() { _ = lock.Release() }()

	var rc int
	captured := testutil.CaptureStderr(t, func() {
		_, rc = cliutil.AcquireRepoLock(dir, "aiwf test", cliutil.OutputFormat{})
	})
	if rc != cliutil.ExitUsage {
		t.Errorf("rc = %d, want ExitUsage", rc)
	}
	want := "aiwf test: another aiwf process is running on this repo; retry in a moment\n"
	if string(captured) != want {
		t.Errorf("stderr = %q, want %q", captured, want)
	}
}

// TestAcquireRepoLock_Succeeds pins the happy path unaffected by the
// OutputFormat parameter: an uncontended lock still acquires cleanly.
func TestAcquireRepoLock_Succeeds(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	release, rc := cliutil.AcquireRepoLock(dir, "aiwf test", cliutil.OutputFormat{})
	if release == nil {
		t.Fatalf("expected a non-nil release func, rc=%d", rc)
	}
	release()
}
