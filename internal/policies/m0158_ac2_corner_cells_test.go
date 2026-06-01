package policies

import (
	"fmt"
	"testing"

	"github.com/23min/aiwf/internal/workflows/spec"
	"github.com/23min/aiwf/internal/workflows/spec/branch"
)

// TestM0158_AC2_TwelveCornerCellsPresent pins M-0158/AC-2: every
// numbered corner case from E-0030 §"Corner cases" (1..12) is
// registered as a named cell `branch-cell-N` in `branch.Rules()`.
// Cell ids match the corner-case numbers 1:1 for traceability.
//
// Per the user's pre-implementation Q&A: 1:1 mapping (not collapsed
// groupings) so the spec body's AC-2 literal wording holds and a
// reader can navigate from cell id to epic prose by the same number.
func TestM0158_AC2_TwelveCornerCellsPresent(t *testing.T) {
	t.Parallel()

	byID := indexBranchRulesByID(t)
	for n := 1; n <= 12; n++ {
		id := fmt.Sprintf("branch-cell-%d", n)
		if _, ok := byID[id]; !ok {
			t.Errorf("M-0158/AC-2: branch.Rules() missing %q (corner case %d from E-0030 epic body §\"Corner cases\")", id, n)
		}
	}
}

// TestM0158_AC2_CornerCellOutcomesMatchEpic pins the spec-side
// alignment between cell.Outcome and the epic body's enumerated
// outcomes ("preflight rejects" / "finding fires" / "finding
// silent"). A regression that flipped an Outcome (e.g., marking
// branch-cell-4 as Legal) would fire this test even before the
// behavioral tests under internal/verb/ + internal/check/ caught
// the runtime drift.
//
// The expected-outcome map is the canonical M-0158 source: it
// encodes the user's directive *"these corner cases become part
// of these verifications"* as a per-cell legality assertion.
func TestM0158_AC2_CornerCellOutcomesMatchEpic(t *testing.T) {
	t.Parallel()

	want := map[int]spec.Outcome{
		1:  spec.OutcomeIllegal, // AI authorize on main no --branch → refused
		2:  spec.OutcomeIllegal, // AI authorize --branch <typo> → refused
		3:  spec.OutcomeLegal,   // AI authorize on epic ritual → accepted
		4:  spec.OutcomeIllegal, // AI commit on main while bound to epic → fires
		5:  spec.OutcomeLegal,   // AI commit on bound branch → silent
		6:  spec.OutcomeLegal,   // AI commit on bound, scope paused → silent
		7:  spec.OutcomeIllegal, // AI commit on different epic → fires
		8:  spec.OutcomeLegal,   // Human cherry-pick → silent
		9:  spec.OutcomeLegal,   // Human merge → silent
		10: spec.OutcomeLegal,   // --force amend → silent
		11: spec.OutcomeLegal,   // AI commit with no scope → silent
		12: spec.OutcomeIllegal, // worktree-vs-branch mismatch → fires
	}

	byID := indexBranchRulesByID(t)
	for n, wantOutcome := range want {
		id := fmt.Sprintf("branch-cell-%d", n)
		got, ok := byID[id]
		if !ok {
			continue // covered by TwelveCornerCellsPresent above; don't double-report
		}
		if got.Outcome != wantOutcome {
			t.Errorf("M-0158/AC-2: %s.Outcome = %v; want %v (per E-0030 epic §\"Corner cases\" #%d)", id, got.Outcome, wantOutcome, n)
		}
	}
}

// TestM0158_AC2_IllegalCellsCarryExpectedErrorCode asserts that
// every Illegal corner-case cell carries an ExpectedErrorCode
// matching the cell's actual kernel emission point. The triplet
// (branch-context-required, branch-not-found, isolation-escape)
// covers the layer-4 illegal cells; a regression that left an
// Illegal cell's code empty would surface here.
func TestM0158_AC2_IllegalCellsCarryExpectedErrorCode(t *testing.T) {
	t.Parallel()

	wantCode := map[int]string{
		1:  "branch-context-required",
		2:  "branch-not-found",
		4:  "isolation-escape",
		7:  "isolation-escape",
		12: "isolation-escape",
	}

	byID := indexBranchRulesByID(t)
	for n, want := range wantCode {
		id := fmt.Sprintf("branch-cell-%d", n)
		got, ok := byID[id]
		if !ok {
			continue
		}
		if got.ExpectedErrorCode != want {
			t.Errorf("M-0158/AC-2: %s.ExpectedErrorCode = %q; want %q", id, got.ExpectedErrorCode, want)
		}
	}
}

// indexBranchRulesByID returns branch.Rules() keyed by Rule.ID.
// Helper used by all M-0158 cell-existence tests. The function
// fatals if any branch.Rules() entry has an empty ID — a
// regression there would silently mask the presence assertions
// in the tests that use this helper.
func indexBranchRulesByID(t *testing.T) map[string]spec.Rule {
	t.Helper()
	out := map[string]spec.Rule{}
	for _, r := range branch.Rules() {
		if r.ID == "" {
			t.Fatalf("branch.Rules() contains entry with empty ID: %+v", r)
		}
		if _, dup := out[r.ID]; dup {
			t.Fatalf("branch.Rules() contains duplicate ID %q", r.ID)
		}
		out[r.ID] = r
	}
	return out
}
