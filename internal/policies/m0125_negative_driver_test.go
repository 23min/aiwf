package policies

import (
	"bytes"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cellcoverage"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/workflows/spec"
)

// TestM0125_AC2_NegativeDriver_VerbTimeRejection drives every illegal
// transition through the real binary without force. The declared layer,
// refusal reason, and unchanged HEAD and files must agree with the result.
func TestM0125_AC2_NegativeDriver_VerbTimeRejection(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)

	cases := enumerateIllegalCases(t)
	if len(cases) == 0 {
		t.Fatal("no verb-time Illegal cells enumerated from spec.Rules()")
	}
	if len(cases) < 27 {
		t.Errorf("expected at least 27 verb-time Illegal cells, got %d (spec shrank?)", len(cases))
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.rule.RejectionLayer != spec.RejectionLayerVerbTime {
				t.Errorf("cell %s declares rejection layer %v; its ordinary request must refuse before writing", tc.name, tc.rule.RejectionLayer)
			}
			if len(errorSubstringsFor(tc.rule.ExpectedErrorCode)) == 0 {
				t.Errorf("cell %s has no refusal-message assertion for %q", tc.name, tc.rule.ExpectedErrorCode)
			}
			runNegativeVerbTimeCell(t, tc)
		})
	}
}

// errorSubstringsFor maps each declared refusal code to the kernel's
// operator-facing refusal text. The driver rejects codes without a mapping.
func errorSubstringsFor(code string) []string {
	switch code {
	case "fsm-transition-illegal":
		return []string{
			"cannot transition to", // non-terminal & terminal cases both
			"no cancel target",     // CancelTarget returns "" for terminal
			"--tests requires a phase change",
			// No arm for "is already at terminal status": cancel converges
			// there, so that phrasing is a NoOp message, and matching it
			// would bless a NoOp as a valid refusal.
		}
	case "gap-addressed-has-resolver", "acs-tdd-audit":
		return []string{code}
	case "milestone-done-incomplete-acs":
		return []string{"open AC", "incomplete"}
	case "adr-supersession-mutual":
		return []string{"supersede", "mutual"}
	case "epic-cancel-non-terminal-children":
		// M-0139 guard: EpicCancelNonTerminalChildrenError.Error().
		return []string{"non-terminal child milestone", "epic-cancel-non-terminal-children"}
	case "milestone-cancel-non-terminal-acs":
		// M-0139 guard: MilestoneCancelNonTerminalACsError.Error().
		return []string{"open acceptance criterion", "milestone-cancel-non-terminal-acs"}
	case "epic-promote-non-terminal-children":
		// G-0394 guard: EpicPromoteNonTerminalChildrenError.Error().
		return []string{"non-terminal child milestone", "epic-promote-non-terminal-children"}
	case "milestone-promote-non-terminal-acs":
		// G-0335 guard: MilestonePromoteNonTerminalACsError.Error().
		return []string{"open acceptance criterion", "milestone-promote-non-terminal-acs"}
	}
	return nil
}

func runNegativeVerbTimeCell(t *testing.T, tc illegalCase) {
	t.Helper()
	f := cellcoverage.NewCellFixture(t)
	opts := deriveBringOpts(tc.rule)
	id := bringEntityForCell(t, f, tc.rule, opts)

	evalCtx := spec.EvalContext{}
	// Predicates populate entity state and request context for the shared argument builder.
	for _, p := range tc.rule.Preconditions {
		f.SatisfyPredicate(t, p, id, &evalCtx)
	}

	before := fixtureGitSnapshot(t, f.Root)

	args := buildIllegalVerbArgs(t, tc, id, evalCtx)
	out, runErr := testutil.RunBin(t, f.Root, "", nil, args...)

	if runErr == nil {
		t.Fatalf("aiwf %v expected non-zero exit but succeeded:\n%s", args, out)
	}

	if wants := errorSubstringsFor(tc.rule.ExpectedErrorCode); len(wants) > 0 {
		matched := false
		for _, w := range wants {
			if strings.Contains(out, w) {
				matched = true
				break
			}
		}
		if !matched {
			t.Errorf("ExpectedErrorCode=%q (substrings %v) not found in output\nargs: %v\noutput:\n%s",
				tc.rule.ExpectedErrorCode, wants, args, out)
		}
	}

	if !bytes.Equal(before, fixtureGitSnapshot(t, f.Root)) {
		t.Errorf("HEAD or project files changed after refusal\nargs: %v\noutput:\n%s", args, out)
	}
}

// buildIllegalVerbArgs uses the declared target and metrics request context.
func buildIllegalVerbArgs(t *testing.T, tc illegalCase, id string, ctx spec.EvalContext) []string {
	t.Helper()
	switch tc.rule.Verb {
	case "cancel":
		return []string{"cancel", id}
	case "promote":
		return buildVerbArgs(t, positiveCase{rule: tc.rule, target: tc.rule.ToState}, id,
			extraArgs{testMetrics: ctx.TestMetrics})
	}
	t.Fatalf("buildIllegalVerbArgs: unsupported verb %q", tc.rule.Verb)
	return nil
}
