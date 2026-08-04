---
id: G-0534
title: Sovereign-dispatcher guard predicate is satisfied by a flag-help string
status: open
---
## What's missing

`PolicySovereignDispatchersGuardHumanActor` decides a dispatcher is guarded when
its function body contains `human/`, `actorIsNonHuman`, or `HasPrefix(actor`. The
first of those is a bare substring match over the body text, which includes every
string literal in it — and a flag-help string is a string literal.

Measured across the four dispatchers the policy examines, all four pass on a
help string and nothing else:

| dispatcher | the only `human/` in the body |
| --- | --- |
| `internal/cli/add/add.go` | `--principal` flag help |
| `internal/cli/cancel/cancel.go` | `--principal` flag help |
| `internal/cli/promote/promote.go` | `--principal` flag help |
| `internal/cli/authorize/authorize.go` | `--actor` flag help |

Not one has a human-actor guard in code. The policy is green because the verbs
document their flags, not because they enforce anything.

## Why it matters

The rule the policy is named for — a sovereign act traces to a named human — is
enforced today, at the verb layer: `internal/verb` refuses an `aiwf-force`
trailer from a non-human actor, and `provenance-force-non-human` backs that at
check time. So nothing is currently violated. What is absent is the
dispatcher-level assertion, at the layer where `--actor` is parsed.

The predicate makes that absence self-perpetuating. Any new sovereign verb that
declares a `--principal` or `--actor` flag inherits a passing verdict from the
flag's help text, so the policy admits the dispatchers most likely to need the
guard. It can only fire on a dispatcher that never types the actor-shape prefix
anywhere — including in prose — which is not the shape a real verb takes.

The scope defect this splits from was the acute form: the policy examined no
files. This is the milder one, and it survives review more easily, because the
policy now examines the right files and reports success.

## Options

1. **Narrow the predicate to code references** — match an identifier or a
   comparison against the actor value rather than any occurrence in the body
   text, then resolve whatever fires. Restores the assertion at the layer the
   policy claims. Costs a guard in four dispatchers, and raises the question
   below before any of them can be written.
2. **Delete the policy** and rest on the verb-layer refusal plus
   `provenance-force-non-human`. Defensible — the property is enforced, and a
   chokepoint that cannot fail is worse than an absent one — but it gives up
   defense-in-depth at the layer parsing `--actor`, and deleting a named
   provenance chokepoint deserves a recorded decision rather than a quiet
   removal.
3. **Keep the predicate and document what it proves** — that the dispatcher
   mentions the actor shape, not that it enforces it. Cheapest, and it makes the
   policy honest about a weaker claim than its name implies. The name would have
   to change with it.

No lean recorded. The choice turns on the open question below, which is a design
question about layering rather than about this predicate.

## Open question

What is the dispatcher layer supposed to assert that the verb layer does not
already? Until that is answered, option 1's four guards have no specified
content and option 2 cannot be justified as redundancy. Settle it first;
the predicate follows from the answer.

## Scope

Split from G-0476 while fixing its scope defect. Rescoping the walk from
`cmd/aiwf/` to `internal/cli/` took the policy from examining zero files to
examining ninety, and the measurement above is what that rescope revealed: the
policy still cannot fail for a realistically-shaped dispatcher. G-0476 closes on
the scope; this carries the predicate.
