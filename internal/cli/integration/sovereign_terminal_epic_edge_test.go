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
// The scope the shared fixture opens is incidental to the cases below,
// not load-bearing: the unforced gate returns before a plan exists, so
// it is reached with any actor and no scope at all. The allow-rule that
// refuses an unauthorized non-human actor runs at the apply seam, past
// this refusal. The scope is load-bearing for sovereignForceRepo's own
// `--force` cases precisely because force skips the gate and leaves the
// allow-rule as what answers.
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

// TestSovereignEpicCancel_NonHumanActor_RefusedAtTheVerb is
// M-0324/AC-2: the two cancel edges ADR-0047 rules sovereign, reached
// through `aiwf cancel` rather than through promote.
//
// The distinguishing assertion is *where* the refusal comes from. The
// history audit is transition-shaped rather than verb-shaped, so it
// already observes a cancel — meaning a closed-set entry with no call
// site in cancel would let the act land at exit 0 and fail the next
// push instead of being refused. Unmoved HEAD is what tells those two
// apart, and it is why this runs through the binary.
//
// The refusal must also name `aiwf cancel`. The gate's message is the
// operator's instruction, and one naming a verb they did not run sends
// them to a different command than the one that refused them.
func TestSovereignEpicCancel_NonHumanActor_RefusedAtTheVerb(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)

	cases := []struct {
		name string
		// from names the epic status the cancel is attempted from;
		// both are sovereign per ADR-0047, which argues that
		// `cancelled` is terminal whatever state it is reached from.
		from string
		repo func(*testing.T) (string, string)
	}{
		{name: "from active", from: "active", repo: activeEpicUnderScopeRepo},
		{name: "from proposed", from: "proposed", repo: sovereignForceRepo},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root, binDir := tc.repo(t)
			before := headSHA(t, root)

			out, err := testutil.RunBin(t, root, binDir, nil,
				"cancel", "E-0001", "--actor", "ai/claude", "--principal", "human/peter")

			if err == nil {
				t.Errorf("cancel from %s succeeded for a non-human actor; want refusal\n%s", tc.from, out)
			} else if !testutil.ExitedWithCode(err, 2) {
				t.Errorf("cancel from %s exited %v, want 2\n%s", tc.from, err, out)
			}
			if after := headSHA(t, root); after != before {
				t.Errorf("cancel from %s moved HEAD %s -> %s; the act landed and the audit would "+
					"report it after the fact, which is the record the verb gate exists to prevent",
					tc.from, before[:8], after[:8])
			}
			if !strings.Contains(out, "sovereign act requires a human/ actor") {
				t.Errorf("cancel from %s: refusal did not come from the sovereign gate\n%s", tc.from, out)
			}
			// Asserted as a negative. The CLI prefixes every line of
			// this verb's output with "aiwf cancel:", so a positive
			// check for that string passes on the label alone and would
			// never see the gate's own message naming promote.
			if strings.Contains(out, "aiwf promote") {
				t.Errorf("cancel from %s: refusal names `aiwf promote`, a verb the operator did not "+
					"run, sending them to a different command than the one that refused them\n%s", tc.from, out)
			}
		})
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
