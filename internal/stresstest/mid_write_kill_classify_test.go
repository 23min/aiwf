package stresstest

import "testing"

// mid_write_kill_classify_test.go pins classifyMidWriteKillOutcome —
// the pure decision logic behind MidWriteKillScenario (M-0242/AC-2) —
// against fabricated byte slices, so every branch is exercised
// deterministically rather than depending on a real kill's exact
// timing.

func TestClassifyMidWriteKillOutcome(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		wantOld        []byte
		wantNew        []byte
		got            []byte
		wantViolations int
	}{
		{
			name:           "got matches the pre-write bytes exactly (kill landed before the rename)",
			wantOld:        []byte("old content"),
			wantNew:        []byte("new content"),
			got:            []byte("old content"),
			wantViolations: 0,
		},
		{
			name:           "got matches the fully-written bytes exactly (kill landed after the rename)",
			wantOld:        []byte("old content"),
			wantNew:        []byte("new content"),
			got:            []byte("new content"),
			wantViolations: 0,
		},
		{
			name:           "got is a truncated mix of old and new — a half-written file",
			wantOld:        []byte("old content"),
			wantNew:        []byte("new content"),
			got:            []byte("new cont"),
			wantViolations: 1,
		},
		{
			name:           "got is empty while neither old nor new is empty — a half-written file",
			wantOld:        []byte("old content"),
			wantNew:        []byte("new content"),
			got:            []byte(""),
			wantViolations: 1,
		},
		{
			name:           "got is unrelated garbage — a half-written (or corrupted) file",
			wantOld:        []byte("old content"),
			wantNew:        []byte("new content"),
			got:            []byte("\x00\x01garbage"),
			wantViolations: 1,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := classifyMidWriteKillOutcome(tc.wantOld, tc.wantNew, tc.got)
			if len(got) != tc.wantViolations {
				t.Errorf("violations = %d (%+v), want %d", len(got), got, tc.wantViolations)
			}
		})
	}
}
