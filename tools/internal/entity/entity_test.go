package entity

import (
	"strings"
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

func TestIDFromPath(t *testing.T) {
	tests := []struct {
		path   string
		kind   Kind
		want   string
		wantOk bool
	}{
		// With slug.
		{"work/epics/E-01-platform/epic.md", KindEpic, "E-01", true},
		{"work/epics/E-01-platform/M-001-cache.md", KindMilestone, "M-001", true},
		{"work/gaps/G-001-noise.md", KindGap, "G-001", true},
		{"work/decisions/D-001-format.md", KindDecision, "D-001", true},
		{"work/contracts/C-001-orders/contract.md", KindContract, "C-001", true},
		{"docs/adr/ADR-0001-format.md", KindADR, "ADR-0001", true},

		// Slug-less.
		{"work/epics/E-01/epic.md", KindEpic, "E-01", true},
		{"work/epics/E-01-platform/M-001.md", KindMilestone, "M-001", true},
		{"work/gaps/G-001.md", KindGap, "G-001", true},
		{"docs/adr/ADR-0001.md", KindADR, "ADR-0001", true},

		// Wider id (id-pattern allows ≥canonical pad width).
		{"work/epics/E-100-big/epic.md", KindEpic, "E-100", true},
		{"docs/adr/ADR-12345-future.md", KindADR, "ADR-12345", true},

		// Mismatched kind / shape.
		{"work/epics/E-01-platform/epic.md", KindMilestone, "", false},   // wrong kind for path
		{"work/epics/E-01-platform/notes.md", KindEpic, "", false},       // not epic.md
		{"work/contracts/C-001/contract.md", KindEpic, "", false},        // wrong shape for epic
		{"work/epics/no-id/epic.md", KindEpic, "", false},                // dir not id-prefixed
		{"work/gaps/random.md", KindGap, "", false},                      // filename not id-prefixed
		{"work/epics/E-1/epic.md", KindEpic, "", false},                  // pad below canonical (E needs ≥2)
		{"work/epics/E-01-platform/epic.md", Kind("unknown"), "", false}, // default branch — unknown kind
	}
	for _, tt := range tests {
		t.Run(tt.path+":"+string(tt.kind), func(t *testing.T) {
			got, ok := IDFromPath(tt.path, tt.kind)
			if got != tt.want || ok != tt.wantOk {
				t.Errorf("IDFromPath(%q, %v) = %q, %v; want %q, %v", tt.path, tt.kind, got, ok, tt.want, tt.wantOk)
			}
		})
	}
}

func TestSchemaForKind(t *testing.T) {
	for _, k := range AllKinds() {
		t.Run(string(k), func(t *testing.T) {
			s, ok := SchemaForKind(k)
			if !ok {
				t.Fatalf("SchemaForKind(%v): not found", k)
			}
			if s.Kind != k {
				t.Errorf("Kind = %q, want %q", s.Kind, k)
			}
			if s.IDFormat == "" {
				t.Error("IDFormat empty")
			}
			if len(s.AllowedStatuses) == 0 {
				t.Error("AllowedStatuses empty")
			}
			// Every kind has at least id, title, status as required.
			for _, want := range []string{"id", "title", "status"} {
				found := false
				for _, got := range s.RequiredFields {
					if got == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("RequiredFields missing %q: got %v", want, s.RequiredFields)
				}
			}
			// Every reference field must declare a non-empty cardinality.
			for _, r := range s.References {
				if r.Cardinality != Single && r.Cardinality != Multi {
					t.Errorf("ref %q has invalid cardinality %q", r.Name, r.Cardinality)
				}
			}
		})
	}
}

func TestSchemaForKind_Unknown(t *testing.T) {
	if _, ok := SchemaForKind("nonsense"); ok {
		t.Error("expected SchemaForKind to return ok=false for unknown kind")
	}
}

func TestAllSchemas_OneEntryPerKind(t *testing.T) {
	got := AllSchemas()
	if len(got) != len(AllKinds()) {
		t.Fatalf("AllSchemas length = %d, want %d", len(got), len(AllKinds()))
	}
	for i, k := range AllKinds() {
		if got[i].Kind != k {
			t.Errorf("AllSchemas[%d].Kind = %q, want %q", i, got[i].Kind, k)
		}
	}
}

func TestAllowedStatuses_DelegatesToSchemas(t *testing.T) {
	for _, k := range AllKinds() {
		s, _ := SchemaForKind(k)
		got := AllowedStatuses(k)
		if diff := strings.Join(got, ","); diff != strings.Join(s.AllowedStatuses, ",") {
			t.Errorf("kind %v: AllowedStatuses=%v, schema.AllowedStatuses=%v", k, got, s.AllowedStatuses)
		}
	}
}

func TestIDFormat_DelegatesToSchemas(t *testing.T) {
	for _, k := range AllKinds() {
		s, _ := SchemaForKind(k)
		if got, want := IDFormat(k), s.IDFormat; got != want {
			t.Errorf("kind %v: IDFormat=%q, schema.IDFormat=%q", k, got, want)
		}
	}
}

func TestIsAllowedACStatus(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{"open", true},
		{"met", true},
		{"deferred", true},
		{"cancelled", true},
		{"", false},     // empty-string sentinel is not itself legal
		{"done", false}, // milestone-only status, not an AC status
		{"in_progress", false},
		{"OPEN", false}, // case-sensitive
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := IsAllowedACStatus(tt.status); got != tt.want {
				t.Errorf("IsAllowedACStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestIsAllowedTDDPhase(t *testing.T) {
	tests := []struct {
		phase string
		want  bool
	}{
		{"red", true},
		{"green", true},
		{"refactor", true},
		{"done", true},
		{"", false},
		{"open", false}, // AC status, not a phase
		{"RED", false},  // case-sensitive
	}
	for _, tt := range tests {
		t.Run(tt.phase, func(t *testing.T) {
			if got := IsAllowedTDDPhase(tt.phase); got != tt.want {
				t.Errorf("IsAllowedTDDPhase(%q) = %v, want %v", tt.phase, got, tt.want)
			}
		})
	}
}

func TestIsAllowedTDDPolicy(t *testing.T) {
	tests := []struct {
		policy string
		want   bool
	}{
		{"required", true},
		{"advisory", true},
		{"none", true},
		{"", false}, // absent-field default is `none`, but the empty string itself is not a legal value
		{"strict", false},
		{"None", false}, // case-sensitive
	}
	for _, tt := range tests {
		t.Run(tt.policy, func(t *testing.T) {
			if got := IsAllowedTDDPolicy(tt.policy); got != tt.want {
				t.Errorf("IsAllowedTDDPolicy(%q) = %v, want %v", tt.policy, got, tt.want)
			}
		})
	}
}

func TestACClosedSets_NoEmptyMember(t *testing.T) {
	// Belt-and-braces: confirm none of the AC closed sets accidentally
	// include the empty string as a legal value. Empty is the absent
	// sentinel and must not collide with a real value.
	for _, s := range AllowedACStatuses() {
		if s == "" {
			t.Error("AC status set contains empty string")
		}
	}
	for _, p := range AllowedTDDPhases() {
		if p == "" {
			t.Error("TDD phase set contains empty string")
		}
	}
	for _, p := range AllowedTDDPolicies() {
		if p == "" {
			t.Error("TDD policy set contains empty string")
		}
	}
}

func TestMilestoneSchema_OptionalFieldsIncludeACs(t *testing.T) {
	s, ok := SchemaForKind(KindMilestone)
	if !ok {
		t.Fatal("SchemaForKind(milestone) not found")
	}
	want := map[string]bool{"depends_on": false, "tdd": false, "acs": false}
	for _, f := range s.OptionalFields {
		if _, ok := want[f]; ok {
			want[f] = true
		}
	}
	for f, found := range want {
		if !found {
			t.Errorf("milestone OptionalFields missing %q: got %v", f, s.OptionalFields)
		}
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
