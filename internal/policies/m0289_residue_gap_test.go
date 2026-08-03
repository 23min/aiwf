package policies

// M-0289 AC-3: the residue this milestone declined is a tracked entity, not an
// informal intention.
//
// The sweep stopped at two documents on purpose. Without a gap saying so, the
// next reader meets three doc paths still carrying narrow ids next to a lint
// that forbids them elsewhere, and has to re-derive whether that is a scoping
// decision or an oversight. The body must therefore carry both halves: which
// paths, and why widening rather than placeholdering is the fix there.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// m0289ResidueGap is the gap AC-3 requires. Named by id because the assertion
// is that this specific record exists and says specific things.
const m0289ResidueGap = "G-0517"

func TestM0289_AC3_ResidueGapNamesItsPathsAndReason(t *testing.T) {
	t.Parallel()
	root, tr := sharedRepoTree(t)

	e := tr.ByID(m0289ResidueGap)
	if e == nil {
		t.Fatalf("%s does not resolve through the loader — the deferred residue "+
			"is an intention rather than a tracked entity", m0289ResidueGap)
	}

	raw, err := os.ReadFile(filepath.Join(root, e.Path))
	if err != nil {
		t.Fatalf("reading %s at %s: %v", m0289ResidueGap, e.Path, err)
	}
	body := string(raw)

	// The three paths the sweep declined. A gap naming only some of them
	// leaves the rest looking like an oversight.
	for _, p := range []string{"docs/design", "docs/overview.md", "docs/architecture.md"} {
		if !strings.Contains(body, p) {
			t.Errorf("%s does not name the deferred path %q", m0289ResidueGap, p)
		}
	}

	// The reason, not just the list. "Widen rather than placeholder" is what
	// distinguishes this residue from the corpus that was swept, and it is
	// the part a reader cannot reconstruct from the paths alone.
	for _, phrase := range []string{"widening", "canonical id"} {
		if !strings.Contains(body, phrase) {
			t.Errorf("%s does not explain the fix shape (%q) — without it the "+
				"deferral reads as an oversight rather than a scoping decision",
				m0289ResidueGap, phrase)
		}
	}
}
