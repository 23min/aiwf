package integration

import (
	"strings"
	"testing"
)

// branch_scenarios_ac5_test.go — M-0159/AC-5: real-git E2E
// coverage of `aiwf acknowledge-illegal` silencing
// trailer-verb-unknown findings, end-to-end through the gather→
// consumer wiring. Converts the docstring promise at
// internal/check/trailer_verb_unknown.go:25-29 — "Promotion to
// error is contingent on cleaning history first (potentially
// via `aiwf acknowledge-illegal` for the few intentional
// historical strays, if any)" — into mechanical truth.
//
// AC-3's helper-lift already wired RunTrailerVerbUnknown to
// consume the gather-layer-computed ackedSHAs map (the third
// concrete consumer of WalkAcknowledgedSHAs, alongside
// FSMHistoryConsistent's illegal-transition + forced-untrailered
// subcodes and RunIsolationEscape). AC-3 also pinned the
// unit-level signature consumption in
// internal/check/trailer_verb_unknown_ack_test.go.
//
// AC-5's contribution is the REAL-GIT E2E proof: the same
// silencing semantics observed against a real binary, real git
// history, and the production gather→consumer path — not the
// rule-in-isolation unit shape. A regression to AC-3's lift that
// stopped threading ackedSHAs to RunTrailerVerbUnknown would
// pass the unit tests (which call the function directly with the
// map) but break here. That seam coverage is the AC-5
// load-bearing claim.
//
// Per the M-0159 test discipline ("real-git integration test
// under internal/cli/integration; the test builds aiwf via
// buildAiwfBinary, sets up a real git repo via tempRepo, runs
// verbs as subprocess invocations, and asserts stdout/stderr/
// exit-code/trailers/envelope output"), all scenarios drive the
// real `aiwf check --format=json` against fixtures fabricated
// via raw git + the real `aiwf acknowledge-illegal` verb.
//
// The scenarios fall into four groups:
//
//  1. Baseline positive control — stray-verb commit without any
//     acknowledgment fires the rule. Pins that the rule actually
//     reaches the fixture; a green-on-day-one regression that
//     silently filtered every stray would surface here.
//
//  2. Silencing happy path — stray-verb commit + acknowledge-
//     illegal on that exact SHA → check silent. The AC's core
//     claim.
//
//  3. Per-SHA scoping — stray-verb commit + acknowledge-illegal
//     on a DIFFERENT SHA → original stray STILL fires. A
//     regression to "silence on any ack present" (instead of
//     per-SHA) would pass group 2 and silently over-exempt every
//     stray; this scenario discriminates.
//
//  4. No-history-rewrite — after acknowledgment, the original
//     stray commit's `aiwf-verb: <fabricated>` trailer still
//     reads the original fabricated value. Mirrors AC-4's group
//     3 author-preservation pin: the verb produces a NEW empty
//     commit; it does NOT rewrite the offending commit's
//     trailers, message, or SHA.

// TestBranchScenarios_AC5_TrailerVerbUnknownAckSilencing drives
// the four scenario groups described in the file header. All
// scenarios use the real `aiwf acknowledge-illegal` verb; no
// fake trailer fabrication for the acknowledgment commit.
func TestBranchScenarios_AC5_TrailerVerbUnknownAckSilencing(t *testing.T) {
	t.Parallel()
	RunScenarios(t, []Scenario{
		// AC-5 Group 1: baseline positive control.

		// Stray-verb commit with NO acknowledgment fires the
		// rule. Pins that RunTrailerVerbUnknown actually reaches
		// the fixture under the gather→consumer wiring; a future
		// regression that silently filtered the commit before the
		// rule saw it would surface here.
		//
		// Severity assertion (FindingSeverity: "warning") pins
		// the rule's landing-severity claim at
		// trailer_verb_unknown.go:25-29 — "warning at landing
		// time so the rule introduces without retroactive
		// breakage of existing fabricated trailers in history."
		// A future PR that flipped warning→error without an
		// explicit decision would surface here.
		{
			CellID: "branch-cell-m0159-ac5-c1",
			Name:   "trailer-verb-unknown without acknowledgment fires as warning (M-0159/AC-5: baseline positive control)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				SimulateStrayVerbCommit(t, env, "implement",
					"hand-rolled feat commit with fabricated aiwf-verb")
			},
			Expect: Expectation{
				FindingPresent:  "trailer-verb-unknown",
				FindingSeverity: "warning",
			},
		},

		// AC-5 Group 2: silencing happy path.

		// Stray-verb commit + acknowledge-illegal on its own SHA
		// → check silent. The gather layer's WalkAcknowledgedSHAs
		// picks up the aiwf-force-for: <strayedSHA> trailer on
		// the acknowledgment commit; RunTrailerVerbUnknown's
		// per-SHA check exempts the original stray's finding.
		// This is the AC-5 core claim and the load-bearing
		// truth-conversion for the docstring promise at
		// trailer_verb_unknown.go:25-29.
		{
			CellID: "branch-cell-m0159-ac5-c2",
			Name:   "trailer-verb-unknown acknowledged is silent (M-0159/AC-5: docstring promise mechanical truth)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				strayedSHA := SimulateStrayVerbCommit(t, env, "implement",
					"hand-rolled feat commit with fabricated aiwf-verb")
				AcknowledgeIllegal(t, env, strayedSHA,
					"intentional historical stray; cleaning trailer-verb-unknown noise per docstring promise")
			},
			Expect: Expectation{NoFindingWithCode: "trailer-verb-unknown"},
		},

		// AC-5 Group 3: per-SHA scoping.

		// Stray-verb commit + acknowledge-illegal on a DIFFERENT
		// SHA → original stray STILL fires. Pins per-SHA closed-
		// set scoping at the E2E level: a regression that
		// silenced any trailer-verb-unknown finding when ANY
		// acknowledgment exists in HEAD's history would spuriously
		// pass group 2 and silently over-exempt every stray; this
		// scenario discriminates.
		//
		// The "different SHA" we acknowledge is HEAD~1 at the
		// moment of the ack — the commit immediately preceding the
		// stray. That commit is a legitimate `aiwf add` (or `aiwf
		// init`) commit; acknowledging it is semantically
		// meaningless but populates the ackedSHAs map with a
		// non-empty value that doesn't cover the stray.
		{
			CellID: "branch-cell-m0159-ac5-c3",
			Name:   "trailer-verb-unknown NOT acknowledged on its own SHA still fires (M-0159/AC-5: per-SHA scoping E2E)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				// strayedSHA is deliberately NOT used in the ack
				// below — the whole point of this scenario is
				// that the stray's SHA is left OUT of the
				// ackedSHAs map. The capture-and-discard makes
				// the omission explicit to a reader: the SHA
				// exists, we just chose not to ack it. Per
				// reviewer R3 / refactor-phase note.
				strayedSHA := SimulateStrayVerbCommit(t, env, "implement",
					"hand-rolled feat commit with fabricated aiwf-verb")
				_ = strayedSHA
				// Acknowledge a different SHA (HEAD~1, the
				// commit immediately preceding the stray) so the
				// ackedSHAs map is non-empty but doesn't contain
				// the stray's SHA. A per-SHA regression (silence
				// on any ack present) would spuriously pass.
				priorSHA := strings.TrimSpace(env.MustRunGit("rev-parse", "HEAD~1"))
				AcknowledgeIllegal(t, env, priorSHA,
					"ack a different SHA to populate the map without covering the stray")
			},
			Expect: Expectation{
				FindingPresent:  "trailer-verb-unknown",
				FindingSeverity: "warning",
			},
		},

		// AC-5 Group 4: no-history-rewrite.

		// After acknowledgment, the original stray commit's
		// `aiwf-verb:` trailer still reads the original
		// fabricated value. Mirrors AC-4's group 3 author-
		// preservation pin: the acknowledge-illegal verb
		// produces a NEW empty commit; it does NOT rewrite the
		// offending commit's trailers, message, or SHA.
		//
		// The Expectation framework only filters envelope
		// findings; the trailer-preservation check is an out-of-
		// band assertion inline in Setup, structurally querying
		// the original commit's aiwf-verb trailer via `git log -1
		// --pretty=%(trailers:...)`.
		//
		// `valueonly=true,unfold=true` returns one value per
		// matching trailer separated by newlines; we want exactly
		// one line equal to the fabricated value. Catches both
		// REMOVAL (trailer absent) and OVERRIDE (verb appended a
		// duplicate `aiwf-verb: ...` trailer leaving the original
		// line in place — which would still substring-match the
		// fabricated value but the OPERATIONAL verb would be
		// different). Per M-0159/AC-4 refactor task #73's
		// structural-assertion discipline.
		//
		// The trailer-preservation check alone has a gap: a
		// hypothetical branch-rewrite-with-replacement (the verb
		// rewriting the branch to drop the original commit and
		// add a "fixed" version with the same content but
		// different trailers) would leave the original object in
		// the DB, so `git log -1 <strayedSHA>` still succeeds and
		// shows the original's trailers — silently passing the
		// preservation query even though the BRANCH state shifted.
		// The reachability assertion below closes that gap: the
		// stray's SHA must still be an ancestor of HEAD.
		// `merge-base --is-ancestor` exits 0 if ancestor / 1 if
		// not; MustRunGit Fatals on non-zero exit. Per reviewer T1.
		//
		// The Expect on this row asserts the silencing also
		// holds (same as group 2), so the row covers both the
		// silencing AND the trailer-preservation claim end-to-
		// end.
		{
			CellID: "branch-cell-m0159-ac5-c4",
			Name:   "trailer-verb-unknown acknowledged preserves fabricated aiwf-verb trailer on original stray (M-0159/AC-5: no-history-rewrite)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				strayedSHA := SimulateStrayVerbCommit(t, env, "implement",
					"hand-rolled feat commit with fabricated aiwf-verb")
				AcknowledgeIllegal(t, env, strayedSHA,
					"intentional historical stray; cleaning trailer-verb-unknown noise per docstring promise")
				// Verify the original stray commit's aiwf-verb
				// trailer still reads exactly "implement". A
				// removal, replacement, or duplicate-append would
				// indicate a history-rewrite regression.
				verbValues := env.MustRunGit("log", "-1",
					"--pretty=%(trailers:key=aiwf-verb,valueonly=true,unfold=true)",
					strayedSHA)
				gotVerb := strings.TrimSpace(verbValues)
				if gotVerb != "implement" {
					t.Errorf("original stray commit %s aiwf-verb trailer value = %q; want exactly %q (single line) — a removal, addition, or override of the aiwf-verb trailer is a history-rewrite regression",
						strayedSHA, gotVerb, "implement")
				}
				// Reachability: the original stray must still be
				// an ancestor of HEAD. If a hypothetical buggy
				// verb rewrote the branch to drop the original
				// (object stays in the DB but no ref points at
				// its line of history), the trailer query above
				// would spuriously pass while the BRANCH state
				// silently shifted. MustRunGit Fatals on non-zero
				// exit, so a non-ancestor result fails the test
				// loudly. Per reviewer T1.
				env.MustRunGit("merge-base", "--is-ancestor", strayedSHA, "HEAD")
			},
			Expect: Expectation{NoFindingWithCode: "trailer-verb-unknown"},
		},
	})
}
