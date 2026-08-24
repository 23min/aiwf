package verb

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestWalkBodyLines_IdentityRewriteIsByteIdentical pins that
// walkBodyLines reconstructs its input exactly when rewriteLine changes
// nothing. Splitting on "\n" and rejoining is only faithful if the
// newline is emitted after every line but the last, and that guard is
// written out three times — once in the fence-delimiter branch, once in
// the in-fence branch, once for prose. A body ending on a fence
// delimiter or inside an unterminated fence is what distinguishes the
// first two from the third.
//
// Byte identity rather than a shape assertion, because the function's
// job here is to add nothing: a trailing newline this appends is one the
// caller then writes to an entity file.
func TestWalkBodyLines_IdentityRewriteIsByteIdentical(t *testing.T) {
	t.Parallel()
	identity := func(line string) string { return line }

	bodies := []struct {
		name string
		body string
	}{
		{name: "empty", body: ""},
		{name: "single line, no trailing newline", body: "text"},
		{name: "single line with trailing newline", body: "text\n"},
		{name: "prose ending without a newline", body: "one\ntwo"},
		{name: "ends on the opening fence delimiter", body: "text\n```"},
		{name: "ends inside an unterminated fence", body: "```\ncode"},
		{name: "ends on the closing fence delimiter", body: "```\ncode\n```"},
		{name: "closed fence then prose, no trailing newline", body: "```\ncode\n```\nafter"},
		{name: "fence with a language tag", body: "```go\nx := 1\n```"},
		{name: "blank lines are preserved", body: "a\n\n\nb"},
	}

	for _, tc := range bodies {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := string(walkBodyLines([]byte(tc.body), identity))
			if got != tc.body {
				t.Errorf("walkBodyLines(%q, identity) = %q, want byte-identical input", tc.body, got)
			}
		})
	}
}

// TestWalkBodyLines_FenceContentIsNotRewritten pins the function's
// headline contract: content inside a fenced block reaches the output
// verbatim, while content outside it goes through rewriteLine. The
// rewriter uppercases so the two are distinguishable in one assertion.
func TestWalkBodyLines_FenceContentIsNotRewritten(t *testing.T) {
	t.Parallel()
	body := "before\n```\ninside\n```\nafter"
	want := "BEFORE\n```\ninside\n```\nAFTER"

	got := string(walkBodyLines([]byte(body), strings.ToUpper))
	if got != want {
		t.Errorf("walkBodyLines with ToUpper = %q, want %q", got, want)
	}
}

// regionStrings renders the split for comparison, tagging each region
// with the classification that decides whether a caller may rewrite it.
func regionStrings(regs []linkPathRegion) []string {
	out := make([]string, 0, len(regs))
	for _, r := range regs {
		tag := "out"
		if r.inLinkPath {
			tag = "link"
		}
		out = append(out, tag+":"+r.text)
	}
	return out
}

// TestSplitLinkPathRegions pins where a link-path region starts and
// stops, which is the boundary every caller relies on to leave prose
// alone. The three malformed shapes are the interesting ones: they are
// what separate "found a link path" from "found nothing", and each is a
// documented behaviour rather than an accident.
//
// Every case also asserts that concatenating the regions reproduces the
// input. A splitter that loses or duplicates a byte would corrupt an
// entity body while still classifying correctly, and no classification
// assertion catches that on its own.
func TestSplitLinkPathRegions(t *testing.T) {
	t.Parallel()
	const dest = "work/gaps/G-0001-a.md"

	cases := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "ordinary inline link splits text from destination",
			in:   "see [a](" + dest + ") here",
			want: []string{"out:see [a]", "link:(" + dest + ")", "out: here"},
		},
		{
			name: "prose with no link is one outside region",
			in:   "no links at all",
			want: []string{"out:no links at all"},
		},
		{
			name: "input starting at the ](  delimiter still splits",
			in:   "](" + dest + ")",
			want: []string{"out:]", "link:(" + dest + ")"},
		},
		{
			name: "empty destination is a link-path region, not an unbalanced tail",
			in:   "[a]() [b](" + dest + ")",
			want: []string{"out:[a]", "link:()", "out: [b]", "link:(" + dest + ")"},
		},
		{
			name: "unbalanced open paren leaves the rest outside, unrewritable",
			in:   "[a](" + dest,
			want: []string{"out:[a]", "out:(" + dest},
		},
		{
			name: "two links in one chunk each get their own region",
			in:   "[a](x) [b](y)",
			want: []string{"out:[a]", "link:(x)", "out: [b]", "link:(y)"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			regs := splitLinkPathRegions(tc.in)
			if diff := cmp.Diff(tc.want, regionStrings(regs)); diff != "" {
				t.Errorf("splitLinkPathRegions(%q) regions mismatch (-want +got):\n%s", tc.in, diff)
			}
			var rebuilt strings.Builder
			for _, r := range regs {
				rebuilt.WriteString(r.text)
			}
			if rebuilt.String() != tc.in {
				t.Errorf("regions concatenate to %q, want the input %q", rebuilt.String(), tc.in)
			}
		})
	}
}
