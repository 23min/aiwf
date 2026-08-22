package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Shared access to the embedded skill and ritual bodies. These are the
// canonical authoring locations — the snapshot the binary ships (ADR-0016) —
// so a test reads the embedded bytes directly rather than a duplicated fixture
// under testdata (G-0182).

const (
	aiwfContractSkillPath  = "internal/skills/embedded/aiwf-contract/SKILL.md"
	aiwfAuthorizeSkillPath = "internal/skills/embedded/aiwf-authorize/SKILL.md"
	aiwfCheckSkillPath     = "internal/skills/embedded/aiwf-check/SKILL.md"
	aiwfAddSkillPath       = "internal/skills/embedded/aiwf-add/SKILL.md"
	aiwfShowSkillPath      = "internal/skills/embedded/aiwf-show/SKILL.md"

	aiwfxRecordDecisionFixturePath = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-record-decision/SKILL.md"
	aiwfxWrapMilestoneFixturePath  = "internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-milestone/SKILL.md"
)

// readVerbSkill reads a skill body relative to the repo root.
func readVerbSkill(t *testing.T, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}

// headingLevel returns the number of leading '#' on a markdown heading line,
// or 0 if the line is not a heading.
func headingLevel(ln string) int {
	if !strings.HasPrefix(ln, "#") {
		return 0
	}
	return len(ln) - len(strings.TrimLeft(ln, "#"))
}

// sectionUnder returns the body text from the first heading containing
// headingSub up to (not including) the next heading of the same-or-shallower
// level. It scopes an assertion to one section, so a fact in an unrelated
// section does not satisfy it.
//
// It is not markdown-code-fence-aware: a `#`-prefixed comment line inside a
// fenced block reads as a heading and truncates the returned section early.
func sectionUnder(body, headingSub string) string {
	lines := strings.Split(body, "\n")
	start, level := -1, 0
	for i, ln := range lines {
		if headingLevel(ln) > 0 && strings.Contains(ln, headingSub) {
			start, level = i, headingLevel(ln)
			break
		}
	}
	if start < 0 {
		return ""
	}
	var b strings.Builder
	for i := start + 1; i < len(lines); i++ {
		if l := headingLevel(lines[i]); l > 0 && l <= level {
			break
		}
		b.WriteString(lines[i])
		b.WriteString("\n")
	}
	return b.String()
}
