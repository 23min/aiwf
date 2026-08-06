---
id: G-0544
title: Wire the contract verbs through the provenance decoration layer
status: open
priority: high
discovered_in: M-0291
---
## What's missing

The contract verbs — `contract bind`, `contract unbind`, `contract recipe
install` and `contract recipe remove` — reach the commit seam without passing
through the provenance-decoration layer. Each registers `--actor` but none
registers `--principal`, and no scope is consulted, so a commit by a non-human
actor carries an actor with nothing naming who it acts for.

`aiwf check` reports those commits at error severity —
`provenance-trailer-incoherent` and `provenance-no-active-scope` — so the push
is blocked. The verb succeeds, the push fails, and nothing between the two says
why.

The audit-only path has the same shape and is worth fixing alongside. A verb
run as `--audit-only` by a non-human actor is refused with "aiwf-principal is
required" *even when the principal was supplied*, because the verb consults the
rule set before the CLI layer decorates the plan. The refusal is correct policy
— audit-only is sovereign, so an agent cannot wield it — but the reason names a
rule the act did not violate, and tells the operator to pass a flag they passed.

## Why it matters

Every other mutating verb states its provenance completely. These four state
half of it and are the only mutating verbs an agent can drive while leaving a
commit no human is accountable for. The failure surfaces at the push, far from
the act, and the operator who meets it has no local signal pointing back at the
verb that produced it.

The gap is a hole in the provenance model rather than a missing convenience: a
commit whose actor is an agent and whose principal is absent is malformed by
the design doc's own definition, and these verbs are the only supported way to
produce one.

## Options

1. **Route the four through the provenance roster.** The consistent end state:
   they would be gated exactly like every other mutating verb. The cost is that
   the gate requires an active authorize scope for a non-human actor, and
   `contract recipe install` is setup-shaped — it can legitimately run before
   any entity exists to authorize against. That makes scope-gating it a real
   behavior change, not a tidy-up.
2. **Register `--principal` on the four, without the scope gate.** Smaller, and
   enough for a non-human actor to state complete provenance. It leaves them
   outside the scope gate the other verbs pass, which is a second inconsistency
   traded for the first.
3. **Declare them human-only.** Honest and cheap, but it removes a capability
   automation legitimately uses, and the force-replace semantics these verbs
   carry have nothing to do with sovereignty.

Option 2 is the lean: it closes the malformed-commit hole, which is the actual
defect, without deciding the larger scope-gating question that option 1 opens.

## Scope

Surfaced by M-0291's wrap review. The seam M-0291 built briefly enforced the
whole coherence rule set and so refused all four verbs outright, with no flag
that could satisfy the guard; the seam was narrowed to the sovereign-force
rules, which restored them and left this hole exactly as it was before.
