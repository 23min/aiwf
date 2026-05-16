package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestComputeTitleBudget pins the title-column sizing math for
// `aiwf list`. Closes G-0080's first chokepoint: when the terminal is
// too narrow to fit a row's natural width, the title column is the
// flex; id/status/parent are fixed by content.
//
// Two-axis: nil renderedStatuses falls back to rows[i].Status (the
// pre-glyph behavior — preserved so callers that don't render glyphs
// don't pay for them); an explicit slice widens statusW by the glyph
// prefix runes (the production path through renderListRowsText).
func TestComputeTitleBudget(t *testing.T) {
	t.Parallel()
	rows := []listSummary{
		{ID: "M-0001", Status: "in_progress", Title: "A very long milestone title that exceeds normal width", Parent: "E-0001"},
		{ID: "M-0002", Status: "draft", Title: "Short", Parent: "E-0001"},
	}
	// Without glyph prefixes: idW=6, statusW=11 (in_progress),
	// parentW=6, titleW=54. Natural row width: 6+2+11+2+54+2+6 = 83.
	// With glyph prefixes (each 2 runes wider): statusW=13, natural 85.
	withGlyphs := []string{"→ in_progress", "○ draft"}
	tests := []struct {
		name             string
		rows             []listSummary
		renderedStatuses []string
		termWidth        int
		want             int
	}{
		{"no termWidth disables truncation", rows, nil, 0, 0},
		{"termWidth zero rows returns 0", nil, nil, 80, 0},
		{"wide enough returns 0 (no truncation needed)", rows, nil, 200, 0},
		// At width=83 the row fits exactly — no truncation.
		{"exact-fit returns 0", rows, nil, 83, 0},
		// At width=60: budget = 60 - (6+11+6+6) = 31 runes — above floor.
		{"narrow terminal returns positive budget", rows, nil, 60, 31},
		// At width=30: budget = 30 - 29 = 1, below floor — return 0 so
		// the row falls back to terminal-wrap rather than collapsing.
		{"below floor returns 0", rows, nil, 30, 0},

		// With-glyphs path — statusW gains 2 runes per row's prefix.
		// At width=60: budget = 60 - (6+13+6+6) = 29 (still above floor).
		{"with glyphs widens status, narrows budget", rows, withGlyphs, 60, 29},
		// Glyph-aware exact-fit: 85 columns.
		{"with glyphs exact-fit returns 0", rows, withGlyphs, 85, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := computeTitleBudget(tt.rows, tt.renderedStatuses, tt.termWidth)
			if got != tt.want {
				t.Errorf("computeTitleBudget(%d) = %d, want %d", tt.termWidth, got, tt.want)
			}
		})
	}
}

// TestRenderListRowsText_TruncatesTitleWhenNarrow exercises the
// happy-path TTY-narrow case: a row whose natural width exceeds
// termWidth has its title truncated with the "…" suffix, and the id /
// status / parent columns stay intact. The non-TTY default path (no
// truncation) is what every other list test already exercises.
func TestRenderListRowsText_TruncatesTitleWhenNarrow(t *testing.T) {
	t.Parallel()
	rows := []listSummary{
		{ID: "M-0001", Status: "in_progress", Title: "A very long milestone title that should not wrap mid-row", Parent: "E-0001"},
	}
	var buf bytes.Buffer
	// termWidth=50, statusW with glyph prefix = 13 ("→ in_progress").
	// budget = 50 - (6+13+6+6) = 19 runes.
	renderListRowsText(&buf, rows, 50, false)
	out := buf.String()
	// The id, status (substring), glyph, and parent must appear.
	for _, want := range []string{"M-0001", "in_progress", "→", "E-0001"} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered output missing %q:\n%s", want, out)
		}
	}
	// The original full title must not appear.
	if strings.Contains(out, "A very long milestone title that should not wrap mid-row") {
		t.Errorf("rendered output contains untruncated title:\n%s", out)
	}
	// The ellipsis must appear — that's the truncation marker.
	if !strings.Contains(out, "…") {
		t.Errorf("rendered output missing ellipsis '…' (truncation marker):\n%s", out)
	}
	// Truncate keeps 18 runes + "…": "A very long milest…" (18+1=19).
	if !strings.Contains(out, "A very long milest…") {
		t.Errorf("rendered output missing expected truncated prefix:\n%s", out)
	}
}

// TestRenderListRowsText_NoTruncationOnZeroBudget is the regression
// guard for the non-TTY path. Passing termWidth=0 must reproduce the
// existing rendering byte-for-byte (no "…" inserted), which is what
// every golden test and pipeline consumer depends on.
func TestRenderListRowsText_NoTruncationOnZeroBudget(t *testing.T) {
	t.Parallel()
	rows := []listSummary{
		{ID: "M-0001", Status: "draft", Title: "Whatever-length title here", Parent: "E-0001"},
	}
	var buf bytes.Buffer
	renderListRowsText(&buf, rows, 0, false)
	out := buf.String()
	if !strings.Contains(out, "Whatever-length title here") {
		t.Errorf("zero-budget render dropped or altered title:\n%s", out)
	}
	if strings.Contains(out, "…") {
		t.Errorf("zero-budget render contains ellipsis (should be passthrough):\n%s", out)
	}
}

// TestTruncStatusTitle pins the per-line title-cap helper used by
// renderStatusText for milestone/gap/decision/epic title columns.
func TestTruncStatusTitle(t *testing.T) {
	t.Parallel()
	const prefix = "    →M-0001 — "  // 14 runes
	const tail = "    [in_progress]" // 17 runes
	const longTitle = "A title long enough to overflow"
	tests := []struct {
		name      string
		title     string
		termWidth int
		prefix    string
		tail      string
		want      string
	}{
		{"zero termWidth passthrough", longTitle, 0, prefix, tail, longTitle},
		// 60 - 14 - 17 = 29 available runes; title is 31 runes so it
		// gets truncated.
		{"narrow terminal truncates", longTitle, 60, prefix, tail, "A title long enough to overf…"},
		// 50 - 14 - 17 = 19 available, above the minTitleColumnRunes
		// floor (10) but tight. Keep 18 runes, append "…".
		{"floor-respected with tight room", longTitle, 50, prefix, tail, "A title long enoug…"},
		// 35 - 14 - 17 = 4 — below floor, return original (terminal wraps).
		{"below floor returns original", longTitle, 35, prefix, tail, longTitle},
		// Title already short enough.
		{"short title untouched", "fits", 200, prefix, tail, "fits"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := truncStatusTitle(tt.title, tt.termWidth, tt.prefix, tt.tail)
			if got != tt.want {
				t.Errorf("truncStatusTitle(%q, %d) = %q, want %q",
					tt.title, tt.termWidth, got, tt.want)
			}
		})
	}
}

// TestRenderStatusText_TruncatesAllTitleColumnsWhenNarrow exercises
// the verb-level seam: a non-zero termWidth must reach milestone, gap,
// and decision title columns alike. A single test pinning all three at
// once is the cheapest way to catch a missed call site.
func TestRenderStatusText_TruncatesAllTitleColumnsWhenNarrow(t *testing.T) {
	t.Parallel()
	r := statusReport{
		Date: "2026-05-14",
		InFlightEpics: []statusEpic{{
			ID:     "E-0001",
			Title:  "An epic title that is definitely long enough to overflow the eighty-column terminal width",
			Status: "active",
			Milestones: []statusMilestone{{
				ID:     "M-0001",
				Title:  "A milestone title that is definitely long enough to overflow the eighty-column terminal width",
				Status: "in_progress",
				TDD:    "required",
			}},
		}},
		OpenDecisions: []statusEntity{{
			ID:     "ADR-0001",
			Title:  "A decision title that is definitely long enough to overflow the eighty-column terminal width",
			Status: "proposed",
			Kind:   "adr",
		}},
		OpenGaps: []statusGap{{
			ID:           "G-0001",
			Title:        "A gap title that is definitely long enough to overflow the eighty-column terminal width",
			DiscoveredIn: "M-0001",
		}},
		Health: statusHealthCounts{Entities: 4},
	}
	var buf bytes.Buffer
	if err := renderStatusText(&buf, &r, 80, false); err != nil {
		t.Fatalf("renderStatusText: %v", err)
	}
	out := buf.String()

	// Each title kind must have been truncated (its full text dropped,
	// an ellipsis present somewhere on its row). The cheapest assertion
	// is "ellipsis appears alongside the entity id" — if the verb
	// missed a code path, the ellipsis won't be there on that line.
	for _, ent := range []struct {
		id, fullTitle string
	}{
		{"E-0001", "An epic title that is definitely long enough to overflow the eighty-column terminal width"},
		{"M-0001", "A milestone title that is definitely long enough to overflow the eighty-column terminal width"},
		{"ADR-0001", "A decision title that is definitely long enough to overflow the eighty-column terminal width"},
		{"G-0001", "A gap title that is definitely long enough to overflow the eighty-column terminal width"},
	} {
		if strings.Contains(out, ent.fullTitle) {
			t.Errorf("%s: full title appeared (truncation missed):\n%s", ent.id, out)
		}
		if !lineContaining(out, ent.id, "…") {
			t.Errorf("%s: no ellipsis found on its row (truncation missed):\n%s", ent.id, out)
		}
	}

	// Each row must stay within the termWidth budget. Without
	// truncation the test fixture's lines exceed 80 columns; with
	// truncation they must all fit.
	for _, line := range strings.Split(out, "\n") {
		if runeCount(line) > 80 {
			t.Errorf("row exceeds termWidth=80: %d runes\n%q", runeCount(line), line)
		}
	}
}

// lineContaining returns true when a line of out contains both subA and
// subB. Cheap two-substring co-occurrence check used by the status
// truncation test to assert "the ellipsis is on the line carrying this
// id" without parsing the full line shape.
func lineContaining(out, subA, subB string) bool {
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, subA) && strings.Contains(line, subB) {
			return true
		}
	}
	return false
}

// runeCount returns the count of runes (not bytes) in s — used so the
// row-width assertion above measures display columns, not utf8 bytes.
func runeCount(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}
