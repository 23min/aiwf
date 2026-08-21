package check

import (
	"strings"
	"testing"
)

// TestScanSkillClaudeMDSection covers the shipped-surface rule that a skill
// may name the consumer's own CLAUDE.md but may not cite a section of it.
// The repo whose CLAUDE.md carries those sections is this one, and it never
// ships: init/update materialize rituals into a consumer's .claude/ and
// maintain one import line, so a section citation resolves only here.
func TestScanSkillClaudeMDSection(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want []string
	}{
		{
			name: "section-sign citation fires",
			body: `Per CLAUDE.md §"Working with the user," gates are explicit.`,
			want: []string{`CLAUDE.md §"`},
		},
		{
			name: "possessive section citation fires",
			body: "Not a dispatched subagent per CLAUDE.md's \"Subagent worktree isolation\" section.",
			want: []string{"CLAUDE.md's"},
		},
		{
			name: "italic section citation fires",
			body: `See CLAUDE.md *Provenance model* for identity.`,
			want: []string{"CLAUDE.md *"},
		},
		{
			name: "per-CLAUDE.md idiom fires without a section marker",
			body: `Per CLAUDE.md, don't write an id-shaped label no verb allocated.`,
			want: []string{"Per CLAUDE.md"},
		},
		{
			name: "generic reference to the consumer's own file is clean",
			body: "- Project-specific rules in `CLAUDE.md` (root and any nested ones).",
			want: nil,
		},
		{
			name: "generic mention as a place doctrine lands is clean",
			body: "Summarise the delta: gaps closed, doctrine landed in `CLAUDE.md`.",
			want: nil,
		},
		{
			name: "a link to the consumer file is not a section citation",
			body: "See [the project rules](CLAUDE.md) for more.",
			want: nil,
		},
		{
			name: "repeated identical citations dedupe",
			body: "Per CLAUDE.md §\"A\" here.\nPer CLAUDE.md §\"A\" there.",
			want: []string{`CLAUDE.md §"`},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			findings := ScanSkillClaudeMDSection([]byte(tc.body), "x.md")
			var got []string
			for _, f := range findings {
				if f.Code != CodeSkillClaudeMDSection {
					t.Errorf("code = %q, want %q", f.Code, CodeSkillClaudeMDSection)
				}
				if f.Severity != SeverityError {
					t.Errorf("severity = %q, want error", f.Severity)
				}
				got = append(got, f.Message)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %d findings %q, want %d for %q", len(got), got, len(tc.want), tc.want)
			}
			for i, w := range tc.want {
				if !strings.Contains(got[i], w) {
					t.Errorf("message %q does not name the citation %q", got[i], w)
				}
			}
		})
	}
}

// TestScanSkillClaudeMDSection_LineNumbers pins that a finding names the
// line the citation sits on, so the operator can go straight to it.
func TestScanSkillClaudeMDSection_LineNumbers(t *testing.T) {
	t.Parallel()
	body := "clean line\nanother clean line\nPer CLAUDE.md §\"Gates\" the rule holds.\n"
	got := ScanSkillClaudeMDSection([]byte(body), "x.md")
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	if got[0].Line != 3 {
		t.Errorf("line = %d, want 3", got[0].Line)
	}
	if got[0].Path != "x.md" {
		t.Errorf("path = %q, want x.md", got[0].Path)
	}
}
