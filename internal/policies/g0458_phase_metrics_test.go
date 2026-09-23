package policies

import (
	"bytes"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cellcoverage"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/workflows/spec"
)

func TestG0458_CLIPhaseRepeatPreservesHistory(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	for _, tc := range []struct {
		name, metrics string
		refuse        bool
	}{
		{name: "without-metrics"},
		{name: "blank-metrics", metrics: " \t "},
		{name: "with-metrics", metrics: "pass=3", refuse: true},
		{name: "with-zero-metrics", metrics: "pass=0 fail=0 skip=0", refuse: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := cellcoverage.NewCellFixture(t)
			id := bringTDDPhaseAC(t, f, entity.TDDPhaseDone)
			before := fixtureGitSnapshot(t, f.Root)
			args := []string{"promote", id, "--phase", entity.TDDPhaseDone}
			if tc.metrics != "" {
				args = append(args, "--tests", tc.metrics)
			}
			out, err := testutil.RunBin(t, f.Root, "", nil, args...)
			if (err != nil) != tc.refuse {
				t.Fatalf("refuse=%t, err=%v\n%s", tc.refuse, err, out)
			}
			want := "nothing to change"
			if tc.refuse {
				want = "--tests requires a phase change"
			}
			if !strings.Contains(out, want) {
				t.Errorf("output %q does not contain %q", out, want)
			}
			if after := fixtureGitSnapshot(t, f.Root); !bytes.Equal(before, after) {
				t.Fatal("phase repeat changed HEAD or files")
			}
		})
	}
}

func TestG0458_PhaseCellsDistinguishMetrics(t *testing.T) {
	t.Parallel()
	for _, phase := range entity.AllowedTDDPhases() {
		for _, metrics := range []string{"", "pass=0"} {
			want := spec.OutcomeNoOp
			if metrics != "" {
				want = spec.OutcomeIllegal
			}
			matches := 0
			for _, rule := range spec.LookupRules(spec.KindTDDPhase, phase, "promote") {
				if rule.ToState != phase {
					continue
				}
				applies := true
				for _, p := range rule.Preconditions {
					holds, err := spec.EvaluatePredicate(p, nil, nil, spec.EvalContext{TestMetrics: metrics})
					if err != nil {
						t.Fatal(err)
					}
					applies = applies && holds
				}
				if applies {
					matches++
					if rule.Outcome != want {
						t.Errorf("phase=%s metrics=%q: outcome=%v want=%v", phase, metrics, rule.Outcome, want)
					}
				}
			}
			if matches != 1 {
				t.Errorf("phase=%s metrics=%q: got %d applicable self-target cells, want 1", phase, metrics, matches)
			}
		}
	}
}
