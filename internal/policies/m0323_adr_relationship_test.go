package policies

import (
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

// adrIDInProseRE matches an ADR id as cited in body prose. Width is left
// open rather than pinned at four digits because the kernel's parsers
// tolerate narrower legacy widths on input; comparisons below run through
// entity.Canonicalize so `ADR-47` and `ADR-0047` compare equal.
var adrIDInProseRE = regexp.MustCompile(`\bADR-\d+\b`)

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

// TestM0323_AC1_ProducedADRsResolveAndAreAccepted asserts every ADR the
// epic names under `## ADRs produced` resolves through the loader, is an
// ADR, and is at status accepted.
//
// The claim is a relationship, not a phrase: the test reads the cited id
// out of the epic rather than expecting a literal, so it fails when the
// ADR is missing, when it is some other kind, when it has not been
// ratified, and when the epic's citation drifts to an id that resolves to
// none of those things. It deliberately does not assert that the ADR
// answers M-0323's three questions well — that is content correctness
// over prose, held at review, and a phrase match would pin one reading
// that any rewording breaks.
func TestM0323_AC1_ProducedADRsResolveAndAreAccepted(t *testing.T) {
	t.Parallel()

	_, tr := sharedRepoTree(t)
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
	}
}

// openQuestionsResolutionCells returns the final cell of every data row
// in the epic's `## Open questions` table — the Resolution path column.
// The header row and the `|---|` delimiter row are dropped; a table whose
// shape stops matching that layout yields no cells and fails the caller.
func openQuestionsResolutionCells(t *testing.T, body string) []string {
	t.Helper()
	section := extractMarkdownSection(body, 2, "Open questions")
	if strings.TrimSpace(section) == "" {
		t.Fatalf("%s has no `## Open questions` section", m0323EpicID)
	}
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
	// Row 0 is the header, row 1 the `---` delimiter; the rest are data.
	if len(rows) < 3 {
		t.Fatalf("%s `## Open questions` has no data rows; section reads:\n%s", m0323EpicID, section)
	}
	var out []string
	for _, cells := range rows[2:] {
		out = append(out, cells[len(cells)-1])
	}
	return out
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
