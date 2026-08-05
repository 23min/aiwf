package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/check"
)

// ratifiableSovereignCodes are the finding codes whose only non-
// acknowledgment remedy is to re-run the mutation as a human — which a
// commit already in history cannot be. For these two, an operator who
// is not told about `aiwf acknowledge illegal` is told to do the one
// thing that cannot work, so the remedy has to be reachable from every
// surface that states a fix.
//
// Scoped to these two rather than to every rule an acknowledgment
// clears (which since M-0292 is the whole provenance family): the
// others name a remedy that does work, and mandating the sentence on
// all of them would be noise.
var ratifiableSovereignCodes = []string{
	check.CodeProvenanceForceNonHuman,
	check.CodeProvenanceAuditOnlyNonHuman,
}

// TestM0292_RatificationPathIsReachableFromEveryFixSurface pins the
// discoverability half of M-0292. The capability exists in the kernel;
// this asserts the two surfaces an operator actually reads when they
// meet the finding — the check hint and the aiwf-check skill's finding
// table — name it.
//
// Per CLAUDE.md §"Kernel functionality must be AI-discoverable": a
// capability reachable only by grepping source is undocumented.
func TestM0292_RatificationPathIsReachableFromEveryFixSurface(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)

	skillPath := filepath.Join(root, "internal", "skills", "embedded", "aiwf-check", "SKILL.md")
	raw, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("reading %s: %v", skillPath, err)
	}
	rows := findingTableRows(string(raw))

	for _, code := range ratifiableSovereignCodes {
		hint := check.HintFor(code, "")
		if hint == "" {
			t.Errorf("%s has no hint at all", code)
		} else if !strings.Contains(hint, "acknowledge illegal") {
			t.Errorf("the hint for %s does not name `aiwf acknowledge illegal`, so an operator "+
				"meeting it on a historical commit is pointed only at re-running the mutation, "+
				"which that commit cannot do; got: %s", code, hint)
		}

		row, ok := rows[code]
		if !ok {
			t.Errorf("the aiwf-check skill's finding table has no row for %s", code)
			continue
		}
		if !strings.Contains(row, "acknowledge illegal") {
			t.Errorf("the aiwf-check skill's row for %s does not name `aiwf acknowledge illegal`; "+
				"every other acknowledgment-clearable rule's row does, and this one's only other "+
				"remedy is impossible for a commit already in history; got: %s", code, row)
		}
	}
}

// findingTableRows maps a finding code to the markdown table row whose
// first cell names it. Keying on the leading cell rather than scanning
// for the code anywhere keeps a mention in a neighbouring row — or in
// the surrounding prose — from standing in for the row's own fix text.
func findingTableRows(doc string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(doc, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "|") {
			continue
		}
		cells := strings.Split(strings.Trim(strings.TrimSpace(line), "|"), "|")
		if len(cells) < 2 {
			continue
		}
		code := strings.Trim(strings.TrimSpace(cells[0]), "`")
		if code == "" {
			continue
		}
		out[code] = line
	}
	return out
}
