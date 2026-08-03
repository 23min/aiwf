package verb_test

import (
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
)

// TestSweepNoOps_WriteNothing pins the premise the sweep's recorded
// claim-scope call rests on: a converging sweep writes nothing. If it
// could commit while reporting nothing to do, "the converging path is
// harmless" would stop being an argument.
func TestSweepNoOps_WriteNothing(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		call func(r *runner) (*verb.Result, error)
	}{
		{"archive", func(r *runner) (*verb.Result, error) {
			return verb.Archive(r.ctx, r.root, testActor, "")
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := newRunner(t)
			r.must(verb.Add(r.ctx, r.tree(), entity.KindGap, "Live gap", testActor,
				verb.AddOptions{BodyOverride: bornCompleteFixtureBody(entity.KindGap)}))
			before := headSHA(t, r.root)

			res, err := tc.call(r)
			if err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if !res.NoOp {
				t.Fatalf("%s did not converge on a tree with nothing to sweep: %+v", tc.name, res)
			}
			if res.Plan != nil {
				t.Errorf("%s converged with a plan attached: %+v", tc.name, res.Plan)
			}
			if after := headSHA(t, r.root); after != before {
				t.Errorf("%s advanced HEAD to %s while converging", tc.name, after)
			}
		})
	}
}
