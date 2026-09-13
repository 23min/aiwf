package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// TestSectionSetCorpusRootsExist pins that every root the scan reads is
// really there. The policy is silent on an absent root so its firing fixture
// can run against a synthetic tree, which means a root renamed by a docs
// reorganization would otherwise narrow the scan with nothing said. A scan
// reporting nothing for bytes it never saw is worse than no scan, because
// the clean verdict is what stops the next reader looking.
func TestSectionSetCorpusRootsExist(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	for _, rel := range sectionSetCorpus {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Errorf("section-set corpus root %s is unreadable: %v\n"+
				"The scan skips a root it cannot find, so this one is no longer being read.", rel, err)
		}
	}
}

// TestKindStatedByRow pins the detector itself. The policy reports an
// absence, so against a clean tree it never reaches a matching row — a
// detector that matched nothing at all would satisfy it just as well. These
// cases are what separate the two.
//
// The matching row is built from entity.RequiredSections rather than spelled
// out, so it tracks the set the detector reads and cannot pin a stale shape.
func TestKindStatedByRow(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		line string
		want entity.Kind
	}{
		{"a row carrying the kind's sections names that kind", gapSectionTableRow(), entity.KindGap},
		{
			"a row keyed by a kind but carrying other content is not a restatement",
			"| gap | `what_s_missing`, `why_it_matters`, plus author-added sections |", "",
		},
		{
			"prose naming every section of a kind is not a table row",
			"**Gaps.** `## What's missing` is the defect; `## Why it matters` is the consequence.", "",
		},
		{
			"a row whose first cell names no kind is not a restatement",
			"| R-AUDIT-0085 | `## What's missing` / `## Why it matters` |", "",
		},
		{"a row too short to carry cells is not a restatement", "|", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := kindStatedByRow(tc.line)
			if ok != (tc.want != "") {
				t.Fatalf("kindStatedByRow(%q) matched = %v, want %v", tc.line, ok, tc.want != "")
			}
			if got != tc.want {
				t.Errorf("kindStatedByRow(%q) = %q, want %q", tc.line, got, tc.want)
			}
		})
	}
}

// gapSectionTableRow renders a table row restating the gap kind's section
// set, derived from the owned definition. Shared with the policy's firing
// fixture so both drive the same shape.
func gapSectionTableRow() string {
	var cells []string
	for _, section := range entity.RequiredSections(entity.KindGap) {
		cells = append(cells, "`## "+section+"`")
	}
	return "| gap | " + strings.Join(cells, ", ") + " |"
}
