package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestRenderListRowsText_StatusColumnGetsGlyphPrefix verifies the
// G-0080 palette is applied to every row's status column. Every
// kernel status maps to a glyph; the column value becomes
// "<glyph> <status>" so a glance can identify state without parsing
// the word.
func TestRenderListRowsText_StatusColumnGetsGlyphPrefix(t *testing.T) {
	t.Parallel()
	rows := []listSummary{
		{ID: "E-0001", Status: "active", Title: "An epic", Parent: ""},
		{ID: "M-0001", Status: "in_progress", Title: "A milestone", Parent: "E-0001"},
		{ID: "M-0002", Status: "draft", Title: "Another milestone", Parent: "E-0001"},
		{ID: "M-0003", Status: "done", Title: "Done milestone", Parent: "E-0001"},
		{ID: "M-0004", Status: "cancelled", Title: "Cancelled milestone", Parent: "E-0001"},
		{ID: "G-0001", Status: "open", Title: "An open gap", Parent: ""},
		{ID: "G-0002", Status: "addressed", Title: "An addressed gap", Parent: ""},
	}
	var buf bytes.Buffer
	renderListRowsText(&buf, rows, 0, false)
	out := buf.String()
	for _, want := range []string{
		"→ active",
		"→ in_progress",
		"○ draft",
		"✓ done",
		"✗ cancelled",
		"○ open",
		"✓ addressed",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered output missing glyph-prefixed status %q:\n%s", want, out)
		}
	}
}

// TestRenderListRowsText_BoldHeaderWhenColorEnabled pins the bold
// header behavior: when colorEnabled is true the header row is wrapped
// in the ANSI bold-on / reset sequence; when false the header is plain
// text. The escape sequence is the chokepoint — body rows never gain
// ANSI escapes, so a downstream consumer parsing rows with grep stays
// unaffected.
func TestRenderListRowsText_BoldHeaderWhenColorEnabled(t *testing.T) {
	t.Parallel()
	rows := []listSummary{
		{ID: "M-0001", Status: "draft", Title: "Whatever", Parent: "E-0001"},
	}
	var bufOn bytes.Buffer
	renderListRowsText(&bufOn, rows, 0, true)
	if !strings.Contains(bufOn.String(), "\x1b[1mID") {
		t.Errorf("colorEnabled=true: header missing ANSI bold-on:\n%s", bufOn.String())
	}
	if !strings.Contains(bufOn.String(), "PARENT\x1b[0m") {
		t.Errorf("colorEnabled=true: header missing ANSI reset:\n%s", bufOn.String())
	}
	// Body rows must not carry ANSI escapes.
	rowLines := strings.Split(bufOn.String(), "\n")
	if len(rowLines) < 2 {
		t.Fatalf("expected at least 2 lines (header + 1 row), got %d", len(rowLines))
	}
	if strings.ContainsAny(rowLines[1], "\x1b") {
		t.Errorf("body row carries ANSI escape (should be clean):\n%q", rowLines[1])
	}

	var bufOff bytes.Buffer
	renderListRowsText(&bufOff, rows, 0, false)
	if strings.Contains(bufOff.String(), "\x1b") {
		t.Errorf("colorEnabled=false: output contains ANSI escape (should be plain):\n%q", bufOff.String())
	}
}

// TestRenderStatusText_BoldSectionLabels pins the bold-when-enabled
// behavior on every section label in `aiwf status`. The labels are the
// scanning anchors of the report; bolding them when stdout is a TTY
// (gated through ColorEnabled) gives a visible hierarchy without
// changing piped output.
func TestRenderStatusText_BoldSectionLabels(t *testing.T) {
	t.Parallel()
	r := statusReport{
		Date:   "2026-05-14",
		Health: statusHealthCounts{Entities: 0},
	}
	labels := []string{"In flight", "Roadmap", "Open decisions", "Open gaps", "Warnings", "Recent activity", "Health"}

	var bufOn bytes.Buffer
	if err := renderStatusText(&bufOn, &r, 0, true); err != nil {
		t.Fatalf("renderStatusText: %v", err)
	}
	for _, lbl := range labels {
		want := "\x1b[1m" + lbl + "\x1b[0m"
		if !strings.Contains(bufOn.String(), want) {
			t.Errorf("colorEnabled=true: section label %q missing ANSI bold wrap:\n%s", lbl, bufOn.String())
		}
	}

	var bufOff bytes.Buffer
	if err := renderStatusText(&bufOff, &r, 0, false); err != nil {
		t.Fatalf("renderStatusText: %v", err)
	}
	if strings.Contains(bufOff.String(), "\x1b") {
		t.Errorf("colorEnabled=false: output contains ANSI escape (should be plain):\n%q", bufOff.String())
	}
	// Labels must still appear plain.
	for _, lbl := range labels {
		if !strings.Contains(bufOff.String(), lbl) {
			t.Errorf("colorEnabled=false: section label %q missing from output:\n%s", lbl, bufOff.String())
		}
	}
}

// TestWriteStatusEpicText_MilestoneGlyphPaletteAppliesToAllStates
// verifies every milestone state from the G-0080 palette renders the
// matching marker in `aiwf status`. The existing → / ✓ markers were
// the pre-G-0080 baseline; this test pins the extended palette so a
// future change that drops a marker fails here.
func TestWriteStatusEpicText_MilestoneGlyphPaletteAppliesToAllStates(t *testing.T) {
	t.Parallel()
	e := statusEpic{
		ID:     "E-0001",
		Title:  "Test epic",
		Status: "active",
		Milestones: []statusMilestone{
			{ID: "M-0001", Title: "Draft work", Status: "draft"},
			{ID: "M-0002", Title: "In-progress work", Status: "in_progress"},
			{ID: "M-0003", Title: "Done work", Status: "done"},
			{ID: "M-0004", Title: "Cancelled work", Status: "cancelled"},
		},
	}
	var sb strings.Builder
	writeStatusEpicText(&sb, e, 0)
	out := sb.String()
	for _, want := range []string{
		" ○ M-0001",
		" → M-0002",
		" ✓ M-0003",
		" ✗ M-0004",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected milestone marker %q in output:\n%s", want, out)
		}
	}
}
