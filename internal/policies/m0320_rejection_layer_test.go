package policies

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/23min/aiwf/internal/cellcoverage"
	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/render"
)

// Force bypasses the resolver guard, but the persisted gap still violates
// the resolver rule. This backstop is independent of the ordinary request's layer.
func TestM0320_AC2_ForcedGapResolutionRetainsCheckBackstop(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	f := cellcoverage.NewCellFixture(t)
	id := f.BringEntityToState(t, entity.KindGap, "open", cellcoverage.BringOpts{})
	out, err := testutil.RunBin(t, f.Root, "", nil, "promote", id, "addressed", "--force", "--reason", "exercise the resolver check backstop")
	if err != nil {
		t.Fatalf("forced resolution failed: %v\n%s", err, out)
	}
	out, err = testutil.RunBin(t, f.Root, "", nil, "check", "--format=json")
	if err != nil {
		t.Fatalf("check failed: %v\n%s", err, out)
	}
	var env render.Envelope
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("decode check output: %v\n%s", err, out)
	}
	if !slices.ContainsFunc(env.Findings, func(f check.Finding) bool {
		return f.Code == check.CodeGapAddressedHasResolver && f.EntityID == id
	}) {
		t.Fatalf("missing resolver finding for %s: %+v", id, env.Findings)
	}
}
