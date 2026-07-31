package cliutil_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/repolock"
)

// lock_test.go pins two claims about AcquireRepoLock's refusal paths:
// each honors --format=json with an OutputFormat envelope rather than a
// plain-text stderr line (G-0391), and each stamps that envelope with
// its own error code so a machine consumer separates contention from
// failure by identity rather than by matching message text (G-0467).
// It also pins the path that is not a refusal at all: contention is
// waited out rather than reported, up to the timeout.

// envelopeError is the subset of the JSON error envelope these tests
// assert on.
type envelopeError struct {
	Tool   string `json:"tool"`
	Status string `json:"status"`
	Error  struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// decodeEnvelope parses captured stdout as an error envelope and checks
// the two fields every refusal shares.
func decodeEnvelope(t *testing.T, captured []byte) envelopeError {
	t.Helper()
	var env envelopeError
	if err := json.Unmarshal(captured, &env); err != nil {
		t.Fatalf("stdout did not parse as a JSON envelope: %v\n%s", err, captured)
	}
	if env.Tool != "aiwf" || env.Status != "error" {
		t.Errorf("envelope tool/status = %q/%q, want aiwf/error", env.Tool, env.Status)
	}
	return env
}

// TestAcquireRepoLock_JSONEnvelopeOnBusy holds the repo lock itself
// (a zero-timeout Acquire against the same dir always returns
// ErrBusy while a lock is held), then confirms AcquireRepoLock's
// busy path emits a JSON error envelope on stdout when asked, carrying
// the retryable-contention code.
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

	env := decodeEnvelope(t, captured)
	if env.Error.Code != cliutil.CodeRepoLockBusy {
		t.Errorf("error.code = %q, want %q", env.Error.Code, cliutil.CodeRepoLockBusy)
	}
	if !strings.Contains(env.Error.Message, "another aiwf process is running") {
		t.Errorf("error.message = %q, want it to name the busy-lock condition", env.Error.Message)
	}
}

// TestAcquireRepoLock_JSONEnvelopeOnAcquireFailure drives the other
// refusal arm: a root that cannot be locked at all (a path with no
// directory behind it) fails Acquire with something other than ErrBusy,
// which must exit ExitInternal under its own code — the distinction a
// caller needs to know that retrying is pointless.
func TestAcquireRepoLock_JSONEnvelopeOnAcquireFailure(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-dir")

	var rc int
	captured := testutil.CaptureStdout(t, func() {
		_, rc = cliutil.AcquireRepoLock(missing, "aiwf test", cliutil.OutputFormat{Format: "json"})
	})
	if rc != cliutil.ExitInternal {
		t.Errorf("rc = %d, want ExitInternal", rc)
	}

	env := decodeEnvelope(t, captured)
	if env.Error.Code != cliutil.CodeRepoLockAcquireFailed {
		t.Errorf("error.code = %q, want %q", env.Error.Code, cliutil.CodeRepoLockAcquireFailed)
	}
	if !strings.Contains(env.Error.Message, "acquiring repo lock") {
		t.Errorf("error.message = %q, want it to name the acquisition failure", env.Error.Message)
	}
}

// TestAcquireRepoLock_CodeSpellings pins the two codes' exact wire
// values against literals, not against the constants that produce
// them. Every other assertion in this file compares the emitted code
// to the same constant the emitter stamped, which pins *which* refusal
// carries *which* identity but leaves the strings themselves free to
// change. They are not free: `aiwf --help` names them, and a consumer
// scripting "retry on contention" matches them, so a rename is a
// breaking change to a published contract and has to fail here rather
// than ship green.
//
// Two distinct non-empty literals also subsume the properties a
// consumer switching on the code depends on — that the two differ, and
// that neither is the empty string the envelope carried before.
func TestAcquireRepoLock_CodeSpellings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"busy", cliutil.CodeRepoLockBusy, "repo-lock-busy"},
		{"acquire failure", cliutil.CodeRepoLockAcquireFailed, "repo-lock-acquire-failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.got != tt.want {
				t.Errorf("code = %q, want %q — renaming it breaks every consumer matching the published string", tt.got, tt.want)
			}
		})
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

// TestAcquireRepoLock_WaitsForAReleasedLock pins what the two
// busy-refusal arms above cannot: a held lock is waited for, not
// refused on sight. Both of those hold the lock for the whole call, so
// they are satisfied by a wait of any length including none — reduce
// the wait to zero and every other test on the every-push path stays
// green.
//
// The wait is bounded from below rather than measured: the lock is held
// for a fixed window after the contender starts, and a contender that
// returns inside that window did not wait long enough. That direction
// is the safe one, since load can only delay the contender, never hurry
// it. The other direction has lockTimeout minus the window to absorb —
// the holder's own timer must fire before the contender's budget runs
// out. How long the wait actually is, and how many contenders get
// through, are properties of the machine and are asserted nowhere
// (G-0468).
func TestAcquireRepoLock_WaitsForAReleasedLock(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	const heldFor = 500 * time.Millisecond

	release := testutil.HoldRepoLock(t, dir)
	// The release func is idempotent, so this frees the lock on every
	// early exit and is a no-op after the deliberate release below.
	defer release()

	type acquisition struct {
		release func()
		rc      int
	}
	started := make(chan struct{})
	done := make(chan acquisition, 1)
	go func() {
		close(started)
		rel, rc := cliutil.AcquireRepoLock(dir, "aiwf test", cliutil.OutputFormat{})
		done <- acquisition{rel, rc}
	}()
	<-started

	select {
	case got := <-done:
		if got.release != nil {
			got.release()
			t.Fatal("AcquireRepoLock took the lock while another holder had it")
		}
		t.Fatalf("AcquireRepoLock refused with rc=%d while the lock was held; a contended lock must be waited for, not refused on sight — this also fires if lockTimeout has dropped below the %v this test holds for", got.rc, heldFor)
	case <-time.After(heldFor):
	}

	release()

	select {
	case got := <-done:
		if got.release == nil {
			t.Fatalf("AcquireRepoLock refused with rc=%d after the holder released; it should have taken the lock", got.rc)
		}
		got.release()
	case <-time.After(15 * time.Second):
		t.Fatal("AcquireRepoLock did not return within 15s of the lock being released")
	}
}
