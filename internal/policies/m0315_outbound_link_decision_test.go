package policies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

// outboundLinkExtendingADR is the decision record M-0315/AC-2 requires:
// the one that settles whether ADR-0033's path-link commitment reaches a
// moved entity's own outbound links.
const outboundLinkExtendingADR = "ADR-0046"

// TestM0315_AC2_OutboundLinkDecisionReachableFromADR0033 asserts the
// relationship M-0315/AC-2 asks for, in both directions that make it
// real: the extending decision resolves to an accepted ADR, and
// ADR-0033's own References section names it.
//
// Retires when either ADR reaches a terminal status: `accepted →
// superseded` is a legal transition, and a superseding record carries the
// commitment forward, so this test is what the supersession deletes
// rather than something it has to keep satisfying.
//
// A relationship check rather than a phrase assertion: it compares two
// artefacts, so deleting the record, un-accepting it, or dropping the
// citation from ADR-0033 each turn it red, while rewording either
// document's prose does not. The AC's bar is the record "being reachable
// from ADR-0033" — a test that only confirmed the record exists would
// leave the reachability half unpinned, since `body-prose-id` fires on a
// dangling id but never on a missing one.
func TestM0315_AC2_OutboundLinkDecisionReachableFromADR0033(t *testing.T) {
	t.Parallel()

	root, tr := sharedRepoTree(t)

	extending := tr.ByID(outboundLinkExtendingADR)
	if extending == nil {
		t.Fatalf("%s does not resolve via tr.ByID — M-0315/AC-2 requires the decision record to exist", outboundLinkExtendingADR)
	}
	if extending.Kind != entity.KindADR {
		t.Errorf("%s resolves to a %s entity, want an adr", outboundLinkExtendingADR, extending.Kind)
	}
	// An extension that is still proposed has not settled anything, which
	// is what the AC asks the record to do.
	if extending.Status != entity.StatusAccepted {
		t.Errorf("%s has status %q, want %q — a decision that settles the question is accepted, not pending",
			outboundLinkExtendingADR, extending.Status, entity.StatusAccepted)
	}

	base := tr.ByID("ADR-0033")
	if base == nil {
		t.Fatal("ADR-0033 does not resolve via tr.ByID")
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(base.Path)))
	if err != nil {
		t.Fatalf("reading ADR-0033 at %s: %v", base.Path, err)
	}
	refs := sectionBody(string(raw), "\n## References")
	if refs == "" {
		t.Fatalf("ADR-0033 has no `## References` section to carry the citation")
	}
	if !strings.Contains(refs, outboundLinkExtendingADR) {
		t.Errorf("ADR-0033's `## References` section does not name %s, so the extending decision is not reachable from the decision it extends:\n%s",
			outboundLinkExtendingADR, refs)
	}
	// ADR-0033 stays the operative record for everything else it decided;
	// the extension widens it rather than replacing it.
	if base.Status != entity.StatusAccepted {
		t.Errorf("ADR-0033 has status %q, want %q — the extension must not supersede it", base.Status, entity.StatusAccepted)
	}
}
