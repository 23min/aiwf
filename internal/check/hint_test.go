package check

import (
	"strings"
	"testing"
)

// TestHint_AreaRequired pins M-0178/AC-6 (hint half): the area-required
// finding carries a remediation hint pointing operators at `aiwf set-area`
// (the M-0183 tag verb). Removing the hint entry reddens this test (and
// the PolicyFindingCodesHaveHints chokepoint).
func TestHint_AreaRequired(t *testing.T) {
	t.Parallel()
	h := HintFor(CodeAreaRequired, "")
	if h == "" {
		t.Fatal("expected a hint for area-required, got empty")
	}
	if !strings.Contains(h, "set-area") {
		t.Errorf("hint %q should point at `aiwf set-area`", h)
	}
}
