package verb

// linkrewrite_suffix_test.go — M-0251/AC-1 tests for #fragment/?query
// suffix preservation in RewriteLinkDestinations. A destination
// carrying a suffix has its bare-path portion split off, resolved
// against the move set exactly as a suffix-free destination is, and
// the original suffix reattached verbatim on a rewrite. Also re-runs
// M-0245/AC-1's untouched-region cases (URL, code span, fenced code
// block, prose) with a suffix-bearing destination added to each, to
// prove suffix support doesn't leak past the existing masking
// boundaries.

import "testing"

func TestRewriteLinkDestinations_FragmentQuerySuffixPreserved(t *testing.T) {
	t.Parallel()
	moves := []EntityMove{
		{From: "work/gaps/G-0045-old-slug.md", To: "work/gaps/archive/G-0045-old-slug.md"},
	}
	linkingFileRoot := "work/epics/E-0001-foo/epic.md"

	tests := []struct {
		name        string
		linkingFile string
		body        string
		want        string
	}{
		{
			name:        "fragment-only, root-relative, matching move",
			linkingFile: linkingFileRoot,
			body:        "[the gap](work/gaps/G-0045-old-slug.md#some-heading) for context.\n",
			want:        "[the gap](work/gaps/archive/G-0045-old-slug.md#some-heading) for context.\n",
		},
		{
			name:        "query-only, root-relative, matching move",
			linkingFile: linkingFileRoot,
			body:        "[the gap](work/gaps/G-0045-old-slug.md?raw=true) for context.\n",
			want:        "[the gap](work/gaps/archive/G-0045-old-slug.md?raw=true) for context.\n",
		},
		{
			name:        "query and fragment combined, root-relative, matching move",
			linkingFile: linkingFileRoot,
			body:        "[the gap](work/gaps/G-0045-old-slug.md?raw=true#some-heading) for context.\n",
			want:        "[the gap](work/gaps/archive/G-0045-old-slug.md?raw=true#some-heading) for context.\n",
		},
		{
			name:        "fragment-only, relative flavor, matching move",
			linkingFile: "docs/adr/ADR-0004-uniform-archive-convention.md",
			body:        "[the gap](../../work/gaps/G-0045-old-slug.md#uniform-archive) for context.\n",
			want:        "[the gap](../../work/gaps/archive/G-0045-old-slug.md#uniform-archive) for context.\n",
		},
		{
			name:        "query and fragment, relative flavor, matching move",
			linkingFile: "docs/adr/ADR-0004-uniform-archive-convention.md",
			body:        "[the gap](../../work/gaps/G-0045-old-slug.md?x=1#uniform-archive) for context.\n",
			want:        "[the gap](../../work/gaps/archive/G-0045-old-slug.md?x=1#uniform-archive) for context.\n",
		},
		{
			name:        "fragment-only, non-matching destination stays byte-identical",
			linkingFile: linkingFileRoot,
			body:        "[another gap](work/gaps/G-0099-other-slug.md#not-moved)\n",
			want:        "[another gap](work/gaps/G-0099-other-slug.md#not-moved)\n",
		},
		{
			name:        "query-only, non-matching destination stays byte-identical",
			linkingFile: linkingFileRoot,
			body:        "[another gap](work/gaps/G-0099-other-slug.md?x=1)\n",
			want:        "[another gap](work/gaps/G-0099-other-slug.md?x=1)\n",
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

// TestRewriteLinkDestinations_UntouchedRegions_WithSuffix re-runs
// M-0245/AC-1's untouched-region cases (URL, code span, fenced code
// block, plain prose) with a suffix-bearing destination added to
// each, proving suffix support doesn't leak past the existing
// fence/code-span/URL masking boundaries.
func TestRewriteLinkDestinations_UntouchedRegions_WithSuffix(t *testing.T) {
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
			name: "URL-shaped destination with fragment untouched",
			body: "[the gap](https://example.com/work/gaps/G-0045-old-slug.md#heading)\n",
			want: "[the gap](https://example.com/work/gaps/G-0045-old-slug.md#heading)\n",
		},
		{
			name: "inline code span with suffix untouched",
			body: "The path is `work/gaps/G-0045-old-slug.md#heading` literally.\n",
			want: "The path is `work/gaps/G-0045-old-slug.md#heading` literally.\n",
		},
		{
			name: "fenced code block with suffix untouched",
			body: "```\n[the gap](work/gaps/G-0045-old-slug.md#heading)\n```\n",
			want: "```\n[the gap](work/gaps/G-0045-old-slug.md#heading)\n```\n",
		},
		{
			name: "prose bare mention with suffix untouched",
			body: "See work/gaps/G-0045-old-slug.md#heading as plain text.\n",
			want: "See work/gaps/G-0045-old-slug.md#heading as plain text.\n",
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
