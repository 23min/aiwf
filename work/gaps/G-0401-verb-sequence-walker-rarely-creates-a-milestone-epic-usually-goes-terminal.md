---
id: G-0401
title: 'verb-sequence walker rarely creates a milestone: epic usually goes terminal'
status: open
discovered_in: M-0250
---
## What's missing

`internal/stresstest/verb_sequence.go`'s `Run` walks one entity per
kind, in `entity.AllKinds()` order (epic, milestone, ADR, gap,
decision, contract). The milestone is created with `--epic <the
just-walked epic>`, so if that epic ends its own walk on a terminal
status (`done`/`cancelled`), milestone creation is refused and this
scenario's own G-0398 tolerance (`isEpicAlreadyTerminalRefusal` /
`isEpicAlreadyArchivedRefusal`) skips it for that run — a known,
accepted outcome, not a violation.

Empirically, at the walker's own realistic step counts (6, matching
`internal/stresstest/verb_sequence_test.go`'s existing pinned seeds;
12, `cmd/stresstest/registry.go`'s registered default), the epic ends
terminal — and the milestone gets skipped — for every seed checked.
Root cause: `stepPromote` draws its target status uniformly from the
kind's *full* closed status set (`entity.AllowedStatuses`), not just
the legal-from-current subset (deliberately, so FSM-illegal targets
get exercised too per M-0241/AC-1). For an epic, that status set is
`{proposed, active, done, cancelled}` — from `proposed`, a single
promote draw has a 1-in-4 chance of landing directly on `cancelled`;
from `active`, a 2-in-4 chance of landing on a terminal status. Over
even a handful of promote draws, the cumulative probability of
reaching terminal is high.

## Why it matters

M-0250/AC-2 extended the walker's per-kind operation table with
`move`, `archive`, `rename`, and `retitle` — but `move` is only ever
selectable when the milestone entity exists (it's the only kind
`verb.Move` accepts). If the milestone is skipped nearly every real
run, `move`'s practical exercise inside `cmd/stresstest run
--scenario verb-sequence` is far rarer than the scenario's own step
count would suggest — the AC's literal bar ("reachable with nonzero
probability during a walk of realistic length") is met and pinned by
a dedicated unit test, but the *scenario's own effectiveness* at
actually catching a `move`-shaped bug in real usage is diminished by
this coupling.

## Direction

Worth a real decision, not assumed. Candidate directions, roughly in
order of invasiveness:

- Create the milestone *before* fully walking the epic (interleave
  their walks, or create the milestone right after the epic's own
  creation and only then walk both), so the milestone's own existence
  doesn't depend on the epic surviving its full walk non-terminal.
- Bias `stepPromote`'s target draw away from terminal statuses when
  another kind's creation still depends on the current entity staying
  non-terminal (adds real complexity — the walker would need to know
  about cross-kind dependencies it currently doesn't model at all).
- Accept the coupling as a property of "one entity per kind, walked
  independently" and instead give `move` its own dedicated concurrency
  scenario the milestone's AC-4 already provides — meaning `move`'s
  real coverage comes from that dedicated scenario, not primarily from
  the sequential walker, and this gap becomes lower priority.

## Scope

Whichever direction is chosen: `internal/stresstest/verb_sequence.go`
(`Run`'s per-kind loop and/or `stepPromote`'s target-selection logic),
plus updated tests confirming the milestone is reached with
meaningfully higher probability at the registered scenario's real
step count.