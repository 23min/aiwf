---
id: G-0546
title: Verb trailer sets are completed by the CLI layer after the verb returns
status: open
discovered_in: M-0291
---
## What's missing

A verb's provenance trailer set is incomplete when the verb returns. The CLI
layer appends the principal, on-behalf-of, authorized-by and scope-ends
trailers to the plan afterwards, so `internal/cli/cliutil` mutates a
`verb.Plan` the verb layer produced. A second group of verbs assembles a
complete set itself and never passes through that decoration.

Two assembly shapes therefore exist for one concept, and the only point
downstream of both is the commit seam.

## Why it matters

The split is a layering inversion: the lower layer's value object is completed
by the higher one. Its practical cost is that no verb can validate its own
provenance — a guard placed inside a verb would see no principal and refuse
every legitimately authorized agent. That is why sovereign-force enforcement
sits at the commit seam rather than where the operator's request still exists,
and it is why a refusal there speaks in trailer keys rather than in flags the
operator typed.

The seam is the right structural backstop regardless. What the split costs is
the *option* of refusing earlier, with a message phrased in the operator's own
terms.

## Options

1. **Move provenance into the verb layer.** The verb owns a complete trailer
   set; the CLI passes provenance in rather than decorating after. This touches
   every dispatcher's plan construction and has to relocate the scope-loading
   git-log walk, so it is epic-sized, not milestone-sized.
2. **Add force to the allow-gate.** Narrower: the allow-gate already lives in
   the verb layer, already receives actor, principal and scopes, and already
   returns coded errors. Wiring force into it would let the refusal name
   `--force` and the actor at the point of request, leaving the commit seam as
   the backstop it should be. Roughly the size of M-0291's own change.
3. **Leave it.** The seam holds; only the message quality and the layering are
   affected.

Option 2 is the lean if this is picked up on its own — it buys most of the
benefit for a fraction of option 1's reach.

## Scope

Named by the design lens of M-0291's wrap review as the deeper smell behind
that milestone's seam choice, and recorded here rather than as a justification
inside the ADR — the current shape is correct, but it is correct given this
constraint, not independently of it.
