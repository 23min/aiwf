package verb_test

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/verb"
)

// TestAddForceIsInertWithoutAGateToBypass pins the distinction the
// kernel's provenance record turns on: the human-only rule is keyed on
// the `aiwf-force:` trailer, not on the `--force` flag.
//
// The two are not the same, and `add` is where they come apart. Its
// flag bypasses the born-complete body gate, and the verb stamps the
// trailer only when the flag actually bypassed something — epic and
// milestone have no such gate, so the flag is inert there, nothing is
// overridden, and no sovereign act is recorded to refuse.
//
// docs/design/design-decisions.md states the rule in those terms, and
// this is its mechanical evidence. A record claiming the kernel refuses
// the *flag* would be false in exactly this case.
//
// The test runs as a non-human actor deliberately, so both halves are
// asserted at once: no force trailer in the plan, and Apply accepting
// it. If the trailer ever became unconditional, Apply would refuse and
// this fails at the seam rather than on the assertion.
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
				"bypass. The flag now records a sovereign act where it overrides nothing, so "+
				"the trailer-not-flag distinction in docs/design/design-decisions.md no longer "+
				"holds", kind, tl.Key, tl.Value)
		}
	}
}
