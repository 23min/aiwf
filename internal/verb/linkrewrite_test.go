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

// TestRelativeFromDir_SamePath pins the degenerate case where dir and
// target are identical: relativeFromDir returns ".". Not reachable
// through RewriteLinkDestinations under EntityMove's contract (To is
// always a file path, never equal to some other file's bare
// directory), so exercised directly.
func TestRelativeFromDir_SamePath(t *testing.T) {
	t.Parallel()
	if got := relativeFromDir("work/gaps", "work/gaps"); got != "." {
		t.Errorf("relativeFromDir(same, same) = %q, want %q", got, ".")
	}
}
