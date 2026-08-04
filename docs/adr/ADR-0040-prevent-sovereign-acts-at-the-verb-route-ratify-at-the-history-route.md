---
id: ADR-0040
title: Prevent sovereign acts at the verb route, ratify at the history route
status: proposed
---
## Context

The kernel states in five places that `--force` is human-only, and enforced it
in one verb of four. A non-human actor could force an epic to `active` — the
canonical sovereign act — at exit 0, and learn of the violation only when the
pre-push check walked git history. By then the act was in the log, and no verb
could clear it.

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
not carry here, and the reason is worth stating rather than leaving to be
re-derived: a converging verb writes no commit, so it emits no trailer, so it
has no coherence to violate. The case that forced the second seam there cannot
arise.

## Decision

The trailer-coherence guard runs at `verb.Apply`, ahead of any filesystem work,
and refuses before anything is written.

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
self-assembling verbs and the cell-coverage fixture reach `Apply` without
passing through the dispatcher layer. Placing it there also makes no-bypass
structural rather than policed: any caller reaching `Apply` is covered without
being enumerated.

## Consequences

The guard enforces the whole coherence rule set, not the force rule alone,
because the rule set is what the function checks. Most of that costs nothing
new: the history-walking check already reported every rule but one at error
severity, so a trailer set that newly fails at the seam is one the push already
rejected, and only the point at which the operator learns it has moved earlier.

The exception is `audit-only-with-force`, which no history-walking rule covers.
The seam is its first enforcement anywhere outside the audit-only verb's own
call.

A non-human actor cannot in practice reach `force-non-human` as the reported
rule. Getting past the allow-rule at all requires an active scope, whose
`aiwf-on-behalf-of` trips `force-with-on-behalf-of` earlier in the order, and
the function returns its first violation only. The act is refused either way;
what a caller must not do is assert on a specific rule name to prove force is
enforced, because that pins the rule order rather than the behavior.

Refusals move from push time to verb time. Automation that was forcing as a
non-human actor was already blocked at push, so no working pipeline breaks —
but a pipeline that treated the verb's exit code as success and the push as a
separate concern now fails a step earlier.
