package check

import (
	"testing"

	"github.com/23min/aiwf/internal/scope"
)

// isolation_escape_ack_test.go — M-0159/AC-3 red phase: pin that
// RunIsolationEscape accepts an ackedSHAs map[string]bool 4th
// parameter and silences findings whose Commit appears in it.
//
// Compile-RED today: current RunIsolationEscape has 3 params
// (commits, oracle, cherryPicked); these tests call it with 4. The
// red signal is `too many arguments in call to RunIsolationEscape`.
//
// Green phase: lift walkAcknowledgedSHAs to internal/check/acks.go,
// add the 4th param here, exempt observations whose c.SHA matches
// in the ackedSHAs map (same shape as illegalTransitionFindings'
// existing M-0136/AC-2 exemption at fsm_history_consistent.go:348).
//
// Behavioral shape mirrors M-0136/AC-2's ack-silencing on FSM
// history: per-SHA scoping (an ack for SHA-X does NOT exempt other
// escape commits — closed-set guarantee, no "exempt everything"
// knob), and a nil/empty map means "no acknowledgments" so the
// rule polices as usual (backward-compat default behavior).

// TestRunIsolationEscape_AC3_AckedSHASilencesEscape pins the
// happy path of the AC-3 lift consumption: when a commit that
// would otherwise fire isolation-escape (AI actor on non-bound
// branch with an active scope) has its SHA in the ackedSHAs map,
// the finding is silenced. The mechanical evidence that
// RunIsolationEscape is now wired to consume the lifted helper's
// output.
//
// Mirrors the canonical AC-1 happy-path fixture
// (TestIsolationEscape_AC1_AICommitOnMainFires) so the only
// behavioral delta is the ackedSHAs presence — change one thing
// at a time so a regression points at exactly the right line.
func TestRunIsolationEscape_AC3_AckedSHASilencesEscape(t *testing.T) {
	t.Parallel()

	commits := []scope.Commit{
		makeAuthorizeOpenCommit("auth0001", "E-0001", "human/peter", "ai/claude", "epic/E-0001-engine"),
		makeAICommit("c0000001", "E-0001", "ai/claude", "edit-body"),
	}
	oracle := fakeOracle{
		"auth0001": {"epic/E-0001-engine"},
		"c0000001": {"main"}, // would-be escape (AI on main, bound to ritual branch)
	}
	ackedSHAs := map[string]bool{
		"c0000001": true, // ack on the escaping commit's SHA
	}

	findings := RunIsolationEscape(commits, oracle, nil, ackedSHAs)
	if len(findings) != 0 {
		t.Fatalf("expected 0 findings (ack silences the escape per AC-3 lift consumption); got %d: %+v", len(findings), findings)
	}
}

// TestRunIsolationEscape_AC3_AckedMapWithoutEscapeSHA_StillFires
// is the positive control: an ackedSHAs map that does NOT contain
// the escape commit's SHA must NOT silence the finding. Pins the
// per-SHA closed-set guarantee — there is no "exempt everything"
// knob; the ack is scoped to the specific SHA it names.
//
// Without this control a green-phase implementation that silenced
// on "ackedSHAs is non-empty" (rather than "this commit's SHA is
// in ackedSHAs") would pass the happy-path test above and silently
// over-exempt every escape. The pair is the sabotage-verification
// hook for the AC-3 consumption.
func TestRunIsolationEscape_AC3_AckedMapWithoutEscapeSHA_StillFires(t *testing.T) {
	t.Parallel()

	commits := []scope.Commit{
		makeAuthorizeOpenCommit("auth0001", "E-0001", "human/peter", "ai/claude", "epic/E-0001-engine"),
		makeAICommit("c0000001", "E-0001", "ai/claude", "edit-body"),
	}
	oracle := fakeOracle{
		"auth0001": {"epic/E-0001-engine"},
		"c0000001": {"main"},
	}
	ackedSHAs := map[string]bool{
		"unrelated-sha-xyz": true, // ack on a different SHA — must not silence c0000001
	}

	findings := RunIsolationEscape(commits, oracle, nil, ackedSHAs)
	if len(findings) != 1 {
		t.Fatalf("expected exactly 1 finding (per-SHA scoping: ack on unrelated SHA must not exempt c0000001); got %d: %+v", len(findings), findings)
	}
	if findings[0].EntityID != "E-0001" {
		t.Errorf("EntityID = %q; want %q", findings[0].EntityID, "E-0001")
	}
}
