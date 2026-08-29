package integration

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// activeEpicUnderScopeRepo returns sovereignScopedRepo's tree with
// E-0001 already promoted to active by a human, on trunk, before the
// epic branch is cut.
//
// The scope that fixture opens is load-bearing here for the same reason
// it is there: without an active scope authorizing ai/claude, the
// allow-rule refuses a non-human actor with provenance-no-active-scope
// before the sovereign gate is ever consulted, and a test built without
// it would pass while proving nothing about the gate.
func activeEpicUnderScopeRepo(t *testing.T) (root, binDir string) {
	t.Helper()
	return sovereignScopedRepo(t, [][]string{{"promote", "E-0001", "active"}})
}

// TestSovereignEpicDone_NonHumanActor_RefusedBeforeCommit is
// M-0324/AC-1: ADR-0047 rules every edge into a terminal epic status
// sovereign, and this covers active → done, the edge promote reaches.
//
// The unmoved-HEAD assertion is the load-bearing half, and it is
// substantive only at this seam. Driving verb.Promote directly would
// never call Apply, so HEAD could not move whether the gate fired or
// not, and the assertion would pass vacuously. Through the binary a
// missing gate really does close the epic — which is the record
// ADR-0040 exists to prevent, and which an exit-code assertion alone
// cannot tell apart from a refusal.
//
// M-0001 is left non-terminal deliberately. The epic-terminal cascade
// guard would refuse this promote too, but it runs after the sovereign
// gate, so reaching the refusal with a draft child present is what
// shows which guard answered.
func TestSovereignEpicDone_NonHumanActor_RefusedBeforeCommit(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)

	root, binDir := activeEpicUnderScopeRepo(t)
	before := headSHA(t, root)

	out, err := testutil.RunBin(t, root, binDir, nil,
		"promote", "E-0001", "done", "--actor", "ai/claude", "--principal", "human/peter")

	if err == nil {
		t.Errorf("promote to done succeeded for a non-human actor; want refusal\n%s", out)
	} else if !testutil.ExitedWithCode(err, 2) {
		// A sovereign-actor refusal is a plain verb error — neither Coded
		// nor internal — so FinishVerbOutcome reports it as ExitUsage, the
		// same code authorize's own human-actor check already returns.
		t.Errorf("exited %v, want 2\n%s", err, out)
	}
	if after := headSHA(t, root); after != before {
		t.Errorf("HEAD moved %s -> %s; the epic closed before the guard refused",
			before[:8], after[:8])
	}
	if !strings.Contains(out, "sovereign act requires a human/ actor") {
		t.Errorf("refusal does not name the sovereign-act rule, so it came from some other guard\n%s", out)
	}
}

// TestSovereignEpicDone_HumanActorSucceeds is AC-1's other half: the
// gate is scoped to the actor, not to the edge. A human closes the epic
// with no flag at all, because the transition is FSM-legal and
// sovereignty is about who declares it.
//
// Without this case the refusal above is satisfied by a gate that
// refuses everyone, which would close the edge to its intended user.
func TestSovereignEpicDone_HumanActorSucceeds(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)

	root, binDir := activeEpicUnderScopeRepo(t)
	// The epic-terminal cascade guard refuses while a child milestone is
	// non-terminal. It sits behind the sovereign gate, so the refusal
	// case above never reaches it; this case has to dispose of M-0001 to
	// reach the write at all.
	if out, err := testutil.RunBin(t, root, binDir, nil,
		"cancel", "M-0001", "--reason", "not needed for this fixture"); err != nil {
		t.Fatalf("aiwf cancel M-0001: %v\n%s", err, out)
	}

	before := headSHA(t, root)
	if out, err := testutil.RunBin(t, root, binDir, nil, "promote", "E-0001", "done"); err != nil {
		t.Fatalf("a human actor must still close the epic: %v\n%s", err, out)
	}
	if after := headSHA(t, root); after == before {
		t.Error("HEAD did not move; the promote reported success without committing")
	}
}
