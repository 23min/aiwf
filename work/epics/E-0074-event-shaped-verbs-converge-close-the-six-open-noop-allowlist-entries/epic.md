---
id: E-0074
title: 'Event-shaped verbs converge: close the six OPEN NoOp allowlist entries'
status: proposed
---
## Goal

Finish the same-state convergence M-0281 started. That milestone converged twelve
operator-facing verbs and shipped
`internal/policies/verb_result_noop_invariant.go` to keep the convention from
rotting — but it left six allowlist entries marked OPEN, each an explicit IOU
rather than a by-design exemption. The policy therefore claims a discipline it
does not yet enforce everywhere, which is the same half-rolled-out condition
M-0281 was itself filed against.

Definition of done is **zero OPEN entries in that allowlist** — but nothing asserts
that today: the allowlist's `Reason` strings are read by no test, so "no OPEN
entries" is currently a grep over a comment. Making the bar real is part of this
epic's scope, not a precondition it inherits. And an entry rewritten with a
by-design reason rather than a NoOp assertion changes no behavior and adds no test,
so the rewrite branch needs its own evidence bar to satisfy the
mechanical-evidence rule.

Addresses G-0458, G-0459 and G-0460.

## Scope

The six OPEN entries. Only one ordering edge is real — G-0460 before the `Authorize`
entry; the other four are independent and can land in any order or in parallel.

**G-0460 gates the `Authorize` entry only.** A repeat
`aiwf authorize <id> --to <agent>` exits 0, appends a second commit, and leaves two
simultaneously-active scopes with no check finding. The allowlist entry says
convergence may be the wrong fix, because a second grant can be a legitimate new
event.

What has to be decided is narrower than the entry implies. Multiple parallel active
scopes are already defined behavior: `docs/design/provenance-model.md` states that a
human may hold several at once and that the kernel resolves a match to the
most-recently-opened scope deterministically, and `verb.Allow` implements exactly
that. So resolution is not ambiguous and there is no missing invariant to establish.
The open question is whether an *exactly-duplicate* re-grant is a distinct event or
a same-state input — which is answerable directly, and does not gate the other five
entries.

**G-0459** — four of the six entries. One open question runs through them: what the
duplicate guard keys on. `acknowledge illegal`'s existing guard keys on the SHA
alone, so a re-run with a *corrected* `--reason` is silently discarded (measured).
For the `--audit-only` trio, whose entire payload is `--reason`, "mirror that guard"
is therefore a decision rather than a mechanical port. Five event-shaped verbs append a fresh
record on an identical re-run: `acknowledge mistag`, `authorize --to`,
`promote --audit-only`, `promote <id>/AC-N --phase --audit-only`, and
`cancel --audit-only`. `acknowledge illegal` already received a HEAD-walk duplicate
guard in M-0281; these did not. `acknowledge mistag` is the closest analogue —
`check.WalkAcknowledgedMistags` already walks HEAD for exactly the relevant
commits, so the detection capability exists and is unused. The `--audit-only` trio
needs a guard keyed on an existing audit *record*, because their precondition is
that the entity already sits at the target state.

**Last, G-0458** — `promote <id>/AC-N --phase <same-phase>` refuses via the
TDD-phase FSM. This is a decision, not a defect: the phase ladder is audit-bearing
evidence and the verb carries a `--tests` payload, so convergence needs a
deliberate metrics carve-out rather than a mechanical repeat. Resolve by converting
with that carve-out, or by rewriting the allowlist entry with a by-design reason.
It goes last because it cannot be decided by implementing it.

## Out of scope

- **Write-scope preconditions** — whether a verb should run at all against a
  frontmatter-dirty entity (G-0463, G-0466), tracked as E-0075. Same layer,
  different axis, kept separate so a broad precondition change does not ride along
  with mechanical dedup work.

  One coupling is not separable, though, and it runs the other way: E-0075's first
  decision is where its precondition sits relative to the same-state comparison, and
  `PromoteACPhase` — G-0458's target — writes frontmatter, so it falls inside
  E-0075's route list and under that decision. E-0075's decision should be settled
  before this epic writes code.
- **Prelude error-envelope uniformity** (G-0456). Shares the word "uniformity" and
  nothing else.
- **Rewriting duplicate records already in history.** These guards prevent new
  duplicates; existing commits stay as the record of what happened. This does *not*
  exclude detection: G-0460 asks for a check rule so an already-divergent tree is
  reported rather than silently carried, and that belongs in scope.
