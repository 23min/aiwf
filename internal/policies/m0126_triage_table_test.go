package policies

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// triageTablePath returns the absolute path to M-0126's deliverable
// (the docs/pocv3/ disposition table). TRIAGE.md is not an aiwf
// entity itself, but it lives alongside the parent epic's spec in
// the epic's directory — so its directory is resolved through the
// loader via the parent epic's id (E-0034), not a hardcoded slug,
// so the lookup survives an eventual archive sweep of the epic per
// PolicyNoHardcodedEntityPaths.
func triageTablePath(t *testing.T) string {
	t.Helper()
	root, tr := sharedRepoTree(t)
	e := tr.ByID("E-0034")
	if e == nil {
		t.Fatal("triage table: parent epic E-0034 not found in tree (active or archive)")
	}
	return filepath.Join(root, filepath.Dir(e.Path), "TRIAGE.md")
}

// loadTriageTable reads the triage table as a single string.
func loadTriageTable(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(triageTablePath(t))
	if err != nil {
		t.Fatalf("reading triage table: %v", err)
	}
	return string(data)
}

// triageRow is one parsed data row of the triage table.
type triageRow struct {
	file        string
	disposition string
	target      string
	rationale   string
}

// parseTriageRows extracts every data row from the table's "## Table"
// section. A data row is any line whose first cell, once trimmed and
// unbackticked, starts with "docs/pocv3/" — this excludes the header
// and separator rows without depending on their exact text. The
// table's own rows are a historical record (the executed contract);
// they always cite the old docs/pocv3/ paths as the source column,
// regardless of where docs/pocv3/ content lives today (see M-0127).
func parseTriageRows(t *testing.T, body string) []triageRow {
	t.Helper()
	section := extractMarkdownSection(body, 2, "Table")
	if section == "" {
		t.Fatal("triage table: no \"## Table\" section found")
	}

	var rows []triageRow
	for _, line := range strings.Split(section, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") {
			continue
		}
		fields := splitTableRow(trimmed)
		if len(fields) != 4 {
			continue
		}
		file := unbacktick(strings.TrimSpace(fields[0]))
		if !strings.HasPrefix(file, "docs/pocv3/") {
			continue // header or separator row
		}
		rows = append(rows, triageRow{
			file:        file,
			disposition: strings.TrimSpace(fields[1]),
			target:      strings.TrimSpace(fields[2]),
			rationale:   strings.TrimSpace(fields[3]),
		})
	}
	return rows
}

// unbacktick strips one leading and trailing backtick, if present.
func unbacktick(s string) string {
	s = strings.TrimPrefix(s, "`")
	s = strings.TrimSuffix(s, "`")
	return s
}

// closedSetDispositions is the four-value disposition vocabulary
// M-0126's spec commits to.
var closedSetDispositions = map[string]bool{
	"relocate":              true,
	"archive":               true,
	"supersede-with-entity": true,
	"delete":                true,
}

// TestM0126_AC2_EveryRowHasDispositionTargetRationale asserts M-0126's
// AC-2: every row carries a non-empty disposition (from the closed
// set), target, and rationale.
func TestM0126_AC2_EveryRowHasDispositionTargetRationale(t *testing.T) {
	t.Parallel()
	body := loadTriageTable(t)
	rows := parseTriageRows(t, body)
	if len(rows) == 0 {
		t.Fatal("AC-2: no data rows parsed from the triage table")
	}

	for _, r := range rows {
		if !closedSetDispositions[r.disposition] {
			t.Errorf("AC-2: %s has disposition %q, not one of the four closed-set values", r.file, r.disposition)
		}
		if r.target == "" {
			t.Errorf("AC-2: %s has an empty target", r.file)
		}
		if r.rationale == "" {
			t.Errorf("AC-2: %s has an empty rationale", r.file)
		}
	}
}

// entityIDPattern matches a canonical kernel entity id (e.g. G-0433,
// M-0126, ADR-0016) inside a target cell that may carry markdown
// emphasis (e.g. "**G-0433**").
var entityIDPattern = regexp.MustCompile(`[A-Z]+-\d{4}`)

// TestM0126_AC5_SupersedeRowsPairedWithEntity asserts M-0126's AC-5:
// every row marked supersede-with-entity carries a target that names
// a real entity id (one that resolves in the live tree); every row
// marked delete carries a non-empty rationale (the explicit
// justification the milestone's constraints require).
func TestM0126_AC5_SupersedeRowsPairedWithEntity(t *testing.T) {
	t.Parallel()
	_, tr := sharedRepoTree(t)
	body := loadTriageTable(t)
	rows := parseTriageRows(t, body)
	if len(rows) == 0 {
		t.Fatal("AC-5: no data rows parsed from the triage table")
	}

	var supersedeCount, deleteCount int
	for _, r := range rows {
		switch r.disposition {
		case "supersede-with-entity":
			supersedeCount++
			id := entityIDPattern.FindString(r.target)
			if id == "" {
				t.Errorf("AC-5: %s is supersede-with-entity but target %q names no entity id", r.file, r.target)
				continue
			}
			if tr.ByID(id) == nil {
				t.Errorf("AC-5: %s targets entity %s, which does not resolve in the tree", r.file, id)
			}
		case "delete":
			deleteCount++
			if r.rationale == "" {
				t.Errorf("AC-5: %s is marked delete with no explicit justification", r.file)
			}
		}
	}
	if supersedeCount == 0 && deleteCount == 0 {
		t.Log("AC-5: no supersede-with-entity or delete rows in the current table — check is vacuously satisfied")
	}
}

// TestM0126_AC4_OpenQuestion1ResolvedAndRecorded asserts M-0126's
// AC-4: Open Question #1 from E-0034 (docs/archive/ absorption) is
// resolved and recorded in a "Triage rationale" section of the
// milestone spec.
func TestM0126_AC4_OpenQuestion1ResolvedAndRecorded(t *testing.T) {
	t.Parallel()
	root, tr := sharedRepoTree(t)
	e := tr.ByID("M-0126")
	if e == nil {
		t.Fatal("AC-4: M-0126 not found in tree (active or archive)")
	}
	data, err := os.ReadFile(filepath.Join(root, e.Path))
	if err != nil {
		t.Fatalf("AC-4: reading M-0126 at %s: %v", e.Path, err)
	}

	section := extractMarkdownSection(string(data), 2, "Triage rationale")
	if section == "" {
		t.Fatal("AC-4: M-0126 has no \"## Triage rationale\" section")
	}
	if !strings.Contains(section, "Open Question #1") {
		t.Error("AC-4: \"Triage rationale\" section does not mention Open Question #1")
	}
	if !strings.Contains(section, "docs/archive/pocv3") {
		t.Error("AC-4: \"Triage rationale\" section does not record the docs/archive/pocv3 resolution")
	}
}
