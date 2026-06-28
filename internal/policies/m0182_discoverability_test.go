package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// aiwfAreaSkillPath is the embedded topical area skill body (M-0182).
const aiwfAreaSkillPath = "internal/skills/embedded/aiwf-area/SKILL.md"

// readAreaSkill returns the aiwf-area skill body or fails the test.
func readAreaSkill(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, aiwfAreaSkillPath))
	if err != nil {
		t.Fatalf("read %s: %v", aiwfAreaSkillPath, err)
	}
	return string(data)
}

// TestAreaSkill_ExistsWithFrontmatter pins M-0182/AC-1: a topical aiwf-area
// skill exists and carries the `name:` / `description:` frontmatter the host's
// discovery surface needs. The broader skill-coverage policy validates the
// name-matches-directory and mentions-resolve invariants for every skill; this
// pins the specific deliverable.
func TestAreaSkill_ExistsWithFrontmatter(t *testing.T) {
	t.Parallel()
	body := readAreaSkill(t)
	if !strings.Contains(body, "name: aiwf-area") {
		t.Error("aiwf-area skill must carry `name: aiwf-area` frontmatter")
	}
	if !strings.Contains(body, "description:") {
		t.Error("aiwf-area skill must carry a non-empty `description:` frontmatter field")
	}
}

// TestAreaSkill_TeachesMentalModel pins M-0182/AC-2: the skill installs the
// operate-everywhere-but-aiwf-constrains mental model in a dedicated section
// that names the constraints — the closed member set, `areas.required`, and the
// mistag check. Structural (section-scoped), not a flat grep.
func TestAreaSkill_TeachesMentalModel(t *testing.T) {
	t.Parallel()
	body := readAreaSkill(t)
	sec := markdownSection(body, "## The mental model")
	if sec == "" {
		t.Fatal("aiwf-area skill has no `## The mental model` section (M-0182/AC-2)")
	}
	for _, want := range []string{"areas.required", "members", "mistag"} {
		if !strings.Contains(sec, want) {
			t.Errorf("mental-model section should name %q (a constraint aiwf enforces):\n%s", want, sec)
		}
	}
	low := strings.ToLower(sec)
	if !strings.Contains(low, "everywhere") && !strings.Contains(low, "anywhere") {
		t.Errorf("mental-model section should state you operate everywhere/anywhere in code:\n%s", sec)
	}
}

// TestAreaSkill_TeachesLifecycle pins M-0182/AC-3: the skill ties the area
// lifecycle together in a dedicated section that names each verb — choosing at
// `aiwf add` (incl. `--path-hint`), remediating with `aiwf set-area`,
// verification by mistag, and the `aiwf acknowledge` escape. The skill-coverage
// body-resolution rule separately guarantees every `aiwf <verb>` mention is a
// real verb.
func TestAreaSkill_TeachesLifecycle(t *testing.T) {
	t.Parallel()
	body := readAreaSkill(t)
	sec := markdownSection(body, "## The area lifecycle")
	if sec == "" {
		t.Fatal("aiwf-area skill has no `## The area lifecycle` section (M-0182/AC-3)")
	}
	for _, want := range []string{"aiwf add", "--path-hint", "aiwf set-area", "mistag", "aiwf acknowledge"} {
		if !strings.Contains(sec, want) {
			t.Errorf("lifecycle section should name %q:\n%s", want, sec)
		}
	}
}
