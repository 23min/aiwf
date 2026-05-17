package gitops_test

import (
	"testing"

	"github.com/23min/aiwf/internal/gitops"
	"github.com/google/go-cmp/cmp"
)

// TestParseTrailers_IsExported pins M-0113/AC-1: the trailer parser
// is callable from outside the gitops package as gitops.ParseTrailers.
// Before M-0113 the parser was the unexported parseTrailers and this
// test would fail to compile. After the rename, this test proves the
// canonical home is established at the documented name.
func TestParseTrailers_IsExported(t *testing.T) {
	t.Parallel()
	in := "aiwf-verb: add\naiwf-entity: M-0113\naiwf-actor: human/peter\n"
	got := gitops.ParseTrailers(in)
	want := []gitops.Trailer{
		{Key: "aiwf-verb", Value: "add"},
		{Key: "aiwf-entity", Value: "M-0113"},
		{Key: "aiwf-actor", Value: "human/peter"},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("gitops.ParseTrailers mismatch (-want +got):\n%s", diff)
	}
}
