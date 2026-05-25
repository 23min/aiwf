package entity

import (
	"strings"
	"testing"
)

// TestValidateTransition_FSMTransitionIllegalCode pins AC-2: an illegal
// transition of a *recognized* (kind, from) is reported as a Coded
// error carrying CodeFSMTransitionIllegal — extracted structurally via
// entity.Code, not by scanning the message. Legal transitions return
// nil; malformed input (unknown kind / unrecognized from) returns a
// non-Coded error, because it is not an FSM-legality refusal. The
// message contract is asserted alongside the code so the conversion
// cannot silently break M-0125's binary-level substring driver or the
// existing transition_test substring cases. FSM data is the real
// transitions map (cross-kind, both terminal and not-allowed shapes).
func TestValidateTransition_FSMTransitionIllegalCode(t *testing.T) {
	t.Parallel()

	t.Run("illegal not-allowed transitions carry the code", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name     string
			kind     Kind
			from, to string
		}{
			{"epic", KindEpic, "proposed", "done"},
			{"milestone", KindMilestone, "draft", "done"},
			{"adr", KindADR, "proposed", "superseded"},
			{"gap", KindGap, "open", "retired"},
			{"decision", KindDecision, "proposed", "superseded"},
			{"contract", KindContract, "proposed", "deprecated"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				err := ValidateTransition(tc.kind, tc.from, tc.to)
				code, ok := Code(err)
				if !ok || code != CodeFSMTransitionIllegal.ID {
					t.Fatalf("Code(%v) = (%q, %v), want (%q, true)", err, code, ok, CodeFSMTransitionIllegal.ID)
				}
				if msg := err.Error(); !strings.Contains(msg, "cannot transition to") || !strings.Contains(msg, "allowed:") {
					t.Errorf("not-allowed message %q missing expected shape", msg)
				}
			})
		}
	})

	t.Run("illegal terminal transitions carry the code", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name     string
			kind     Kind
			from, to string
		}{
			{"epic", KindEpic, "done", "active"},
			{"milestone", KindMilestone, "cancelled", "in_progress"},
			{"adr", KindADR, "superseded", "accepted"},
			{"gap", KindGap, "addressed", "open"},
			{"decision", KindDecision, "rejected", "accepted"},
			{"contract", KindContract, "retired", "accepted"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				err := ValidateTransition(tc.kind, tc.from, tc.to)
				code, ok := Code(err)
				if !ok || code != CodeFSMTransitionIllegal.ID {
					t.Fatalf("Code(%v) = (%q, %v), want (%q, true)", err, code, ok, CodeFSMTransitionIllegal.ID)
				}
				if msg := err.Error(); !strings.Contains(msg, "is terminal") {
					t.Errorf("terminal message %q missing 'is terminal'", msg)
				}
			})
		}
	})

	t.Run("legal transitions return nil", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name     string
			kind     Kind
			from, to string
		}{
			{"epic", KindEpic, "proposed", "active"},
			{"milestone", KindMilestone, "draft", "in_progress"},
			{"adr", KindADR, "proposed", "accepted"},
			{"gap", KindGap, "open", "addressed"},
			{"decision", KindDecision, "accepted", "superseded"},
			{"contract", KindContract, "accepted", "deprecated"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				if err := ValidateTransition(tc.kind, tc.from, tc.to); err != nil {
					t.Errorf("ValidateTransition(%s, %q, %q) = %v, want nil", tc.kind, tc.from, tc.to, err)
				}
			})
		}
	})

	t.Run("malformed input is not a coded FSM error", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name     string
			kind     Kind
			from, to string
		}{
			{"unknown kind", Kind("bogus"), "x", "y"},
			{"unrecognized from", KindEpic, "weird", "active"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				err := ValidateTransition(tc.kind, tc.from, tc.to)
				if err == nil {
					t.Fatalf("ValidateTransition(%s, %q, %q) = nil, want error", tc.kind, tc.from, tc.to)
				}
				if code, ok := Code(err); ok {
					t.Errorf("malformed-input error carried code %q; want not-coded (not an FSM-legality refusal)", code)
				}
			})
		}
	})
}
