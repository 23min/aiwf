package check

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/tree"
	"github.com/23min/aiwf/internal/trunk"
)

func TestCrossBranchIndex_GroupsByCanonicalID(t *testing.T) {
	t.Parallel()
	tr := &tree.Tree{CrossBranchHits: []trunk.RefHit{
		{Kind: entity.KindGap, ID: "G-0500", Path: "work/gaps/G-0500-bar.md", Ref: "refs/heads/sibling"},
		// Narrow-legacy (3-digit) form of the same id — must group
		// under the same canonical key as the canonical-width hit
		// above (gap ids canonicalize at 4 digits; 3 is the minimum
		// idPatterns accepts, per entity.idPatterns[KindGap]).
		{Kind: entity.KindGap, ID: "G-500", Path: "work/gaps/G-500-bar.md", Ref: "refs/remotes/origin/feature"},
		{Kind: entity.KindGap, ID: "G-0009", Path: "work/gaps/G-0009-baz.md", Ref: "refs/heads/sibling"},
	}}

	idx := crossBranchIndex(tr)
	if got := len(idx[entity.Canonicalize("G-0500")]); got != 2 {
		t.Errorf("len(idx[G-0500]) = %d, want 2 (canonical + narrow-legacy hit grouped together)", got)
	}
	if got := len(idx[entity.Canonicalize("G-0009")]); got != 1 {
		t.Errorf("len(idx[G-0009]) = %d, want 1", got)
	}
	if _, known := idx[entity.Canonicalize("G-9999")]; known {
		t.Error("idx[G-9999] should be absent — no hit carries that id")
	}
}

func TestCrossBranchIndex_NilHits_EmptyIndex(t *testing.T) {
	t.Parallel()
	idx := crossBranchIndex(&tree.Tree{})
	if len(idx) != 0 {
		t.Errorf("idx = %+v, want empty for nil CrossBranchHits", idx)
	}
}

func TestJoinRefNames_MultipleDistinctRefs(t *testing.T) {
	t.Parallel()
	got := joinRefNames([]trunk.RefHit{
		{Ref: "refs/heads/sibling"},
		{Ref: "refs/remotes/origin/feature"},
	})
	want := "refs/heads/sibling, refs/remotes/origin/feature"
	if got != want {
		t.Errorf("joinRefNames = %q, want %q", got, want)
	}
}

func TestJoinRefNames_DedupesRepeatedRef(t *testing.T) {
	// Defensive: two hits carrying the identical ref (e.g. the same id
	// appearing twice within one ref's scan) must not repeat the ref
	// name in the formatted message.
	t.Parallel()
	got := joinRefNames([]trunk.RefHit{
		{Ref: "refs/heads/sibling"},
		{Ref: "refs/heads/sibling"},
	})
	want := "refs/heads/sibling"
	if got != want {
		t.Errorf("joinRefNames = %q, want deduped %q", got, want)
	}
}

func TestJoinRefNames_Empty(t *testing.T) {
	t.Parallel()
	if got := joinRefNames(nil); got != "" {
		t.Errorf("joinRefNames(nil) = %q, want empty string", got)
	}
}
