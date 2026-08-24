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

// TestRewriteLinkDestinations_RelativeDestinations pins AC-2: a
// relative destination (`../work/…`, any `../` depth) is recomputed
// against the linking file's own directory so the rewritten link
// resolves to the target's new location.
func TestRewriteLinkDestinations_RelativeDestinations(t *testing.T) {
	t.Parallel()
	moves := []EntityMove{
		{From: "work/gaps/G-0045-old-slug.md", To: "work/gaps/archive/G-0045-old-slug.md"},
	}

	tests := []struct {
		name        string
		linkingFile string
		body        string
		want        string
	}{
		{
			// Golden fixture reproducing the ADR-rot shape from the epic
			// context: an ADR two directories above work/ links into a
			// gap with a sibling-directory relative path.
			name:        "ADR-rot shape: two dirs up into work/gaps",
			linkingFile: "docs/adr/ADR-0004-uniform-archive-convention.md",
			body:        "See [the loader gap](../../work/gaps/G-0045-old-slug.md) for context.\n",
			want:        "See [the loader gap](../../work/gaps/archive/G-0045-old-slug.md) for context.\n",
		},
		{
			name:        "one dir up",
			linkingFile: "docs/foo.md",
			body:        "[gap](../work/gaps/G-0045-old-slug.md)\n",
			want:        "[gap](../work/gaps/archive/G-0045-old-slug.md)\n",
		},
		{
			name:        "three dirs up",
			linkingFile: "work/epics/E-0001-foo/AC-notes/deep.md",
			body:        "[gap](../../../gaps/G-0045-old-slug.md)\n",
			want:        "[gap](../../../gaps/archive/G-0045-old-slug.md)\n",
		},
		{
			name:        "root-relative destination still works alongside relative resolution",
			linkingFile: "docs/adr/ADR-0004-uniform-archive-convention.md",
			body:        "[gap](work/gaps/G-0045-old-slug.md)\n",
			want:        "[gap](work/gaps/archive/G-0045-old-slug.md)\n",
		},
		{
			name:        "relative destination to a non-moved entity untouched",
			linkingFile: "docs/adr/ADR-0004-uniform-archive-convention.md",
			body:        "[gap](../../work/gaps/G-0099-other-slug.md)\n",
			want:        "[gap](../../work/gaps/G-0099-other-slug.md)\n",
		},
		{
			// linkingFile has no directory component (path.Dir returns
			// "."), exercising the repo-root case of the relative
			// resolver — e.g. a top-level README linking into work/.
			name:        "linking file at repo root",
			linkingFile: "README.md",
			body:        "[gap](./work/gaps/G-0045-old-slug.md)\n",
			want:        "[gap](work/gaps/archive/G-0045-old-slug.md)\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := string(RewriteLinkDestinations([]byte(tt.body), tt.linkingFile, moves))
			if got != tt.want {
				t.Errorf("RewriteLinkDestinations(%q, linkingFile=%q) = %q, want %q", tt.body, tt.linkingFile, got, tt.want)
			}
		})
	}
}

// TestPathSegments pins pathSegments' full contract directly: "" and
// "." (the two spellings of "no directory") both split to nil, and a
// real path splits on "/". "" is not reachable through
// RewriteLinkDestinations (dir is always path.Dir(linkingFile), which
// never returns ""), so this is the only exercise of that arm.
func TestPathSegments(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{name: "empty string", in: "", want: nil},
		{name: "dot", in: ".", want: nil},
		{name: "single segment", in: "work", want: []string{"work"}},
		{name: "nested", in: "work/gaps/archive", want: []string{"work", "gaps", "archive"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := pathSegments(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("pathSegments(%q) = %v, want %v", tt.in, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("pathSegments(%q)[%d] = %q, want %q", tt.in, i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestRelativeFromDir pins the shapes the common-prefix walk has to
// survive, none of which RewriteLinkDestinations reaches under
// EntityMove's contract (To is always a file path, never equal to some
// other file's bare directory, and never a prefix of one).
//
// The prefix case is the load-bearing one: the walk indexes both slices
// on every iteration, so it has to stop at the shorter of the two. A
// bound that only consults dir's length reads past the end of target
// and panics rather than returning "..".
func TestRelativeFromDir(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		dir    string
		target string
		want   string
	}{
		{name: "identical dir and target", dir: "work/gaps", target: "work/gaps", want: "."},
		{name: "target is a segment-prefix of dir", dir: "work/gaps", target: "work", want: ".."},
		{name: "target is two segments above dir", dir: "work/epics/E-0001", target: "work", want: "../.."},
		{name: "dir is a segment-prefix of target", dir: "work", target: "work/gaps/G-0001-a.md", want: "gaps/G-0001-a.md"},
		{name: "sibling directories", dir: "work/gaps", target: "work/epics/E-0001/epic.md", want: "../epics/E-0001/epic.md"},
		{name: "repo root as dir", dir: ".", target: "work/gaps/G-0001-a.md", want: "work/gaps/G-0001-a.md"},
		{name: "no shared prefix at all", dir: "docs/adr", target: "work/gaps/G-0001-a.md", want: "../../work/gaps/G-0001-a.md"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := relativeFromDir(tc.dir, tc.target); got != tc.want {
				t.Errorf("relativeFromDir(%q, %q) = %q, want %q", tc.dir, tc.target, got, tc.want)
			}
		})
	}
}

// TestIsRepoPathDestination pins the destination shapes whose
// classification nothing else constrains. Everything the predicate
// rejects is returned byte-identical by the caller, so a wrong
// rejection is invisible rather than loud.
//
// The site-absolute, protocol-relative, angle-bracket, `https` and
// `mailto` shapes are deliberately absent. The archive test that moves
// an entity carrying every non-path destination shape and asserts each
// survives byte-identical already constrains them through the exported
// surface (outbound_linkrewrite_test.go, currently
// TestArchive_MovedEntityKeepsItsOwnRelativeLinksResolving); a row here
// would assert the same outcome a second time and drift from it
// independently.
//
// The leading-colon case is the one worth stating: RFC 3986 requires a
// scheme to begin with a letter, so `:foo` carries no scheme and is an
// ordinary relative path.
func TestIsRepoPathDestination(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		bare string
		want bool
	}{
		{name: "empty destination names no file", bare: "", want: false},
		{name: "leading whitespace cannot be reproduced faithfully", bare: " work/gaps/G-0001-a.md", want: false},
		{name: "trailing whitespace likewise", bare: "work/gaps/G-0001-a.md ", want: false},
		{name: "leading colon is not a scheme, so it is a repo path", bare: ":foo", want: true},
		{name: "colon after the first slash is a filename character", bare: "work/gaps/a:b.md", want: true},
		{name: "root-relative entity path", bare: "work/gaps/G-0001-a.md", want: true},
		{name: "dot-dot relative entity path", bare: "../work/gaps/G-0001-a.md", want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := isRepoPathDestination(tc.bare); got != tc.want {
				t.Errorf("isRepoPathDestination(%q) = %v, want %v", tc.bare, got, tc.want)
			}
		})
	}
}
