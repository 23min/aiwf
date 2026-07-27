---
id: M-0281
title: Same-state mutating-verb inputs return NoOp
status: in_progress
parent: E-0073
tdd: required
acs:
    - id: AC-1
      title: promote to current status returns NoOp instead of an error
      status: met
      tdd_phase: done
    - id: AC-2
      title: cancel of an already-terminal entity returns NoOp
      status: met
      tdd_phase: done
    - id: AC-3
      title: move to the current parent epic returns NoOp
      status: open
      tdd_phase: red
    - id: AC-4
      title: acknowledge-illegal on an acknowledged SHA is NoOp, appends no duplicate commit
      status: open
      tdd_phase: red
    - id: AC-5
      title: rename to current slug and retitle to current title return NoOp
      status: open
      tdd_phase: red
    - id: AC-6
      title: verb_result_noop_invariant policy pins same-state NoOp across mutating verbs
      status: open
      tdd_phase: red
---

## Goal

Extend the NoOp-on-same-state convention already shipped in `archive`,
`rewidth`, `contract bind`, and `contract recipe install` to the six mutating
verbs that still return a Go error when the requested change already equals
current state. An operator who re-runs `aiwf promote M-… done` — interactively,
or from a forgotten script — should get a clean "already done" no-op at exit 0,
matching the kernel's atomic / single-commit / FSM-policed safety, not a
confusing second-run error. The tradeoff between fail-loud and idempotent
resolves toward convergence here, consistent with the four verbs that already
behave this way.

`acknowledge-illegal` is additionally a **correctness** fix, not only a UX one:
re-running it against an already-acknowledged SHA today appends a duplicate empty
audit commit — the "re-running creates duplicates" smell the scorecard's C2 noted.

The convergence is codified as a policy invariant (AC-6) so it cannot rot back to
one-of as new verbs land. For the field-mutation verbs (`move`, `rename`,
`retitle`, `acknowledge-illegal`) this completes a pattern already live in four
verbs. For the FSM-transition verbs (`promote`, `cancel`) it is a kernel-semantics
change — same-status is reclassified from FSM refusal to a first-class NoOp,
recorded in ADR-0036 — because the one-directional FSM encoded "same-status is
refused" across four correctness surfaces (the FSM, the legal-workflow spec, the
negative-driver, the stresstest oracle) that this milestone updates to model NoOp.

**Design note — the `promote` wrinkle.** `promote` already has a legitimate
same-status *mutation*: when a resolver flag is supplied it sets a resolver
pointer without changing status, deliberately skipping `ValidateTransition`
(`internal/verb/promote.go`). The NoOp guard must therefore fire only when the
target status equals current **and** no other field is changing, so it never
swallows a resolver-pointer write.

**Design note — the seam.** `verb.Result.NoOp` maps to exit 0 with `NoOpMessage`
on stdout; the CLI layer already surfaces NoOp for `archive`/`rewidth`. Each verb
is covered at the CLI seam (drive `run([]string{"<verb>", …})`), not just at the
verb layer, so a verb-layer NoOp that the CLI wiring drops would be caught.

## Acceptance criteria

### AC-1 — promote to current status returns NoOp instead of an error

`promote <id> <current-status>` with no other field changing returns
`Result.NoOp == true` (exit 0, descriptive message), not an
`FSMTransitionError`. The same-status path *with* a resolver flag continues to
perform the resolver-pointer mutation — the guard distinguishes "nothing is
changing" from "status is unchanged but a pointer is being set."

### AC-2 — cancel of an already-terminal entity returns NoOp

`cancel <id>` on an entity already at a terminal status returns
`Result.NoOp == true` instead of the current "already at terminal" error.

### AC-3 — move to the current parent epic returns NoOp

`move <M-id> --epic <epic>` where the milestone is already under that epic returns
`Result.NoOp == true` instead of the current "already under epic" error.

### AC-4 — acknowledge-illegal on an acknowledged SHA is NoOp, appends no duplicate commit

`acknowledge illegal <sha>` against a SHA already acknowledged returns
`Result.NoOp == true` and writes **no** commit — closing the duplicate-empty-
audit-commit path. The assertion checks both the NoOp result and that the commit
count is unchanged.

### AC-5 — rename to current slug and retitle to current title return NoOp

`rename <id> <current-slug>` and `retitle <id> <current-title>` each return
`Result.NoOp == true` instead of their current "matches the current slug" /
"title already" errors.

### AC-6 — verb_result_noop_invariant policy pins same-state NoOp across mutating verbs

`internal/policies/verb_result_noop_invariant.go` asserts, at the AST level, that
every mutating verb in `internal/verb/` has at least one test case that drives it
with same-state input and asserts `Result.NoOp == true`. By-design-additive verbs
(`add`, `authorize-open`, `edit-body --body-file`) are allowlisted, each with a
one-line rationale.

## Decisions made during implementation

- **ADR-0036** — Same-status FSM transitions converge to NoOp, not refusal.
  Surfaced while implementing AC-1: the one-directional FSM classified a
  same-status promote as illegal-and-refused across four correctness surfaces, so
  making it a NoOp is a kernel-semantics decision, not a local guard.

## Work log

### AC-1 — promote same-status returns NoOp
NoOp guard in `verb.Promote` gated on no resolver flag; same-status reclassified
from FSM refusal to a first-class NoOp and the four FSM-legality oracles updated
to model it. commit b0ea0a17 · tests: verb + CLI-seam + negative-driver +
stresstest classify, all green.

### AC-2 — cancel of a terminal entity returns NoOp
`verb.Cancel`'s terminal guard flips from a coded FSM refusal to a NoOp (Option A:
any terminal, not only cancel's cancel-class terminal). The concurrent-milestone-
race oracle now models the cancel NoOp — the "one cancel wins" invariant moves
from ok-count to commit-count. commit 8533f85f · tests: verb (2 new, 2 updated) +
CLI-seam + race oracle (scenario 4×) + classify unit, all green.

