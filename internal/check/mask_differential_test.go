package check

// M-0287 AC-3: proseMask and proseAndCodeMask are one walker under two
// settings, so a change meant for the shipped-surface rule can silently move
// body-prose-id — where a backticked id-shape is how an entity body
// legitimately discusses id syntax, and firing on it would be a defect.
//
// The differential below is what "proseMask is unchanged" means once the
// walker is shared: the two masks agree on every construct except code.
// Asserting the masks directly pins that at the seam where it can break,
// rather than inferring it from downstream findings.

import (
	"strings"
	"testing"
)

func TestMasks_DifferOnlyOnCodeConstructs(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		src  string
		// token appears exactly once in src, so a Contains check on the
		// masked projection is unambiguous.
		token string
		// inProse / inProseAndCode: whether that mask keeps the token
		// scannable. Every row where the two differ is a code construct;
		// every row where they agree is the invariant half.
		inProse        bool
		inProseAndCode bool
	}{
		// Prose: both masks keep it.
		{"plain prose", "See M-0001 for the example.", "M-0001", true, true},
		{"link label is prose", "See [ADR-0004](docs/x.md) here.", "ADR-0004", true, true},
		{"html comment renders verbatim", "<!-- see M-0002 -->", "M-0002", true, true},

		// Code constructs: the one axis the masks differ on.
		{"inline code span", "Run `aiwf show M-0003`.", "M-0003", false, true},
		{"double-backtick span", "The shape ``M-0004`` is discussed.", "M-0004", false, true},
		{"fenced block", "```\naiwf show M-0005\n```\n", "M-0005", false, true},
		{"indented code block", "Example:\n\n    aiwf show M-0006\n", "M-0006", false, true},
		{"unclosed fence runs to EOF", "Prose.\n\n```\nM-0007 inside\n", "M-0007", false, true},

		// Non-prose link carriers: blanked by BOTH. This is the doc-link
		// carve-out, and widening the scan to code must not have widened it
		// to these.
		{"link destination", "See [the rule](docs/adr/ADR-0008-x.md).", "ADR-0008", false, false},
		{"link title", "See [label](https://example.com \"about G-0009\").", "G-0009", false, false},
		{"reference definition", "See [x][r].\n\n[r]: work/gaps/G-0010-x.md", "G-0010", false, false},
		{"autolink", "<https://example.com/G-0011.md>", "G-0011", false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			src := []byte(tc.src)
			if got := strings.Count(tc.src, tc.token); got != 1 {
				t.Fatalf("fixture must contain %q exactly once, found %d", tc.token, got)
			}

			prose := proseMask(src)
			proseCode := proseAndCodeMask(src)

			if got := strings.Contains(prose, tc.token); got != tc.inProse {
				t.Errorf("proseMask kept %q = %v, want %v\nmasked: %q", tc.token, got, tc.inProse, prose)
			}
			if got := strings.Contains(proseCode, tc.token); got != tc.inProseAndCode {
				t.Errorf("proseAndCodeMask kept %q = %v, want %v\nmasked: %q", tc.token, got, tc.inProseAndCode, proseCode)
			}
		})
	}
}
