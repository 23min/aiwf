---
id: D-0060
title: Scope the apply-seam guard to the sovereign-force rules
status: proposed
relates_to:
    - M-0291
---
## Question

M-0291 put the trailer-coherence guard at `verb.Apply` and had it enforce the
whole coherence rule set, on the reasoning that the rule set is what the
function checks and that widening cost nothing in reach — the history-walking
check already reported all but one rule at error severity, so a set newly
failing at the seam was one the push already rejected.

The blast radius was recorded as zero because no existing test broke. That was
true about the suite and false about the behavior: the suite never exercised a
contract verb under a non-human actor.

So: which rules belong at a verb-time refusal?

## Decision

Enforce only the rules predicated on a force trailer —
`force-non-human`, `force-with-on-behalf-of` and `audit-only-with-force` — in
`CheckSovereignForceCoherence`. `CheckTrailerCoherence` keeps the full set for
its existing callers. The commit seam refuses sovereign acts; every other way a
trailer set can be incoherent stays the push's business.

## Reasoning

A refusal has to be satisfiable. The four contract verbs — `contract bind`,
`contract unbind`, and both recipe verbs — never pass through the
provenance-decoration layer, so they carry no `aiwf-principal`, and none of them
registers a flag that could supply one. Enforcing the whole rule set closed all
four to non-human actors with no invocation that could pass, which the
milestone's own constraints forbid. That is not a refusal, it is a dead end.

The subset is principled rather than convenient, and the principle is
satisfiability: a rule belongs at the seam only if every verb that can reach the
seam has some invocation that satisfies it. The force rules qualify because a
verb emitting no force trailer satisfies them vacuously. The principal rules do
not, for the reason measured above. A trailer set incomplete for a reason no
invocation could fix is still malformed — but it is malformed in a way the push
already reports and the operator can already act on.

Satisfiability rather than sovereignty, deliberately. Audit-only is sovereign
too — the kernel says so in its own refusal message and in its audit
catalogue — and `audit-only-non-human` is *not* at the seam, because it fails
the same satisfiability test for the same reason. A criterion of "the sovereign
rules" would have given the wrong answer for the very next rule, and it would
have been the author of that rule who discovered it.

The criterion also survives its own repair. Once G-0544 gives the contract
verbs a way to state a principal, the principal rules become satisfiable and
could move to the seam without re-deriving any of this.

Three alternatives lost.

- **Give the contract verbs a `--principal` flag.** This closes the real hole,
  and it is the lean recorded in G-0544 — but it is a provenance-policy change
  that deserves its own branch and review, not one improvised at a wrap.
- **Route them through the full provenance roster.** The most consistent end
  state, and the most expensive: the roster's gate requires an active authorize
  scope for a non-human actor, and `contract recipe install` legitimately runs
  before any entity exists to authorize against.
- **Accept the closure and amend the constraint.** Rejected outright. The
  constraint reserved these verbs from the *force* rule, and the guard was not
  refusing them on a force rule — it was refusing them on provenance
  completeness, a hole that predates this milestone. Amending a constraint to
  match a regression records the regression as intent.

The narrowing loses one thing worth naming: `principal-missing-for-non-human`
was the seam's only backstop for the import path, which skips the allow-gate.
It was never enforced there before this milestone either, so nothing regresses —
but the seam no longer covers it.

## Follow-ups

- G-0544 — wire the contract verbs through the provenance decoration layer, so
  a non-human actor can state complete provenance on them.
