package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// aiwfAcknowledgeSkillPath is the embedded aiwf-acknowledge verb-skill body.
const aiwfAcknowledgeSkillPath = "internal/skills/embedded/aiwf-acknowledge/SKILL.md"

// TestAcknowledgeSkill_TeachesBothSubverbs pins M-0181/AC-7: the topical
// aiwf-acknowledge skill documents BOTH subverbs — `illegal` and `mistag` —
// each under its own `## aiwf acknowledge <sub>` section that names the finding
// the subverb addresses and shows the command. Structural (markdownSection-
// scoped), not a flat substring grep, so a section in the wrong place or a
// missing subverb fails.
func TestAcknowledgeSkill_TeachesBothSubverbs(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, aiwfAcknowledgeSkillPath))
	if err != nil {
		t.Fatalf("read %s: %v", aiwfAcknowledgeSkillPath, err)
	}
	body := string(data)

	illegal := markdownSection(body, "## aiwf acknowledge illegal")
	if illegal == "" {
		t.Fatal("aiwf-acknowledge skill has no `## aiwf acknowledge illegal` section")
	}
	if !strings.Contains(illegal, "illegal-transition") {
		t.Error("the illegal section should explain the fsm-history-consistent/illegal-transition subcode it addresses")
	}
	if !strings.Contains(illegal, "aiwf acknowledge illegal") {
		t.Error("the illegal section should show the `aiwf acknowledge illegal` command")
	}

	mistag := markdownSection(body, "## aiwf acknowledge mistag")
	if mistag == "" {
		t.Fatal("aiwf-acknowledge skill has no `## aiwf acknowledge mistag` section (M-0181/AC-7: the skill must teach both subverbs)")
	}
	if !strings.Contains(mistag, "area-mistag") {
		t.Error("the mistag section should reference the area-mistag finding it suppresses")
	}
	if !strings.Contains(mistag, "aiwf acknowledge mistag") {
		t.Error("the mistag section should show the `aiwf acknowledge mistag` command")
	}
}

// TestAreaMistagFinding_StructurallyDocumented pins M-0181/AC-7: the area-mistag
// finding code is documented as a ROW in the aiwf-check skill's
// "Findings (warnings)" table (the structural upgrade over the bare
// discoverability policy), and carries a hint. Mirrors the M-0180/AC-6 pin for
// the path-axis codes; the self-guard keeps the row assertion from passing
// vacuously if markdownSection ever over-extends.
func TestAreaMistagFinding_StructurallyDocumented(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, aiwfCheckSkillPath))
	if err != nil {
		t.Fatalf("read %s: %v", aiwfCheckSkillPath, err)
	}
	body := string(data)

	warnings := markdownSection(body, "## Findings (warnings)")
	if warnings == "" {
		t.Fatal("aiwf-check skill has no `## Findings (warnings)` section")
	}
	if strings.Contains(warnings, "## Provenance findings") {
		t.Fatal("warnings section over-extends past `## Provenance findings`: markdownSection scoping regressed, so the row assertion below would be vacuous")
	}
	row := "| `area-mistag` |"
	if !strings.Contains(warnings, row) {
		t.Errorf("aiwf-check `Findings (warnings)` section has no table row for area-mistag (looked for %q)", row)
	}

	// The hint table must carry an actionable entry for the code.
	hintData, err := os.ReadFile(filepath.Join(root, "internal", "check", "hint.go"))
	if err != nil {
		t.Fatalf("read hint.go: %v", err)
	}
	if !strings.Contains(string(hintData), `"area-mistag":`) {
		t.Error("internal/check/hint.go has no hint entry for area-mistag")
	}
}
