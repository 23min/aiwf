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
		{KindContract, "proposed", "accepted"},
		{KindContract, "accepted", "deprecated"},
		{KindContract, "deprecated", "retired"},
		{KindContract, "proposed", "rejected"},
		{KindContract, "accepted", "rejected"},
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
		{"contract jump", KindContract, "proposed", "deprecated", "cannot transition"},
		{"contract from terminal", KindContract, "rejected", "accepted", "terminal"},
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
		{KindContract, "rejected"},
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

// TestIsLegalACTransition_AllPairs enumerates every (from, to) pair
// across the closed AC status set plus a few negative cases. Self-
// transitions are illegal (no rest-state transitions exist in the FSM).
// `deferred` and `cancelled` are terminal — every outgoing pair from
// either is illegal.
func TestIsLegalACTransition_AllPairs(t *testing.T) {
	tests := []struct {
		from, to string
		want     bool
	}{
		// open → ...
		{"open", "open", false},
		{"open", "met", true},
		{"open", "deferred", true},
		{"open", "cancelled", true},
		// met → ...
		{"met", "open", false},
		{"met", "met", false},
		{"met", "deferred", true},
		{"met", "cancelled", true},
		// deferred → * — terminal.
		{"deferred", "open", false},
		{"deferred", "met", false},
		{"deferred", "deferred", false},
		{"deferred", "cancelled", false},
		// cancelled → * — terminal.
		{"cancelled", "open", false},
		{"cancelled", "met", false},
		{"cancelled", "deferred", false},
		{"cancelled", "cancelled", false},
		// Negative cases.
		{"", "met", false},
		{"open", "", false},
		{"open", "in_progress", false}, // milestone status, not an AC status
		{"draft", "open", false},       // milestone status as `from`
	}
	for _, tt := range tests {
		t.Run(tt.from+"->"+tt.to, func(t *testing.T) {
			if got := IsLegalACTransition(tt.from, tt.to); got != tt.want {
				t.Errorf("IsLegalACTransition(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

// TestIsLegalTDDPhaseTransition_AllPairs enumerates every (from, to)
// pair across the linear TDD phase set. The linearity rules out skip-
// ahead (red → done) and backwards moves (green → red). `refactor` is
// optional — `green → done` is legal. Empty string ("pre-cycle") may
// only enter at red — entering at green or later from absent would
// bypass the red-discipline that the audit relies on.
func TestIsLegalTDDPhaseTransition_AllPairs(t *testing.T) {
	tests := []struct {
		from, to string
		want     bool
	}{
		// "" → ... (pre-cycle entry).
		{"", "red", true},
		{"", "green", false},
		{"", "refactor", false},
		{"", "done", false},
		{"", "", false},
		// red → ...
		{"red", "red", false},
		{"red", "green", true},
		{"red", "refactor", false}, // must go through green
		{"red", "done", false},     // must go through green
		// green → ...
		{"green", "red", false},
		{"green", "green", false},
		{"green", "refactor", true},
		{"green", "done", true}, // refactor is optional
		// refactor → ...
		{"refactor", "red", false},
		{"refactor", "green", false},
		{"refactor", "refactor", false},
		{"refactor", "done", true},
		// done → * — terminal.
		{"done", "red", false},
		{"done", "green", false},
		{"done", "refactor", false},
		{"done", "done", false},
		// Negative cases.
		{"red", "", false},
		{"open", "green", false}, // AC status, not a phase
		{"red", "in_progress", false},
	}
	for _, tt := range tests {
		t.Run(tt.from+"->"+tt.to, func(t *testing.T) {
			if got := IsLegalTDDPhaseTransition(tt.from, tt.to); got != tt.want {
				t.Errorf("IsLegalTDDPhaseTransition(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

// TestEveryAllowedACStatusHasTransitionEntry mirrors the existing
// drift guard for the kind FSMs: every status in AllowedACStatuses
// must have a transition entry (possibly empty), so adding a new
// status without wiring its FSM row fails loudly.
func TestEveryAllowedACStatusHasTransitionEntry(t *testing.T) {
	for _, status := range AllowedACStatuses() {
		if _, ok := acTransitions[status]; !ok {
			t.Errorf("AC status %q has no FSM entry in acTransitions", status)
		}
	}
}

// TestEveryAllowedTDDPhaseHasTransitionEntry mirrors the drift guard
// for AC statuses. Every phase in AllowedTDDPhases must have a
// transition entry (possibly empty).
func TestEveryAllowedTDDPhaseHasTransitionEntry(t *testing.T) {
	for _, phase := range AllowedTDDPhases() {
		if _, ok := tddPhaseTransitions[phase]; !ok {
			t.Errorf("TDD phase %q has no FSM entry in tddPhaseTransitions", phase)
		}
	}
}

func TestMilestoneCanGoDone(t *testing.T) {
	tests := []struct {
		name        string
		acs         []AcceptanceCriterion
		wantCanGo   bool
		wantOpenIDs []string
	}{
		{
			name:        "nil entity is permissive",
			acs:         nil,
			wantCanGo:   true,
			wantOpenIDs: nil,
		},
		{
			name:        "empty acs slice",
			acs:         []AcceptanceCriterion{},
			wantCanGo:   true,
			wantOpenIDs: nil,
		},
		{
			name: "all acs met",
			acs: []AcceptanceCriterion{
				{ID: "AC-1", Status: "met"},
				{ID: "AC-2", Status: "met"},
			},
			wantCanGo:   true,
			wantOpenIDs: nil,
		},
		{
			name: "one open ac blocks",
			acs: []AcceptanceCriterion{
				{ID: "AC-1", Status: "met"},
				{ID: "AC-2", Status: "open"},
				{ID: "AC-3", Status: "met"},
			},
			wantCanGo:   false,
			wantOpenIDs: []string{"AC-2"},
		},
		{
			name: "multiple open acs all reported",
			acs: []AcceptanceCriterion{
				{ID: "AC-1", Status: "open"},
				{ID: "AC-2", Status: "met"},
				{ID: "AC-3", Status: "open"},
			},
			wantCanGo:   false,
			wantOpenIDs: []string{"AC-1", "AC-3"},
		},
		{
			name: "deferred and cancelled are acceptable terminals",
			acs: []AcceptanceCriterion{
				{ID: "AC-1", Status: "met"},
				{ID: "AC-2", Status: "deferred"},
				{ID: "AC-3", Status: "cancelled"},
			},
			wantCanGo:   true,
			wantOpenIDs: nil,
		},
		{
			name: "open ac among terminals still blocks",
			acs: []AcceptanceCriterion{
				{ID: "AC-1", Status: "deferred"},
				{ID: "AC-2", Status: "open"},
				{ID: "AC-3", Status: "cancelled"},
			},
			wantCanGo:   false,
			wantOpenIDs: []string{"AC-2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Entity{Kind: KindMilestone, ID: "M-007", ACs: tt.acs}
			canGo, openIDs := MilestoneCanGoDone(m)
			if canGo != tt.wantCanGo {
				t.Errorf("canGoDone = %v, want %v", canGo, tt.wantCanGo)
			}
			if !equalStringSlices(openIDs, tt.wantOpenIDs) {
				t.Errorf("openACs = %v, want %v", openIDs, tt.wantOpenIDs)
			}
		})
	}
}

func TestMilestoneCanGoDone_NilEntity(t *testing.T) {
	canGo, openIDs := MilestoneCanGoDone(nil)
	if !canGo {
		t.Error("nil entity should permit milestone-done")
	}
	if openIDs != nil {
		t.Errorf("nil entity should produce nil openACs, got %v", openIDs)
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
