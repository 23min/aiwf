# Epic wrap — E-0062

**Date:** 2026-07-11
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0062-correctness-stress-harness
**Merge commit:** a0b43436

## Milestones delivered

- M-0240 — Harness skeleton: driver, scenario interface, streaming report (merged 23065ee8)
- M-0241 — Property sequences and multi-worktree contention scenarios (merged 99569f8f)
- M-0242 — Fault injection via external observation (merged 8b71dc5a)
- M-0243 — Named scenarios from G-0212 and G-0269 (merged cc440aae)
- M-0244 — Concurrent-writer test at scale; triage process (merged f41bc4d6)
- M-0249 — Scenario registry: wire cmd/stresstest run to the real catalog (merged 238275af)
- M-0250 — Register the verb-sequence walker; extend it to move/archive/rename/retitle (merged 592520f8)

## Summary

Built an on-demand, real-git/real-process stress harness (`cmd/stresstest` + `internal/stresstest`) that drives the compiled `aiwf` binary as a subprocess against disposable repos, covering four correctness-risk mechanisms: true simultaneity (goroutine/process fan-out racing `repolock`), divergent worktrees reconciled later, fault injection via external observation, and a sequential FSM random walk. The catalog grew from a driver skeleton with a single placeholder scenario to 14 registered, real scenarios selectable via `cmd/stresstest run --scenario <name>|all`, each with a deterministic pass/fail oracle. Every milestone listed above landed with mechanical AC evidence; two milestones (M-0249, M-0250) closed follow-up gaps discovered mid-epic rather than deferring them.

## ADRs ratified

- (none)

## Decisions captured

- D-0033 — M-0240 driver: bespoke cmd/stresstest, not go test -tags=stress
- D-0034 — DAG-scoped acknowledge-illegal exemption trades off against rebase durability
- D-0035 — Diagnostic-log passthrough plus a resumable cursor, not a scalar correlation id

## Follow-ups carried forward

- G-0212 — data-loss audit for verb composition across the kernel surface: M-0243 converted its named scenarios into real harness coverage; the broader audit (explicitly scoped as "future epic" work in its own title) remains open beyond that.
- G-0269 — mutating verbs lack a HEAD-drift guard against shared-worktree session races: the harness's `head-drift` scenario is deliberately expected-red until this guard ships (confirmed still red in a fresh 15-repeat run at wrap time) — it's a live regression trap for G-0269's own fix, not a harness defect.
- G-0398 — `aiwf add milestone` accidentally-not-purposefully refuses under a terminal epic: a second symptom (the archived-parent variant) surfaced during M-0250's own work and is now tolerated by the harness's verb-sequence walker, but the underlying aiwf precondition this gap asks for is still unbuilt.
- G-0400 — stress scenario catalog exercises only 10 of 38 aiwf verbs: M-0250 closed part of this (list/archive/rename/retitle/move all gained coverage), but its own open questions (import, contract sub-verbs, worktree add) were deliberately left for a future milestone.
- G-0401 — verb-sequence walker rarely creates a milestone because the epic's own random walk usually reaches a terminal status first, making `move`'s practical exercise inside the sequential walk rarer than intended; a real design question for whoever picks this up next.

Closed during this wrap (fully resolved by landed work, not carried forward): **G-0397** (cmd/stresstest run had no way to select any real scenario — resolved by M-0249's registry, commit 930d391e) and **G-0399** (VerbSequenceScenario wasn't registered — resolved by M-0250/AC-1, commit 59f00c89).

## Handoff

The harness is real, on-demand, and covers the four mechanisms the epic scoped. What's ready for whoever plans the next stress-testing work: G-0400's remaining verb gaps (import, contract sub-verbs, worktree add) and G-0401's walker-effectiveness question are the two natural next-epic candidates. G-0398's own aiwf-side fix (a dedicated precondition on `add milestone`/`import` against a terminal parent epic) is a separate, smaller unit of work that doesn't need a new epic.
