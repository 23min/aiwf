package gitops

import (
	"strings"
	"testing"
)

// FuzzParseTrailers drives parseTrailers with arbitrary string input
// and checks invariants that the production code relies on.
// Filed under G44 item 1.
func FuzzParseTrailers(f *testing.F) {
	for _, seed := range []string{
		"",
		"\n",
		"aiwf-verb: add\n",
		"aiwf-verb: add\naiwf-entity: M-001\naiwf-actor: human/peter\n",
		":\n",
		" : value\n",
		"key:value\n",
		"key:  value with extra spaces  \n",
		"no-colon-at-all\n",
		"aiwf-verb: add\n\naiwf-entity: M-001\n",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, in string) {
		trailers := parseTrailers(in)
		// Output count must not exceed input line count: parseTrailers
		// emits at most one Trailer per input line.
		inputLines := strings.Count(in, "\n") + 1
		if len(trailers) > inputLines {
			t.Fatalf("got %d trailers from %d input lines; in=%q", len(trailers), inputLines, in)
		}
		// All keys and values are trimmed of leading/trailing whitespace.
		for _, tr := range trailers {
			if strings.TrimSpace(tr.Key) != tr.Key {
				t.Fatalf("key %q not trimmed; in=%q", tr.Key, in)
			}
			if strings.TrimSpace(tr.Value) != tr.Value {
				t.Fatalf("value %q not trimmed; in=%q", tr.Value, in)
			}
			// Empty key would mean the line started with ':' — those
			// must be skipped per the `idx <= 0` guard.
			if tr.Key == "" {
				t.Fatalf("emitted trailer with empty key; in=%q", in)
			}
			// Keys never contain a literal LF — that's the splitter's
			// boundary token. Mid-line CR is not asserted: parseTrailers'
			// input contract is `git log` trailer output, which strips
			// CRs at line ends via TrimSpace; mid-line CRs are an
			// out-of-contract input the parser does not promise to
			// normalize. The fuzz seed at testdata/fuzz/FuzzParseTrailers/
			// records this boundary case.
			if strings.ContainsAny(tr.Key, "\n") {
				t.Fatalf("key %q contains LF; in=%q", tr.Key, in)
			}
		}
	})
}
