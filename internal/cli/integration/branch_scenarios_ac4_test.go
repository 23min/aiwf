package integration

import (
	"strings"
	"testing"

	"github.com/23min/aiwf/internal/cli/cliutil/testutil"
)

// branch_scenarios_ac4_test.go — M-0159/AC-4: real-git E2E
// coverage of `aiwf acknowledge-illegal` silencing both
// isolation-escape (M-0106) and forced-untrailered
// (fsm-history-consistent subcode) findings. Closes G-0208 +
// G-0214 + G-0196 — the asymmetry where acknowledge-illegal
// silenced illegal-transition but NOT the other two subcodes
// the verb's --help promised it would cover.
//
// AC-4's load-bearing claim per the spec body:
//
//	"acknowledge-illegal extended to cover isolation-escape
//	 AND forced-untrailered subcodes via the shared helper.
//	 Real-git E2E: AI escape → aiwf acknowledge-illegal <sha>
//	 --reason → aiwf check silent; AI authorship preserved on
//	 original commit."
//
// The scenarios fall into three groups:
//
//  1. Isolation-escape silencing (already wired by M-0159/AC-3's
//     lift — gather-layer ackedSHAs flows to RunIsolationEscape).
//     These scenarios pin the E2E end-to-end; they pass on day
//     one of AC-4 but a regression to AC-3's gather-side wiring
//     would surface here.
//
//  2. Forced-untrailered silencing (RED until AC-4 GREEN lands
//     the forcedUntraileredFindings(observations, ackedSHAs)
//     signature change and threads the gather-layer ackedSHAs
//     to it). These scenarios fail today and pass after the
//     GREEN-phase code change.
//
//  3. Author preservation (the AC's "AI authorship preserved on
//     original commit" claim): asserts that after
//     acknowledge-illegal, the original escape/untrailered
//     commit's aiwf-actor trailer still reads "ai/claude" — the
//     no-history-rewrite principle in concrete form.
//
// All scenarios use the real `aiwf acknowledge-illegal` verb;
// no fake fixture for the trailer. The acknowledgment commit's
// aiwf-force-for trailer is what the gather-layer's
// WalkAcknowledgedSHAs picks up; this surface tests the
// production path through to silencing.

// TestBranchScenarios_AC4_AckSilencing drives real-git E2E
// scenarios for the acknowledge-illegal-silences contract.
// Author preservation is verified inline in the Setup
// function for the specific scenario that asserts it
// (the Expectation framework only filters envelope findings;
// author preservation is an out-of-band assertion).
func TestBranchScenarios_AC4_AckSilencing(t *testing.T) {
	t.Parallel()
	RunScenarios(t, []Scenario{
		// AC-4 Group 1: isolation-escape silencing.

		// Isolation-escape acknowledged → silent. The gather
		// layer's WalkAcknowledgedSHAs picks up the
		// aiwf-force-for trailer on the acknowledgment commit;
		// RunIsolationEscape's per-SHA check exempts the
		// original escape's finding. This passes today on the
		// AC-3 lift; the scenario pins it end-to-end so a
		// regression to the gather→consumer wiring surfaces.
		{
			Name: "isolation-escape acknowledged is silent (M-0159/AC-4: E2E for AC-3 lift)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				escapeSHA := SimulateAIEscape(t, env, "E-0001", "AI body edit escaping the bound branch")
				AcknowledgeIllegal(t, env, escapeSHA, "pre-rule era escape; squash-merge collapsed the intermediate steps")
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape"},
		},

		// Isolation-escape acknowledged on a DIFFERENT SHA →
		// the original escape STILL fires. Pins per-SHA
		// closed-set scoping at the E2E level: a regression
		// that silenced any isolation-escape when ANY
		// acknowledgment exists would surface here.
		{
			Name: "isolation-escape NOT acknowledged on its own SHA still fires (M-0159/AC-4: per-SHA scoping E2E)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				escapeSHA := SimulateAIEscape(t, env, "E-0001", "AI body edit escaping the bound branch")
				// Acknowledge a DIFFERENT SHA (the authorize
				// commit's, which isn't even illegal) so the
				// ackedSHAs map is non-empty but doesn't
				// contain escapeSHA. A per-SHA regression
				// (silence on any ack present) would
				// spuriously pass.
				_ = escapeSHA
				authorizeSHA := strings.TrimSpace(env.MustRunGit("rev-parse", "HEAD~1"))
				AcknowledgeIllegal(t, env, authorizeSHA, "ack a different SHA to populate the map without covering the escape")
			},
			Expect: Expectation{FindingPresent: "isolation-escape"},
		},

		// AC-4 Group 2: forced-untrailered silencing.
		// RED until AC-4 GREEN extends forcedUntraileredFindings
		// to consume ackedSHAs.

		// Forced-untrailered acknowledged → silent. After AC-4
		// GREEN, forcedUntraileredFindings receives the
		// gather-computed ackedSHAs and exempts the
		// acknowledged commit. The acknowledge-illegal verb's
		// --help today only names illegal-transition; AC-4
		// closes the asymmetry G-0214 documented.
		//
		// RED state: forcedUntraileredFindings does not consume
		// ackedSHAs, so the predicate still fires on the
		// acknowledged commit. The expectation "NO
		// fsm-history-consistent/forced-untrailered finding"
		// fails today and passes after GREEN lands.
		{
			Name: "forced-untrailered acknowledged is silent (M-0159/AC-4: G-0214 asymmetry closed)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				untrailedSHA := SimulateForcedUntrailedActivate(t, env, "E-0001")
				AcknowledgeIllegal(t, env, untrailedSHA, "fabricated promote was the only way to reach the rule's predicate end-to-end")
			},
			Expect: Expectation{
				NoFindingWithCode: "fsm-history-consistent",
				FindingSubcode:    "forced-untrailered",
			},
		},

		// Forced-untrailered acknowledged on a DIFFERENT SHA →
		// the original untrailered commit STILL fires. Pins
		// per-SHA closed-set scoping for the forced-untrailered
		// path at E2E level.
		//
		// RED state: same as the silencing scenario above,
		// fired in the same direction — today the predicate
		// fires regardless of ack presence, so the
		// "FindingPresent: fsm-history-consistent
		// /forced-untrailered" assertion will PASS today (a
		// spurious pass for the wrong reason — it fires
		// because ackedSHAs isn't consumed, not because
		// per-SHA scoping works). After GREEN lands, the
		// pass becomes the right reason: the ack on a
		// different SHA correctly does NOT silence this one.
		{
			Name: "forced-untrailered NOT acknowledged on its own SHA still fires (M-0159/AC-4: per-SHA scoping E2E)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				untrailedSHA := SimulateForcedUntrailedActivate(t, env, "E-0001")
				_ = untrailedSHA
				// Acknowledge a different SHA so the
				// ackedSHAs map is non-empty but doesn't
				// cover the untrailered commit.
				addSHA := strings.TrimSpace(env.MustRunGit("rev-parse", "HEAD~1"))
				AcknowledgeIllegal(t, env, addSHA, "ack a different SHA to populate the map without covering the offender")
			},
			Expect: Expectation{
				FindingPresent: "fsm-history-consistent",
				FindingSubcode: "forced-untrailered",
			},
		},

		// AC-4 Group 3: author preservation.
		//
		// The AC's "AI authorship preserved on original commit"
		// claim is a no-history-rewrite principle: the
		// acknowledge-illegal verb produces a NEW empty commit
		// carrying aiwf-force-for: <sha> — it does NOT rewrite
		// the original escape commit's trailers, message, or
		// SHA. The original commit's aiwf-actor trailer should
		// still read "ai/claude" after acknowledgment.
		//
		// The Expectation framework only filters envelope
		// findings; author preservation is verified inline
		// in Setup via `git show <sha>` against the original
		// escape commit. A regression that rewrote the
		// original (impossible via the verb today, but the
		// rule needs the pin so a future verb-flag never
		// breaks it) would Fatal here.
		//
		// The Expect on this row asserts the silencing also
		// holds (same as the first scenario), so the row
		// covers both the silencing AND the preservation
		// claim end-to-end.
		{
			Name: "isolation-escape acknowledged preserves AI authorship on the original escape commit (M-0159/AC-4: no-history-rewrite)",
			Setup: func(t *testing.T, env *ScenarioEnv) {
				t.Helper()
				env.MustRunBin("add", "epic", "--title", "Engine")
				OpenBoundScope(t, env, "E-0001", "epic/E-0001-engine")
				escapeSHA := SimulateAIEscape(t, env, "E-0001", "AI body edit escaping the bound branch")
				AcknowledgeIllegal(t, env, escapeSHA, "pre-rule era escape")
				// Verify the original escape commit's
				// aiwf-actor trailer still reads exactly
				// "ai/claude". Structural query via
				// `git log -1 --pretty=%(trailers:...)` —
				// catches both REMOVAL (trailer absent) and
				// OVERRIDE (verb appended a duplicate
				// `aiwf-actor: human/X` trailer leaving the
				// original line in place — would still
				// substring-match `aiwf-actor: ai/claude`
				// but the OPERATIONAL actor is now the
				// override). Per first-reviewer N1 / M-0159/AC-4
				// refactor task #73.
				//
				// `valueonly=true,unfold=true` returns one
				// value per matching trailer separated by
				// newlines; we want exactly one line equal
				// to "ai/claude".
				actorValues := env.MustRunGit("log", "-1",
					"--pretty=%(trailers:key=aiwf-actor,valueonly=true,unfold=true)",
					escapeSHA)
				gotActor := strings.TrimSpace(actorValues)
				if gotActor != "ai/claude" {
					t.Errorf("original escape commit %s aiwf-actor trailer value = %q; want exactly %q (single line) — a removal, addition, or override of the aiwf-actor trailer is a history-rewrite regression",
						escapeSHA, gotActor, "ai/claude")
				}
				// Also pin that the original commit's SHA
				// is still resolvable (object exists). The
				// `git log -1` above would have failed if
				// the SHA was unreachable, so this is
				// implicit; but an explicit `cat-file -t`
				// documents the intent.
				typeOut, _ := testutil.RunGit(env.Root, "cat-file", "-t", escapeSHA)
				if strings.TrimSpace(typeOut) != "commit" {
					t.Errorf("original escape commit %s is not a reachable commit object after acknowledgment; type = %q (history rewrite)",
						escapeSHA, strings.TrimSpace(typeOut))
				}
			},
			Expect: Expectation{NoFindingWithCode: "isolation-escape"},
		},
	})
}
