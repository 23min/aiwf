package verb_test

import (
	"fmt"
	"testing"

	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/gitops"
	"github.com/23min/aiwf/internal/verb"
)

func acPhaseFixture(t *testing.T, phase string) *runner {
	t.Helper()
	r := acFixture(t, 1)
	for _, next := range []string{entity.TDDPhaseRed, entity.TDDPhaseGreen, entity.TDDPhaseRefactor, entity.TDDPhaseDone} {
		r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-0001/AC-1", next, testActor, "", false, nil))
		if next == phase {
			return r
		}
	}
	t.Fatalf("unsupported fixture phase %q", phase)
	return nil
}

func TestPromoteACPhase_SamePhase_ReturnsNoOp(t *testing.T) {
	t.Parallel()
	for _, phase := range entity.AllowedTDDPhases() {
		t.Run(phase, func(t *testing.T) {
			t.Parallel()
			r := acPhaseFixture(t, phase)
			before := countCommits(t, r.root)
			res, err := verb.PromoteACPhase(r.ctx, r.tree(), "M-0001/AC-1", phase, testActor, "", false, nil)
			if err != nil {
				t.Fatalf("same phase: %v", err)
			}
			if !res.NoOp || res.Plan != nil {
				t.Fatalf("want NoOp with no plan, got %+v", res)
			}
			if want := fmt.Sprintf("M-0001/AC-1 is already at TDD phase %s; nothing to change", phase); res.NoOpMessage != want {
				t.Errorf("message %q, want %q", res.NoOpMessage, want)
			}
			if got := countCommits(t, r.root); got != before {
				t.Errorf("commits %s, want %s", got, before)
			}
		})
	}
}

func TestPromoteACPhase_SamePhaseForce_ReturnsNoOp(t *testing.T) {
	t.Parallel()
	r := acPhaseFixture(t, entity.TDDPhaseRed)
	res, err := verb.PromoteACPhase(r.ctx, r.tree(), "M-0001/AC-1", entity.TDDPhaseRed, testActor, "repeat", true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.NoOp || res.Plan != nil {
		t.Fatalf("forced repeat should converge without a plan: %+v", res)
	}
}

func TestPromoteACPhase_SamePhaseMetricsRefused(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		metrics gitops.TestMetrics
		force   bool
	}{
		{name: "zero-counts"},
		{name: "new-counts", metrics: gitops.TestMetrics{Pass: 3}},
		{name: "forced-counts", metrics: gitops.TestMetrics{Pass: 3}, force: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := acPhaseFixture(t, entity.TDDPhaseRed)
			before := countCommits(t, r.root)
			res, err := verb.PromoteACPhase(r.ctx, r.tree(), "M-0001/AC-1", entity.TDDPhaseRed, testActor, "repeat", tc.force, &tc.metrics)
			if code, ok := entity.Code(err); !ok || code != entity.CodeFSMTransitionIllegal.ID {
				t.Fatalf("want typed transition refusal, result=%+v err=%v", res, err)
			}
			if want := "M-0001/AC-1 is already at TDD phase red; --tests requires a phase change; omit --tests to converge without recording metrics"; err.Error() != want {
				t.Errorf("operator message %q, want %q", err.Error(), want)
			}
			if res != nil {
				t.Errorf("refused request produced result %+v", res)
			}
			if got := countCommits(t, r.root); got != before {
				t.Errorf("commits %s, want %s", got, before)
			}
		})
	}
}

func TestPromoteACPhase_AdvanceStillRecordsMetrics(t *testing.T) {
	t.Parallel()
	r := acPhaseFixture(t, entity.TDDPhaseRed)
	res := r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-0001/AC-1", entity.TDDPhaseGreen, testActor, "", false, &gitops.TestMetrics{Pass: 3}))
	if res.NoOp {
		t.Fatal("phase advance reported NoOp")
	}
	if got := r.tree().ByID("M-0001").ACs[0].TDDPhase; got != entity.TDDPhaseGreen {
		t.Fatalf("phase=%s, want green", got)
	}
	trailers, err := gitops.HeadTrailers(r.ctx, r.root)
	if err != nil {
		t.Fatal(err)
	}
	for _, tr := range trailers {
		if tr.Key == "aiwf-tests" {
			if tr.Value != "pass=3 fail=0 skip=0" {
				t.Errorf("metrics=%q", tr.Value)
			}
			return
		}
	}
	t.Fatal("phase advance omitted metrics trailer")
}

func TestPromoteACPhase_DirtyMilestoneCannotConverge(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, from, to, phase string }{
		{"phase", "tdd_phase: red", "tdd_phase: green", entity.TDDPhaseGreen},
		{"body", "## Goal", "## Goal\n\nUncommitted prose.", entity.TDDPhaseRed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := acPhaseFixture(t, entity.TDDPhaseRed)
			path := dirtyEntity(t, r, "M-0001", tc.from, tc.to)
			res, err := verb.PromoteACPhase(r.ctx, r.tree(), "M-0001/AC-1", tc.phase, testActor, "", false, nil)
			assertClaimRefused(t, res, err, path)
		})
	}
}

func TestPromoteACPhase_AbsentOrUnknownPhaseCannotConverge(t *testing.T) {
	t.Parallel()
	for _, phase := range []string{"", "unknown"} {
		t.Run(phase, func(t *testing.T) {
			t.Parallel()
			r := acFixture(t, 1)
			if phase != "" {
				r.must(verb.PromoteACPhase(r.ctx, r.tree(), "M-0001/AC-1", entity.TDDPhaseRed, testActor, "", false, nil))
				dirtyEntity(t, r, "M-0001", "tdd_phase: red", "tdd_phase: "+phase)
				commitFixture(t, r.root, "test: seed unrecognized phase")
			}
			res, err := verb.PromoteACPhase(r.ctx, r.tree(), "M-0001/AC-1", phase, testActor, "", false, nil)
			if code, ok := entity.Code(err); !ok || code != entity.CodeFSMTransitionIllegal.ID {
				t.Fatalf("want typed refusal, result=%+v err=%v", res, err)
			}
		})
	}
}
