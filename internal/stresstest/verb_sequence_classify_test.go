package stresstest

import (
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/entity"
)

// verb_sequence_classify_test.go pins classifyVerbSequenceStep and
// classifyCheckFindings — the pure decision logic behind
// VerbSequenceScenario (M-0241/AC-1) — against fabricated envelopes,
// so every branch is exercised deterministically rather than hoping
// a random walk's seed happens to hit it.

func TestClassifyVerbSequenceStep(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		kind           entity.Kind
		current        string
		target         string
		before, after  int
		env            verbEnvelope
		wantNext       string
		wantViolations int
	}{
		{
			name:    "illegal transition correctly refused as fsm-transition-illegal, no commit",
			kind:    entity.KindADR,
			current: "accepted",
			target:  "proposed",
			before:  2, after: 2,
			env:            verbEnvelope{Status: "error", Error: &verbEnvelopeError{Code: entity.CodeFSMTransitionIllegal.ID}},
			wantNext:       "accepted",
			wantViolations: 0,
		},
		{
			name:    "illegal transition NOT refused (status ok) is a violation",
			kind:    entity.KindADR,
			current: "accepted",
			target:  "proposed",
			before:  2, after: 3,
			env:            verbEnvelope{Status: "ok"},
			wantNext:       "accepted", // current stays put — an FSM-illegal "success" is never trusted to advance bookkeeping
			wantViolations: 1,          // accepted as fsm-illegal; commit count is otherwise consistent with the (bogus) success
		},
		{
			name:    "illegal transition refused as fsm-transition-illegal but a commit still landed",
			kind:    entity.KindADR,
			current: "accepted",
			target:  "proposed",
			before:  2, after: 3,
			env:            verbEnvelope{Status: "error", Error: &verbEnvelopeError{Code: entity.CodeFSMTransitionIllegal.ID}},
			wantNext:       "accepted",
			wantViolations: 1,
		},
		{
			name:    "illegal transition refused with a DIFFERENT error code — still a violation, not an FSM refusal",
			kind:    entity.KindADR,
			current: "accepted",
			target:  "proposed",
			before:  2, after: 2,
			env:            verbEnvelope{Status: "error", Error: &verbEnvelopeError{Code: "some-other-code"}}, //enums:ignore deliberately fabricated non-code for the test, not a real finding
			wantNext:       "accepted",
			wantViolations: 1,
		},
		{
			name:    "legal transition succeeds with exactly one commit",
			kind:    entity.KindADR,
			current: "proposed",
			target:  "accepted",
			before:  2, after: 3,
			env:            verbEnvelope{Status: "ok"},
			wantNext:       "accepted",
			wantViolations: 0,
		},
		{
			name:    "legal transition succeeds but landed zero commits — a violation",
			kind:    entity.KindADR,
			current: "proposed",
			target:  "accepted",
			before:  2, after: 2,
			env:            verbEnvelope{Status: "ok"},
			wantNext:       "accepted",
			wantViolations: 1,
		},
		{
			name:    "legal transition succeeds but landed two commits — a violation",
			kind:    entity.KindADR,
			current: "proposed",
			target:  "accepted",
			before:  2, after: 4,
			env:            verbEnvelope{Status: "ok"},
			wantNext:       "accepted",
			wantViolations: 1,
		},
		{
			name:    "legal transition refused by an orthogonal business rule (e.g. gap's addressed-resolver gate) — no commit, current unchanged, not a violation",
			kind:    entity.KindGap,
			current: "open",
			target:  "addressed",
			before:  2, after: 2,
			env:            verbEnvelope{Status: "error", Error: &verbEnvelopeError{Code: check.CodeGapAddressedHasResolver}},
			wantNext:       "open",
			wantViolations: 0,
		},
		{
			name:    "legal transition refused by an orthogonal business rule but a commit still landed — a violation",
			kind:    entity.KindGap,
			current: "open",
			target:  "addressed",
			before:  2, after: 3,
			env:            verbEnvelope{Status: "error", Error: &verbEnvelopeError{Code: check.CodeGapAddressedHasResolver}},
			wantNext:       "open",
			wantViolations: 1,
		},
		{
			name:    "legal transition incorrectly refused as fsm-transition-illegal — a violation",
			kind:    entity.KindGap,
			current: "open",
			target:  "wontfix",
			before:  2, after: 2,
			env:            verbEnvelope{Status: "error", Error: &verbEnvelopeError{Code: entity.CodeFSMTransitionIllegal.ID}},
			wantNext:       "open",
			wantViolations: 1,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			next, violations := classifyVerbSequenceStep(tc.kind, tc.current, tc.target, tc.before, tc.after, tc.env)
			if next != tc.wantNext {
				t.Errorf("next = %q, want %q", next, tc.wantNext)
			}
			if len(violations) != tc.wantViolations {
				t.Errorf("violations = %d (%+v), want %d", len(violations), violations, tc.wantViolations)
			}
		})
	}
}

func TestParseVerbEnvelope_ErrorsOnMalformedJSON(t *testing.T) {
	t.Parallel()
	if _, err := parseVerbEnvelope([]string{"check"}, []byte("not valid json")); err == nil {
		t.Fatal("expected an error parsing malformed JSON output")
	}
}

func TestParseVerbEnvelope_DecodesAWellFormedEnvelope(t *testing.T) {
	t.Parallel()
	env, err := parseVerbEnvelope([]string{"check"}, []byte(`{"status":"ok","metadata":{"entity_id":"E-0001"}}`))
	if err != nil {
		t.Fatalf("parseVerbEnvelope: %v", err)
	}
	if env.Status != "ok" || env.Metadata.EntityID != "E-0001" {
		t.Fatalf("unexpected decoded envelope: %+v", env)
	}
}

func TestParseCommitCount_ErrorsOnMalformedOutput(t *testing.T) {
	t.Parallel()
	if _, err := parseCommitCount([]byte("not-a-number\n")); err == nil {
		t.Fatal("expected an error parsing malformed commit-count output")
	}
}

func TestParseCommitCount_ParsesWellFormedOutput(t *testing.T) {
	t.Parallel()
	n, err := parseCommitCount([]byte("42\n"))
	if err != nil {
		t.Fatalf("parseCommitCount: %v", err)
	}
	if n != 42 {
		t.Fatalf("parseCommitCount = %d, want 42", n)
	}
}

func TestClassifyCheckFindings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		findings       []verbEnvelopeFinding
		wantViolations int
	}{
		{
			name:           "no findings",
			findings:       nil,
			wantViolations: 0,
		},
		{
			name: "only baseline expected warnings",
			findings: []verbEnvelopeFinding{
				{Code: check.CodeProvenanceUntrailedScopeUndefined, Severity: "warning"},
				{Code: check.CodeEpicActiveNoDraftedMilestones, Severity: "warning"},
				{Code: check.CodeTerminalEntityNotArchived, Severity: "warning"},
				{Code: check.CodeArchiveSweepPending, Severity: "warning"},
			},
			wantViolations: 0,
		},
		{
			name: "an error-severity finding is always a violation, even a code otherwise expected",
			findings: []verbEnvelopeFinding{
				{Code: check.CodeProvenanceUntrailedScopeUndefined, Severity: "error"},
			},
			wantViolations: 1,
		},
		{
			name: "a warning with an unexpected code is a violation",
			findings: []verbEnvelopeFinding{
				{Code: "some-unexpected-code", Severity: "warning"}, //enums:ignore deliberately fabricated non-code for the test, not a real finding
			},
			wantViolations: 1,
		},
		{
			name: "mixed: one expected, one unexpected",
			findings: []verbEnvelopeFinding{
				{Code: check.CodeProvenanceUntrailedScopeUndefined, Severity: "warning"},
				{Code: "some-unexpected-code", Severity: "warning"}, //enums:ignore deliberately fabricated non-code for the test, not a real finding
			},
			wantViolations: 1,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			violations := classifyCheckFindings(tc.findings)
			if len(violations) != tc.wantViolations {
				t.Errorf("violations = %d (%+v), want %d", len(violations), violations, tc.wantViolations)
			}
		})
	}
}

// TestIsEpicAlreadyTerminalRefusal pins G-0398's exact-match
// tolerance: only a findings slice consisting of precisely the
// epic-terminal-non-terminal-children code, and nothing else,
// qualifies as the known-and-accepted refusal Run's milestone-creation
// step skips past. Any other shape — a different code, this code
// alongside another finding, or no findings at all — must NOT match,
// so a genuinely different (and genuinely unexpected) refusal reason
// still fails the scenario instead of being silently swallowed.
func TestIsEpicAlreadyTerminalRefusal(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		findings []verbEnvelopeFinding
		want     bool
	}{
		{
			name:     "nil findings",
			findings: nil,
			want:     false,
		},
		{
			name: "exactly the epic-terminal-non-terminal-children finding",
			findings: []verbEnvelopeFinding{
				{Code: check.CodeEpicTerminalNonTerminalChildren, Severity: "error"},
			},
			want: true,
		},
		{
			name: "a different single code does not match",
			findings: []verbEnvelopeFinding{
				{Code: "some-other-code", Severity: "error"}, //enums:ignore deliberately fabricated non-code for the test, not a real finding
			},
			want: false,
		},
		{
			name: "the expected code alongside a second finding does not match",
			findings: []verbEnvelopeFinding{
				{Code: check.CodeEpicTerminalNonTerminalChildren, Severity: "error"},
				{Code: "some-other-code", Severity: "warning"}, //enums:ignore deliberately fabricated non-code for the test, not a real finding
			},
			want: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := isEpicAlreadyTerminalRefusal(tc.findings); got != tc.want {
				t.Errorf("isEpicAlreadyTerminalRefusal(%+v) = %v, want %v", tc.findings, got, tc.want)
			}
		})
	}
}

// TestIsEpicAlreadyArchivedRefusal pins the archived-parent variant of
// G-0398 that M-0250/AC-2's new "archive" walk operation newly makes
// reachable: when the epic's own walk both terminates AND archives it
// before the milestone is created, `aiwf add milestone --epic
// <archived-epic>` projects the new milestone as born inside the
// archived directory (archived-entity-not-terminal) alongside the
// epic's own non-terminal-children finding — two codes, not
// isEpicAlreadyTerminalRefusal's one. Exact-match on both codes
// together, mirroring isEpicAlreadyTerminalRefusal's own discipline.
func TestIsEpicAlreadyArchivedRefusal(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		findings []verbEnvelopeFinding
		want     bool
	}{
		{
			name:     "nil findings",
			findings: nil,
			want:     false,
		},
		{
			name: "exactly the archived-epic two-finding combination",
			findings: []verbEnvelopeFinding{
				{Code: check.CodeArchivedEntityNotTerminal, Severity: "error"},
				{Code: check.CodeEpicTerminalNonTerminalChildren, Severity: "error"},
			},
			want: true,
		},
		{
			name: "same two codes in the opposite order still matches",
			findings: []verbEnvelopeFinding{
				{Code: check.CodeEpicTerminalNonTerminalChildren, Severity: "error"},
				{Code: check.CodeArchivedEntityNotTerminal, Severity: "error"},
			},
			want: true,
		},
		{
			name: "only the terminal-in-place single code does not match (that's isEpicAlreadyTerminalRefusal's own shape)",
			findings: []verbEnvelopeFinding{
				{Code: check.CodeEpicTerminalNonTerminalChildren, Severity: "error"},
			},
			want: false,
		},
		{
			name: "only the archived code alone does not match",
			findings: []verbEnvelopeFinding{
				{Code: check.CodeArchivedEntityNotTerminal, Severity: "error"},
			},
			want: false,
		},
		{
			name: "the two expected codes alongside a third finding does not match",
			findings: []verbEnvelopeFinding{
				{Code: check.CodeArchivedEntityNotTerminal, Severity: "error"},
				{Code: check.CodeEpicTerminalNonTerminalChildren, Severity: "error"},
				{Code: "some-other-code", Severity: "warning"}, //enums:ignore deliberately fabricated non-code for the test, not a real finding
			},
			want: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := isEpicAlreadyArchivedRefusal(tc.findings); got != tc.want {
				t.Errorf("isEpicAlreadyArchivedRefusal(%+v) = %v, want %v", tc.findings, got, tc.want)
			}
		})
	}
}
