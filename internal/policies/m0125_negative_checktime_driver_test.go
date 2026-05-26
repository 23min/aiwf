package policies

import (
	"encoding/json"
	"testing"

	"github.com/23min/aiwf/internal/cellcoverage"
	"github.com/23min/aiwf/internal/check"
	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
	"github.com/23min/aiwf/internal/render"
	"github.com/23min/aiwf/internal/workflows/spec"
)

// TestM0125_AC3_NegativeDriver_CheckTimeRejection exercises every
// Illegal cell with RejectionLayer == CheckTime. The driver branches
// per-cell on what kernel chokepoint guards the cell:
//
//  1. **Naturally check-time cells** (neither map has the cell name) —
//     run the verb without --force, expect success, then run
//     `aiwf check --format=json` and assert the rule's
//     ExpectedErrorCode appears in the envelope. End-to-end coverage.
//
//  2. **--force-bypassable verb-time guards** (cell in ac3ForceBypass) —
//     the kernel has a hand-rolled verb-time guard that --force lets
//     us skip; the rule's underlying check still fires post-write as
//     a warning. Run the verb with `--force --reason …`, expect
//     success, then run `aiwf check --format=json` and assert the
//     finding code. End-to-end coverage, just with the --force fixup.
//
//  3. **Unbypassable verb-time rejection** (cell in ac3KnownImplGaps) —
//     the kernel rejects at verb-time AND --force doesn't help
//     (typically because pre-write projectionFindings catches the
//     post-state and aborts before write). We attempt the verb
//     without --force and assert it STILL rejects. This is the
//     staleness teeth: if a future kernel change opens a check-time
//     path for the cell, the verb succeeds, the assertion fails, and
//     the operator is forced to remove the entry from
//     ac3KnownImplGaps so the cell graduates to end-to-end coverage.
//     No `aiwf check` assertion fires for these cells (the illegal
//     state never lands on disk).
//
// Coverage commitment: every check-time Illegal cell in spec.Rules()
// yields one subtest. M-0123 phase 1 produced 2 such cells; the floor
// below catches "check-time cells silently disappeared from spec".
//
// The kernel being stricter than the spec is design-aligned
// (belt-and-suspenders: illegal state never reaches disk). G-0166 is
// the umbrella tracking gap for the spec/impl axis mismatch.
func TestM0125_AC3_NegativeDriver_CheckTimeRejection(t *testing.T) {
	t.Parallel()
	testutil.SkipIfShortOrUnsupported(t)

	cases := enumerateCheckTimeIllegalCases(t)
	if len(cases) == 0 {
		t.Fatal("no check-time Illegal cells enumerated from spec.Rules()")
	}
	if len(cases) < 2 {
		t.Errorf("expected at least 2 check-time Illegal cells, got %d (spec shrank?)", len(cases))
	}

	// Sanity-check ac3KnownImplGaps and ac3ForceBypass keys against
	// the enumeration — symmetric to ac2KnownImplGaps in
	// m0125_negative_driver_test.go. A map entry pointing at a
	// non-existent cell name is silent regression (the supposed
	// behavior is never asserted).
	liveNames := make(map[string]bool, len(cases))
	for _, tc := range cases {
		liveNames[tc.name] = true
	}
	for name := range ac3KnownImplGaps {
		if !liveNames[name] {
			t.Errorf("ac3KnownImplGaps key %q does not match any check-time Illegal cell name; map went stale (predicate authoring change?)", name)
		}
	}
	for name := range ac3ForceBypass {
		if !liveNames[name] {
			t.Errorf("ac3ForceBypass key %q does not match any check-time Illegal cell name; map went stale", name)
		}
		if _, also := ac3KnownImplGaps[name]; also {
			t.Errorf("cell %q is in both ac3KnownImplGaps and ac3ForceBypass; pick one (--force either bypasses the guard or it doesn't)", name)
		}
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runNegativeCheckTimeCell(t, tc)
		})
	}
}

// ac3KnownImplGaps lists Illegal cells whose spec.RejectionLayer is
// CheckTime but where the kernel rejects at verb-time AND --force
// does not bypass (typically projectionFindings catches the
// post-state at write-time regardless of force). The driver attempts
// the verb without --force and asserts it STILL fails — staleness
// teeth: a future kernel softening that lets the verb succeed makes
// the test fire red, forcing the entry to be removed so the cell
// graduates to end-to-end coverage.
//
// The check rule itself remains active as a backstop: hand-editing
// the entity into the illegal state would still trip the rule. The
// driver does not exercise that backstop path — the check rule's
// own unit tests in internal/check/ cover it.
//
// G-0166 is the umbrella tracking gap. See the gap body for the
// resolution shape (reclassify the spec, OR soften the kernel's
// verb-time chokepoint).
var ac3KnownImplGaps = map[string]string{
	// AC.open → met under parent.tdd=required with tdd_phase != done
	// — verb-time rejection via finalizeACPlan's projectionFindings.
	// finalizeACPlan does NOT honor --force for the projection
	// gate; the gate fires on any error-severity finding in the
	// projected post-state, and acs-tdd-audit under tdd:required is
	// error-severity. The illegal state never lands on disk.
	"ac-open-promote-ptddeqrequired-tddphasenedone": "G-0166",
}

// ac3ForceBypass lists Illegal cells whose verb-time guard is
// --force-skippable, letting the driver reach the illegal state and
// exercise the check rule end-to-end. The cell is driven with
// `--force --reason …`; the rule's warning-severity finding fires
// post-write (warning-only projection does not block, so the verb
// completes).
//
// When G-0166 resolves via path (B) (soften the kernel's verb-time
// chokepoint) the entry is removed and the cell becomes a "naturally
// check-time" case driven without --force. When resolved via path
// (A) (reclassify the cell as VerbTime in the spec) the cell drops
// out of the enumeration entirely.
var ac3ForceBypass = map[string]string{
	// gap.open → addressed with self.addressed_by == "" — verb-time
	// rejection via requireResolverForResolutionClass (G-0096) is
	// gated on `if !force`; the warning-severity finding
	// gap-addressed-has-resolver does not trip projectionFindings'
	// HasErrors check. --force --reason "<rationale>" passes both
	// gates and the verb writes the illegal state, then aiwf check
	// fires the rule post-write.
	"gap-open-promote": "G-0166",
}

func enumerateCheckTimeIllegalCases(t *testing.T) []illegalCase {
	t.Helper()
	var out []illegalCase
	rules := spec.Rules()
	for i := range rules {
		rule := rules[i]
		if rule.Outcome != spec.OutcomeIllegal || rule.RejectionLayer != spec.RejectionLayerCheckTime {
			continue
		}
		out = append(out, illegalCase{name: illegalCaseName(rule), rule: rule})
	}
	return out
}

// runNegativeCheckTimeCell branches on which classification the cell
// falls into (see the test docstring). See the per-branch comments
// for the assertion shape; the unifying invariant is: every
// check-time Illegal cell produces SOME mechanical assertion that
// would fire under reasonable breakage.
func runNegativeCheckTimeCell(t *testing.T, tc illegalCase) {
	t.Helper()
	f := cellcoverage.NewCellFixture(t)
	opts := deriveBringOpts(tc.rule)
	id := bringEntityForCell(t, f, tc.rule, opts)

	evalCtx := spec.EvalContext{}
	for _, p := range tc.rule.Preconditions {
		f.SatisfyPredicate(t, p, id, &evalCtx)
	}

	_, isImplGap := ac3KnownImplGaps[tc.name]
	_, isForceBypass := ac3ForceBypass[tc.name]

	args := buildIllegalVerbArgs(t, tc, id)
	if isForceBypass {
		args = append(args, "--force", "--reason", "AC-3 check-time driver: bypass --force-skippable verb-time guard to exercise the post-write check rule")
	}
	verbOut, verbErr := testutil.RunBin(t, f.Root, "", nil, args...)

	if isImplGap {
		// Staleness teeth (no --force, no check assertion). If the
		// verb succeeds, the impl-gap entry is obsolete and the cell
		// should graduate to end-to-end coverage.
		if verbErr == nil {
			t.Errorf("ac3KnownImplGaps[%q] is stale: verb succeeded without --force, meaning the kernel no longer rejects this cell at verb-time. Remove the entry from ac3KnownImplGaps and the cell will be exercised end-to-end.\nverb output:\n%s", tc.name, verbOut)
		}
		return
	}

	// End-to-end path: verb must succeed (with or without --force),
	// then check must fire the rule's ExpectedErrorCode.
	if verbErr != nil {
		t.Fatalf("aiwf %v: verb returned non-zero for check-time cell\noutput:\n%s\nhint: file an ac3KnownImplGaps entry if this is impl divergence, or add to ac3ForceBypass if --force lets the verb proceed.", args, verbOut)
	}

	checkOut, _ := testutil.RunBin(t, f.Root, "", nil, "check", "--format=json")
	var env render.Envelope
	if err := json.Unmarshal([]byte(checkOut), &env); err != nil {
		t.Fatalf("parsing check envelope: %v\noutput:\n%s", err, checkOut)
	}

	// Assert finding code AND EntityID match — binding the
	// finding to the specific entity we just promoted. A
	// code-only check would pass if any unrelated entity in the
	// fixture happened to have the same finding code, which
	// could mask a real test-shape regression (e.g., the verb's
	// mutation didn't actually take effect on the subject
	// entity). For composite-id entities (AC), id is "M-NNNN/AC-N"
	// and the check rule emits the composite directly; for
	// top-level entities, id is the kind-prefixed id.
	want := tc.rule.ExpectedErrorCode
	if !envelopeHasFindingForEntity(env.Findings, want, id) {
		var got []string
		for i := range env.Findings {
			got = append(got, env.Findings[i].Code+"@"+env.Findings[i].EntityID)
		}
		t.Errorf("expected finding code %q for entity %q in check envelope; got code@entity pairs: %v\nargs: %v\nverb output:\n%s\ncheck envelope:\n%s",
			want, id, got, args, verbOut, checkOut)
	}
}

// envelopeHasFindingForEntity returns true iff some finding in the
// envelope matches BOTH the expected code and the expected entity id.
// Binding both fields prevents an unrelated entity's matching finding
// from making the test pass spuriously.
func envelopeHasFindingForEntity(findings []check.Finding, code, entityID string) bool {
	for i := range findings {
		if findings[i].Code == code && findings[i].EntityID == entityID {
			return true
		}
	}
	return false
}
