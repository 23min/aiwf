package stresstest

import (
	"strings"
	"testing"
)

// assertViolations checks that got holds exactly one violation per
// entry in wantSubstrings, each identified by a fragment of its
// message.
//
// Classifier tables assert on the message rather than only on the
// count because the message is the whole of what the harness reports —
// Violation carries nothing else, and it is what a failing scenario
// prints. A table that counted alone would pass a classifier emitting
// the right number of the wrong violations, which sends whoever reads
// the failure toward a defect the run never found.
func assertViolations(t *testing.T, got []Violation, wantSubstrings []string) {
	t.Helper()
	if len(got) != len(wantSubstrings) {
		t.Fatalf("violations = %+v (%d), want %d matching %v", got, len(got), len(wantSubstrings), wantSubstrings)
	}
	for _, want := range wantSubstrings {
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
}
