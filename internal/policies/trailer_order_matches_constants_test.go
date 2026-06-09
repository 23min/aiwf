package policies

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestPolicy_TrailerOrderMatchesConstants is the live-tree
// negative-control: the kernel's own internal/gitops/trailers.go
// must pass — every Trailer* constant is in trailerOrder and
// every identifier in trailerOrder resolves to a Trailer* constant.
//
// A regression that adds a TrailerXyz constant without appending
// to trailerOrder (or vice versa) fails this test with the
// offending identifier named in the violation's Detail.
func TestPolicy_TrailerOrderMatchesConstants(t *testing.T) {
	t.Parallel()
	vs, err := PolicyTrailerOrderMatchesConstants(repoRoot(t))
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	if len(vs) != 0 {
		t.Errorf("expected no drift between Trailer* constants and trailerOrder; got %d violations:", len(vs))
		for _, v := range vs {
			t.Errorf("  %s:%d — %s", v.File, v.Line, v.Detail)
		}
	}
}

// TestPolicy_TrailerOrderMatchesConstants_ConstMissingFromOrder
// is the positive-control for the constant-side drift direction:
// a synthetic trailers.go with a Trailer* constant absent from
// trailerOrder must fire a violation naming that constant.
func TestPolicy_TrailerOrderMatchesConstants_ConstMissingFromOrder(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "internal", "gitops", "trailers.go"), `package gitops

const (
	TrailerVerb  = "aiwf-verb"
	TrailerActor = "aiwf-actor"
	TrailerOrphan = "aiwf-orphan"
)

var trailerOrder = []string{
	TrailerVerb,
	TrailerActor,
}
`)
	vs, err := PolicyTrailerOrderMatchesConstants(root)
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	hit := false
	for _, v := range vs {
		if strings.Contains(v.Detail, "TrailerOrphan") && strings.Contains(v.Detail, "not listed in trailerOrder") {
			hit = true
			if v.Policy != "trailer-order-matches-constants" {
				t.Errorf("violation has unexpected Policy=%q", v.Policy)
			}
			if v.Line <= 0 {
				t.Errorf("violation reports non-positive line %d", v.Line)
			}
		}
	}
	if !hit {
		t.Errorf("expected violation naming TrailerOrphan as missing from trailerOrder; got %+v", vs)
	}
}

// TestPolicy_TrailerOrderMatchesConstants_PhantomInOrder is the
// positive-control for the order-side drift direction: a synthetic
// trailers.go with trailerOrder referencing an identifier that
// isn't a Trailer* constant must fire a violation naming that
// identifier.
func TestPolicy_TrailerOrderMatchesConstants_PhantomInOrder(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "internal", "gitops", "trailers.go"), `package gitops

const (
	TrailerVerb  = "aiwf-verb"
	TrailerActor = "aiwf-actor"
	// SomethingElse is not a Trailer* constant.
	SomethingElse = "something-else"
)

var trailerOrder = []string{
	TrailerVerb,
	TrailerActor,
	SomethingElse,
}
`)
	vs, err := PolicyTrailerOrderMatchesConstants(root)
	if err != nil {
		t.Fatalf("policy returned error: %v", err)
	}
	hit := false
	for _, v := range vs {
		if strings.Contains(v.Detail, "SomethingElse") && strings.Contains(v.Detail, "not a Trailer* string constant") {
			hit = true
		}
	}
	if !hit {
		t.Errorf("expected violation naming SomethingElse as phantom in trailerOrder; got %+v", vs)
	}
}
