package stresstest

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/repolock"
)

// concurrent_writer_at_scale_retry_test.go — G-0424. Pins the lock-busy
// retry logic ConcurrentWriterAtScaleScenario uses so a concurrent
// `aiwf cancel` that loses the repo-lock race (repolock.ErrBusy →
// ExitUsage) is retried to completion instead of aborting the whole run.
// The pure decision helpers are unit-tested against fabricated envelopes
// here; the real-binary retry seam is pinned by the held-lock integration
// test at the bottom.

// completedEnvelope / busyEnvelope / errorEnvelope build the three
// --format=json cancel envelope shapes the retry classifier must
// distinguish. busyEnvelope's message is sourced from repolock.ErrBusy so
// the fixture can't drift from the sentinel parseBusyEnvelope matches on.
func completedEnvelope(correlationID string) []byte {
	return []byte(fmt.Sprintf(
		`{"status":"ok","result":{"status":"cancelled"},"metadata":{"correlation_id":%q}}`, correlationID))
}

func busyEnvelope(correlationID string) []byte {
	return []byte(fmt.Sprintf(
		`{"status":"error","error":{"message":%q},"metadata":{"correlation_id":%q}}`,
		repolock.ErrBusy.Error()+"; retry in a moment", correlationID))
}

func errorEnvelope(message, correlationID string) []byte {
	return []byte(fmt.Sprintf(
		`{"status":"error","error":{"message":%q},"metadata":{"correlation_id":%q}}`, message, correlationID))
}

func TestParseBusyEnvelope(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		out       []byte
		wantBusy  bool
		wantCorro string
	}{
		{"lock-busy error envelope", busyEnvelope("c2"), true, "c2"},
		{"ok envelope is not busy", completedEnvelope("c1"), false, ""},
		{"error envelope with a different message is not busy", errorEnvelope("entity not found", "c3"), false, ""},
		{"error envelope with no error field is not busy", []byte(`{"status":"error","metadata":{"correlation_id":"c4"}}`), false, ""},
		{"unparseable stdout is not busy", []byte("not json"), false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			env, busy := parseBusyEnvelope(tt.out)
			if busy != tt.wantBusy {
				t.Fatalf("parseBusyEnvelope busy = %v, want %v", busy, tt.wantBusy)
			}
			if tt.wantBusy && env.Metadata.CorrelationID != tt.wantCorro {
				t.Fatalf("parseBusyEnvelope correlation id = %q, want %q", env.Metadata.CorrelationID, tt.wantCorro)
			}
		})
	}
}

func TestClassifyCancelOutcome(t *testing.T) {
	t.Parallel()
	exit2 := errors.New("exit status 2")
	tests := []struct {
		name            string
		out             []byte
		runErr          error
		wantID          string
		wantBusy        bool
		wantErrContains string
	}{
		{"success", completedEnvelope("c1"), nil, "c1", false, ""},
		{"lock-busy loss is retryable, not an error", busyEnvelope("c2"), exit2, "c2", true, ""},
		{"non-busy usage exit is a real error", errorEnvelope("entity not found", "c3"), exit2, "", false, "running aiwf cancel"},
		{"unparseable stdout on a failing exit is a real error", []byte("not json"), exit2, "", false, "running aiwf cancel"},
		{"unparseable stdout on a clean exit is a parse error", []byte("not json"), nil, "", false, "parsing aiwf"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			id, busy, err := classifyCancelOutcome("G-NNNN", tt.out, tt.runErr)
			if id != tt.wantID {
				t.Errorf("correlation id = %q, want %q", id, tt.wantID)
			}
			if busy != tt.wantBusy {
				t.Errorf("busy = %v, want %v", busy, tt.wantBusy)
			}
			switch {
			case tt.wantErrContains == "" && err != nil:
				t.Errorf("unexpected error: %v", err)
			case tt.wantErrContains != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErrContains)):
				t.Errorf("error = %v, want it to contain %q", err, tt.wantErrContains)
			}
		})
	}
}

// attemptStep scripts one retryWhileBusy attempt outcome.
type attemptStep struct {
	id   string
	busy bool
	err  error
}

func TestRetryWhileBusy(t *testing.T) {
	t.Parallel()
	errBoom := errors.New("boom")
	tests := []struct {
		name       string
		steps      []attemptStep
		alwaysBusy bool // ignore steps; every attempt reports busy (budget-exhaustion case)
		budget     int
		wantIDs    []string
		wantErr    string
	}{
		{
			name:    "success on first attempt",
			steps:   []attemptStep{{"s", false, nil}},
			budget:  busyRetryBudget,
			wantIDs: []string{"s"},
		},
		{
			name:    "busy attempts retried until success, all ids retained",
			steps:   []attemptStep{{"b1", true, nil}, {"b2", true, nil}, {"s", false, nil}},
			budget:  busyRetryBudget,
			wantIDs: []string{"b1", "b2", "s"},
		},
		{
			name:       "budget exhausted by persistent contention is a real error",
			alwaysBusy: true,
			budget:     3,
			wantErr:    "after 3 attempts",
		},
		{
			name:    "attempt error on the first try aborts immediately",
			steps:   []attemptStep{{"", false, errBoom}},
			budget:  busyRetryBudget,
			wantErr: "boom",
		},
		{
			name:    "attempt error mid-sequence discards accumulated ids",
			steps:   []attemptStep{{"b1", true, nil}, {"", false, errBoom}},
			budget:  busyRetryBudget,
			wantErr: "boom",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			calls := 0
			attempt := func() (string, bool, error) {
				calls++
				if tt.alwaysBusy {
					return fmt.Sprintf("b%d", calls), true, nil
				}
				step := tt.steps[calls-1]
				return step.id, step.busy, step.err
			}
			ids, err := retryWhileBusy(attempt, tt.budget)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want it to contain %q", err, tt.wantErr)
				}
				if ids != nil {
					t.Fatalf("ids = %v, want nil on error", ids)
				}
				// Budget exhaustion must call attempt exactly budget times —
				// pins the loop bound against an off-by-one.
				if tt.alwaysBusy && calls != tt.budget {
					t.Fatalf("attempt called %d times, want budget=%d", calls, tt.budget)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tt.wantIDs, ids); diff != "" {
				t.Fatalf("ids mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestConcurrentWriterAtScaleScenario_RealBinary_RunRetriesPastLockBusy is
// the real-binary seam for G-0424: an independently-held repo lock forces
// the actor's first `aiwf cancel` attempt to lose the race (exit 2 /
// repolock.ErrBusy); the scenario must retry past it, complete the cancel,
// and account for every attempt's diagnostic line (busy retries included)
// so both the classifier and the exact-line-count invariant stay clean.
func TestConcurrentWriterAtScaleScenario_RealBinary_RunRetriesPastLockBusy(t *testing.T) {
	t.Parallel()
	skipIfUnsupported(t)
	bin := sharedTestBinary(t)
	holderBin := sharedLockHolderBinary(t)
	dir := t.TempDir()

	s := NewConcurrentWriterAtScaleScenario(bin, 1, 1)
	if err := s.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	logPath := filepath.Join(dir, "diag.log")

	// Hold the repo lock from an independent, killable process so the
	// actor's first cancel attempt polls for the full lock timeout and then
	// exits busy. Stdin is wired and never written to (mirroring
	// LockKillScenario) so the holder blocks instead of reading EOF and
	// exiting immediately.
	holder := exec.Command(holderBin, dir) //nolint:gosec // holderBin is a path this package's own BuildLockHolder just produced, not attacker-controlled input
	stdout, err := holder.StdoutPipe()
	if err != nil {
		t.Fatalf("wiring holder stdout: %v", err)
	}
	stdinW, err := holder.StdinPipe()
	if err != nil {
		t.Fatalf("wiring holder stdin: %v", err)
	}
	defer func() { _ = stdinW.Close() }()
	if startErr := holder.Start(); startErr != nil {
		t.Fatalf("starting holder: %v", startErr)
	}
	if readyErr := waitForReady(stdout); readyErr != nil {
		_ = holder.Process.Kill()
		_ = holder.Wait()
		t.Fatalf("holder never reported ACQUIRED: %v", readyErr)
	}

	// Release the lock only once the actor's first attempt has actually gone
	// busy — detected by its verb.failed line landing in the shared log —
	// rather than after a fixed sleep a slow subprocess start could race
	// (the very load condition this fix targets). Guaranteeing a real busy
	// loss before release makes >=1 retry deterministic under any load.
	go func() {
		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) {
			raw, _ := os.ReadFile(logPath)
			if strings.Contains(string(raw), `"msg":"verb.failed"`) {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		_ = holder.Process.Kill()
		_ = holder.Wait()
	}()

	if runErr := s.Run(dir); runErr != nil {
		t.Fatalf("Run errored instead of retrying past the busy lock: %v", runErr)
	}
	if v := s.Verify(dir); len(v) != 0 {
		t.Fatalf("unexpected violations after retrying past lock-busy: %+v", v)
	}
	if len(s.wantRunIDs) < 2 {
		t.Fatalf("expected >=2 diagnostic invocations (>=1 busy retry + the success), got %d", len(s.wantRunIDs))
	}

	raw, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading shared diagnostic log: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) != len(s.wantRunIDs) {
		t.Fatalf("log has %d lines, want %d (one per real invocation, busy retries included)", len(lines), len(s.wantRunIDs))
	}
}
