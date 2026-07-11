package stresstest

import "testing"

// disk_fault_classify_test.go pins classifyDiskFaultOutcome — the
// pure decision logic behind DiskFaultScenario (M-0242/AC-4) —
// against fabricated outcomes, so every branch is exercised
// deterministically rather than depending on real filesystem timing.

func TestClassifyDiskFaultOutcome(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		outcome        diskFaultOutcome
		wantViolations int
	}{
		{
			name: "clean refusal: error status, no corruption, no stray temp file, no extra commit",
			outcome: diskFaultOutcome{
				env:            verbEnvelope{Status: "error", Error: &verbEnvelopeError{Message: "permission denied"}},
				beforeBytes:    []byte("old"),
				afterBytes:     []byte("old"),
				beforeCommits:  2,
				afterCommits:   2,
				strayTempFiles: nil,
			},
			wantViolations: 0,
		},
		{
			name: "write unexpectedly succeeded (status ok) — a violation",
			outcome: diskFaultOutcome{
				env:           verbEnvelope{Status: "ok"},
				beforeBytes:   []byte("old"),
				afterBytes:    []byte("old"),
				beforeCommits: 2,
				afterCommits:  2,
			},
			wantViolations: 1,
		},
		{
			name: "error status but no Error payload — a malformed envelope, a violation",
			outcome: diskFaultOutcome{
				env:           verbEnvelope{Status: "error", Error: nil},
				beforeBytes:   []byte("old"),
				afterBytes:    []byte("old"),
				beforeCommits: 2,
				afterCommits:  2,
			},
			wantViolations: 1,
		},
		{
			name: "error message contains a panic marker — a violation",
			outcome: diskFaultOutcome{
				env:           verbEnvelope{Status: "error", Error: &verbEnvelopeError{Message: "panic: runtime error"}},
				beforeBytes:   []byte("old"),
				afterBytes:    []byte("old"),
				beforeCommits: 2,
				afterCommits:  2,
			},
			wantViolations: 1,
		},
		{
			name: "error message contains a goroutine stack-trace marker — a violation",
			outcome: diskFaultOutcome{
				env:           verbEnvelope{Status: "error", Error: &verbEnvelopeError{Message: "goroutine 1 [running]:"}},
				beforeBytes:   []byte("old"),
				afterBytes:    []byte("old"),
				beforeCommits: 2,
				afterCommits:  2,
			},
			wantViolations: 1,
		},
		{
			name: "entity file bytes changed despite the refusal — corruption, a violation",
			outcome: diskFaultOutcome{
				env:           verbEnvelope{Status: "error", Error: &verbEnvelopeError{Message: "permission denied"}},
				beforeBytes:   []byte("old"),
				afterBytes:    []byte("corrupted"),
				beforeCommits: 2,
				afterCommits:  2,
			},
			wantViolations: 1,
		},
		{
			name: "a stray temp file was left behind — a violation",
			outcome: diskFaultOutcome{
				env:            verbEnvelope{Status: "error", Error: &verbEnvelopeError{Message: "permission denied"}},
				beforeBytes:    []byte("old"),
				afterBytes:     []byte("old"),
				beforeCommits:  2,
				afterCommits:   2,
				strayTempFiles: []string{"G-0001-x.md.aiwf-tmp-123"},
			},
			wantViolations: 1,
		},
		{
			name: "a commit landed despite the refusal — a partial mutation, a violation",
			outcome: diskFaultOutcome{
				env:           verbEnvelope{Status: "error", Error: &verbEnvelopeError{Message: "permission denied"}},
				beforeBytes:   []byte("old"),
				afterBytes:    []byte("old"),
				beforeCommits: 2,
				afterCommits:  3,
			},
			wantViolations: 1,
		},
		{
			name: "every check fails at once — five violations, not short-circuited",
			outcome: diskFaultOutcome{
				env:            verbEnvelope{Status: "ok"},
				beforeBytes:    []byte("old"),
				afterBytes:     []byte("corrupted"),
				beforeCommits:  2,
				afterCommits:   3,
				strayTempFiles: []string{"stray"},
			},
			wantViolations: 4, // status-ok subsumes the malformed-envelope/panic-marker checks (both gated on status=="error")
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := classifyDiskFaultOutcome(tc.outcome)
			if len(got) != tc.wantViolations {
				t.Errorf("violations = %d (%+v), want %d", len(got), got, tc.wantViolations)
			}
		})
	}
}
