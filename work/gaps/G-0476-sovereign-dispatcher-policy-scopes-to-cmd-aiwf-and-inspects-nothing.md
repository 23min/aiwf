---
id: G-0476
title: Sovereign-dispatcher policy scopes to cmd/aiwf and inspects nothing
status: open
discovered_in: E-0075
---
## What's missing

`PolicySovereignDispatchersGuardHumanActor` asserts that every CLI dispatcher
parsing a sovereign act guards on a human actor. It scopes its walk to `cmd/aiwf/`.
That directory now contains only `main.go`; every verb dispatcher lives under
`internal/cli/<verb>/` after the M-0115 relocation. The policy therefore inspects
nothing and cannot fire.

Measured: a synthetic guardless dispatcher naming both `"force"` and `"reason"` —
the policy's own trigger for the sovereign-FSM-bypass meaning — added to
`internal/cli/promote/promote.go`, where the real `promote` dispatcher lives, leaves
`TestPolicy_SovereignDispatchersGuardHumanActor` passing.

Seven files under `internal/cli/` declare a `"force"` flag today, including the
`promote`, `cancel` and `authorize` dispatchers the policy names in its own doc
comment as the surfaces it protects. None is examined.

## Why it matters

`--force` is sovereign: the kernel's rule is that a forced act requires a human
actor, and the repo treats that as one of its load-bearing provenance properties.
The verb layer does enforce it — `internal/verb` refuses an `aiwf-force` trailer
from a non-human actor, and `provenance-force-non-human` backs that at check time —
so the property is not currently violated. What is missing is the dispatcher-level
assertion the policy was written to provide, at the layer where `--actor` is parsed.

A vacuously-passing policy is worse than an absent one. It appears in the enforced
set, its test is green, and a reader auditing which sovereign surfaces are covered
finds a policy named for exactly that job. This is the condition G-0264 identified
for dormant `forbidigo` config and fixed with a firing test: a chokepoint that
detects nothing while reporting success.

## Options

1. **Rescope the walk to `internal/cli/`** and re-run. The likely outcome is that
   the policy fires on real dispatchers, so this is a fix plus whatever it finds —
   which is the point, and should be measured before deciding how much work it is.
2. **Add a firing fixture** so the policy proves it can fail, independently of the
   scope fix. This is what would have caught the vacuity when the dispatchers moved,
   and it generalizes: any policy whose path scope is a string prefix can be
   silently orphaned by a relocation.
3. **Delete the policy** and rely on the verb-layer refusal plus
   `provenance-force-non-human`. Defensible — the property is enforced elsewhere —
   but it gives up the dispatcher-level check at the layer that parses the actor,
   and it should be an explicit decision rather than a consequence of the policy
   having quietly stopped working.

Options 1 and 2 together are the lean, with 2 first: prove the policy can fail
before trusting what it says about the wider scope.

## Scope

Surfaced while reviewing E-0075, whose fourth decision concerns adding `--force` to
verbs that lack it — which made the state of the sovereign-act chokepoint worth
checking.

Worth checking as part of this: whether any other policy scopes its walk to a path
prefix that a relocation has since emptied. The failure mode is not specific to this
one.
