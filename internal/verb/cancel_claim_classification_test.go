package verb_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/verb"
)

// M-0284/AC-4. The reproduction that motivated the claim-side seam, plus
// the arm that makes its placement load-bearing.
//
// Cancel is where the two halves of the defect are visible in one verb.
// Its already-terminal convergence and its FSM consult both read the
// loaded tree, which parses the working copy, and both sit downstream of
// the guard. The first drops a mutation; the second reports a status no
// verb wrote as the reason the operator cannot proceed. A guard placed at
// the converge point would fix only the first.

// TestCancel_TerminalStatusHandEditedOntoDisk_RefusesRatherThanConverging
// is AC-4's reproduction verbatim: a gap at `open` in HEAD, `wontfix`
// hand-edited onto disk, then cancel.
//
// This is the input the commit-side guard structurally cannot reach.
// Convergence returns before any plan exists, so verb.Apply — where every
// write-side refusal lives — is on a path nothing walks.
func TestCancel_TerminalStatusHandEditedOntoDisk_RefusesRatherThanConverging(t *testing.T) {
	t.Parallel()
	r := newGapRunner(t)
	path := dirtyEntity(t, r, "G-0001", "status: open", "status: wontfix")
	before := headSHA(t, r.root)

	res, err := verb.Cancel(r.ctx, r.tree(), "G-0001", testActor, "", false)
	assertClaimRefused(t, res, err, path)
	if after := headSHA(t, r.root); after != before {
		t.Errorf("HEAD advanced to %s on a refusal", after)
	}
}

// TestCancel_TerminalStatusCommitted_StillConverges is what makes the
// refusal above about the record rather than about the value. The same
// bytes, committed, take the converging path — so the guard is answering
// "does HEAD agree", not "is this status suspicious". Without this, a
// guard that refused every terminal status would satisfy the test above
// while ending same-state convergence for cancel (ADR-0036).
func TestCancel_TerminalStatusCommitted_StillConverges(t *testing.T) {
	t.Parallel()
	r := newGapRunner(t)
	dirtyEntity(t, r, "G-0001", "status: open", "status: wontfix")
	commitFixture(t, r.root, "fixture: the terminal status, blessed")
	before := headSHA(t, r.root)

	res, err := verb.Cancel(r.ctx, r.tree(), "G-0001", testActor, "", false)
	if err != nil {
		t.Fatalf("Cancel over a committed terminal status: %v", err)
	}
	if !res.NoOp {
		t.Fatalf("cancel did not converge on an already-terminal entity: %+v", res)
	}
	if after := headSHA(t, r.root); after != before {
		t.Errorf("a converging cancel advanced HEAD to %s", after)
	}
}

// TestCancel_UnrecognizedStatusHandEditedOntoDisk_RefusesBeforeTheFSMConsult
// pins the half AC-4 calls load-bearing rather than stylistic: not merely
// that the mutation survives, but that the classification never runs
// against disputed bytes.
//
// An unrecognized status is not terminal, so it falls past the
// already-terminal convergence and reaches the FSM consult, which refuses
// it by name. That refusal is correct about committed state and wrong
// about this one — it reports a status no verb wrote as the reason cancel
// cannot proceed, sending the operator to repair a record that is intact.
// The guard has to beat it, which a converge-point guard would not.
func TestCancel_UnrecognizedStatusHandEditedOntoDisk_RefusesBeforeTheFSMConsult(t *testing.T) {
	t.Parallel()
	r := newGapRunner(t)
	path := dirtyEntity(t, r, "G-0001", "status: open", "status: marinating")

	res, err := verb.Cancel(r.ctx, r.tree(), "G-0001", testActor, "", false)
	assertClaimRefused(t, res, err, path)
	if strings.Contains(err.Error(), "marinating") {
		t.Errorf("the refusal reports a status no verb wrote:\n%v", err)
	}
}

// TestCancel_UnrecognizedStatusCommitted_ReachesTheFSMConsult is the
// ordering control. The guard precedes the FSM consult; it does not stand
// in for it. Committed junk must still be refused on its own terms, and by
// the error that names what is wrong with it.
func TestCancel_UnrecognizedStatusCommitted_ReachesTheFSMConsult(t *testing.T) {
	t.Parallel()
	r := newGapRunner(t)
	dirtyEntity(t, r, "G-0001", "status: open", "status: marinating")
	commitFixture(t, r.root, "fixture: an unrecognized status, blessed")

	res, err := verb.Cancel(r.ctx, r.tree(), "G-0001", testActor, "", false)
	if err == nil {
		t.Fatalf("Cancel accepted an unrecognized status; got %+v", res)
	}
	var claimErr *verb.ClaimDivergenceError
	if errors.As(err, &claimErr) {
		t.Fatalf("a committed status was reported as a claim divergence: %v", err)
	}
	if !strings.Contains(err.Error(), "marinating") {
		t.Errorf("the FSM refusal does not name the offending status:\n%v", err)
	}
}
