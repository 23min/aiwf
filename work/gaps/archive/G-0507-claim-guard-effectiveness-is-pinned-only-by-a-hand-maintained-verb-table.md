---
id: G-0507
title: Claim-guard effectiveness is pinned only by a hand-maintained verb table
status: wontfix
discovered_in: M-0285
---
## What's missing

The claim-side guard can be present and ineffective, and nothing mechanical
distinguishes that from present and working.

Two shapes, both measured against the live tree:

- A guard passed no paths. `guardClaim(ctx, t.Root, id)` with an empty variadic
  compares nothing and satisfies `PolicyClaimGuardPresence`, which reads a
  call-graph edge rather than the argument expression.
- A verb whose same-value path is untested. Only `set-priority`, `promote` and
  `cancel` have a test driving a same-value request over a HEAD-divergent file.
  `TestEveryGuardedVerb_DivergentTarget_Refuses` exercises the non-converging path
  only, so it passes whether or not the guard runs before the convergence.

What stands in for both is the hand-maintained table in
`internal/verb/claim_divergence_guard_test.go`. A newly added verb does not join it.

## Why it matters

The guarantee E-0075 ships is that a verb refuses rather than answering from bytes
no verb wrote. A guard comparing an empty path set satisfies every structural check
while delivering none of that, and the verb reports "already set; nothing to change"
exactly as it did before the epic.

`PolicyClaimGuardPresence` states both bounds in its doc comment, so this is a known
limit rather than a surprise. The gap is that the limit is currently held by a table
someone has to remember to extend.

## Scope

Make the per-verb same-value-over-divergent case mechanical rather than enumerated:
derive the verb list from `noOpClaimScopes` so a new guarded row without a test
fails, rather than listing verbs by hand.

Whether an empty path set can be caught structurally is a separate question — the
argument is an expression, and evaluating it needs more than the AST scan the policy
does today. A runtime assertion inside `guardClaimPaths` may be the cheaper answer.

## References

- M-0285 — `PolicyClaimGuardPresence`, whose doc comment states both bounds
- ADR-0038 — the claim-side seam
