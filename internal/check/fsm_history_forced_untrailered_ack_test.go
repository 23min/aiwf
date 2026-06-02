package check

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
)

// fsm_history_forced_untrailered_ack_test.go — M-0159/AC-4 red
// phase: pin that forcedUntraileredFindings accepts an
// ackedSHAs map[string]bool 2nd parameter and silences findings
// whose Commit SHA appears in it.
//
// Compile-RED today: the current signature is
// forcedUntraileredFindings(observations) — 1 param; these tests
// call it with 2 args. Red signal is `too many arguments in
// call to forcedUntraileredFindings`.
//
// Green-phase scope: extend the predicate's signature to accept
// ackedSHAs (mirroring AC-3's lift extension to
// illegalTransitionFindings); add the `if ackedSHAs[o.Commit]
// { continue }` check inside the per-observation loop. The
// CLI gather layer already computes ackedSHAs once and the
// surrounding FSMHistoryConsistent / fsmHistoryConsistentWithDeps
// receive it; AC-4 GREEN just forwards the existing local value
// to the forced-untrailered predicate the same way M-0136/AC-2
// did for illegal-transition.
//
// AC-4's load-bearing claim per the spec body:
//
//	"acknowledge-illegal extended to cover isolation-escape AND
//	 forced-untrailered subcodes via the shared helper. Real-git
//	 E2E: AI escape → aiwf acknowledge-illegal <sha> --reason →
//	 aiwf check silent; AI authorship preserved on original
//	 commit."
//
// G-0214 documents the asymmetry these tests close: today,
// acknowledge-illegal silences illegal-transition but NOT
// forced-untrailered (the same shape, different rule branch);
// the verb's docstring promises silencing without naming the
// subcode, and the asymmetry has bitten at least one real
// consumer. AC-4 closes the gap at the rule level + writes the
// E2E pin in `internal/cli/integration/branch_scenarios_ac4_test.go`.

// TestForcedUntraileredFindings_AC4_AckedSHASilences pins the
// happy path: a sovereign-act-shape transition by a non-human
// actor without aiwf-force normally fires forced-untrailered;
// when that commit's SHA appears in ackedSHAs (i.e., a current-
// day `aiwf-force-for: <sha>` trailer exists in HEAD's
// reachable history), the finding is silenced. Mirrors the
// per-SHA closed-set scoping the M-0136/AC-2
// illegal-transition exemption uses.
//
// The fixture is the canonical positive case from
// TestForcedUntraileredFindings_FiresOnSovereignActByNonHumanWithoutForce
// (epic proposed → active by ai/claude with no aiwf-force) plus
// an ackedSHAs map naming the commit's SHA. Only the
// ackedSHAs presence differs from the baseline positive test,
// so a regression points at exactly the right line.
func TestForcedUntraileredFindings_AC4_AckedSHASilences(t *testing.T) {
	t.Parallel()
	obs := []statusChange{
		{
			EntityID:   "E-0001",
			EntityKind: entity.KindEpic,
			Commit:     "abc1234567890def",
			Parent:     "0000000000000000",
			Path:       "work/epics/E-0001-x/epic.md",
			Prior:      entity.StatusProposed,
			Next:       entity.StatusActive,
			Trailers:   map[string]string{gitops.TrailerVerb: "promote", gitops.TrailerActor: "ai/claude"},
		},
	}
	ackedSHAs := map[string]bool{
		"abc1234567890def": true,
	}
	got := forcedUntraileredFindings(obs, ackedSHAs)
	if len(got) != 0 {
		t.Fatalf("expected 0 findings (ack silences the forced-untrailered observation per AC-4); got %d: %+v", len(got), got)
	}
}

// TestForcedUntraileredFindings_AC4_AckedMapWithoutCommitSHA_StillFires
// is the positive control: an ackedSHAs map that does NOT
// contain the offending commit's SHA must NOT silence the
// finding. Pins per-SHA closed-set scoping — a green-phase
// regression that silenced on "ackedSHAs is non-empty" (rather
// than per-SHA match) would pass the happy-path test above and
// silently over-exempt every forced-untrailered observation.
func TestForcedUntraileredFindings_AC4_AckedMapWithoutCommitSHA_StillFires(t *testing.T) {
	t.Parallel()
	obs := []statusChange{
		{
			EntityID:   "E-0001",
			EntityKind: entity.KindEpic,
			Commit:     "abc1234567890def",
			Parent:     "0000000000000000",
			Path:       "work/epics/E-0001-x/epic.md",
			Prior:      entity.StatusProposed,
			Next:       entity.StatusActive,
			Trailers:   map[string]string{gitops.TrailerVerb: "promote", gitops.TrailerActor: "ai/claude"},
		},
	}
	ackedSHAs := map[string]bool{
		"unrelated-sha-deadbeef0000": true,
	}
	got := forcedUntraileredFindings(obs, ackedSHAs)
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 finding (per-SHA scoping: ack on unrelated SHA must not exempt abc1234567890def); got %d: %+v", len(got), got)
	}
	if got[0].Subcode != "forced-untrailered" {
		t.Errorf("Subcode = %q; want %q", got[0].Subcode, "forced-untrailered")
	}
}
