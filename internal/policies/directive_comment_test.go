package policies

import (
	"strings"
	"testing"
)

// TestHasDirectiveComment pins the escape convention both //history:ok and
// //exec:ok obey: the marker opens the comment, whitespace separates it from
// a reason, and the reason is mandatory.
//
// The table runs against both markers because the two policies share one
// matcher — a case that holds for one and not the other would mean the
// conventions had drifted, which is what sharing the matcher makes
// unrepresentable.
func TestHasDirectiveComment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// raw is a template; every MARKER is replaced with the marker under test.
		raw  string
		want bool
	}{
		{"absent", "// a plain comment", false},
		{"marker with reason", "//MARKER legacy on-disk format still in the wild", true},
		{"tab before the reason", "//MARKER\tlegacy on-disk format", true},
		{"bare marker is not an escape", "//MARKER", false},
		{"marker with only spaces after it", "//MARKER   ", false},

		// The marker must open the comment. A comment that merely names the
		// escape is documentation, not a directive — and this repo documents
		// both markers in Go doc comments, so the match-anywhere reading
		// silences a gate by writing about it.
		{"marker mid-line is not an escape", "// see below //MARKER supported older release", false},
		{"prose naming the escape is not an escape", "// use MARKER when the format is legacy", false},

		// Only whitespace separates the marker from its reason, which is what
		// keeps a longer word starting with the marker's letters from reading
		// as the directive with a one-syllable reason. The rows carrying text
		// after the longer word are the ones that isolate the separator: with
		// nothing after it, the empty-reason check rejects the input anyway.
		{"longer word opening with the marker is not an escape", "//MARKERay", false},
		{"longer word with a reason after it is not an escape", "//MARKERay because the format is legacy", false},
		{"punctuation does not separate the marker from a reason", "//MARKER-ish still parsed on read", false},
		{"a colon does not separate the marker from a reason", "//MARKER: still parsed on read", false},

		// The `//` prefix is required, so a block comment is not a directive.
		// That callers pass whole comments — never one interior line of a
		// block — is what makes this hold end to end; the seam-level case is
		// in TestAddedCommentLines_EscapeScope.
		{"a comment opening with a block marker is not a directive", "/* MARKER legacy on-disk format */", false},
	}

	for _, marker := range []string{historyOKMarker, execOKMarker, coverageIgnoreMarker} {
		for _, tt := range tests {
			t.Run(marker+"/"+tt.name, func(t *testing.T) {
				t.Parallel()
				raw := strings.ReplaceAll(tt.raw, "MARKER", marker)
				if got := hasDirectiveComment(raw, marker); got != tt.want {
					t.Errorf("hasDirectiveComment(%q, %q) = %v, want %v", raw, marker, got, tt.want)
				}
			})
		}
	}
}

// TestHasDirectiveComment_MarkersDoNotCrossMatch pins that each policy's
// escape is inert against every other marker in the family, so annotating an
// exec-mode call cannot silence a history finding or a coverage one.
//
// Each marker annotates a different property, and the three gates fire on
// different evidence; one directive standing in for another would exempt a
// block nobody examined.
func TestHasDirectiveComment_MarkersDoNotCrossMatch(t *testing.T) {
	t.Parallel()

	family := map[string]string{
		historyOKMarker:      "legacy on-disk format",
		execOKMarker:         "the mode is the subject",
		coverageIgnoreMarker: "unreachable in fixtures",
	}

	for written, reason := range family {
		raw := "//" + written + " " + reason
		for asked := range family {
			if asked == written {
				continue
			}
			if hasDirectiveComment(raw, asked) {
				t.Errorf("a //%s directive must not satisfy the %s escape; %q matched", written, asked, raw)
			}
		}
	}
}
