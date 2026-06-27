package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// aiwfCheckSkillPath is the embedded aiwf-check verb-skill body — the finding
// catalog the aiwf binary materializes into .claude/skills/aiwf-check/. Per
// G-0182, AC content assertions read the embedded bytes directly rather than a
// duplicated fixture.
const aiwfCheckSkillPath = "internal/skills/embedded/aiwf-check/SKILL.md"

// TestAreaPathFindings_StructurallyDocumented pins M-0180/AC-6: the two
// path-axis finding codes are documented as ROWS in the aiwf-check skill's
// "Findings (warnings)" table — the structural upgrade over
// PolicyFindingCodesAreDiscoverable, which only proves a code is mentioned
// somewhere — and the now-observable `paths` schema carries a note toward the
// full areas-schema reference (G-0288).
func TestAreaPathFindings_StructurallyDocumented(t *testing.T) {
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
	// Guard the scoping itself, so the row assertions below cannot pass
	// vacuously: if markdownSection ever regressed to return the whole file,
	// the extracted slice would contain the NEXT section's heading and "in the
	// warnings section" would collapse to "anywhere in the file". This makes
	// the structural claim self-verifying rather than assumed.
	if strings.Contains(warnings, "## Provenance findings") {
		t.Fatal("warnings section over-extends past `## Provenance findings`: markdownSection scoping regressed, so the table-row assertions below would be vacuous")
	}
	// Structural: each code must be the leading cell of a table row INSIDE the
	// warnings section, not merely text somewhere in the file.
	for _, code := range []string{"area-dead-glob", "area-overlap"} {
		row := "| `" + code + "` |"
		if !strings.Contains(warnings, row) {
			t.Errorf("aiwf-check `Findings (warnings)` section has no table row for %q (looked for %q)", code, row)
		}
	}

	// The now-observable `paths` schema note (toward G-0288). Scope the
	// schema-field and forward-reference assertions to the note region so this
	// stays structural, not a whole-file grep.
	const noteMarker = "Areas `paths` schema"
	noteStart := strings.Index(body, noteMarker)
	if noteStart == -1 {
		t.Fatalf("aiwf-check skill has no %q note (M-0180/AC-6, toward G-0288)", noteMarker)
	}
	note := body[noteStart:]
	if end := strings.Index(note, "\n\n"); end != -1 {
		note = note[:end]
	}
	for _, want := range []string{"areas.members", "paths", "G-0288"} {
		if !strings.Contains(note, want) {
			t.Errorf("%q note does not mention %q", noteMarker, want)
		}
	}
}
