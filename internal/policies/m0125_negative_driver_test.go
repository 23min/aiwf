package policies

import (
	"context"
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cellcoverage"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/entity"
	"github.com/23min/aiwf/internal/verb"
	"github.com/23min/aiwf/internal/workflows/spec"
)

// TestM0125_AC2_NegativeDriver_VerbTimeRejection exercises every
// Illegal cell with RejectionLayer == VerbTime by driving the real
// `aiwf` binary through a subprocess against a per-cell fixture. For
// each cell the driver:
//
//   - Builds the fixture and brings the subject entity to the cell's
//     FromState (reusing AC-1's precondition pipeline).
//   - Captures the HEAD SHA before the cell-under-test invocation.
//   - Executes the verb via testutil.RunBin, expecting non-zero exit.
//   - Confirms HEAD is unchanged after rejection (the verb's commit
//     never landed — kernel-level rollback).
//   - Confirms the error output names the right rejection reason via
//     the kernel-side substring mapping below.
//
// The substring map is the test's side of the rule's `ExpectedErrorCode`
// — the kernel emits human-readable error text, not literal rule codes,
// so this map translates the spec's logical code to a kernel-emitted
// substring. A kernel error-message change that drops the substring
// fails the test loudly; this is intentional (we want to know when the
// kernel rephrases an enforcement boundary).
//
// Coverage commitment: every verb-time Illegal cell in spec.Rules()
// yields one subtest; M-0123 phase 1 pinned the floor at 27 cells
// (15 explicit OutcomeIllegal struct literals carrying
// RejectionLayer: RejectionLayerVerbTime + 12 terminalIllegal helper
// invocations; the 2 RejectionLayerCheckTime cells are out of scope
// here and covered by AC-3).
func TestM0125_AC2_NegativeDriver_VerbTimeRejection(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)

	cases := enumerateVerbTimeIllegalCases(t)
	if len(cases) == 0 {
		t.Fatal("no verb-time Illegal cells enumerated from spec.Rules()")
	}
	if len(cases) < 27 {
		t.Errorf("expected at least 27 verb-time Illegal cells, got %d (spec shrank?)", len(cases))
	}

	// Sanity-check ac2KnownImplGaps keys against the enumeration:
	// detect map staleness if predicate authoring changes shift cell
	// names (subtest names derive from illegalCaseName + precondition
	// signature). A skip referencing a non-existent cell is a silent
	// regression — the supposed-to-be-skipped cell would run unguarded.
	liveNames := make(map[string]bool, len(cases))
	for _, tc := range cases {
		liveNames[tc.name] = true
	}
	for name := range ac2KnownImplGaps {
		if !liveNames[name] {
			t.Errorf("ac2KnownImplGaps key %q does not match any verb-time Illegal cell name; map went stale (predicate authoring change?)", name)
		}
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if gap, ok := ac2KnownImplGaps[tc.name]; ok {
				// Staleness teeth (M-0125/AC-2 retrofit, symmetric to
				// AC-3's runNegativeCheckTimeCell impl-gap branch).
				// The cell's spec axis is VerbTime but the kernel
				// currently does NOT reject — the gap tracks the
				// missing chokepoint. Drive the verb and assert it
				// STILL succeeds (gap is still open). If the kernel
				// learns to reject (gap closes), the verb fails, the
				// assertion fires red, and the operator is forced to
				// remove the entry so the cell graduates to
				// end-to-end coverage via runNegativeVerbTimeCell.
				runImplGapStalenessVerbTime(t, tc, gap)
				return
			}
			runNegativeVerbTimeCell(t, tc)
		})
	}
}

// ac2KnownImplGaps lists Illegal cells whose spec.RejectionLayer is
// VerbTime but where the kernel currently does not enforce the
// rejection at verb-time. Each entry pins the gap entity that tracks
// the impl work needed to close the divergence. When a gap is
// addressed (the kernel learns to reject at verb-time) the
// corresponding entry here is removed; the cell's subtest then
// participates in coverage automatically. New impl gaps surfacing in
// the future get filed and added here in the same commit.
//
// Discovered by M-0125's negative-driver dry-run against the kernel:
// the cells listed succeed when they should fail. They are NOT spec
// errors — the spec was deliberately authored to mark these as
// verb-time chokepoints (per D-0003, D-0004, D-0007, and the audit
// catalog citations on each cell).
var ac2KnownImplGaps = map[string]string{
	// Cross-entity non-terminal-child/AC checks — verb proceeds with the
	// cancel even when the spec's precondition (non-terminal child or
	// open AC) holds. Tracked by G-0139 (filed at M-0123 wrap referencing
	// D-0003 + D-0004; G-0162 was a duplicate filed at M-0125 and
	// cancelled wontfix on consolidation).
	"epic-proposed-cancel-anychildstatusnotinmilestoneterminalset": "G-0139",
	"epic-active-cancel-anychildstatusnotinmilestoneterminalset":   "G-0139",
	"milestone-draft-cancel-anychildacstatuseqopen":                "G-0139",
	"milestone-in_progress-cancel-anychildacstatuseqopen":          "G-0139",
	// CancelTarget(ADR/Decision, accepted) returns "rejected" but the
	// FSM forbids accepted→rejected. The verb's cancel path bypasses
	// FSM. Tracked by G-0163.
	"adr-accepted-cancel": "G-0163",
	// ac-evidence-missing is unsupported by the verb — no --evidence
	// flag, no PromoteOptions.Evidence, no validation. Tracked by G-0140
	// (filed at M-0123 wrap referencing D-0005; G-0164 was a duplicate
	// filed at M-0125 and cancelled wontfix on consolidation).
	"ac-open-promote": "G-0140",
}

func enumerateVerbTimeIllegalCases(t *testing.T) []illegalCase {
	t.Helper()
	var out []illegalCase
	rules := spec.Rules()
	for i := range rules {
		rule := rules[i]
		if rule.Outcome != spec.OutcomeIllegal || rule.RejectionLayer != spec.RejectionLayerVerbTime {
			continue
		}
		out = append(out, illegalCase{name: illegalCaseName(rule), rule: rule})
	}
	return out
}

// errorSubstringsFor maps the spec's ExpectedErrorCode to one or more
// kernel-emitted phrasings the verb-time error message may contain.
// Test passes if the output contains AT LEAST ONE of the listed
// substrings (logical OR). The kernel does not emit the rule codes as
// literal strings (those are spec/check identifiers); instead each
// rejection path produces a Go error with hand-rolled wording. One
// spec code can correspond to several kernel error phrasings — e.g.
// fsm-transition-illegal surfaces as both "cannot transition to" (non-
// terminal-with-no-edge) and "no cancel target" (terminal-state cancel).
//
// Only codes whose cells run today appear here. Codes whose cells are
// currently skipped under ac2KnownImplGaps (epic/milestone cancel-non-
// terminal-children/acs, ac-evidence-missing) intentionally have no
// entry — there's no live test to anchor the substring choice. When a
// gap closes and the cell un-skips, the case is added in the same
// commit (the unmapped-code branch falls through to "non-zero exit +
// rollback only," which would let the test pass on wrong-reason
// rejection — adding the case is required for full assertion strength).
func errorSubstringsFor(code string) []string {
	switch code {
	case "fsm-transition-illegal":
		return []string{
			"cannot transition to", // non-terminal & terminal cases both
			"no cancel target",     // CancelTarget returns "" for terminal
			"is already at terminal status",
		}
	case "milestone-done-incomplete-acs":
		return []string{"open AC", "incomplete"}
	case "adr-supersession-mutual":
		return []string{"supersede", "mutual"}
	case "authorize-kind-not-allowed":
		return []string{"authorize", "not allowed"}
	}
	return nil
}

func runNegativeVerbTimeCell(t *testing.T, tc illegalCase) {
	t.Helper()
	f := cellcoverage.NewCellFixture(t)
	opts := deriveBringOpts(tc.rule)
	id := bringEntityForCell(t, f, tc.rule, opts)

	evalCtx := spec.EvalContext{}
	// All live Illegal-cell predicates today flow through SatisfyPredicate
	// directly — no Illegal cell uses self.target-state or carries the
	// non-empty form of self.addressed_by / self.superseded_by, so the
	// M-0124 driver's verb-arg-shaping switch isn't needed here. If a
	// future Illegal cell introduces such a precondition, copy the
	// shape from m0124_positive_driver_test.go::runPositiveCell.
	for _, p := range tc.rule.Preconditions {
		f.SatisfyPredicate(t, p, id, &evalCtx)
	}

	headBefore, err := testutil.RunGit(f.Root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse HEAD before: %v\n%s", err, headBefore)
	}
	headBefore = strings.TrimSpace(headBefore)

	args := buildIllegalVerbArgs(t, tc, id)
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

	headAfter, err := testutil.RunGit(f.Root, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse HEAD after: %v\n%s", err, headAfter)
	}
	headAfter = strings.TrimSpace(headAfter)
	if headBefore != headAfter {
		t.Errorf("HEAD moved from %s to %s after expected-rejection verb\nargs: %v\noutput:\n%s",
			headBefore, headAfter, args, out)
	}
}

// buildIllegalVerbArgs constructs the CLI args for a verb-time Illegal
// cell. No live Illegal cell uses verb-arg-shaped preconditions
// (self.target-state / self.addressed_by:non-empty /
// self.superseded_by:non-empty), so this is simpler than M-0124's
// buildVerbArgs — there are no `--by` / `--superseded-by` flag paths
// here. If a future Illegal cell introduces such a precondition,
// re-introduce the extras/evalCtx parameters and copy the shape from
// m0124_positive_driver_test.go::buildVerbArgs.
func buildIllegalVerbArgs(t *testing.T, tc illegalCase, id string) []string {
	t.Helper()
	switch tc.rule.Verb {
	case "cancel":
		return []string{"cancel", id}
	case "authorize":
		return []string{"authorize", id, "--to", "ai/claude"}
	case "promote":
		if tc.rule.Kind == spec.KindTDDPhase {
			return []string{"promote", id, "--phase", anyOtherTDDPhase(tc.rule.FromState)}
		}
		return []string{"promote", id, deriveIllegalPromoteTarget(t, tc.rule)}
	}
	t.Fatalf("buildIllegalVerbArgs: unsupported verb %q", tc.rule.Verb)
	return nil
}

// deriveIllegalPromoteTarget picks the target for an Illegal promote
// cell. The first FSM-allowed transition wins (covers cells where the
// FromState is non-terminal and the cell's other precondition triggers
// rejection — e.g. ADR.accepted with self.superseded_by=="" rejects on
// the verb's --superseded-by-required guard before FSM check fires).
// Otherwise: a kind-domain status (terminal-FromState cells — any
// target trips fsm-transition-illegal).
func deriveIllegalPromoteTarget(t *testing.T, rule spec.Rule) string {
	t.Helper()
	allowed := entity.AllowedTransitions(rule.Kind, rule.FromState)
	if len(allowed) > 0 {
		return allowed[0]
	}
	return anyKindDomainStatus(t, rule.Kind)
}

func anyKindDomainStatus(t *testing.T, k entity.Kind) string {
	t.Helper()
	switch k {
	case entity.KindEpic:
		return entity.StatusActive
	case entity.KindMilestone:
		return entity.StatusInProgress
	case entity.KindADR, entity.KindDecision:
		return entity.StatusAccepted
	case entity.KindGap:
		return entity.StatusAddressed
	case entity.KindContract:
		return entity.StatusAccepted
	case spec.KindAC:
		return entity.StatusMet
	}
	t.Fatalf("anyKindDomainStatus: no domain status for kind %q", k)
	return ""
}

func anyOtherTDDPhase(current string) string {
	if current == entity.TDDPhaseDone {
		return entity.TDDPhaseRed
	}
	return entity.TDDPhaseDone
}

// runImplGapStalenessVerbTime drives a cell tracked by an entry in
// ac2KnownImplGaps and asserts the verb STILL succeeds — the kernel
// currently does NOT reject this Illegal cell at verb-time (the gap
// tracks the missing chokepoint). If the kernel learns to reject
// (the tracked gap closes), the verb starts failing, the assertion
// fires red, and the operator is forced to remove the
// ac2KnownImplGaps entry so the cell graduates to end-to-end coverage.
//
// Symmetric to AC-3's runNegativeCheckTimeCell impl-gap branch, just
// inverted:
//
//   - AC-3 impl-gap cells: kernel rejects when spec says CheckTime is
//     the chokepoint (the kernel is stricter than spec). Staleness =
//     "verb still rejects."
//   - AC-2 impl-gap cells: kernel does NOT reject when spec says
//     VerbTime is the chokepoint (the kernel is more permissive than
//     spec). Staleness = "verb still succeeds."
//
// Replaces the previous t.Skipf path (no per-cell assertion) so the
// divergence-tracking is two-way: ac2KnownImplGaps entries can become
// stale (closed gap) and the test surfaces the staleness rather than
// silently continuing to skip.
func runImplGapStalenessVerbTime(t *testing.T, tc illegalCase, gap string) {
	t.Helper()
	f := cellcoverage.NewCellFixture(t)
	opts := deriveBringOpts(tc.rule)
	id := bringEntityForCell(t, f, tc.rule, opts)

	// Optional per-cell fixture customization: avoids incidental
	// projection-finding rejections that would obscure the
	// gap-tracked chokepoint's status. Runs after bringEntityForCell
	// and before SatisfyPredicate / verb drive.
	if setup, ok := ac2ImplGapFixtureSetup[tc.name]; ok {
		setup(t, f, id)
	}

	evalCtx := spec.EvalContext{}
	for _, p := range tc.rule.Preconditions {
		f.SatisfyPredicate(t, p, id, &evalCtx)
	}

	args := buildIllegalVerbArgs(t, tc, id)
	out, verbErr := testutil.RunBin(t, f.Root, "", nil, args...)
	if verbErr != nil {
		t.Errorf("ac2KnownImplGaps[%q] (tracking %s) is stale: verb returned non-zero, meaning the kernel has learned to reject this cell at verb-time. Remove the entry from ac2KnownImplGaps and the cell will be exercised end-to-end by runNegativeVerbTimeCell.\nargs: %v\nverb output:\n%s", tc.name, gap, args, out)
	}
}

// ac2ImplGapFixtureSetup holds optional per-cell fixture
// customization for ac2KnownImplGaps cells whose default fixture
// state triggers an incidental rejection (typically a projection-
// finding rejection in finalizeACPlan) that would mask the
// gap-tracked chokepoint's actual status. The customization runs
// after bringEntityForCell and before SatisfyPredicate / verb drive.
//
// Most ac2KnownImplGaps cells don't need this (their default fixture
// reaches the verb cleanly); they have no entry here and the
// staleness assertion runs unmodified.
//
// History: introduced during the AC-4 retrofit (replacing t.Skipf
// with staleness teeth surfaced that ac-open-promote was incidentally
// rejected via projectionFindings's acs-tdd-audit error, even though
// G-0140's missing chokepoint — `--evidence` flag at verb time — is
// still genuinely absent. Without this customization the naive
// staleness check would fire false-positive for ac-open-promote, when
// in fact the gap is still open).
var ac2ImplGapFixtureSetup = map[string]func(t *testing.T, f *cellcoverage.CellFixture, id string){
	// ac-open-promote (G-0140): cell is "AC.open.promote with
	// self.evidence empty" — verb-time chokepoint at the evidence
	// flag is missing. But the default fixture (acAt(open)) seeds AC
	// at phase=red under parent.tdd=required, and finalizeACPlan's
	// projection catches acs-tdd-audit as an error, returning
	// non-zero before the (missing) evidence chokepoint would fire.
	// Advance phase to done so the projection finding doesn't fire
	// and the staleness check correctly asserts "verb still succeeds
	// = G-0140 still open." When G-0140 closes (verb gains an
	// --evidence chokepoint), the verb will reject for the cell's
	// actual reason and the staleness check will fire red as
	// designed.
	"ac-open-promote": func(t *testing.T, f *cellcoverage.CellFixture, id string) {
		t.Helper()
		ctx := context.Background()
		f.Must(verb.PromoteACPhase(ctx, f.Tree(), id, entity.TDDPhaseGreen, "human/test", "AC-2 staleness fixture: bypass incidental acs-tdd-audit projection", false, nil))
		f.Must(verb.PromoteACPhase(ctx, f.Tree(), id, entity.TDDPhaseDone, "human/test", "AC-2 staleness fixture: bypass incidental acs-tdd-audit projection", false, nil))
	},
}
