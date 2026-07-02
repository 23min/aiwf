package check_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/tree"
)

// TestScanSkillBodyID pins the skill-body id-reference rule (G-0299): a
// shipped skill body must cite no real (digit-bearing) entity id. The rule
// is the mirror image of body-prose-id — here a digit-bearing strict-form
// id is the defect and a canonical letter-N placeholder is correct.
//
// The scanner reuses the body-prose-id prose-mask, so tokens inside code
// constructs and inside non-prose link carriers (destinations) are exempt
// by construction; that is what gives the ADR/design doc-link carve-out for
// free (the id lives in the link destination, the visible text is prose).
func TestScanSkillBodyID(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		body     string
		wantFire bool
	}{
		// AC-1 — fires on a real digit-bearing id in prose.
		{name: "bare real bare id in prose", body: "See M-0001 for the worked example.", wantFire: true},
		{name: "bare real ADR id in prose", body: "This follows ADR-0004 for archiving.", wantFire: true},
		{name: "real composite id in prose", body: "The criterion M-0001/AC-1 is met.", wantFire: true},
		{name: "id inside a filesystem path in prose", body: "Edit work/epics/E-0044/M-0185-foo.md by hand.", wantFire: true},

		// AC-1 — silent on a clean body.
		{name: "clean prose, no id-shapes", body: "Run the verb and confirm exactly one commit.", wantFire: false},

		// AC-2 — silent on canonical letter-N placeholders.
		{name: "canonical bare placeholder", body: "Use the canonical G-NNNN placeholder shape.", wantFire: false},
		{name: "canonical composite placeholder", body: "Address it as M-NNNN/AC-N in prose.", wantFire: false},

		// AC-2 — silent on code-masked id-shapes.
		{name: "real id in an inline code span", body: "Reference the canonical id (`M-0001`, not `M-1`).", wantFire: false},
		{
			name:     "real id in a fenced code block",
			body:     "Example:\n\n```\naiwf show M-0001\n```\n",
			wantFire: false,
		},

		// AC-2 — the ADR/design doc-link carve-out: the id rides in the
		// link DESTINATION (a non-prose carrier the mask exempts), the
		// visible text is descriptive prose.
		{name: "id in a docs doc-link destination", body: "See the [uniform archive convention](docs/adr/ADR-0004-uniform-archive.md) for the rule.", wantFire: false},

		// AC-2 — firing contrast: citing the id inline as the visible link
		// TEXT is an inline citation, not a carve-out, so it fires.
		{name: "id as visible link text", body: "See [ADR-0004](docs/adr/ADR-0004-uniform-archive.md).", wantFire: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := check.ScanSkillBodyID([]byte(tc.body), "internal/skills/embedded/aiwf-demo/SKILL.md")
			if tc.wantFire {
				if len(got) == 0 {
					t.Fatalf("expected a skill-body-id finding, got none\nbody: %q", tc.body)
				}
				for _, f := range got {
					if f.Code != check.CodeSkillBodyID {
						t.Errorf("finding code = %q, want %q", f.Code, check.CodeSkillBodyID)
					}
					if f.Severity != check.SeverityError {
						t.Errorf("finding severity = %q, want %q", f.Severity, check.SeverityError)
					}
				}
			} else if len(got) != 0 {
				t.Fatalf("expected no findings, got %d:\n%+v\nbody: %q", len(got), got, tc.body)
			}
		})
	}
}

// TestScanSkillBodyID_DedupesPerToken pins that one bad token mentioned
// many times in a single body produces one finding, not one per mention —
// mirroring the body-prose-id dedupe contract.
func TestScanSkillBodyID_DedupesPerToken(t *testing.T) {
	t.Parallel()
	body := "M-0001 here, and M-0001 again, and once more M-0001."
	got := check.ScanSkillBodyID([]byte(body), "internal/skills/embedded/aiwf-demo/SKILL.md")
	if len(got) != 1 {
		t.Fatalf("expected exactly one deduped finding, got %d:\n%+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "M-0001") {
		t.Errorf("message %q does not name the offending token", got[0].Message)
	}
}

// TestSkillBodyIDReference_Seam drives the tree-walking rule through
// check.Run against an on-disk SKILL.md fixture, exercising the seam the
// byte-level scanner test cannot: the directory walk, the frontmatter
// split, and the body-relative-to-file-relative line adjustment. Per
// CLAUDE.md "Test the seam, not just the layer".
func TestSkillBodyIDReference_Seam(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	skillDir := filepath.Join(root, "internal", "skills", "embedded", "aiwf-demo")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Frontmatter on lines 1-4; the real id sits on file line 8, so a
	// passing line assertion proves the offset adjustment, not just the
	// body-relative line.
	skill := "---\n" +
		"name: aiwf-demo\n" +
		"description: A synthetic demo skill for the seam test.\n" +
		"---\n" +
		"\n" +
		"# aiwf-demo\n" +
		"\n" +
		"This body cites M-0001, a real id, and must fire.\n"
	skillPath := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(skill), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}

	got := check.Run(&tree.Tree{Root: root}, nil)

	var hits []check.Finding
	for _, f := range got {
		if f.Code == check.CodeSkillBodyID {
			hits = append(hits, f)
		}
	}
	if len(hits) != 1 {
		t.Fatalf("expected exactly one skill-body-id finding, got %d:\n%+v", len(hits), got)
	}
	if want := filepath.Join("internal", "skills", "embedded", "aiwf-demo", "SKILL.md"); hits[0].Path != want {
		t.Errorf("finding path = %q, want repo-relative %q", hits[0].Path, want)
	}
	if hits[0].Line != 8 {
		t.Errorf("finding line = %d, want file-relative 8", hits[0].Line)
	}
}
