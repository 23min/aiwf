package policies

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// wf-ritual honesty / reframe tests (M-0199 / G-0309, G-0297, G-0294). Each
// pins one corrected fact in a wf-* engineering ritual. The edited skills
// live under internal/skills/embedded-rituals/**, the canonical authoring
// location, so these tests assert against the bytes the binary embeds.
//
// These are doc-shaped assertions — there is no kernel set to source-derive
// — so each is scoped to the named section (heading order or section-local
// content), never a flat body grep, per CLAUDE.md §"Substring assertions are
// not structural assertions".

const (
	wfTddCycleFixturePath   = "internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-tdd-cycle/SKILL.md"
	wfReviewCodeFixturePath = "internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-review-code/SKILL.md"
	wfDocLintFixturePath    = "internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-doc-lint/SKILL.md"
)

// headingIndexContaining returns the line index of the first markdown
// heading line whose text contains sub, or -1. Restricting the match to
// heading lines (not any line) is what makes a positional order assertion
// structural rather than a substring coincidence.
func headingIndexContaining(body, sub string) int {
	lines := strings.Split(body, "\n")
	levels := headingLevels(body)
	for i, ln := range lines {
		if levels[i] > 0 && strings.Contains(ln, sub) {
			return i
		}
	}
	return -1
}

// headingLevels returns one entry per line of section: that line's markdown
// heading level, or 0 when the line is not a heading. Lines inside a fenced
// code block are 0 whatever their leading hashes — skill bodies carry fenced
// command examples throughout, and a shell comment in one is content rather
// than a heading. Matches how extractMarkdownSection finds a section's bounds,
// so a count taken here and a section extracted there agree about the same
// bytes.
func headingLevels(section string) []int {
	lines := strings.Split(section, "\n")
	out := make([]int, len(lines))
	inFence := false
	for i, ln := range lines {
		if strings.HasPrefix(strings.TrimSpace(ln), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		out[i] = headingLevel(ln)
	}
	return out
}

// TestHeadingLevels covers the scanner every sub-heading assertion in this
// package reads through: plain headings, non-headings, and the fenced-block
// case that separates a shell comment from a heading.
func TestHeadingLevels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		section string
		want    []int
	}{
		{
			name:    "headings report their level",
			section: "## Two\n### Three\n#### Four",
			want:    []int{2, 3, 4},
		},
		{
			name:    "prose and blanks are not headings",
			section: "prose\n\n  indented",
			want:    []int{0, 0, 0},
		},
		{
			name:    "hashes inside a fence are content",
			section: "## Real\n```bash\n### not a heading\n```\n### Real",
			want:    []int{2, 0, 0, 0, 3},
		},
		{
			name:    "an unclosed fence swallows the rest",
			section: "## Real\n```\n### swallowed",
			want:    []int{2, 0, 0},
		},
		{
			name:    "an indented fence marker still toggles",
			section: "  ```\n### inside\n  ```\n### outside",
			want:    []int{0, 0, 0, 3},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if diff := cmp.Diff(tc.want, headingLevels(tc.section)); diff != "" {
				t.Errorf("headingLevels() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// countSubHeadings returns how many lines in section are markdown headings
// at exactly the given level.
func countSubHeadings(section string, level int) int {
	n := 0
	for _, lvl := range headingLevels(section) {
		if lvl == level {
			n++
		}
	}
	return n
}

// TestWfTddCycle_RecordFollowsEvidence pins AC-1 (G-0309): the RECORD step
// (which promotes the AC to `met`) is narrated after the branch-coverage
// audit and the vacuity check — the "done" judgment sits after the evidence.
func TestWfTddCycle_RecordFollowsEvidence(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfTddCycleFixturePath)

	audit := headingIndexContaining(body, "Branch-coverage audit")
	vacuity := headingIndexContaining(body, "Vacuity check")
	record := headingIndexContaining(body, "RECORD")

	if audit < 0 {
		t.Fatal("wf-tdd-cycle has no 'Branch-coverage audit' heading")
	}
	if vacuity < 0 {
		t.Fatal("wf-tdd-cycle has no 'Vacuity check' heading")
	}
	if record < 0 {
		t.Fatal("wf-tdd-cycle has no 'RECORD' heading")
	}
	if record < audit {
		t.Errorf("RECORD heading (line %d) precedes the branch-coverage audit (line %d); the done-judgment must follow the evidence (G-0309)", record, audit)
	}
	if record < vacuity {
		t.Errorf("RECORD heading (line %d) precedes the vacuity check (line %d); the done-judgment must follow the evidence (G-0309)", record, vacuity)
	}
}

// TestWfDocLint_SevenHeuristicsPlusStandaloneScan pins the "What it checks"
// section carrying exactly seven heuristics. The count is the property: the
// repo-wide path-leak scan is a distinct section, and folding it back in as an
// eighth heuristic is the drift G-0294 records.
func TestWfDocLint_SevenHeuristicsPlusStandaloneScan(t *testing.T) {
	t.Parallel()
	body := readVerbSkill(t, wfDocLintFixturePath)

	checks := sectionUnder(body, "What it checks")
	if checks == "" {
		t.Fatal("wf-doc-lint has no 'What it checks' section")
	}
	if got := countSubHeadings(checks, 3); got != 7 {
		t.Errorf("'What it checks' has %d ### sub-headings; the doc heuristics must be exactly seven (path-leak moves to its own section)", got)
	}
}
