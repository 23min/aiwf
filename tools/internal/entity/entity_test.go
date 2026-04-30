package entity

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestAllowedStatuses(t *testing.T) {
	tests := []struct {
		kind Kind
		want []string
	}{
		{KindEpic, []string{"proposed", "active", "done", "cancelled"}},
		{KindMilestone, []string{"draft", "in_progress", "done", "cancelled"}},
		{KindADR, []string{"proposed", "accepted", "superseded", "rejected"}},
		{KindGap, []string{"open", "addressed", "wontfix"}},
		{KindDecision, []string{"proposed", "accepted", "superseded", "rejected"}},
		{KindContract, []string{"proposed", "accepted", "deprecated", "retired", "rejected"}},
	}
	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			got := AllowedStatuses(tt.kind)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("AllowedStatuses(%s) mismatch (-want +got):\n%s", tt.kind, diff)
			}
		})
	}
}

func TestIsAllowedStatus(t *testing.T) {
	tests := []struct {
		kind   Kind
		status string
		want   bool
	}{
		{KindEpic, "active", true},
		{KindEpic, "in_progress", false}, // milestone-only status
		{KindMilestone, "in_progress", true},
		{KindMilestone, "active", false}, // epic-only status
		{KindGap, "open", true},
		{KindGap, "rejected", false},
		{KindContract, "accepted", true},
		{KindContract, "done", false},
		{KindEpic, "", false},
	}
	for _, tt := range tests {
		t.Run(string(tt.kind)+"/"+tt.status, func(t *testing.T) {
			if got := IsAllowedStatus(tt.kind, tt.status); got != tt.want {
				t.Errorf("IsAllowedStatus(%s, %q) = %v, want %v", tt.kind, tt.status, got, tt.want)
			}
		})
	}
}

func TestValidateID(t *testing.T) {
	tests := []struct {
		kind    Kind
		id      string
		wantErr bool
	}{
		{KindEpic, "E-01", false},
		{KindEpic, "E-99", false},
		{KindEpic, "E-100", false}, // accepts growth past pad width
		{KindEpic, "E-1", true},    // below pad width
		{KindEpic, "E01", true},    // missing dash
		{KindEpic, "E-01a", true},  // suffix not allowed
		{KindMilestone, "M-001", false},
		{KindMilestone, "M-1234", false},
		{KindMilestone, "M-99", true}, // below pad width
		{KindADR, "ADR-0001", false},
		{KindADR, "ADR-001", true}, // below pad width
		{KindGap, "G-001", false},
		{KindDecision, "D-001", false},
		{KindContract, "C-001", false},
	}
	for _, tt := range tests {
		t.Run(string(tt.kind)+"/"+tt.id, func(t *testing.T) {
			err := ValidateID(tt.kind, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateID(%s, %q) err = %v, wantErr = %v", tt.kind, tt.id, err, tt.wantErr)
			}
		})
	}
}

func TestKindFromID(t *testing.T) {
	tests := []struct {
		id     string
		want   Kind
		wantOk bool
	}{
		{"E-01", KindEpic, true},
		{"M-001", KindMilestone, true},
		{"ADR-0001", KindADR, true},
		{"G-001", KindGap, true},
		{"D-001", KindDecision, true},
		{"C-001", KindContract, true},
		{"X-01", "", false},
		{"", "", false},
		{"E-1", "", false},    // below pad width
		{"M-007a", "", false}, // suffix-form rejected
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got, ok := KindFromID(tt.id)
			if got != tt.want || ok != tt.wantOk {
				t.Errorf("KindFromID(%q) = %v, %v; want %v, %v", tt.id, got, ok, tt.want, tt.wantOk)
			}
		})
	}
}

func TestPathKind(t *testing.T) {
	tests := []struct {
		path   string
		want   Kind
		wantOk bool
	}{
		{"work/epics/E-01-platform/epic.md", KindEpic, true},
		{"work/epics/E-01-platform/M-001-cache.md", KindMilestone, true},
		{"work/epics/E-01-platform/M-001.md", KindMilestone, true},
		{"work/gaps/G-001-noise.md", KindGap, true},
		{"work/decisions/D-001-format.md", KindDecision, true},
		{"work/contracts/C-001-orders/contract.md", KindContract, true},
		{"docs/adr/ADR-0001-format.md", KindADR, true},

		// Negative cases — files that should be skipped.
		{"README.md", "", false},
		{"work/epics/E-01-platform/notes.md", "", false},
		{"work/epics/E-01-platform/sub/something.md", "", false}, // too deep
		{"work/gaps/random.md", "", false},
		{"work/contracts/C-001-orders/schema/api.yaml", "", false},
		{"docs/adr/notes.md", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, ok := PathKind(tt.path)
			if got != tt.want || ok != tt.wantOk {
				t.Errorf("PathKind(%q) = %v, %v; want %v, %v", tt.path, got, ok, tt.want, tt.wantOk)
			}
		})
	}
}
