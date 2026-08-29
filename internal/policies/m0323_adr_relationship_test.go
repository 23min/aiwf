package policies

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// m0323EpicID is the epic whose planning surface M-0323's two ACs bind
// together. Both tests below are relationship checks between two
// artefacts — the epic body and the ADR it cites — so neither hardcodes
// which ADR id is expected: the id is read out of the epic and resolved
// against the tree. A reallocate that renumbers the ADR and rewrites the
// citation keeps both green; one that rewrites only half does not.
const m0323EpicID = "E-0090"

// adrIDInProseRE matches an ADR id as cited in body prose; epicIDInProseRE
// does the same for an epic. Width is left open rather than pinned at four
// digits because the kernel's parsers tolerate narrower legacy widths on
// input; comparisons below run through entity.Canonicalize so `ADR-47` and
// `ADR-0047` compare equal.
var (
	adrIDInProseRE  = regexp.MustCompile(`\bADR-\d+\b`)
	epicIDInProseRE = regexp.MustCompile(`\bE-\d+\b`)
)

// loadM0323Epic reads the epic body by resolving its id through the
// loader, per CLAUDE.md §"Policy tests that read entity files resolve via
// the loader" — Tree.ByID spans active and archive, so the tests survive
// the archive sweep that follows the epic reaching a terminal status.
func loadM0323Epic(t *testing.T) string {
	t.Helper()
	root, tr := sharedRepoTree(t)
	e := tr.ByID(m0323EpicID)
	if e == nil {
		t.Fatalf("%s not found in tree (active or archive)", m0323EpicID)
	}
	data, err := os.ReadFile(filepath.Join(root, e.Path))
	if err != nil {
		t.Fatalf("reading %s at %s: %v", m0323EpicID, e.Path, err)
	}
	return string(data)
}

// producedADRIDs returns the canonicalized ADR ids cited in the epic's
// `## ADRs produced` section, in first-seen order. Fails the test when
// the section is absent or cites none — an epic that produced an ADR and
// does not say which has broken the relationship AC-1 asserts.
func producedADRIDs(t *testing.T, body string) []string {
	t.Helper()
	section := extractMarkdownSection(body, 2, "ADRs produced")
	if strings.TrimSpace(section) == "" {
		t.Fatalf("%s has no `## ADRs produced` section", m0323EpicID)
	}
	seen := map[string]bool{}
	var out []string
	for _, raw := range adrIDInProseRE.FindAllString(section, -1) {
		id := entity.Canonicalize(raw)
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	if len(out) == 0 {
		t.Fatalf("%s `## ADRs produced` cites no ADR id; section reads:\n%s", m0323EpicID, section)
	}
	return out
}

// TestM0323_AC1_ProducedADRsResolveAndNameTheEpic asserts every ADR the
// epic names under `## ADRs produced` resolves through the loader, is an
// ADR, is at status accepted, and names the epic back.
//
// The claim is a relationship, not a phrase: the test reads the cited id
// out of the epic rather than expecting a literal, so it fails when the
// ADR is missing, when it is some other kind, when it has not been
// ratified, and when the epic's citation drifts to an id that resolves to
// none of those things.
//
// The back-reference conjunct is what makes the pairing identify a
// specific ADR rather than any ratified one. Without it, repointing the
// epic at an unrelated accepted ADR satisfies every other clause while
// the milestone's actual deliverable goes unreferenced. It survives a
// reallocate of either id because that verb rewrites cross-references on
// both sides.
//
// It deliberately does not assert that the ADR answers M-0323's three
// questions well — that is content correctness over prose, held at
// review, and a phrase match would pin one reading that any rewording
// breaks.
func TestM0323_AC1_ProducedADRsResolveAndNameTheEpic(t *testing.T) {
	t.Parallel()

	root, tr := sharedRepoTree(t)
	body := loadM0323Epic(t)

	for _, id := range producedADRIDs(t, body) {
		e := tr.ByID(id)
		if e == nil {
			t.Errorf("%s `## ADRs produced` cites %s, which does not resolve via tr.ByID", m0323EpicID, id)
			continue
		}
		if e.Kind != entity.KindADR {
			t.Errorf("%s `## ADRs produced` cites %s, which resolves to a %s entity, expected adr",
				m0323EpicID, id, e.Kind)
			continue
		}
		if e.Status != entity.StatusAccepted {
			t.Errorf("%s cites %s, whose status is %q, expected %q",
				m0323EpicID, id, e.Status, entity.StatusAccepted)
		}
		adrBody, err := os.ReadFile(filepath.Join(root, e.Path))
		if err != nil {
			t.Errorf("reading %s at %s: %v", id, e.Path, err)
			continue
		}
		if !namesEntity(string(adrBody), epicIDInProseRE, m0323EpicID) {
			t.Errorf("%s cites %s, but %s's body never names %s — the citation is one-way, "+
				"so it identifies no particular ADR", m0323EpicID, id, id, m0323EpicID)
		}
	}
}

// namesEntity reports whether body cites want, comparing canonicalized so
// a narrower legacy width in either position still matches.
func namesEntity(body string, re *regexp.Regexp, want string) bool {
	want = entity.Canonicalize(want)
	for _, raw := range re.FindAllString(body, -1) {
		if entity.Canonicalize(raw) == want {
			return true
		}
	}
	return false
}

// errNoDelimiterRow reports a table whose second row is not the
// `|---|---|` delimiter markdown requires. It is a distinct error rather
// than a generic parse failure because it is the shape that previously
// failed open: without the check, a table missing its delimiter silently
// loses its first data row from the audit while the caller still reads as
// evidence that every row was audited.
var errNoDelimiterRow = errors.New("second table row is not a delimiter")

// errTooFewRows reports a section carrying no table, or one without a
// single data row under its header and delimiter.
var errTooFewRows = errors.New("table has too few rows")

// delimiterCellRE matches one cell of a markdown table delimiter row.
// GFM requires one or more hyphens per cell with optional leading and
// trailing colons for alignment, so `|-|-|` is as legal as `|---|---|`
// and a stricter pattern would red-flag a lawful reformat.
var delimiterCellRE = regexp.MustCompile(`^:?-+:?$`)

// parseResolutionCells returns the final cell of every data row in a
// markdown table, given the section text containing it. Row 0 is the
// header and row 1 the delimiter; both are dropped, and the delimiter is
// verified rather than assumed.
//
// Pure and error-returning so the malformed-table cases can be exercised
// against synthetic input; the *testing.T wrapper below is what the live
// assertion calls.
func parseResolutionCells(section string) ([]string, error) {
	var rows [][]string
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		var cells []string
		for _, c := range strings.Split(strings.Trim(line, "|"), "|") {
			cells = append(cells, strings.TrimSpace(c))
		}
		rows = append(rows, cells)
	}
	if len(rows) < 3 {
		return nil, fmt.Errorf("%w: got %d, need a header, a delimiter and at least one data row", errTooFewRows, len(rows))
	}
	for _, c := range rows[1] {
		if !delimiterCellRE.MatchString(c) {
			return nil, fmt.Errorf("%w: %q", errNoDelimiterRow, strings.Join(rows[1], "|"))
		}
	}
	var out []string
	for _, cells := range rows[2:] {
		out = append(out, cells[len(cells)-1])
	}
	return out, nil
}

// openQuestionsResolutionCells returns the Resolution path cell of every
// data row in the epic's `## Open questions` table.
func openQuestionsResolutionCells(t *testing.T, body string) []string {
	t.Helper()
	section := extractMarkdownSection(body, 2, "Open questions")
	if strings.TrimSpace(section) == "" {
		t.Fatalf("%s has no `## Open questions` section", m0323EpicID)
	}
	cells, err := parseResolutionCells(section)
	if err != nil {
		t.Fatalf("%s `## Open questions`: %v; section reads:\n%s", m0323EpicID, err, section)
	}
	return cells
}

// TestParseResolutionCells covers the malformed-table shapes the live
// assertion must refuse rather than silently skip. The no-delimiter case
// is the one that matters: dropping that row from a real table used to
// remove a data row from the audit with the suite still green.
func TestParseResolutionCells(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		section string
		want    []string
		wantErr error
	}{
		{
			name:    "well formed table yields one cell per data row",
			section: "| Q | B? | R |\n|---|---|---|\n| a | yes | ADR-0001 |\n| b | no | ADR-0001 |",
			want:    []string{"ADR-0001", "ADR-0001"},
		},
		{
			name:    "colon anchored delimiter is accepted",
			section: "| Q | R |\n|:---|---:|\n| a | ADR-0001 |",
			want:    []string{"ADR-0001"},
		},
		{
			name:    "single hyphen delimiter is accepted, as GFM allows",
			section: "| Q | R |\n|-|-|\n| a | ADR-0001 |",
			want:    []string{"ADR-0001"},
		},
		{
			name:    "two hyphen delimiter is accepted",
			section: "| Q | R |\n|--|--|\n| a | ADR-0001 |",
			want:    []string{"ADR-0001"},
		},
		{
			name:    "missing delimiter row is refused, not silently skipped",
			section: "| Q | B? | R |\n| a | yes | still open |\n| b | no | ADR-0001 |",
			wantErr: errNoDelimiterRow,
		},
		{
			name:    "header and delimiter with no data row is refused",
			section: "| Q | R |\n|---|---|",
			wantErr: errTooFewRows,
		},
		{
			name:    "prose with no table at all is refused",
			section: "All resolved — see the ADR.",
			wantErr: errTooFewRows,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseResolutionCells(tc.section)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("parseResolutionCells() error = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseResolutionCells() unexpected error: %v", err)
			}
			if strings.Join(got, "\x1f") != strings.Join(tc.want, "\x1f") {
				t.Errorf("parseResolutionCells() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestM0323_AC2_OpenQuestionsRouteToTheProducedADR asserts every row of
// the epic's `## Open questions` table names an ADR that `## ADRs
// produced` also names, as its resolution path.
//
// This is the second half of the relationship AC-1 opens. It fails when a
// row still points somewhere else, and — the case worth having a test for
// — when a question is added after the ADR lands and left unrouted, which
// no reading of either document on its own would catch.
func TestM0323_AC2_OpenQuestionsRouteToTheProducedADR(t *testing.T) {
	t.Parallel()

	body := loadM0323Epic(t)

	produced := map[string]bool{}
	for _, id := range producedADRIDs(t, body) {
		produced[id] = true
	}

	for i, cell := range openQuestionsResolutionCells(t, body) {
		cited := adrIDInProseRE.FindAllString(cell, -1)
		if len(cited) == 0 {
			t.Errorf("%s `## Open questions` row %d names no ADR as its resolution path: %q",
				m0323EpicID, i+1, cell)
			continue
		}
		for _, raw := range cited {
			if id := entity.Canonicalize(raw); !produced[id] {
				t.Errorf("%s `## Open questions` row %d routes to %s, which `## ADRs produced` does not name",
					m0323EpicID, i+1, id)
			}
		}
	}
}
