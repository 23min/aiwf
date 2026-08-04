---
id: ADR-0040
title: Prevent sovereign acts at the verb route, ratify at the history route
status: accepted
---
## Context

The kernel states across its guidance, its design docs and its own code
comments that `--force` is human-only, and enforced it in one verb of four. A
non-human actor could force an epic to `active` — the canonical sovereign act —
at exit 0, and learn of the violation only when the pre-push check walked git
history. By then the act was in the log, and no verb could clear it.

The guard existed. `CheckTrailerCoherence` was reached from two verbs and no
others. It did not drift there by inattention: a verb's trailer set is
incomplete when the verb returns. The CLI layer appends `aiwf-principal`,
`aiwf-on-behalf-of`, `aiwf-authorized-by` and `aiwf-scope-ends` to the plan
afterwards, so a guard placed inside a verb sees no principal and refuses every
legitimately authorized non-human actor. The two verbs that could call it are
absent from the provenance roster and assemble a complete set themselves.

Two trailer-assembly shapes therefore exist, and they meet at exactly one point
downstream of both: `verb.Apply`.

ADR-0038 faced a structurally similar question about verb writes and needed two
seams — one claim-side, one commit-side — because a converging verb returns
before a plan exists and so never reaches the commit path. That reasoning does
not carry here: a converging verb writes no commit, so it emits no trailer, so
it has no coherence to violate. The case that forced the second seam there
cannot arise.

## Decision

The sovereign-force coherence guard runs at `verb.Apply`, ahead of any
filesystem work, and refuses before anything is written.

Sovereign acts are **prevented at the verb route and ratifiable at the history
route.** Both halves are load-bearing:

- **The verb route is closed.** Every site constructing a sovereign
  `aiwf-force` trailer refuses a non-human actor before committing. Refusal
  leaves `HEAD` where it was; a guard that reports after writing produces
  exactly the record this closes.
- **The history route stays open to ratification.** The check rule walks git
  history and fires on commits no verb produced — an imported repo, a
  hand-crafted commit, history predating the guard. Those cannot be prevented
  retroactively, so they are ratified instead, by a human, with a written
  reason, in a separate commit that leaves the acknowledged one untouched.

The guard sits in `verb.Apply` rather than in the CLI layer because the
self-assembling verbs reach `Apply` without passing through the dispatcher
layer, so a CLI-layer placement would leave exactly the paths this closes.
Placing it there also makes no-bypass structural rather than policed: any
caller reaching `Apply` is covered without being enumerated.

## Consequences

The guard enforces the rules predicated on a force trailer, not the whole
coherence rule set. The scope is load-bearing rather than incidental: a verb
whose trailer set is incomplete for a reason unrelated to force may have no
invocation that could complete it. The contract verbs are the measured case —
they never pass through the provenance-decoration layer and register no flag
that could supply a principal, so a seam enforcing the whole set closed all
four to non-human actors outright. Sovereignty is what this seam exists to
enforce, and force is what makes an act sovereign; everything else a trailer
set can get wrong is the push's business.

Two of the three rules in that subset are backstops rather than live paths.
`audit-only-with-force` sits behind a flag mutex that refuses `--force`
alongside `--audit-only` as a usage error, before a plan exists to carry
either. `force-with-on-behalf-of` needs a set carrying force and on-behalf-of
without a non-human actor, which the decoration layer does not assemble — it
adds on-behalf-of only for a non-human actor inside a scope. Both stay in the
subset because the seam also sees trailer sets no dispatcher built.

The rule order is chosen, not inherited. A non-human actor reaches this seam
only through an active scope, and a scope always carries `aiwf-on-behalf-of`,
so whichever force rule sits first decides the sentence the operator reads.
`force-non-human` is checked first, so the refusal names what the operator did
wrong instead of a pair of trailer keys they never typed. A caller proving that
force is enforced should still assert the refusal rather than the rule name:
the order is a deliberate choice about the message, and a future rule could
change which one speaks first without weakening the guarantee.

A refusal here is a legality refusal, not an internal failure. It exits with
the findings code and carries the same finding identifier `aiwf check` reports
for the same act once it has landed, so one consumer routes on a denial without
needing to know which of the two moments produced it.

Refusals move from push time to verb time. Automation that was forcing as a
non-human actor was already blocked at push, so no working pipeline breaks —
but a pipeline that treated the verb's exit code as success and the push as a
separate concern now fails a step earlier.
