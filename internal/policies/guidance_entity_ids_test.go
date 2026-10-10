package policies

import (
	"strings"
	"testing"
)

// TestPolicy_GuidanceIDsThroughLinks holds the live tree to M-0336 AC-2: no
// development guidance document carries an id-shaped token outside a link
// destination.
func TestPolicy_GuidanceIDsThroughLinks(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyGuidanceIDsThroughLinks)
}

// TestGuidanceIDViolations pins the scan's rules (M-0336 AC-2): an id in a
// link destination passes and one in the visible text fails; a canonical
// letter-N placeholder passes; and a code span is in scope, since a command
// example cites a placeholder, never a real id.
func TestGuidanceIDViolations(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, doc, want string
	}{
		{"an id in a link destination passes", "See [the logging ADR](docs/adr/ADR-0017-logging.md).\n", ""},
		{"an id in a link's text fails", "See [ADR-0017](docs/adr/ADR-0017-logging.md).\n", `"ADR-0017"`},
		{"a canonical placeholder passes", "Run `aiwf show G-NNNN`.\n", ""},
		{"a real id in a code span fails", "Run `aiwf show G-0001`.\n", `"G-0001"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			vs := guidanceIDViolations(map[string]string{"CLAUDE.md": tc.doc})
			if tc.want == "" {
				if len(vs) != 0 {
					t.Errorf("violations = %v, want none", vs)
				}
				return
			}
			if len(vs) != 1 || !strings.Contains(vs[0].Detail, tc.want) {
				t.Errorf("violations = %v, want one naming %s", vs, tc.want)
			}
		})
	}
}
