---
id: G-0067
title: wf-tdd-cycle is LLM-honor-system advisory; no mechanical RED-first guard
status: open
discovered_in: M-0066
---

## What's missing

A mechanical chokepoint that enforces TDD's red-first ordering and branch-coverage HARD RULE so the kernel's correctness stops depending on LLM behavior under load. Today both rules live only inside the `wf-tdd-cycle` skill — advisory text the assistant is asked to follow. When session context grows or an AC's setup is heavy, the assistant drifts off the discipline and nothing notices: tests get written after the implementation; defensive branches ship untested; the cycle ends green on output but red on process.

Concrete failure mode demonstrated on M-0066/AC-1: the assistant wrote `internal/check/entity_body.go` (~165 lines, full per-kind dispatch + heading locator + emptiness logic) before any test existed. Mid-stream the assistant noticed and corrected by neutering the implementation, confirming the 7 firing subcases failed red against the empty stub, then restoring. Output was sound (10 tests pass, lint clean). Process was broken. Branch-coverage audit then declared "the file-read error path, the malformed-Split path, the unknown-Kind path, and two `scanACBodies` clearing arms are untested but follow the existing `acs.go` pattern" — which is a false rationalization: the pattern in `acs.go` is itself a violation of the HARD RULE, not an acceptable precedent.

Candidate mechanisms (rough, ordered cheap to invasive):

- **Branch-coverage subagent**: `wf-tdd-cycle`'s audit step delegates to a fresh subagent with only the diff and the AC contract. Subagent enumerates branches, points at tests, returns a list. Bounded scope, fresh eyes, no investment in the impl just written. Fastest path; addresses the specific M-0066/AC-1 failure.
- **AC-scope cap as planning discipline**: when an AC's expected diff is >~50 lines of impl, split into smaller ACs. The atomic unit of TDD is the cycle, not the milestone; large ACs naturally drift off red-first regardless of how strict the skill is.
- **`aiwf promote --phase green` runs the test suite and refuses on any new test that doesn't fail-then-pass**: the kernel checks that some commit between the AC's add and the green-promote contains a test that, if run against the parent of the impl commit, would fail. Real chokepoint, mechanical, but expensive (runs tests at promote time) and language-specific.
- **`aiwf-red-commit:` trailer on the AC**: `aiwf promote --phase red --commit <SHA>` records the SHA of the failing-test commit; promote-to-green refuses unless that SHA is reachable from the parent of the green commit. Pins ordering as a deliberate act without checking content shape.
- **Dedicated TDD-cycle subagent for the whole cycle** (most invasive): every `wf-tdd-cycle` invocation runs in a fresh subagent context with only the AC contract; main agent gets back the diff + test results. Kills the long-session-drift failure mode entirely; coordination overhead and warmup cost are real.

## Why it matters

CLAUDE.md's load-bearing principle is "the framework's correctness must not depend on the LLM's behavior." The wf-tdd-cycle skill — and behind it the entire `tdd: required` AC discipline — currently does. When the LLM follows the skill, things work; when the LLM drifts, the standing checks (`aiwf check`, `acs-tdd-audit`) confirm the AC reached `met` with `phase: done` but cannot tell that the test was written after the implementation, or that defensive branches were never exercised. The AC-7 of M-0069 (G-0040 follow-up) sized this concern at the structural level — the audit is "tests exist for a `done` AC" not "tests preceded the implementation" — but didn't propose a chokepoint.

Beyond M-0066: every future `tdd: required` milestone is exposed to the same drift, and the larger the AC the more likely the drift. M-0067/AC-1 was small enough that red-first happened naturally; M-0066/AC-1 was big enough (per-kind dispatch, fixture suite per kind, hint registration) that it didn't. As the kernel's check rules grow more elaborate, the "AC big enough to drift" boundary will be hit more often, not less.

A second compounding angle is the branch-coverage HARD RULE specifically. The rule's failure mode is silent: an untested defensive branch passes the AC's promote because `aiwf check` doesn't see coverage. That branch can ship subtly wrong (returning early on a path the test suite never exercises) and the standing check never catches it. The discipline-debt accumulates exactly where the kernel's other guards are blindest.

This gap is the prerequisite for trusting `tdd: required` as a guarantee rather than as aspiration. Until it's addressed, a `met` AC under `tdd: required` means "the LLM said it followed the discipline," not "the discipline mechanically held."
