package entity

import (
	"strings"
	"testing"
)

func TestValidateTransition_Allowed(t *testing.T) {
	tests := []struct {
		kind Kind
		from string
		to   string
	}{
		{KindEpic, "proposed", "active"},
		{KindEpic, "active", "done"},
		{KindEpic, "active", "cancelled"},
		{KindMilestone, "draft", "in_progress"},
		{KindMilestone, "in_progress", "done"},
		{KindADR, "proposed", "accepted"},
		{KindADR, "accepted", "superseded"},
		{KindGap, "open", "addressed"},
		{KindGap, "open", "wontfix"},
		{KindDecision, "proposed", "rejected"},
		{KindContract, "draft", "published"},
		{KindContract, "published", "deprecated"},
		{KindContract, "deprecated", "retired"},
		{KindContract, "published", "retired"},
	}
	for _, tt := range tests {
		t.Run(string(tt.kind)+"/"+tt.from+"->"+tt.to, func(t *testing.T) {
			if err := ValidateTransition(tt.kind, tt.from, tt.to); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateTransition_Forbidden(t *testing.T) {
	tests := []struct {
		name      string
		kind      Kind
		from      string
		to        string
		errorPart string // substring expected in the error message
	}{
		{"epic skip-ahead", KindEpic, "proposed", "done", "cannot transition"},
		{"milestone backwards", KindMilestone, "in_progress", "draft", "cannot transition"},
		{"adr from terminal", KindADR, "rejected", "accepted", "terminal"},
		{"contract jump", KindContract, "draft", "deprecated", "cannot transition"},
		{"unknown source status", KindEpic, "weird", "active", "not a recognized"},
		{"unknown kind", Kind("widget"), "proposed", "active", "unknown kind"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTransition(tt.kind, tt.from, tt.to)
			if err == nil {
				t.Fatal("want error, got nil")
			}
			if !strings.Contains(err.Error(), tt.errorPart) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errorPart)
			}
		})
	}
}

func TestCancelTarget(t *testing.T) {
	tests := []struct {
		kind Kind
		want string
	}{
		{KindEpic, "cancelled"},
		{KindMilestone, "cancelled"},
		{KindADR, "rejected"},
		{KindDecision, "rejected"},
		{KindGap, "wontfix"},
		{KindContract, "retired"},
	}
	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			if got := CancelTarget(tt.kind); got != tt.want {
				t.Errorf("CancelTarget(%s) = %q, want %q", tt.kind, got, tt.want)
			}
		})
	}
}

// TestEveryAllowedStatusHasTransitionEntry guards against a kind's
// status set drifting away from its FSM. Every status in
// AllowedStatuses(k) must have a transition entry (possibly empty).
func TestEveryAllowedStatusHasTransitionEntry(t *testing.T) {
	for _, k := range AllKinds() {
		t.Run(string(k), func(t *testing.T) {
			fsm := transitions[k]
			for _, status := range AllowedStatuses(k) {
				if _, ok := fsm[status]; !ok {
					t.Errorf("status %q in AllowedStatuses(%s) has no FSM entry", status, k)
				}
			}
		})
	}
}
