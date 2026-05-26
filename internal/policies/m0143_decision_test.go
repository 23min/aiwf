package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestM0143_AC1_Decision is M-0143/AC-1: the envelope-representation +
// exit-code question is governed by an accepted decision (D-0013) that
// resolves via the loader, carries its named sections with non-empty
// prose, and records the chosen representation (status:error + an error
// object) and the exit-code (ExitFindings) inside its Resolution section.
//
// Per CLAUDE.md *Testing* §"Substring assertions are not structural
// assertions", the representation/exit-code literals are asserted inside
// the extracted `## Resolution` section, not flat over the whole file.
func TestM0143_AC1_Decision(t *testing.T) {
	t.Parallel()
	root, tr := sharedRepoTree(t)

	e := tr.ByID("D-0013")
	if e == nil {
		t.Fatal("AC-1: D-0013 not found in tree (active or archive)")
	}
	if e.Status != "accepted" {
		t.Errorf("AC-1: D-0013 status = %q, want accepted", e.Status)
	}

	data, err := os.ReadFile(filepath.Join(root, e.Path))
	if err != nil {
		t.Fatalf("reading D-0013 at %s: %v", e.Path, err)
	}
	body := string(data)

	for _, name := range []string{"Context", "Resolution", "Consequences"} {
		section := extractMarkdownSection(body, 2, name)
		if section == "" {
			t.Errorf("AC-1: D-0013 must have a `## %s` section", name)
			continue
		}
		if !hasNonEmptyProse(section) {
			t.Errorf("AC-1: D-0013 `## %s` section is empty / placeholder only", name)
		}
	}

	// The Resolution records the representation (status:error + an error
	// object) and the exit-code (ExitFindings) — asserted inside the
	// section, case-sensitively for the identifier literals.
	resolution := extractMarkdownSection(body, 2, "Resolution")
	for _, lit := range []string{"ExitFindings", "error", "status"} {
		if !strings.Contains(resolution, lit) {
			t.Errorf("AC-1: `## Resolution` must name %q (representation + exit-code)", lit)
		}
	}
}
