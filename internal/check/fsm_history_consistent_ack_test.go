package check

import (
	"context"
	"testing"

	"github.com/23min/aiwf/internal/tree"
)

// fsm_history_consistent_ack_test.go — M-0159/AC-3 red phase: pin
// that FSMHistoryConsistent accepts an ackedSHAs map[string]bool
// 4th parameter populated by the CLI gather layer. AC-3 names
// fsm-history-consistent as one of the THREE consumers driven
// through the lifted helper; the corresponding RED for the other
// two (RunIsolationEscape, RunTrailerVerbUnknown) lives in the
// sibling _ack_test.go files.
//
// Compile-RED today: the current public surface is
// FSMHistoryConsistent(ctx, root, tree) — 3 args. The internal
// walk computes ackedSHAs at fsm_history_consistent.go:174 via
// the soon-to-be-lifted walkAcknowledgedSHAs. This test calls the
// function with the 4-arg shape that the green phase must adopt
// so the gather layer's single-compute can flow into this rule
// without re-walking.
//
// Without this compile-RED, a green phase that lifts the helper
// and wires RunIsolationEscape / RunTrailerVerbUnknown to consume
// the gather-layer-computed map but leaves FSMHistoryConsistent's
// public signature unchanged (still re-walking internally) would
// satisfy 2 of the 3 named consumers — defeating the AC's
// "ackedSHAs ... populated by the CLI gather layer ... single
// parameter ... three consumers" claim. The compile-RED forces
// the third consumer's signature to flip in lockstep.
//
// The green-phase implementation has discretion on how the public
// API stays backward-compatible (e.g., a public 3-arg wrapper
// that delegates to an internal 4-arg shape using the lifted
// helper internally for callers that don't have ackedSHAs yet,
// while the gather layer adopts the 4-arg shape directly). What
// is NOT discretionary: the rule must HAVE the 4-arg seam so the
// gather layer can flow ackedSHAs through. This test pins the
// seam's existence; how the wrapper is named is a green-phase
// choice that downstream tests don't need to constrain.

// TestFSMHistoryConsistent_M0159AC3_AcceptsAckedSHAsParam pins
// that the rule HAS a callable surface accepting ackedSHAs as a
// 4th positional argument — same shape as the other two AC-3
// consumers. The body assertion is intentionally minimal: this
// is the seam pin, not a behavior pin (the behavioral surface
// for FSM-history ack-silencing already lives in the M-0136/AC-2
// tests at internal/check/fsm_history_acknowledgment_test.go:
// TestFSMHistoryConsistent_AC2_AcknowledgmentExemptsIllegalTransition
// and TestFSMHistoryConsistent_AC2_AcknowledgmentScopedToTarget).
//
// Note: the milestone-qualified test name (M0159AC3 rather than
// AC3) disambiguates from M-0136's own AC-3 test in the same
// package per the second-reviewer naming-collision note.
//
// On a fresh empty repo with an empty tree, the rule returns no
// findings regardless of the ackedSHAs map content — the
// hasGitCommits short-circuit returns nil. The compile-RED is
// the signal: today the function only accepts 3 args.
func TestFSMHistoryConsistent_M0159AC3_AcceptsAckedSHAsParam(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	tr := &tree.Tree{}
	ackedSHAs := map[string]bool{}

	got := FSMHistoryConsistent(context.Background(), root, tr, ackedSHAs)
	if len(got) != 0 {
		t.Fatalf("FSMHistoryConsistent on empty fixture returned non-empty findings: %+v", got)
	}
}
