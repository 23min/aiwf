package policies

// M-0288 AC-3: six shipped passages teach a rule by naming an id shape the
// kernel REJECTS. Rewriting them mechanically — swapping each rejected token
// for the canonical placeholder — would turn every one of them into an
// instruction to write the shape it exists to forbid, so each is rewritten to
// DESCRIBE its shape instead of spelling an instance.
//
// Both halves below are load-bearing, and neither stands alone. The
// zero-findings half alone passes for a rewrite that clears the tokens by
// deleting the instruction; the structural half alone passes for a passage
// that keeps the instruction and the offending token together. Only the pair
// says "the rule is still taught, and it no longer exhibits what it forbids".

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
)

// documentingPassage is one shipped passage whose subject is a rejected id
// shape, together with the section it must keep living in and the phrases
// that carry its instruction.
type documentingPassage struct {
	// relPath is relative to internal/skills/.
	relPath string
	// level and heading scope the structural assertion to one section, so a
	// surviving phrase elsewhere in the file cannot stand in for the passage
	// (CLAUDE.md §"Substring assertions are not structural assertions").
	level   int
	heading string
	// stopBefore truncates the section at this literal heading line. Needed
	// only where the passage sits under an H1 whose section otherwise runs to
	// EOF, swallowing later H2 sections and weakening the scope to a file grep.
	stopBefore string
}

var m0288DocumentingPassages = []documentingPassage{
	{
		relPath: "embedded/aiwf-check/SKILL.md",
		level:   2,
		heading: "Findings (errors)",
	},
	{
		relPath: "embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-plan-epic/SKILL.md",
		level:   2,
		heading: "Anti-patterns",
	},
	{
		relPath: "embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-plan-milestones/SKILL.md",
		level:   2,
		heading: "Anti-patterns",
	},
	{
		relPath:    "embedded-guidance/aiwf-guidance.md",
		level:      1,
		heading:    "aiwf — standing guidance",
		stopBefore: "## Code-health priming",
	},
	{
		relPath: "embedded/aiwf-reallocate/SKILL.md",
		level:   2,
		heading: "What to run",
	},
	{
		relPath: "embedded/aiwf-history/SKILL.md",
		level:   2,
		heading: "Composite ids and prefix matching",
	},
}

// TestM0288_AC3_DocumentingPassagesDescribeRatherThanExhibit runs both halves
// over every passage.
func TestM0288_AC3_DocumentingPassagesDescribeRatherThanExhibit(t *testing.T) {
	t.Parallel()
	skillsRoot := filepath.Join(repoRoot(t), "internal", "skills")

	for _, p := range m0288DocumentingPassages {
		t.Run(p.relPath, func(t *testing.T) {
			t.Parallel()
			raw, err := os.ReadFile(filepath.Join(skillsRoot, filepath.FromSlash(p.relPath)))
			if err != nil {
				t.Fatalf("reading shipped passage: %v", err)
			}

			// Whole-file, frontmatter included — the same input the
			// registered walker hands the rule at pre-push.
			for _, f := range check.ScanSkillBodyID(raw, p.relPath) {
				t.Errorf("line %d: %s", f.Line, f.Message)
			}

			section := extractMarkdownSection(string(raw), p.level, p.heading)
			if p.stopBefore != "" {
				// strings.SplitN returns its input unchanged when the
				// separator is absent, so a missing anchor would silently
				// widen the scope to the whole section — here, the whole
				// file. Fail loud instead, mirroring the heading guard below.
				if !strings.Contains(section, p.stopBefore) {
					t.Fatalf("stopBefore anchor %q is missing — the section scope would silently widen to the whole file", p.stopBefore)
				}
				section = strings.SplitN(section, p.stopBefore, 2)[0]
			}
			if strings.TrimSpace(section) == "" {
				t.Fatalf("section %q (level %d) is missing — the passage lost its home", p.heading, p.level)
			}
		})
	}
}
