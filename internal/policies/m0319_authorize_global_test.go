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

func TestM0319_AC1_AuthorizeHasNoTransitionCells(t *testing.T) {
	t.Parallel()
	for _, r := range spec.Rules() {
		if r.Verb == "authorize" {
			t.Errorf("authorize has a transition cell: %+v", r)
		}
	}
}

func TestM0319_AC1_GlobalKindRestriction(t *testing.T) {
	t.Parallel()
	var rules []spec.Rule
	for _, r := range spec.GlobalRules() {
		if r.ExpectedErrorCode == "authorize-kind-not-allowed" {
			rules = append(rules, r)
		}
	}
	if len(rules) != 1 {
		t.Fatalf("got %d global kind restrictions, want one", len(rules))
	}
	rule := rules[0]
	if rule.Kind != "" || rule.FromState != "" || rule.ToState != "" || rule.Verb != "authorize" || rule.Sources.Decision != "D-0007" || rule.Outcome != spec.OutcomeIllegal || rule.RejectionLayer != spec.RejectionLayerVerbTime || !rule.BlockingStrict {
		t.Fatalf("kind restriction must be coordinate-free, source D-0007, and refuse at verb time: %+v", rule)
	}
	for _, kind := range entity.AllKinds() {
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()
			applies := true
			for _, p := range rule.Preconditions {
				holds, err := spec.EvaluatePredicate(p, &entity.Entity{Kind: kind}, nil, spec.EvalContext{})
				if err != nil {
					t.Fatal(err)
				}
				applies = applies && holds
			}
			want := kind != entity.KindEpic && kind != entity.KindMilestone
			if applies != want {
				t.Errorf("restriction applies=%t, want=%t", applies, want)
			}
		})
	}
}

func TestM0319_AC1_ExcludedKindsRefuseWithoutSideEffects(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)
	for _, kind := range []entity.Kind{entity.KindADR, entity.KindGap, entity.KindDecision, entity.KindContract} {
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()
			f := cellcoverage.NewCellFixture(t)
			state := "proposed"
			if kind == entity.KindGap {
				state = "open"
			}
			id := f.BringEntityToState(t, kind, state, cellcoverage.BringOpts{})
			before := fixtureGitSnapshot(t, f.Root)
			out, err := testutil.RunBin(t, f.Root, "", nil, "authorize", id, "--to", "ai/claude")
			if err == nil || !strings.Contains(out, "authorize-kind-not-allowed") {
				t.Fatalf("want kind refusal, err=%v output=%s", err, out)
			}
			if !bytes.Equal(before, fixtureGitSnapshot(t, f.Root)) {
				t.Fatal("refusal changed HEAD or project files")
			}
		})
	}
}
