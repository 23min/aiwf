package stresstest

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/23min/aiwf/internal/cli/cliutil"
	"github.com/23min/aiwf/internal/repolock"
)

// concurrent_writer_at_scale_retry_hermetic_test.go — the hermetic half
// of ConcurrentWriterAtScaleScenario's lock-busy retry logic (G-0424,
// G-0467): four pure decision helpers exercised against fabricated
// envelopes. Every oracle here is a comparison over in-memory bytes with
// no subprocess, no clock and no contention, so the lane rule in
// CLAUDE.md §"Stress-test harness" keeps the file untagged. Its sibling
// concurrent_writer_at_scale_retry_test.go carries `//go:build stress`
// because it drives the same helpers through a real binary against a
// genuinely held lock.

// These four constructors build the --format=json cancel envelope
// shapes the retry classifier must distinguish. busyEnvelope's code is
// the production constant and its prose the production sentinel, so
// neither can drift from what a real busy exit emits; the "; retry in a
// moment" suffix the CLI appends is restated here, and is the one part
// that can.
func completedEnvelope(correlationID string) []byte {
	return fmt.Appendf(nil,
		`{"status":"ok","result":{"status":"cancelled"},"metadata":{"correlation_id":%q}}`, correlationID)
}

func busyEnvelope(correlationID string) []byte {
	return codedErrorEnvelope(cliutil.CodeRepoLockBusy, repolock.ErrBusy.Error()+"; retry in a moment", correlationID)
}

func errorEnvelope(message, correlationID string) []byte {
	return fmt.Appendf(nil,
		`{"status":"error","error":{"message":%q},"metadata":{"correlation_id":%q}}`, message, correlationID)
}

// codedErrorEnvelope is errorEnvelope's coded sibling: a failing exit
// whose refusal carries a machine-readable identity.
func codedErrorEnvelope(code, message, correlationID string) []byte {
	return fmt.Appendf(nil,
		`{"status":"error","error":{"code":%q,"message":%q},"metadata":{"correlation_id":%q}}`,
		code, message, correlationID)
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
		{"error envelope with no code is not busy", errorEnvelope("entity not found", "c3"), false, ""},
		{
			"error envelope with a different code is not busy",
			codedErrorEnvelope(cliutil.CodeRepoLockAcquireFailed, "acquiring repo lock: permission denied", "c5"),
			false, "",
		},
		{
			// Identity, not wording, is the discriminator: the busy prose
			// alone must not classify as contention.
			"busy message without the busy code is not busy",
			errorEnvelope(repolock.ErrBusy.Error()+"; retry in a moment", "c6"),
			false, "",
		},
		{"error envelope with no error field is not busy", []byte(`{"status":"error","metadata":{"correlation_id":"c4"}}`), false, ""},
		{
			// The status conjunct earns its place here: an envelope can
			// decode with an error object hanging off a non-error status,
			// and only a status of "error" means the verb actually refused.
			"ok status carrying a busy error object is not busy",
			[]byte(`{"status":"ok","error":{"code":"repo-lock-busy"},"metadata":{"correlation_id":"c7"}}`),
			false, "",
		},
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

func TestEnvelopeErrorDetail(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		out  []byte
		want string
	}{
		{
			name: "coded error envelope names both code and message",
			out:  codedErrorEnvelope(cliutil.CodeRepoLockAcquireFailed, "acquiring repo lock: permission denied", "c1"),
			want: " (envelope repo-lock-acquire-failed: acquiring repo lock: permission denied)",
		},
		{
			name: "uncoded error envelope still carries its message",
			out:  errorEnvelope("entity not found", "c2"),
			want: " (envelope: entity not found)",
		},
		{"a success envelope has no error to report", completedEnvelope("c3"), ""},
		{"unparseable stdout adds nothing", []byte("not json"), ""},
		{"empty stdout adds nothing", nil, ""},
		{
			// Well-formed JSON that fails to decode into verbEnvelope:
			// encoding/json fills the fields it could before erroring, so
			// the error object here is real but the envelope as a whole
			// never parsed. Quoting it would attribute to the subprocess an
			// account it did not give.
			name: "an envelope that failed to decode is not quoted, even carrying an error object",
			out:  []byte(`{"status":"error","error":{"code":"x","message":"m"},"findings":"not-an-array"}`),
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := envelopeErrorDetail(tt.out); got != tt.want {
				t.Errorf("envelopeErrorDetail = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClassifyCancelOutcome(t *testing.T) {
	t.Parallel()
	exit2 := errors.New("exit status 2")
	// Every input is deterministic, so the error is asserted whole
	// rather than by substring: an exact want also pins how the pieces
	// compose — that the envelope detail lands once, after the wrapped
	// run error — which a contains-check cannot see.
	tests := []struct {
		name          string
		out           []byte
		runErr        error
		wantID        string
		wantBusy      bool
		wantErr       string // exact match
		wantErrPrefix string // prefix match, for errors whose tail is another package's wording
	}{
		{name: "success", out: completedEnvelope("c1"), wantID: "c1"},
		{
			name:     "lock-busy loss is retryable, not an error",
			out:      busyEnvelope("c2"),
			runErr:   exit2,
			wantID:   "c2",
			wantBusy: true,
		},
		{
			name:   "non-busy usage exit is a real error carrying the envelope message",
			out:    errorEnvelope("entity not found", "c3"),
			runErr: exit2,
			// The run error names only "exit status 2"; the envelope message
			// is the sole account of the cause that survives into a CI log.
			wantErr: `running aiwf cancel G-NNNN: exit status 2 (envelope: entity not found)`,
		},
		{
			name:    "a coded non-busy refusal carries its code too",
			out:     codedErrorEnvelope(cliutil.CodeRepoLockAcquireFailed, "acquiring repo lock: permission denied", "c5"),
			runErr:  errors.New("exit status 3"),
			wantErr: `running aiwf cancel G-NNNN: exit status 3 (envelope repo-lock-acquire-failed: acquiring repo lock: permission denied)`,
		},
		{
			name:    "unparseable stdout on a failing exit reports the run error alone",
			out:     []byte("not json"),
			runErr:  exit2,
			wantErr: `running aiwf cancel G-NNNN: exit status 2`,
		},
		{
			// The only case asserted by prefix: the tail is
			// encoding/json's own wording, which is not this package's
			// contract to pin.
			name:          "unparseable stdout on a clean exit is a parse error",
			out:           []byte("not json"),
			wantErrPrefix: `parsing aiwf [cancel G-NNNN] JSON output: `,
		},
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
			if tt.wantErr == "" && tt.wantErrPrefix == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("err = nil, want %q", tt.wantErr+tt.wantErrPrefix)
			}
			if tt.wantErr != "" && err.Error() != tt.wantErr {
				t.Errorf("error = %q, want %q", err.Error(), tt.wantErr)
			}
			if tt.wantErrPrefix != "" && !strings.HasPrefix(err.Error(), tt.wantErrPrefix) {
				t.Errorf("error = %q, want it to start with %q", err.Error(), tt.wantErrPrefix)
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
