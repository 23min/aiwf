package verb

// linkrewrite_test.go — M-0245 tests for the shared move-based
// link-destination rewrite primitive. AC-1 pins the masking
// contract (rewrite only a matched link destination; leave prose,
// inline code, fenced code, URLs, and non-matching links untouched).
// AC-2 (relative destinations) and AC-3 (idempotence + property
// test) land in their own test functions as the primitive grows.

import "testing"

func TestRewriteLinkDestinations_PreservedRegionsAndRewriteCase(t *testing.T) {
	t.Parallel()
	moves := []EntityMove{
		{From: "work/gaps/G-0045-old-slug.md", To: "work/gaps/archive/G-0045-old-slug.md"},
	}
	linkingFile := "work/epics/E-0001-foo/epic.md"

	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "rewrite case: link to a moved entity is rewritten",
			body: "See [the gap](work/gaps/G-0045-old-slug.md) for context.\n",
			want: "See [the gap](work/gaps/archive/G-0045-old-slug.md) for context.\n",
		},
		{
			name: "prose bare id mention untouched",
			body: "See G-0045 for context, and also work/gaps/G-0045-old-slug.md as plain text.\n",
			want: "See G-0045 for context, and also work/gaps/G-0045-old-slug.md as plain text.\n",
		},
		{
			name: "inline code span untouched",
			body: "The path is `work/gaps/G-0045-old-slug.md` literally.\n",
			want: "The path is `work/gaps/G-0045-old-slug.md` literally.\n",
		},
		{
			name: "fenced code block untouched",
			body: "```\n[the gap](work/gaps/G-0045-old-slug.md)\n```\n",
			want: "```\n[the gap](work/gaps/G-0045-old-slug.md)\n```\n",
		},
		{
			name: "URL-shaped destination untouched",
			body: "[the gap](https://example.com/work/gaps/G-0045-old-slug.md)\n",
			want: "[the gap](https://example.com/work/gaps/G-0045-old-slug.md)\n",
		},
		{
			name: "link to a non-moved entity untouched",
			body: "[another gap](work/gaps/G-0099-other-slug.md)\n",
			want: "[another gap](work/gaps/G-0099-other-slug.md)\n",
		},
		{
			name: "link text preserved verbatim, only destination rewritten",
			body: "[G-0045: fix the thing](work/gaps/G-0045-old-slug.md)\n",
			want: "[G-0045: fix the thing](work/gaps/archive/G-0045-old-slug.md)\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := string(RewriteLinkDestinations([]byte(tt.body), linkingFile, moves))
			if got != tt.want {
				t.Errorf("RewriteLinkDestinations(%q) = %q, want %q", tt.body, got, tt.want)
			}
		})
	}
}

func TestRewriteLinkDestinations_NoMoves_BodyUnchanged(t *testing.T) {
	t.Parallel()
	body := "See [the gap](work/gaps/G-0045-old-slug.md) for context.\n"
	got := string(RewriteLinkDestinations([]byte(body), "work/epics/E-0001-foo/epic.md", nil))
	if got != body {
		t.Errorf("RewriteLinkDestinations with no moves = %q, want unchanged %q", got, body)
	}
}
