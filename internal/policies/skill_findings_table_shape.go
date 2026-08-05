package policies

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// skillFixClauseMarker is the inline shape remediation takes inside a
// meaning cell: a `Fix:` clause appended to the prose.
const skillFixClauseMarker = "Fix:"

// pluralCells agrees the violation message's noun with its count, since
// a one-cell row is a real input — a line of `|` with no separator.
func pluralCells(n int) string {
	if n == 1 {
		return " cell"
	}
	return " cells"
}

// markdownCells splits one markdown table row into its cells. The split
// honors `\|`, which is how a cell carries a literal pipe, so a row
// using one is not read as having an extra column.
//
// The leading and trailing pipes bound the row rather than separating
// cells, so the empty fields they produce are dropped. A line that is
// not a table row returns nil.
//
// A lone `|` is both of those pipes at once. It is a truncated row
// rather than a row of no cells, so it reads as one empty cell and
// reaches the caller's shape check like any other malformed row.
func markdownCells(line string) []string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
		return nil
	}
	if len(line) < 2 {
		return []string{""}
	}
	var cells []string
	var cur strings.Builder
	escaped := false
	for _, r := range line[1 : len(line)-1] {
		switch {
		case escaped:
			cur.WriteRune(r)
			escaped = false
		case r == '\\':
			cur.WriteRune(r)
			escaped = true
		case r == '|':
			cells = append(cells, cur.String())
			cur.Reset()
		default:
			cur.WriteRune(r)
		}
	}
	return append(cells, cur.String())
}

// PolicySkillFindingsTableShape asserts the aiwf-check skill's findings
// tables document what a code means and leave what to do about it to
// the hint the tool prints.
//
// `applyHints` fills every finding's Hint from hintTable at emission
// time, and two policies hold that surface to its job:
// PolicyFindingCodesHaveHints requires every emitted code to have one,
// and the finding-hints-name-command check requires each to name the
// remediation command. A fix column in the skill is therefore a second
// copy of guidance the operator already has on screen — one that no
// check re-derives, that ships into every consumer repo, and that a
// reader weighs as authoritative while it drifts.
//
// So a findings-table row carries exactly two cells, and no cell
// carries an inline `Fix:` clause — the two shapes a remediation copy
// takes. Both are mechanical; what a meaning cell should say is not,
// and is held at review like the rest of the skill's prose.
//
// Scope is a section whose heading declares a severity, which is the
// same classifier PolicySkillTableSeverityPlacement routes rows by, so
// the two policies agree on what counts as a findings table. Tables
// elsewhere in the skill are free to have any shape — the hook table
// under "What to run" is three columns and the id-rule matrix is five.
// A heading renamed until it declares nothing takes its rows out of
// this policy's scope, but that rename is itself a violation of
// PolicySkillTableSeverityPlacement, so it does not go unreported.
//
// The property is an aiwf-repo authoring invariant: the skill is
// materialized rather than authored in a consumer tree, and the hint
// table it defers to lives in internal/check. So it is a CI-tier policy
// rather than an aiwf check finding.
func PolicySkillFindingsTableShape(root string) ([]Violation, error) {
	data, err := os.ReadFile(filepath.Join(root, skillCheckPath))
	if err != nil {
		return nil, err
	}

	var out []Violation
	heading := ""
	for i, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "## ") {
			heading = strings.TrimSpace(line)
		}
		if skillSectionClass(heading) == "" {
			continue
		}
		cells := markdownCells(line)
		if cells == nil {
			continue
		}
		if len(cells) != 2 {
			out = append(out, Violation{
				Policy: "skill-findings-table-shape",
				File:   skillCheckPath,
				Line:   i + 1,
				Detail: "row under " + strconv.Quote(heading) + " has " + strconv.Itoa(len(cells)) +
					pluralCells(len(cells)) + "; a findings table is `Code | Meaning` — remediation " +
					"belongs in hintTable (internal/check/hint.go), which the tool prints with the finding",
			})
			continue
		}
		if strings.Contains(cells[1], skillFixClauseMarker) {
			out = append(out, Violation{
				Policy: "skill-findings-table-shape",
				File:   skillCheckPath,
				Line:   i + 1,
				Detail: "meaning cell under " + strconv.Quote(heading) + " carries an inline " +
					strconv.Quote(skillFixClauseMarker) + " clause; remediation belongs in hintTable " +
					"(internal/check/hint.go), which the tool prints with the finding",
			})
		}
	}
	return out, nil
}
