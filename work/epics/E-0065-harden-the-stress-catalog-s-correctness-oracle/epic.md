---
id: E-0065
title: Harden the stress catalog's correctness oracle
status: done
---

## Goal

Close the blind spots identified in G-0410: broaden the stress catalog's
"`aiwf check` must stay clean" oracle beyond `verb-sequence`, and add a
concurrent-race mode that exercises promote/cancel/AC operations against
shared entity state, so a missing domain-specific verb-time guard can't ship
silently.

## Context

E-0062 built the correctness stress harness and its scenario catalog: real
git/real-process scenarios, each judged by a deterministic pass/fail oracle.
G-0410, discovered near that epic's close, found two structural gaps in that
oracle coverage. First, `verb-sequence`'s generic "no check finding beyond a
curated baseline" oracle only catches a regression in a domain-specific
promote guard when a companion check-rule backstop also exists — confirmed
empirically against G-0335, where the pre-fix binary passed 30/30
`verb-sequence` runs despite a real bypass of the open-AC cancel guard.
Second, the concurrency-focused scenarios (`parallel-branch-reallocate`,
`cross-worktree-id-race`, `force-override-durability`,
`promote-on-wrong-branch-detection`, `reachability-isolation`) each assert on
exactly one scenario-specific finding code rather than "no unexpected
finding," so a check-rule regression surfacing as a side effect of one of
those scenarios would go unnoticed even though `aiwf check` ran. Third, no
scenario contests a single shared entity via concurrent promote/cancel/AC
operations — the existing concurrency scenarios
(`concurrent-writer-at-scale`, `concurrent-move`, `concurrent-id-allocation`)
each race distinct entities per actor, never a shared one.

A sibling gap, G-0400, raised the stress catalog's raw verb-coverage breadth
(several verbs are never exercised by any scenario). That's explicitly out of
scope here — those verbs are used infrequently enough that broader coverage
isn't a current priority.

## Scope

### In scope

- Broaden the "check must stay clean beyond a curated baseline" oracle —
  mirroring `verb-sequence`'s `verbSequenceExpectedWarnings` pattern — onto
  the scenarios whose end-state is expected to be a coherent, loadable tree:
  `parallel-branch-reallocate`, `cross-worktree-id-race`,
  `force-override-durability`, `promote-on-wrong-branch-detection`,
  `reachability-isolation`, `archive-during-active-scope`,
  `cross-worktree-edit-body-race`, `concurrent-move`,
  `concurrent-writer-at-scale`, `concurrent-id-allocation`. Each scenario
  gets its own curated baseline, not a shared one.
- Add a concurrent-race mode that points N walkers at shared entity state
  instead of isolated disposable repos, reusing `concurrent-writer-at-scale`'s
  subprocess fan-out harness, with oracle logic that distinguishes a
  legitimate race outcome (one actor wins an FSM-legal transition, the other
  observes a legal rejection) from an actual guard violation.

### Out of scope

- G-0400's raw verb-coverage breadth (`rename-area`, `set-area`, `rewidth`,
  `import`, `worktree-add`, the `contract-*` verbs, and the read-only /
  administrative verbs) — deferred, not a current priority.
- `disk-fault`, `lock-kill`, `mid-write-kill`, `head-drift` — these
  scenarios' whole point is a torn-write or crash-recovery intermediate state
  that `aiwf check`'s vocabulary doesn't model; broadening the check-clean
  oracle onto them would be a category error.

## Constraints

- Reuse existing harness primitives (`concurrent-writer-at-scale`'s goroutine
  + subprocess fan-out pattern) rather than building a new concurrency
  mechanism from scratch.
- Each scenario's expected-warnings baseline is derived empirically from
  repeated `make stress` runs, not guessed or copied wholesale from
  `verb-sequence`'s baseline.

## Success criteria

- [x] G-0410 is closed.
- [x] Every scenario listed in the *In scope* broadening list above asserts
      its own curated check-clean baseline instead of a single pinned
      finding code.
- [x] A stress scenario exists that races concurrent promote/cancel/AC
      operations against shared entity state, with an oracle that
      distinguishes a legitimate race from a guard violation.
- [x] Re-running the new concurrent scenario against a synthetic
      G-0335-shaped regression (a domain-specific guard removed with no
      check-rule backstop) fails the run, demonstrating the blind spot
      G-0410 identified is closed.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| What exact criteria distinguish a "legitimate race outcome" from a "guard violation" for the concurrent-mode oracle? | yes | Resolved during the concurrent-mode milestone's design, informed by observed FSM-legal race outcomes across the target entity kinds. |

## Milestones

- `M-0257` — Broaden the check-clean oracle across ten stress scenarios · depends on: —
- `M-0258` — Race concurrent promote/cancel/AC operations against shared entity state · depends on: `M-0257`

## References

- G-0410 — stress catalog can't detect a missing domain-specific promote guard
- G-0400 — stress scenario catalog exercises only 10 of 38 aiwf verbs (raw verb-coverage breadth, deferred)
- E-0062 — Correctness stress harness (prior epic; this epic hardens its scenario catalog's oracle)
