package check_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/tree"
)

// TestSkillBodyIDReference_CarveOutsPreserved drives the registered walker
// seam (check.Run over a real on-disk tree), not the scanner in isolation, so
// it pins what an operator actually gets from `aiwf check`.
//
// The doc-link carve-out is the one place a real id is sanctioned in a shipped
// surface: the id rides in the link DESTINATION, a non-prose carrier, while the
// visible text stays descriptive. Code constructs are NOT a carve-out — a real
// id in a command example ships and rots exactly as one in prose does — so the
// firing cases below are as load-bearing as the exempt ones. Together they fix
// the boundary from both sides, which a one-sided test cannot do.
func TestSkillBodyIDReference_CarveOutsPreserved(t *testing.T) {
	t.Parallel()

	const skillDir = "internal/skills/embedded/aiwf-x"

	cases := []struct {
		name      string
		relPath   string
		content   string
		wantFires bool
	}{
		{
			name:    "inline code span and fenced block fire",
			relPath: skillDir + "/SKILL.md",
			content: "---\n" +
				"name: aiwf-x\n" +
				"description: A synthetic demo skill.\n" +
				"---\n\n# aiwf-x\n\n" +
				"Run `aiwf show M-0001` to inspect the entity.\n\n" +
				"```\naiwf show M-0001\n```\n",
			wantFires: true,
		},
		{
			name:    "ADR doc-link destination exempt",
			relPath: skillDir + "/SKILL.md",
			content: "---\n" +
				"name: aiwf-x\n" +
				"description: A synthetic demo skill.\n" +
				"---\n\n# aiwf-x\n\n" +
				"See the [archive rule](docs/adr/ADR-0004-foo.md) for the design.\n",
			wantFires: false,
		},
		{
			name:    "code span inside the description field fires",
			relPath: skillDir + "/SKILL.md",
			content: "---\n" +
				"name: aiwf-x\n" +
				"description: Runs `aiwf show M-0001` and returns the matching row.\n" +
				"---\n\n# aiwf-x\n\nA clean body.\n",
			wantFires: true,
		},
		{
			// The carve-out is a prose-level construct: it exempts a link
			// DESTINATION, and a fenced block has no links to speak of —
			// CommonMark hands its content to the renderer as literal text.
			// So an id written inside a fence is visible text and fires,
			// which is the same answer the carve-out's own rationale gives:
			// there, the id is what the reader sees.
			name:    "doc-link syntax inside a fenced block is literal text and fires",
			relPath: skillDir + "/SKILL.md",
			content: "---\n" +
				"name: aiwf-x\n" +
				"description: A synthetic demo skill.\n" +
				"---\n\n# aiwf-x\n\n" +
				"```\nSee [the rule](docs/adr/ADR-0004-foo.md).\n```\n",
			wantFires: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			full := filepath.Join(root, filepath.FromSlash(tc.relPath))
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			if err := os.WriteFile(full, []byte(tc.content), 0o644); err != nil {
				t.Fatalf("write fixture: %v", err)
			}

			var hits []check.Finding
			for _, f := range check.Run(&tree.Tree{Root: root}, nil) {
				if f.Code == check.CodeSkillBodyID {
					hits = append(hits, f)
				}
			}
			if tc.wantFires && len(hits) == 0 {
				t.Fatalf("expected a skill-body-id finding, got none\ncontent:\n%s", tc.content)
			}
			if !tc.wantFires && len(hits) != 0 {
				t.Fatalf("carve-out defeated: expected no skill-body-id findings, got %d:\n%+v\ncontent:\n%s", len(hits), hits, tc.content)
			}
		})
	}
}
