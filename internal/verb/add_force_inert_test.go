package verb_test

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/verb"
)

// TestAddForceIsInertWithoutAGateToBypass pins the premise behind the
// one exemption in PolicySovereignDispatchersGuardHumanActor
// (M-0293/AC-3).
//
// `aiwf add --force` is not unconditionally a sovereign act. The verb
// stamps the force trailer only when the flag actually bypassed the
// born-complete body gate; epic and milestone have no such gate, so the
// flag is inert there, no trailer is written, and the apply seam has
// nothing to refuse — a non-human actor's invocation succeeds.
//
// That is why `add`'s dispatcher carries no flag-keyed pre-check, where
// promote, cancel and authorize each do: theirs emit the trailer
// whenever the flag is set, so a pre-check refuses exactly what the
// seam would. One keyed on `add`'s flag would refuse invocations the
// kernel permits.
//
// The test runs as a non-human actor deliberately, so both halves are
// asserted at once: no force trailer in the plan, and Apply accepting
// it. If the trailer ever became unconditional, Apply would refuse and
// this fails on the seam rather than on the assertion — either way the
// exemption has lost its justification and `add` should take the guard
// like its three siblings.
func TestAddForceIsInertWithoutAGateToBypass(t *testing.T) {
	t.Parallel()

	const nonHuman = "ai/claude"

	t.Run("epic", func(t *testing.T) {
		t.Parallel()
		r := newRunner(t)
		res := r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Inert force", nonHuman, verb.AddOptions{
			Force:  true,
			Reason: "a reason the flag has no gate to spend",
		}))
		assertNoForceTrailer(t, res, entity.KindEpic)
	})

	t.Run("milestone", func(t *testing.T) {
		t.Parallel()
		r := newRunner(t)
		r.must(verb.Add(r.ctx, r.tree(), entity.KindEpic, "Parent", testActor, verb.AddOptions{}))

		res := r.must(verb.Add(r.ctx, r.tree(), entity.KindMilestone, "Inert force", nonHuman, verb.AddOptions{
			EpicID: "E-0001",
			TDD:    "none",
			Force:  true,
			Reason: "a reason the flag has no gate to spend",
		}))
		assertNoForceTrailer(t, res, entity.KindMilestone)
	})
}

func assertNoForceTrailer(t *testing.T, res *verb.Result, kind entity.Kind) {
	t.Helper()
	for _, tl := range res.Plan.Trailers {
		if tl.Key == gitops.TrailerForce {
			t.Fatalf("aiwf add %s stamped %s=%q with --force on a kind that has no body gate to "+
				"bypass. The flag is no longer inert here, so a dispatcher pre-check keyed on it "+
				"would no longer over-refuse — revisit the exemption in "+
				"internal/policies/sovereign.go", kind, tl.Key, tl.Value)
		}
	}
}
