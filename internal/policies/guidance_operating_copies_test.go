package policies

import (
	"maps"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/projectguidance"
)

// TestPolicy_GuidanceNoOperatingCopies holds the live tree to M-0336 AC-1:
// no development guidance document restates a shipped operating rule.
func TestPolicy_GuidanceNoOperatingCopies(t *testing.T) {
	t.Parallel()
	runPolicy(t, PolicyGuidanceNoOperatingCopies)
}

// TestOperatingCopyViolations pins what counts as a copy (M-0336 AC-1): a
// rendered managed block carrying a rule is generated output and passes,
// while a handwritten statement of it in the same file fails; an anchor's
// fragments count only within one paragraph or list item, read across its
// line wraps; and the router and the handwritten documents it links to are
// guidance too, while a routed output aiwf owns is not.
func TestOperatingCopyViolations(t *testing.T) {
	t.Parallel()
	const rule = "Decide one thing at a time."
	block := projectguidance.GuidanceStartMarker + "\n" + rule + "\n" + projectguidance.GuidanceEndMarker + "\n"
	cases := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{"a rendered block carrying the rule passes", map[string]string{"CLAUDE.md": "# Rules\n\n" + block}, ""},
		{"a handwritten copy beside the rendered block fails", map[string]string{"CLAUDE.md": "# Rules\n\n" + rule + "\n\n" + block}, "one-decision-at-a-time"},
		{"fragments in separate list items pass", map[string]string{"AGENTS.md": "- Run aiwf reallocate on a collision.\n- Never use git mv.\n"}, ""},
		{"fragments in one paragraph fail", map[string]string{"AGENTS.md": "Run aiwf reallocate on a collision,\nnever git mv.\n"}, "reallocate-not-git-mv"},
		{"a copy capitalised at the start of a sentence fails", map[string]string{"AGENTS.md": "Never suggest the user pause.\n"}, "never-suggest-pause"},
		{"fragments in separate paragraphs pass", map[string]string{"AGENTS.md": "Run aiwf reallocate on a collision.\n\nNever use git mv.\n"}, ""},
		{"items numbered past nine are separate paragraphs", map[string]string{"AGENTS.md": "10. Run aiwf reallocate on a collision.\n11. Never use git mv.\n"}, ""},
		{"a line opening with bold text is no list item", map[string]string{"AGENTS.md": "Each mutating action is its own\n**approval gate**.\n"}, "gate-per-mutation"},
		{"a blank line of spaces or a CRLF ending separates paragraphs", map[string]string{"AGENTS.md": "Run aiwf reallocate on a collision.\r\n  \r\nNever use git mv.\r\n"}, ""},
		{"a last paragraph with no closing newline is read", map[string]string{"AGENTS.md": "Never suggest the user pause."}, "never-suggest-pause"},
		{"a list marker in mid-line splits nothing", map[string]string{"AGENTS.md": "Run aiwf reallocate on a collision; the\nrule - never git mv - holds.\n"}, "reallocate-not-git-mv"},
		{"fragments split across a line wrap fail", map[string]string{"AGENTS.md": "Decide one thing at a\ntime.\n"}, "one-decision-at-a-time"},
		{"fragments split across an indented continuation fail", map[string]string{"AGENTS.md": "- Decide one thing at a\n  time.\n"}, "one-decision-at-a-time"},
		{"a copy in the router fails", map[string]string{projectguidance.RouterFile: rule + "\n"}, "one-decision-at-a-time"},
		{"a routed document aiwf owns is generated and passes", map[string]string{
			projectguidance.RouterFile:      "- [Go](packs/go/guide.md)\n",
			projectguidance.OwnedRecordFile: `{".guidance/packs/go/guide.md": "x"}`,
			".guidance/packs/go/guide.md":   rule + "\n",
		}, ""},
		{"a copy in a routed document fails", map[string]string{
			projectguidance.RouterFile:     "- [Testing](project/testing.md)\n",
			".guidance/project/testing.md": rule + "\n",
		}, "one-decision-at-a-time"},
	}
	// CommonMark's list markers: a bullet of -, * or +, or a number closed
	// by . or ), followed by a space or a tab. Each item is a paragraph of
	// its own, whichever marker opens it.
	for _, marker := range []string{"- ", "* ", "+ ", "1. ", "1) ", "-\t"} {
		doc := marker + "Run aiwf reallocate on a collision.\n" + marker + "Never use git mv.\n"
		cases = append(cases, struct {
			name  string
			files map[string]string
			want  string
		}{"fragments in separate items marked " + strconv.Quote(marker) + " pass", map[string]string{"AGENTS.md": doc}, ""})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			read := func(p string) (string, bool) { c, ok := tc.files[p]; return c, ok }
			vs := operatingCopyViolations(developmentGuidance(read))
			if tc.want == "" {
				if len(vs) != 0 {
					t.Errorf("violations = %v, want none", vs)
				}
				return
			}
			if len(vs) != 1 || !strings.Contains(vs[0].Detail, tc.want) || !slices.Contains(slices.Collect(maps.Keys(tc.files)), vs[0].File) {
				t.Errorf("violations = %v, want one naming %q", vs, tc.want)
			}
		})
	}
}
