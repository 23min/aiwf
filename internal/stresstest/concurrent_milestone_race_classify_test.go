package stresstest

import (
	"testing"

	"github.com/23min/aiwf/internal/check"
)

// concurrent_milestone_race_classify_test.go pins the two pure helpers
// behind ConcurrentMilestoneRaceScenario (M-0258/AC-1): raceActorArgs
// (which argv one actor's operation builds) and buildRaceOutcome (how
// one actor's decoded envelope reduces to a raceActorOutcome) — both
// split out of Run/launchActor so their branches are deterministically
// unit-testable without depending on real race timing to exercise both
// sides.

func TestRaceActorArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		operation   string
		milestoneID string
		want        []string
	}{
		{
			name:        "promote targets the milestone's AC-1 composite id with the met target",
			operation:   raceOpPromote,
			milestoneID: "M-0007",
			want:        []string{"promote", "M-0007/AC-1", "met", "--format=json"},
		},
		{
			name:        "cancel targets the milestone itself with a reason",
			operation:   raceOpCancel,
			milestoneID: "M-0007",
			want:        []string{"cancel", "M-0007", "--reason", "concurrent-milestone-race probe", "--format=json"},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := raceActorArgs(tc.operation, tc.milestoneID)
			if len(got) != len(tc.want) {
				t.Fatalf("raceActorArgs(%q, %q) = %v, want %v", tc.operation, tc.milestoneID, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("raceActorArgs(%q, %q)[%d] = %q, want %q", tc.operation, tc.milestoneID, i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestBuildRaceOutcome(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		operation     string
		env           verbEnvelope
		wantStatus    string
		wantErrorCode string
	}{
		{
			name:          "a successful actor carries no error code",
			operation:     raceOpPromote,
			env:           verbEnvelope{Status: "ok"},
			wantStatus:    "ok",
			wantErrorCode: "",
		},
		{
			name:      "a refused actor carries the refusal's typed code",
			operation: raceOpCancel,
			env: verbEnvelope{
				Status: "error",
				Error:  &verbEnvelopeError{Code: "milestone-cancel-non-terminal-acs", Message: "refused"},
			},
			wantStatus:    "error",
			wantErrorCode: "milestone-cancel-non-terminal-acs",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := buildRaceOutcome(tc.operation, tc.env)
			if got.operation != tc.operation {
				t.Errorf("operation = %q, want %q", got.operation, tc.operation)
			}
			if got.status != tc.wantStatus {
				t.Errorf("status = %q, want %q", got.status, tc.wantStatus)
			}
			if got.errorCode != tc.wantErrorCode {
				t.Errorf("errorCode = %q, want %q", got.errorCode, tc.wantErrorCode)
			}
		})
	}
}

// TestConcurrentMilestoneRaceExpectedWarnings pins the scenario's own
// baseline map (M-0257/AC-1's convention), derived empirically by
// running the scenario repeatedly (see concurrent_milestone_race.go's
// doc comment): a provenance-scope-undefined warning is always
// accepted noise; the archive-sweep advisory pair is accepted because a
// legitimate race outcome can land the milestone at the terminal
// `cancelled` status, which this scenario never sweeps.
func TestConcurrentMilestoneRaceExpectedWarnings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		findings       []verbEnvelopeFinding
		wantViolations int
	}{
		{name: "no findings", findings: nil, wantViolations: 0},
		{
			name:           "the baseline provenance-scope-undefined warning is accepted",
			findings:       []verbEnvelopeFinding{{Code: check.CodeProvenanceUntrailedScopeUndefined, Severity: "warning"}},
			wantViolations: 0,
		},
		{
			name:           "the baseline archive-sweep-pending warning is accepted",
			findings:       []verbEnvelopeFinding{{Code: check.CodeArchiveSweepPending, Severity: "warning"}},
			wantViolations: 0,
		},
		{
			name:           "the baseline terminal-entity-not-archived warning is accepted",
			findings:       []verbEnvelopeFinding{{Code: check.CodeTerminalEntityNotArchived, Severity: "warning"}},
			wantViolations: 0,
		},
		{
			name:           "an unbaselined warning code is a violation",
			findings:       []verbEnvelopeFinding{{Code: "some-unexpected-code", Severity: "warning"}}, //enums:ignore deliberately fabricated non-code for the test, not a real finding
			wantViolations: 1,
		},
		{
			name:           "an error-severity finding is a violation even for a baselined code",
			findings:       []verbEnvelopeFinding{{Code: check.CodeProvenanceUntrailedScopeUndefined, Severity: "error"}},
			wantViolations: 1,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := classifyAgainstBaseline(tc.findings, concurrentMilestoneRaceExpectedWarnings)
			if len(got) != tc.wantViolations {
				t.Fatalf("violations = %+v, want %d", got, tc.wantViolations)
			}
		})
	}
}
