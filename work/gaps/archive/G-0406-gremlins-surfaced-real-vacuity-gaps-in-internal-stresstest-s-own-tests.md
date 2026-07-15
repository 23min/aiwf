---
id: G-0406
title: gremlins surfaced real vacuity gaps in internal/stresstest's own tests
status: addressed
addressed_by_commit:
    - c648ef45
---
## What's missing

A partial `mutate-hunt` run against `./internal/stresstest` (see
G-0405 — it never finished, but streamed real per-mutant results
before being canceled) surfaced several `LIVED`/`NOT COVERED` mutants
worth triaging. Sorted by severity:

**1. Real wiring-vacuity gap — `cross_worktree_edit_body_race.go:106`**
(`LIVED CONDITIONALS_NEGATION`): `conflicted := runGit(wtA, "merge",
"--no-edit", "actor-b") != nil`. Flipping this to `== nil` survived
every test. Root cause, confirmed by reading both the classify
function and its tests: `classifyCrossWorktreeEditBodyRace`'s own pure
logic IS tested directly with fabricated `conflicted` bool values
(`cross_worktree_edit_body_race_classify_test.go`) — but the real-
binary integration test
(`TestCrossWorktreeEditBodyRaceScenario_RealBinary_ConfirmsObservableOutcome`)
only asserts `result.Passed` (zero violations), never independently
confirming `conflicted`'s computed value matches what the real merge
actually did. Since a real git conflict-marker file typically contains
*both* operators' draft text, `classifyCrossWorktreeEditBodyRace`'s
"clean merge" branch (checking neither draft's content is present)
would ALSO pass by coincidence when fed a genuinely-conflicted file's
content under a flipped `conflicted` value — the exact same wiring-
vacuity shape M-0250's own `checkListInvariant` and
`ConcurrentMoveScenario.Run` were found and fixed for during E-0062's
wrap review (see that milestone's own Work log / Reviewer notes for
the fix pattern: a stand-in-binary test forcing a real, detectable
divergence).

**2. Counter-assertion weakness — `verb_sequence.go:292` and `:307`**
(`LIVED INCREMENT_DECREMENT` on `s.archiveCounter++` and
`s.moveCounter++`): `TestVerbSequenceScenario_RealBinary_
WalkDispatchesEveryOperation`'s assertions (`if s.archiveCounter == 0
{ t.Error(...) }`) only check "not zero," so a `--` mutation
(producing a negative count after the counter's initial zero value)
would still satisfy the assertion. Low severity — these counters exist
purely for test observability, not production logic — but a one-line
fix (assert `> 0`, not `!= 0`) closes it.

**3. Cosmetic, not logic-affecting — `verb_sequence.go:223:48` and
`compose.go:43:101`** (`LIVED ARITHMETIC_BASE` on `i+1` inside
`fmt.Sprintf`/`fmt.Errorf` label text): both mutate a step/line number
used only in a human-readable message string, never a comparison or
control-flow decision. No test asserts on the exact numeral in these
messages, so nothing catches a mutation here — acceptable as-is, but
worth knowing this diagnostic label's accuracy isn't currently pinned
(matters for the "reproduce without re-running the whole walk" goal
these labels serve, per M-0250/AC-3's own design intent).

**Not actionable** — everything else the partial run reported was
either:
- A `LIVED`/`NOT COVERED` on a line **already** carrying a
  `//coverage:ignore defensive` annotation with a stated rationale
  (`cross_worktree_edit_body_race.go:64`, `force_override_durability.go:134,138`,
  `gitrepo.go:16,124`) — the mutation surviving *confirms* the
  existing human judgment that the branch is genuinely defensive/
  unreachable, not a new finding.
- `NOT COVERED` on a tuning constant's own arithmetic
  (`lock_kill.go:37`, `mid_write_kill.go:42` — both `5 * time.Second`
  timeout literals) — no test reasonably needs to assert on the exact
  duration value.
- `TIMED OUT` mutants (`concurrent_writer_at_scale.go:57`,
  `repeat.go:154`, `verb_sequence.go:195`) — gremlins counts these as
  killed by its own convention (an induced hang the timeout catches),
  not a vacuity gap.

## Why it matters

Finding #1 is a real, if likely low-probability-of-triggering,
correctness-signal gap in a scenario that's specifically designed to
test cross-worktree merge-conflict handling — the one property this
scenario exists to verify is exactly the one its own wiring isn't
independently confirmed against.

## Direction

- Fix #1 with the same pattern already used twice in M-0250: a
  deterministic test that forces the `conflicted` variable to a known
  value independent of the real git merge outcome (or, simpler here,
  a test that fabricates a non-conflicting edit pair alongside the
  existing conflicting one, so both real branches of the wiring get
  exercised end-to-end, not just the classify function in isolation).
- Fix #2 with a one-line assertion strengthening (`> 0` not `!= 0`).
- #3 is optional; low priority.

## Scope

`internal/stresstest/cross_worktree_edit_body_race.go` +
`_test.go`/`_classify_test.go`; `internal/stresstest/verb_sequence_test.go`.