---
id: G-0460
title: Repeat authorize leaves two active scopes on one entity, no finding
status: open
priority: high
discovered_in: M-0281
---
## What's missing

Running `aiwf authorize <id> --to <agent> --branch <b>` twice against the same
entity leaves **two simultaneously-`active` scopes on that entity**, and
`aiwf check` reports zero error-severity findings about it.

Measured with a freshly-built binary against a disposable repo: two `authorize`
invocations with identical arguments both exit 0, each lands its own commit, and
`aiwf show <id> --format=json` then reports two scope records, both
`"state": "active"`, differing only by `auth_sha`.

Multiple simultaneously-active scopes are *not* themselves undefined.
`docs/design/provenance-model.md` §"Multiple parallel scopes" states that a human
may authorize the same agent for several scopes at once, and that when more than
one active scope matches a verb "the kernel picks the *most-recently-opened*
scope deterministically and records that one." `verb.Allow` implements exactly
that, walking scopes in reverse insertion order and citing the rule.

What is undefined is narrower: whether an **exactly-duplicate** re-grant — same
entity, same agent, same branch — is a distinct event worth recording or a
same-state input that should converge. Nothing refuses or reconciles it, and no
finding reports the resulting pair.

## Why it matters

The scope is the gate for authorized autonomous work: it records which agent may
act on the entity, under which principal, bound to which branch. Downstream
provenance behavior keys on the active scope — matching an agent's verbs against
it, stamping `aiwf-on-behalf-of:` / `aiwf-authorized-by:` trailers, and deciding
when delegated work is in or out of scope.

Resolution itself is not ambiguous — the design fixes it as most-recently-opened
and the code follows. What the duplicate costs is elsewhere: a pause acts on one
scope, so an entity can be left carrying a paused scope and a live one
simultaneously, an authorization state no operator asked for and no finding
reports. And the second grant's commit records an event that conveys nothing a
reader can act on, which is the same audit-noise class as the duplicate records
in G-0459.

This is separate from same-state verb convergence. Making `authorize` a no-op on
a repeat would hide the condition rather than resolve it, which is why the
companion gap on event-verb duplicate records defers `authorize` to this one.

## Resolution shape

First establish the intended invariant: is more than one active scope per entity
legal? If not, the fix is a verb-time refusal (or an FSM guard) on opening a
scope while one is already active, plus an `aiwf check` rule so an
already-divergent tree is reported rather than silently carried. If multiple
active scopes are intended, then the resolution rule for "the active scope" needs
stating explicitly and every consumer needs auditing against it.
