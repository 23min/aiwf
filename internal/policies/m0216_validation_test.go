package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// loadM0216Spec resolves the M-0216 spec via the tree loader (not a
// hardcoded path) so it survives an archive sweep, per the repo's
// "policy tests must resolve via the loader" rule.
func loadM0216Spec(t *testing.T) string {
	t.Helper()
	root, tr := sharedRepoTree(t)
	e := tr.ByID("M-0216")
	if e == nil {
		t.Fatal("entity M-0216 not found in tree")
	}
	data, err := os.ReadFile(filepath.Join(root, e.Path))
	if err != nil {
		t.Fatalf("loading %s: %v", e.Path, err)
	}
	return string(data)
}

// TestM0216_AC4_ValidationRecordsDelta is the mechanical pin for
// M-0216 AC-4 ("Measured check wall-time delta recorded in
// Validation"): the spec's `## Validation` section must record the
// measured before/after wall-time figures and the byte-identical
// confirmation. Structural — scoped to §Validation, not a flat file
// grep, so the figures must live in the section a reader consults for
// the milestone's measured outcome (per the repo's "substring
// assertions are not structural assertions" rule).
func TestM0216_AC4_ValidationRecordsDelta(t *testing.T) {
	t.Parallel()
	section := extractMarkdownSection(loadM0216Spec(t), 2, "Validation")
	if section == "" {
		t.Fatal("AC-4: M-0216 spec must have a `## Validation` section")
	}
	// The measured wall-time delta: before/after figures + the metric.
	for _, m := range []string{"48.8s", "37.3s", "wall-time"} {
		if !strings.Contains(section, m) {
			t.Errorf("AC-4: §Validation must record wall-time-delta marker %q", m)
		}
	}
	// The byte-identical confirmation (the AC-3 pin, recorded alongside).
	for _, m := range []string{"byte-identical", "34 = 34"} {
		if !strings.Contains(section, m) {
			t.Errorf("AC-4: §Validation must record byte-identical marker %q", m)
		}
	}
}
