package check_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/tree"
)

// TestSkillBodyIDReference_CarveOutsPreserved (M-0227 AC-3) is a regression
// lock: broadening the scan to whole-file *.md (descriptions, templates,
// agent cards, guidance) must not defeat proseMask's exemptions. A real id
// inside an inline code span, a fenced block, or an ADR doc-link
// DESTINATION — including inside the newly-scanned description: field —
// produces no finding. It goes red only if a future mask change breaks a
// carve-out; on arrival it passes because proseMask is unchanged.
func TestSkillBodyIDReference_CarveOutsPreserved(t *testing.T) {
	t.Parallel()

	const skillDir = "internal/skills/embedded/aiwf-x"

	cases := []struct {
		name    string
		relPath string
		content string
	}{
		{
			name:    "inline code span and fenced block exempt",
			relPath: skillDir + "/SKILL.md",
			content: "---\n" +
				"name: aiwf-x\n" +
				"description: A synthetic demo skill.\n" +
				"---\n\n# aiwf-x\n\n" +
				"Run `aiwf show M-0001` to inspect the entity.\n\n" +
				"```\naiwf show M-0001\n```\n",
		},
		{
			name:    "ADR doc-link destination exempt",
			relPath: skillDir + "/SKILL.md",
			content: "---\n" +
				"name: aiwf-x\n" +
				"description: A synthetic demo skill.\n" +
				"---\n\n# aiwf-x\n\n" +
				"See the [archive rule](docs/adr/ADR-0004-foo.md) for the design.\n",
		},
		{
			name:    "code span inside the description field exempt",
			relPath: skillDir + "/SKILL.md",
			content: "---\n" +
				"name: aiwf-x\n" +
				"description: Runs `aiwf show M-0001` and returns the matching row.\n" +
				"---\n\n# aiwf-x\n\nA clean body.\n",
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
			if len(hits) != 0 {
				t.Fatalf("carve-out defeated: expected no skill-body-id findings, got %d:\n%+v\ncontent:\n%s", len(hits), hits, tc.content)
			}
		})
	}
}
