---
id: D-0020
title: M-0161/AC-5 cell-5 orphan acknowledgment deferred to verb extension
status: accepted
relates_to:
    - M-0161
    - G-0205
    - G-0226
---
## Context

M-0161/AC-5 (G-0205) landed `WalkOrphanedAICommits` + `RunOrphanedAICommits` at commit `4ece6f6a`, surfacing AI-actor commits orphaned by non-fast-forward updates on ritual branches as the new `isolation-escape-orphaned-ai-commit` warning.

The AC-5 body's matrix at lines 354-364 enumerates 7 cells. Cell 5 reads:

> "Force-push orphans AI commit, then `aiwf acknowledge-illegal <sha>` → silent (override takes effect)"

And AC-5 body line 349 asserts the composition:

> "The existing `aiwf acknowledge-illegal <sha>` verb silences the warning per its existing mechanism (writes an empty commit with `aiwf-force-for: <sha>` + human actor + reason); no new override path needed."

This composition is **false at the time of AC-5 landing**.

`aiwf acknowledge-illegal` hard-requires the target SHA to be reachable from HEAD via `git merge-base --is-ancestor <sha> HEAD`. A force-push orphan is, by construction, unreachable — that's the definitional property the new rule names. The verb refuses with exit 2 ("SHA … is not reachable from HEAD") on every orphan.

Evidence: the RED test `TestForcePushOrphan_AC5_Matrix/AC-5_cell_5:_force-push_orphans_AI_commit_+_aiwf_acknowledge-illegal_→_silent` failed during the AC-5 cycle with exactly that refusal. The full transcript shows the orphan's SHA, the verb's stderr message, and the exit code.

The M-0161/AC-2 reviewer pass discipline (subagent, M-0160 lineage) caught this asymmetry at AC-5 wrap and recommended the deferral be recorded as a mid-flight architectural decision rather than left as an in-test comment that risks rotting.

## Decision

**Defer cell-5 ack-composition coverage to a verb-design cycle. AC-5 ships without the composition; the AC-5 body's matrix row for cell 5 carries a deferral note pointing at [G-0226](../gaps/G-0226-aiwf-acknowledge-illegal-hard-requires-sha-reachable-from-head.md).**

Three resolution paths are documented in G-0226; the choice belongs to a future cycle that scopes the verb-side surface explicitly:

1. **`--allow-unreachable` flag on existing verb.** Simplest path; preserves the verb's single-surface promise. Sovereign-gated (`--reason` + human actor).
2. **`aiwf acknowledge-orphan <sha>` new verb.** Lifts the reachability constraint cleanly; keeps the existing verb's noisy-on-typo behavior. Adds a second surface.
3. **Rewrite the AC-5 body's composition claim.** Treat orphans as unsilenceable by design; accept indefinite operator-discipline-only handling.

Rationale for deferring rather than choosing now:

1. **The verb's reachability check is correct for its original use case.** Loosening it requires reasoning about every caller, not just AC-5. A verb-design cycle scopes that reasoning; AC-5 does not.
2. **No active operational burden.** The af1051d1-class kernel-repo orphan is a single warning, non-blocking. The rule's coverage of the broader force-push-orphan class is intact (it fires correctly on every orphan); only the ack composition is unavailable.
3. **AC-5's core deliverable stands.** The detection mechanism — the load-bearing surface — works. The composition is a separable affordance; deferring it does not invalidate the rule.
4. **YAGNI on the choice.** Until a real operator hits this and prefers one resolution over another, the design space stays open. Choosing now would hard-code a guess.

## Concrete sequencing

- **AC-5 wrap (now):** record this decision; file G-0226; update the AC-5 body matrix row for cell 5 to point at the gap; rewrite the in-test carve-out comment with the real D and G ids.
- **A future verb-design cycle:** pick path 1, 2, or 3 from G-0226; land the implementation; re-add the cell-5 E2E scenario; update the AC-5 body matrix and the M-0161 Decisions section.
- **The kernel-repo's af1051d1 orphan:** remains as residual debt until the verb extension lands. The warning is non-blocking; documented in the AC-5 commit message and in G-0226's "Observed in" section.

## Why not the alternatives

- **Alternative A: pick path 1 (`--allow-unreachable`) now and land it under M-0161/AC-5.** Rejected — AC-5's scope is the detection rule; adding a verb-extension under the same AC conflates two design surfaces and pushes M-0161 wrap further out. The decision belongs to a verb-design cycle.
- **Alternative B: leave the deferral as an in-test comment only, no D-NNN.** Rejected per CLAUDE.md §"AC promotion requires mechanical evidence" + the milestone's Decisions-section discipline. An in-test comment rots; a D-NNN survives.
- **Alternative C: amend the AC-5 body to remove cell 5 entirely from the matrix and the composition claim.** Rejected — the matrix row exists for a real concern (operators will eventually need to silence orphans). Removing it loses that signal. Pointing at the gap preserves the deliverable surface while making the unavailability explicit.

## References

- [G-0226](../gaps/G-0226-aiwf-acknowledge-illegal-hard-requires-sha-reachable-from-head.md) — the verb-side gap this defers to
- M-0161/AC-5 (G-0205) — the cycle that surfaced the dependency
- M-0161/AC-5 commit `4ece6f6a` — the substantive feat commit landing the detection rule
- AC-5 body line 349 — the false-composition claim
- AC-5 body line 362 — the cell-5 matrix row this decision routes to G-0226
- [`internal/cli/integration/isolation_escape_force_push_scenarios_test.go`](../../internal/cli/integration/isolation_escape_force_push_scenarios_test.go) lines 114-135 — the in-test carve-out comment (updated post-decision to cite D-0020 + G-0226)
- [`internal/verb/acknowledgeillegal.go`](../../internal/verb/acknowledgeillegal.go) — the verb whose reachability check is the chokepoint
- D-0019 — AC-3's oracle-coverage decision; D-0020 follows the same fail-shut-on-correctness pattern by surfacing the gap rather than silently failing
