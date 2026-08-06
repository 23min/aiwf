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

// assertDocumentedAsFindingRow asserts each code is documented as a
// table ROW in a severity-declaring findings section of the aiwf-check
// skill — the structural upgrade over PolicyFindingCodesAreDiscoverable,
// which only proves a code is mentioned somewhere in the file.
//
// It deliberately does not name the section. Which one is correct is
// derived from the severity the rule emits, and
// PolicySkillTableSeverityPlacement is the single surface that decides
// it; naming a section here too would mean two places to update when a
// rule's severity changes, and one of them would eventually be wrong.
func assertDocumentedAsFindingRow(t *testing.T, root string, codes ...string) {
	t.Helper()
	rows, err := loadSkillFindingRows(root)
	if err != nil {
		t.Fatalf("read aiwf-check skill: %v", err)
	}
	for _, code := range codes {
		row, ok := rows[code]
		if !ok {
			t.Errorf("aiwf-check skill has no findings-table row for %q", code)
			continue
		}
		class, unambiguous := row.class()
		if !unambiguous || class == "" {
			t.Errorf("%q is documented under %s, which declares no single severity; it must sit in one severity-declaring findings section",
				code, row.sections())
		}
	}
}

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

	assertDocumentedAsFindingRow(t, root, "area-dead-glob", "area-overlap")

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
	for _, want := range []string{"areas.members", "paths"} {
		if !strings.Contains(note, want) {
			t.Errorf("%q note does not mention %q", noteMarker, want)
		}
	}
}

// TestAreaCoverageFinding_StructurallyDocumented pins M-0185/AC-7: the
// area-unslotted finding code is documented as a ROW in the aiwf-check skill's
// "Findings (warnings)" table (the structural upgrade over the substring-level
// finding-codes-discoverable policy), and the new areas.coverage_roots knob
// carries a schema note toward the full areas-block reference (G-0288).
func TestAreaCoverageFinding_StructurallyDocumented(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, aiwfCheckSkillPath))
	if err != nil {
		t.Fatalf("read %s: %v", aiwfCheckSkillPath, err)
	}
	body := string(data)

	// area-coverage-root-missing / area-coverage-no-paths are the M-0185
	// AC-8 misconfiguration findings.
	assertDocumentedAsFindingRow(t, root, "area-unslotted", "area-coverage-root-missing", "area-coverage-no-paths")

	// The coverage_roots schema note. Scope assertions to the note region so
	// this stays structural, not a whole-file grep. The note once carried a
	// real-id back-reference; G-0299 removed real ids from shipped skill
	// bodies, so the pins are now the config key + the finding code it drives.
	const noteMarker = "Areas `coverage_roots` schema"
	noteStart := strings.Index(body, noteMarker)
	if noteStart == -1 {
		t.Fatalf("aiwf-check skill has no %q note", noteMarker)
	}
	note := body[noteStart:]
	if end := strings.Index(note, "\n\n"); end != -1 {
		note = note[:end]
	}
	for _, want := range []string{"areas.coverage_roots", "area-unslotted"} {
		if !strings.Contains(note, want) {
			t.Errorf("%q note does not mention %q", noteMarker, want)
		}
	}
}
