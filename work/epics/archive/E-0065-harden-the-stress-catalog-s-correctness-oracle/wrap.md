# Epic wrap — E-0065

**Date:** 2026-07-15
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0065-harden-the-stress-catalog-s-correctness-oracle
**Merge commit:** 545e89fc

## Milestones delivered

- M-0257 — Broaden the check-clean oracle across ten stress scenarios (merged b3215932)
- M-0258 — Race concurrent promote/cancel/AC operations against shared entity state (merged 6045c452)

## Summary

Closes the two structural gaps G-0410 identified in the stress catalog's correctness oracle. M-0257 generalized `verb-sequence`'s "no check finding beyond a curated baseline" pattern (via a shared `classifyAgainstBaseline` helper) onto the ten scenarios that previously asserted only a single pinned finding code, each deriving its own baseline empirically, and added a synthetic regression test proving the broadened oracle actually catches a reintroduced finding. M-0258 added the concurrent-race mode the epic's success criteria called for: `ConcurrentMilestoneRaceScenario` races real subprocess actors against one shared milestone+AC via promote/cancel, with a two-signal oracle (`classifyMilestoneRaceOutcomes`) that distinguishes a legitimate race from a guard violation by outcome-shape/refusal-reason plus real commit-order causality — the signal that actually catches the G-0335 shape, since final state alone cannot. AC-3 proved the oracle's value empirically: a disposable, isolated `git worktree` copy with both the verb-time guard and its check-rule backstop removed reliably surfaces the regression (~13–19/30 attempts across three independent measurements), with zero false positives against the healthy binary. Scope held to the epic spec as planned — no scope shifted mid-flight, the one open question (defining "legitimate race" vs. "guard violation") resolved during M-0258's own design per the epic spec's own resolution path.

## ADRs ratified

- none

## Decisions captured

- none — the one candidate mid-flight decision (a narrow `internal/verb` exit-code consistency fix `classifyMilestoneRaceOutcomes` depends on) was judged, in M-0258's own Work log, not to rise to ADR/D-NNN level: a mechanically necessary fix with its rationale already captured in the commit message and the milestone spec, not an architectural choice.

## Follow-ups carried forward

- G-0414 — stale test naming in `promote-on-wrong-branch-detection`'s real-binary test (discovered during M-0257's review; deferred, not blocking)
- G-0400 — stress scenario catalog exercises only 10 of 38 aiwf verbs (pre-existing, explicitly out of scope for this epic per its own spec)

## Doc findings

Scoped `wf-doc-lint` pass over the full 48-commit, 35-file change-set: no findings — zero files under `docs/`, `README.md`, or `CONTRIBUTING.md` intersect this epic's change-set.

## Handoff

The stress catalog's correctness oracle is now broad (ten scenarios' own curated baselines, not one shared pinned code) and deep (a concurrent-race mode with a commit-order-causality oracle, proven against a real regression). G-0410 is closed. What's deliberately left open: G-0400's raw verb-coverage breadth remains a separate, lower-priority initiative; the `disk-fault`/`lock-kill`/`mid-write-kill`/`head-drift` scenarios remain outside the check-clean-oracle broadening by design (their whole point is a state `aiwf check`'s vocabulary doesn't model). No new scenario or harness work is pending from this epic.
