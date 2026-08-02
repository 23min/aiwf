package check

// M-0287 AC-1: a real entity id in a shipped surface is the defect wherever it
// sits. A command example and a fenced transcript are the places such a
// citation actually accumulates, so exempting them exempts the whole
// population — measured at 50 of the tree's 50 real-id citations, none of
// which sits in plain prose.
//
// The link-destination case below is the boundary these tests defend: widening
// the scan to code constructs must not also swallow the doc-link carve-out,
// where the id legitimately rides in the destination and the visible text stays
// descriptive.

import (
	"strings"
	"testing"
)

func TestScanSkillBodyID_RealIDInCodeConstruct(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		content string
		wantTok string
		// wantLine is the 1-based line the finding must carry. Pinning it
		// (rather than only the token) is what proves the mask preserved byte
		// offsets instead of collapsing the content it now copies through.
		wantLine int
	}{
		{
			name:     "inline code span",
			content:  "# Title\n\nRun `aiwf promote E-0033 active` to activate.\n",
			wantTok:  "E-0033",
			wantLine: 3,
		},
		{
			name:     "fenced block",
			content:  "# Title\n\n```bash\naiwf show M-0001\n```\n",
			wantTok:  "M-0001",
			wantLine: 4,
		},
		{
			name:     "fenced block with a comment line",
			content:  "# Title\n\n```bash\n# closes M-0058\naiwf check\n```\n",
			wantTok:  "M-0058",
			wantLine: 4,
		},
		{
			name:     "indented code block",
			content:  "# Title\n\n    aiwf show G-0301\n",
			wantTok:  "G-0301",
			wantLine: 3,
		},
		{
			name:     "html comment",
			content:  "# Title\n\n<!-- see ADR-0008 for the width rule -->\n",
			wantTok:  "ADR-0008",
			wantLine: 3,
		},
		{
			name:     "plain prose still fires",
			content:  "# Title\n\nThis supersedes E-0038 entirely.\n",
			wantTok:  "E-0038",
			wantLine: 3,
		},
		{
			name:     "composite id in a code span",
			content:  "# Title\n\nPromote with `aiwf promote M-0007/AC-3 met`.\n",
			wantTok:  "M-0007/AC-3",
			wantLine: 3,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ScanSkillBodyID([]byte(tc.content), "shipped.md")
			if len(got) != 1 {
				t.Fatalf("want exactly 1 finding for %q, got %d: %+v", tc.wantTok, len(got), got)
			}
			f := got[0]
			if f.Code != CodeSkillBodyID {
				t.Errorf("code = %q, want %q", f.Code, CodeSkillBodyID)
			}
			if f.Line != tc.wantLine {
				t.Errorf("line = %d, want %d (content:\n%s)", f.Line, tc.wantLine, tc.content)
			}
			if !strings.Contains(f.Message, tc.wantTok) {
				t.Errorf("message %q does not name the token %q", f.Message, tc.wantTok)
			}
		})
	}
}

// TestScanSkillBodyID_DocLinkCarveOutSurvivesCodeScanning (M-0287 AC-1) is the
// boundary case: the doc-link carve-out is the one place a real id is
// sanctioned, and it must stay silent now that code constructs are scanned.
// Citing the id as the visible link TEXT is an inline citation, not a
// carve-out, so that arm still fires.
func TestScanSkillBodyID_DocLinkCarveOutSurvivesCodeScanning(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		content   string
		wantFires bool
	}{
		{
			name:      "inline link destination is exempt",
			content:   "# Title\n\nSee [the width ADR](docs/adr/ADR-0008-canonical-id-width.md).\n",
			wantFires: false,
		},
		{
			name:      "reference definition is exempt",
			content:   "# Title\n\nSee [the width ADR][w].\n\n[w]: docs/adr/ADR-0008-canonical-id-width.md\n",
			wantFires: false,
		},
		{
			name:      "id as visible link text still fires",
			content:   "# Title\n\nSee [ADR-0008](docs/adr/canonical-id-width.md).\n",
			wantFires: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ScanSkillBodyID([]byte(tc.content), "shipped.md")
			if tc.wantFires && len(got) == 0 {
				t.Fatalf("expected a finding, got none\ncontent:\n%s", tc.content)
			}
			if !tc.wantFires && len(got) != 0 {
				t.Fatalf("carve-out defeated: expected no findings, got %d: %+v\ncontent:\n%s", len(got), got, tc.content)
			}
		})
	}
}
