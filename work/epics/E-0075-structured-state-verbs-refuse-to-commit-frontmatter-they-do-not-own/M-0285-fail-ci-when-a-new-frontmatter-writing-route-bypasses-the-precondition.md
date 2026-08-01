---
id: M-0285
title: Fail CI when a new frontmatter-writing route bypasses the precondition
status: done
parent: E-0075
depends_on:
    - M-0284
tdd: required
acs:
    - id: AC-1
      title: A new frontmatter-writing route that bypasses the seam fails CI by name
      status: met
      tdd_phase: done
    - id: AC-2
      title: The invariant's exemption list carries one reasoned entry per exempt route
      status: met
      tdd_phase: done
---

## Goal

Make the precondition's coverage a mechanical property rather than a convention,
so a frontmatter-writing route added later cannot quietly bypass the seam.

## Context

M-0283 and M-0284 route the verbs that exist today. Nothing stops the next one
from being written without the guard, and a coverage rule that depends on a
reviewer noticing is the condition this repo holds is not a guarantee at all.

The precedent is already in the tree and is a close fit.
`internal/policies/projection_findings_presence.go` asserts that every verb
either calls `projectionFindings` or appears on a reviewed allowlist with its own
specific reason, and it fails CI on a verb falling outside both — the design is
documented at the top of `internal/verb/verb.go`. This milestone builds the same
shape for the write-scope seam.

One known limitation to design around rather than inherit:
`verb_result_noop_invariant.go`, M-0281's analogous policy, scans *exported*
entry points, so an unexported branch is invisible to it. That is a real gap in
the model being copied, not a detail of its implementation.

## Approach

Follow `projection_findings_presence.go`: presence check plus a reviewed
one-entry-per-exemption allowlist, failing CI on anything outside both. Write the
firing fixture first — a route shaped like a real one that bypasses the seam —
so red is a live state rather than a formality, which is also what the
firing-fixture meta-gate expects of every policy's construction line.

The exemption list is authored from the calls M-0283 and M-0284 already recorded,
not re-derived. Where those milestones wrote down a reason for a route being out,
that reason is what the allowlist entry carries.

## Acceptance criteria

### AC-1 — A new frontmatter-writing route that bypasses the seam fails CI by name

A route that writes entity frontmatter without passing through the precondition
fails the policy suite, and the finding names the route so the failure is
actionable without reading the policy's source.

Scope coverage is part of the claim, not a separate concern: the analogous
same-state policy scans only exported entry points, so an unexported branch
escapes it. This invariant covers the routes it claims to cover, or states in its
own doc comment which shapes it cannot see.

### AC-2 — The invariant's exemption list carries one reasoned entry per exempt route

Every exempt route has its own entry with its own specific reason — not a shared
category label, and not a bare route name. The reasons come from the calls
M-0283 and M-0284 recorded, so the allowlist and the milestone bodies say the
same thing.

An entry whose reason has stopped being true is the failure mode worth guarding:
this mirrors what E-0077 is doing for dormant `dupl` exemptions, where an
exemption outliving the condition it exempts is treated as a defect rather than
as harmless.

## Constraints

- Every reachable branch of the policy is exercised, and the policy's own
  construction line is covered by a fixture that makes it fire — the
  firing-fixture meta-gate requires this of every policy, and a chokepoint that
  never fires in test is the thing it exists to prevent.
- The exemption list is authored from M-0283's and M-0284's recorded calls.
  Re-deriving reasons here would fork them from the bodies that decided them.
- The invariant is kernel-internal. It adds no consumer-facing surface and so
  needs no skill, `--help` text, or completion wiring.

## Design notes

- `projection_findings_presence.go` is the shape to copy; `verb.go`'s package
  doc is the shape to copy for how the exemption reasons get written down.
- `verb_result_noop_invariant.go`'s exported-only scan is a known limitation of
  that policy, not of the model. This invariant sees exported and unexported
  package-level functions alike, which is what puts the four composite-id routes
  inside it. The shape it does not see is a method, and such a row is reported as
  unvouchable rather than passed over.

## Out of scope

- Any change to the guard itself. If the invariant reveals an unrouted verb, that
  is a finding against M-0283 or M-0284, not new design here.
- After-the-fact detection of laundering already in history (G-0480). This
  invariant governs the source tree; that rule governs the commit log.

## Dependencies

- M-0284 — the sweeps and nested paths must be routed, or the invariant fails
  against them on the day it lands.
- M-0283 — the seam itself.

## References

- E-0075 — the parent epic
- `internal/policies/projection_findings_presence.go` — the precedent shape
- `internal/policies/verb_result_noop_invariant.go` — M-0281's analogue, and its
  exported-only limitation
- `internal/verb/verb.go` — where the projection-findings exemption reasons live
- E-0077 — the dormant-exemption problem this milestone's AC-2 anticipates


## Work log

### AC-1 — A new frontmatter-writing route that bypasses the seam fails CI by name

`PolicyClaimGuardPresence` requires every claim-scope row naming a file to reach
the wrapper that scope names, above the convergence it protects · commit 244ef84f7,
strengthened by 1e582ab28 · tests 41/41

### AC-2 — The invariant's exemption list carries one reasoned entry per exempt route

The exempt rows are checked for a distinct reason that is not the route's own name,
and for dormancy — an exemption that has since wired a guard · same two commits

## Decisions made during implementation

- **The invariant derives its subject from `noOpClaimScopes` rather than carrying a
  list of its own.** AC-2 asked for an exemption list authored from what the prior
  milestones recorded; reusing that ledger satisfies it exactly rather than
  approximately, and a second list would fork the reasons it was meant to preserve.
- **The requirement names the two wrappers, not the shared comparison they
  delegate to.** Both reach `guardClaimPaths`, which differs only in its
  `exemptAbsentFromHEAD` argument — and that argument is what the scope decides.
  Targeting the shared function would let a direct call apply the aiwf.yaml
  exemption to an entity claim, which is the reproduction the guard exists to
  refuse.
- **Placement is checked, not only presence.** A call-graph edge cannot say where
  the guard sits. A guard below the converging return is present on every edge and
  never runs on the input it exists to refuse.

## Validation

- `make check-fast` — exit 0 (race suite, lint, vet)
- `make coverage-gate` — exit 0 (diff-scoped coverage audit, firing-fixture meta-gate)
- `aiwf check` — 0 errors
- Mutation probe, two rounds. Round one: 7 mutations to the helpers, all caught.
  Round two, after the review: 10 mutations including the wiring and two applied to
  `internal/verb` itself — the guard moved below its converging return, and a direct
  `guardClaimPaths(..., true, ...)` call — all caught. Both rounds restored the
  files byte-identical.

## Deferrals

- G-0506 — `PromoteACPhase` computes its FSM refusal from working-copy bytes with no
  claim guard. Reproduced with a real binary. Outside this milestone per its own
  Out of scope: a finding against the milestones that built the seam.
- G-0507 — the guard's *effectiveness* (paths actually passed, same-value-over-divergent
  behaviour per verb) is pinned only by a hand-maintained table a new verb does not join.
- G-0508 — four policies now carry near-copies of the `internal/verb` scan, and they
  have already drifted.
- G-0509 — the epic's user-visible refusal is absent from `CHANGELOG.md`.

## Reviewer notes

**AC-1's `met` preceded the evidence that now backs it.** The status was promoted on
a suite whose every test drove the policy's helpers directly, plus a live assertion
that the policy reports no violations. That combination is satisfied by a policy
consulting nothing: measured, disconnecting the ledger *and* deleting a real verb's
guard left the suite green. The seam tests in the corrective commit close it — the
same disconnection now fails three of them. The AC FSM has no `met → open` edge and
`--force` is a human-only act, so the record is this note rather than a status
change; the claim is true as of 1e582ab28.

**AC-2's phase ladder was stamped in one burst.** Both criteria are one policy file
covered by one test batch: the tests were written and watched to fail before any
implementation existed, which is readable in the file, but only AC-1's `red` was
stamped live. AC-2's ladder went in afterwards and is indistinguishable in
`aiwf history` from one back-stamped after the fact.

**The scan's bounds are disclosed in the policy's doc comment rather than closed.**
It sees package-level functions, so a converging method is reported as unvouchable
rather than checked — an interlock that holds only because the sibling ledger's scan
does see methods, now pinned by a test and a load-bearing comment. It reads
call-shaped edges, so a call in a dead branch or to a shadowing local counts. And it
does not evaluate the paths a guard receives.

**Design review recommended collapsing `callGraph` into `verbFunc`;** declined here.
The change is sound but touches the same four scan copies G-0508 covers, and doing
it piecemeal inside this milestone would leave the general condition open while
looking addressed.

**The walker's matching was corrected as part of this work.** It compared raw source
text, which matched `applyDirRename(` as a call to `Rename` — so a verb consulting
nothing could read as reaching whatever `Rename` reaches. Measured across all
package-level functions under `internal/verb`, the AST replacement reports no reach
the text scan did not, and two allowlisted verdicts changed from reaching to not,
which makes their exemptions do real work.
