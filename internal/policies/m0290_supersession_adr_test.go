package policies

// M-0290/AC-5: the retirement's ADR records which of the
// canonical-width ADR's clauses lapse, and the original points back at
// it.
//
// The supersession is clause-wise, so it cannot ride the kernel's
// supersession machinery: `aiwf promote <id> superseded --superseded-by`
// flips the original's status wholesale and writes reciprocal
// frontmatter. Doing that here would assert the canonical-width policy
// no longer holds, when four of its runtime claims are live — parser
// tolerance, allocator emit, renderer canonicalization, and the
// existence of a drift finding. The original therefore stays
// `accepted`, and the reciprocal link is prose, which is what this
// test checks for.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/entity"
)

const (
	// supersedingADR retires the width-migration verb.
	supersedingADR = "ADR-0039"
	// canonicalWidthADR is the one it supersedes clause-wise.
	canonicalWidthADR = "ADR-0008"
)

// TestM0290_AC5_SupersedingADRIsAcceptedAndNamesTheLapsedClauses reads
// both ADRs through the loader (never a hardcoded path — archive
// sweeps move files) and checks the record a reader of the original
// needs in order to tell which of its clauses still bind.
func TestM0290_AC5_SupersedingADRIsAcceptedAndNamesTheLapsedClauses(t *testing.T) {
	t.Parallel()
	root, tr := sharedRepoTree(t)

	adr := tr.ByID(supersedingADR)
	if adr == nil {
		t.Fatalf("AC-5: %s does not resolve in the planning tree", supersedingADR)
	}
	if adr.Status != entity.StatusAccepted {
		t.Errorf("AC-5: %s status = %q, want %q — a proposed ADR records no decision",
			supersedingADR, adr.Status, entity.StatusAccepted)
	}

	body := mustReadEntityBody(t, root, adr.Path)

	// The decision itself: clause-wise, not wholesale.
	for _, want := range []string{canonicalWidthADR, "clause-wise"} {
		if !strings.Contains(body, want) {
			t.Errorf("AC-5: %s body does not mention %q; the scope of the supersession is the decision",
				supersedingADR, want)
		}
	}

	// Both halves must be enumerated. A reader who is told only what
	// lapsed still cannot tell whether the rest survived by decision or
	// by omission.
	lapse := extractMarkdownSection(body, 3, "Clauses that lapse")
	if strings.TrimSpace(lapse) == "" {
		t.Fatal("AC-5: no `### Clauses that lapse` section — this test would assert nothing")
	}
	stand := extractMarkdownSection(body, 3, "Clauses that stand, unchanged")
	if strings.TrimSpace(stand) == "" {
		t.Fatal("AC-5: no `### Clauses that stand, unchanged` section — a reader cannot tell what still binds")
	}

	// Every clause named as lapsed, scoped to that section so a passing
	// mention elsewhere in the body cannot satisfy it.
	for _, clause := range []string{"Migration", "Reversal", "Drift control", "No nagging on pre-migration trees"} {
		if !strings.Contains(lapse, clause) {
			t.Errorf("AC-5: the lapsed-clause section does not name %q", clause)
		}
	}
	// And every runtime claim that survives, likewise scoped.
	for _, clause := range []string{"Parser tolerance", "Allocator behavior", "Renderer canonicalization"} {
		if !strings.Contains(stand, clause) {
			t.Errorf("AC-5: the surviving-clause section does not name %q, so the record leaves it ambiguous", clause)
		}
	}
}

// TestM0290_AC5_CanonicalWidthADRCarriesTheReciprocalLink pins the back
// reference. Without it a reader arriving at the original has no signal
// that part of it has lapsed — which is the whole failure this AC
// exists to prevent.
func TestM0290_AC5_CanonicalWidthADRCarriesTheReciprocalLink(t *testing.T) {
	t.Parallel()
	root, tr := sharedRepoTree(t)

	orig := tr.ByID(canonicalWidthADR)
	if orig == nil {
		t.Fatalf("AC-5: %s does not resolve in the planning tree", canonicalWidthADR)
	}
	// Status stays accepted: the supersession is clause-wise, and the
	// runtime claims it carries are still authoritative.
	if orig.Status != entity.StatusAccepted {
		t.Errorf("AC-5: %s status = %q, want %q — a clause-wise supersession must not retire the whole ADR",
			canonicalWidthADR, orig.Status, entity.StatusAccepted)
	}
	// Scoped to the preamble — the text before the first section
	// heading — rather than the whole body. A reference buried in
	// References at the bottom is not a notice: the failure this
	// guards against is a reader who lands on the original, reads its
	// migration section, and never learns it has lapsed.
	body := mustReadEntityBody(t, root, orig.Path)
	preamble := body
	if i := strings.Index(body, "\n## "); i >= 0 {
		preamble = body[:i]
	}
	if !strings.Contains(preamble, supersedingADR) {
		t.Errorf("AC-5: %s does not reference %s above its first section heading, so a reader of the original "+
			"can read the lapsed clauses without ever learning they lapsed", canonicalWidthADR, supersedingADR)
	}
}

// mustReadEntityBody reads an entity file whose repo-relative path came
// from the loader, so an archive sweep that moves the file is followed
// rather than hardcoded around.
func mustReadEntityBody(t *testing.T, root, relPath string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, relPath))
	if err != nil {
		t.Fatalf("reading entity at %s: %v", relPath, err)
	}
	return string(raw)
}
