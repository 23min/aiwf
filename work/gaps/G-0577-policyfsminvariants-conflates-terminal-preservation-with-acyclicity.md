---
id: G-0577
title: PolicyFSMInvariants conflates terminal preservation with acyclicity
status: open
priority: medium
discovered_in: M-0300
---
## What's missing

`PolicyFSMInvariants` guards two different properties with one mechanism. It
runs DFS cycle-detection over every kind FSM and over the `ac-status` and
`tdd-phase` composite FSMs, and its own comment gives the motivating example as
`cancelled → active` — resurrecting a terminal state. Those are not the same
property, and separating them changes what the policy is worth.

Measured by mutating one edge per clean clone and running the full suite:

| Mutation | Class | What fails across the whole repo |
|---|---|---|
| contract `deprecated → accepted` | non-terminal cycle | `TestPolicy_FSMInvariants` and nothing else — 69 of 70 packages pass |
| AC `met → open` | non-terminal cycle | the policy, plus one hardcoded pair-table case |
| epic `active → proposed` | non-terminal cycle | the policy, plus one predicate test |
| epic `done → active` | terminal resurrection | ~15 failures across four packages |
| ADR `superseded → accepted` | terminal resurrection | terminal-set tests, directly |
| epic `cancelled → active` | terminal resurrection | the policy's CancelTarget arm *and* the cycle check |

So terminal resurrection is already covered several times over — `IsTerminal` is
derived and consumed at roughly twenty call sites, and giving a terminal an
out-edge flips its terminality and breaks unrelated tests immediately.
Non-terminal demotion has exactly one guard: this cycle check. The contract case
above is the load-bearing rule R-FP-0042, and nothing but the cycle check
notices when it is violated.

The obvious-looking simplification is therefore backwards, and it is worse than
backwards — it is vacuous. Restating the rule as "no terminal state has an
outgoing edge" cannot fire at all, because `IsTerminal` *derives* terminality
from out-degree, so the restatement reduces to "no state with zero out-edges has
an out-edge". Implemented against the kernel's own predicates and run over all
six mutations above, it reports zero violations — including `cancelled → active`,
the example the policy was written for, since `cancelled` stops being terminal
the instant it gains the edge.

## Why it matters

Any future proposal to add a backward edge to any FSM meets this policy, and the
first instinct is to reshape the rule so the edge passes. The measurement above
is what makes that instinct checkable rather than arguable, and it points the
other way: the acyclicity half is the load-bearing half.

The shape worth adopting instead splits the two:

- **Terminal preservation, bound to a declared set.** Assert `IsTerminal(kind,
  s)` agrees with an explicitly written terminal set, both directions. This is a
  real assertion precisely because terminality is otherwise derived — it becomes
  a drift check between a declared spec and its implementation. A declared set
  already exists in the verb layer's audit-only path and in a terminal-set test
  table; neither is reachable from a policy today, and no AC analogue exists.
- **Acyclicity, kept, with a named allowlist.** Keep the DFS. On a back-edge,
  consult an explicit table of permitted edges, each carrying a rationale and a
  cited decision entity. Default stays "no cycles", and each exception is one
  reviewed, greppable line — the pattern this repo already uses for coverage
  ignores, comment-history exceptions, and the NoOp-exempt verb list.

Exempting the sub-element FSMs wholesale is the wrong resolution: it would admit
`deferred → open` and `cancelled → open` on acceptance criteria, and
`IsTerminalACStatus` drives the AC cancel path's convergence guard, so flipping
a terminal silently changes what `aiwf cancel` does with no test to catch it.

Separately, the policy's own comment cites a commitment that lives elsewhere
than stated, and the vertex list it builds duplicates the entry states — the
duplicate is absorbed by the visited-colour check and is harmless, but the
"vertices visited in the order given" contract in the cycle detector's doc is
looser than it reads.
