package stresstest

import (
	"reflect"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
)

// TestParseDiagLog pins parseDiagLog's line-by-line decoding directly
// against fabricated byte content — including a deliberately malformed
// line — since genuine O_APPEND tearing isn't reproducible on demand
// from a real subprocess run.
func TestParseDiagLog(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		raw               string
		wantParseFailures []string
		wantLogRunIDs     []string
	}{
		{
			name:              "every line parses cleanly",
			raw:               `{"run_id":"aaa"}` + "\n" + `{"run_id":"bbb"}` + "\n",
			wantParseFailures: nil,
			wantLogRunIDs:     []string{"aaa", "bbb"},
		},
		{
			name:              "a malformed (torn) line is recorded verbatim, not fatal to the scan",
			raw:               `{"run_id":"aaa"}` + "\n" + `{"run_id":"bbb"` + "\n" + `{"run_id":"ccc"}` + "\n",
			wantParseFailures: []string{`{"run_id":"bbb"`},
			wantLogRunIDs:     []string{"aaa", "ccc"},
		},
		{
			name:              "empty file",
			raw:               "",
			wantParseFailures: nil,
			wantLogRunIDs:     nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotFailures, gotRunIDs, err := parseDiagLog([]byte(tc.raw))
			if err != nil {
				t.Fatalf("parseDiagLog: %v", err)
			}
			if !reflect.DeepEqual(gotFailures, tc.wantParseFailures) {
				t.Errorf("parseFailures = %+v, want %+v", gotFailures, tc.wantParseFailures)
			}
			if !reflect.DeepEqual(gotRunIDs, tc.wantLogRunIDs) {
				t.Errorf("logRunIDs = %+v, want %+v", gotRunIDs, tc.wantLogRunIDs)
			}
		})
	}
}

// concurrent_writer_at_scale_classify_test.go pins
// classifyConcurrentWriterAtScale — the pure decision logic behind
// ConcurrentWriterAtScaleScenario (M-0244/AC-1) — against fabricated
// parse-failure/run-id data, so every branch is exercised
// deterministically rather than depending on a real concurrent-process
// run's exact interleaving.

func TestClassifyConcurrentWriterAtScale(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		parseFailures  []string
		logRunIDs      []string
		wantRunIDs     []string
		wantSubstrings []string // nil means no violations expected
	}{
		{
			name:           "clean run: every wanted run_id appears exactly once, no parse failures",
			parseFailures:  nil,
			logRunIDs:      []string{"aaa", "bbb", "ccc"},
			wantRunIDs:     []string{"aaa", "bbb", "ccc"},
			wantSubstrings: nil,
		},
		{
			name:           "a line failed to parse (interleaved or torn), isolated from the wanted-run_id checks",
			parseFailures:  []string{`{"run_id":"aaa"`},
			logRunIDs:      []string{"bbb", "ccc"},
			wantRunIDs:     []string{"bbb", "ccc"},
			wantSubstrings: []string{`did not parse as valid JSON (interleaved or torn): "{\"run_id\":\"aaa\""`},
		},
		{
			name:           "a wanted run_id never appeared in the log (missing outcome line)",
			parseFailures:  nil,
			logRunIDs:      []string{"aaa", "bbb"},
			wantRunIDs:     []string{"aaa", "bbb", "ccc"},
			wantSubstrings: []string{"run_id ccc (one real aiwf cancel invocation's own correlation id) appears 0 times in the shared diagnostic log, want exactly 1"},
		},
		{
			name:           "a wanted run_id appeared twice (duplicated outcome line)",
			parseFailures:  nil,
			logRunIDs:      []string{"aaa", "aaa", "bbb"},
			wantRunIDs:     []string{"aaa", "bbb"},
			wantSubstrings: []string{"run_id aaa (one real aiwf cancel invocation's own correlation id) appears 2 times in the shared diagnostic log, want exactly 1"},
		},
		{
			name:           "a logged run_id matches none of this run's actors (foreign/corrupted value)",
			parseFailures:  nil,
			logRunIDs:      []string{"aaa", "zzz"},
			wantRunIDs:     []string{"aaa"},
			wantSubstrings: []string{"run_id zzz appears 1 time(s) in the shared diagnostic log but does not match any of this run's actors' own correlation ids"},
		},
		{
			name:          "every check fails at once",
			parseFailures: []string{"not json"},
			logRunIDs:     []string{"aaa", "aaa", "zzz"},
			wantRunIDs:    []string{"aaa", "bbb"},
			wantSubstrings: []string{
				`did not parse as valid JSON (interleaved or torn): "not json"`,
				"run_id aaa (one real aiwf cancel invocation's own correlation id) appears 2 times in the shared diagnostic log, want exactly 1",
				"run_id bbb (one real aiwf cancel invocation's own correlation id) appears 0 times in the shared diagnostic log, want exactly 1",
				"run_id zzz appears 1 time(s) in the shared diagnostic log but does not match any of this run's actors' own correlation ids",
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := classifyConcurrentWriterAtScale(tc.parseFailures, tc.logRunIDs, tc.wantRunIDs)
			if len(got) != len(tc.wantSubstrings) {
				t.Fatalf("violations = %+v, want %d matching %v", got, len(tc.wantSubstrings), tc.wantSubstrings)
			}
			for _, want := range tc.wantSubstrings {
				found := false
				for _, v := range got {
					if strings.Contains(v.Message, want) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("no violation contained %q; got %+v", want, got)
				}
			}
		})
	}
}

// TestConcurrentWriterAtScaleExpectedWarnings pins M-0257/AC-1's
// broadened check-clean baseline for this scenario.
func TestConcurrentWriterAtScaleExpectedWarnings(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		findings       []verbEnvelopeFinding
		wantViolations int
	}{
		{name: "no findings", findings: nil, wantViolations: 0},
		{
			name: "every baseline warning is accepted, including a repeated terminal-entity-not-archived per gap",
			findings: []verbEnvelopeFinding{
				{Code: check.CodeArchiveSweepPending, Severity: "warning"},
				{Code: check.CodeTerminalEntityNotArchived, Severity: "warning", EntityID: "G-0001"},
				{Code: check.CodeTerminalEntityNotArchived, Severity: "warning", EntityID: "G-0002"},
				{Code: check.CodeProvenanceUntrailedScopeUndefined, Severity: "warning"},
			},
			wantViolations: 0,
		},
		{
			name:           "an unbaselined warning code is a violation",
			findings:       []verbEnvelopeFinding{{Code: "some-unexpected-code", Severity: "warning"}}, //enums:ignore deliberately fabricated non-code for the test, not a real finding
			wantViolations: 1,
		},
		{
			name:           "an error-severity finding is a violation even for a baselined code",
			findings:       []verbEnvelopeFinding{{Code: check.CodeTerminalEntityNotArchived, Severity: "error"}},
			wantViolations: 1,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := classifyAgainstBaseline(tc.findings, concurrentWriterAtScaleExpectedWarnings)
			if len(got) != tc.wantViolations {
				t.Fatalf("violations = %+v, want %d", got, tc.wantViolations)
			}
		})
	}
}
