# Epic wrap — E-0081

**Date:** 2026-08-12
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0081-one-table-owns-entity-body-sections-every-other-surface-follows-it
**Merge commit:** 16809c856

## Milestones delivered

- M-0305 — Single-source the per-kind section set and retire Approach (merged 22616e8e)
- M-0306 — Every remaining surface follows the owned set, and the epic nesting is fixed (merged 2a564519)
- M-0307 — Route the body-scaffold instruction through the verb covering every kind — **cancelled**; its scope landed directly as `4d94db6c` after a preflight found the milestone patch-sized

## Summary

Eleven surfaces stated which body sections each entity kind carries, and they
disagreed in both directions. `entity.RequiredSections` is now the single definition:
the `aiwf add` scaffold renders from it, the `entity-body-empty` rule reads it, and
every remaining surface either derives from it or is checked against it by test.
`Approach` left the milestone set, having entered it by transcription rather than by
decision.

The shipped prose templates were reconciled with the scaffold rather than collapsed
into it — they stay a superset, now a checked one. The epic template's out-of-scope
heading moved to top level, so an epic drafted from the ritual finally carries an
`out_of_scope` key in `aiwf show --format=json`; a mandated title H1 the scaffold never
writes became optional; and seven `(optional)` markers that were folding into JSON keys
came out.

Scope shifted twice, both times toward less. Two of M-0306's five acceptance criteria
were cancelled after their content had already landed, because the checks they mandated
enforced tidiness rather than preventing any consumer-visible failure. M-0307 was
cancelled outright once a preflight measured its premise: the defect was real and worse
than the spec stated, but the fix was one commit rather than a milestone.

## ADRs ratified

- ADR-0043 — Enforce body-section membership at the write seams, never tree-wide

## Decisions captured

- D-0065 — Whether `Context` takes the milestone required-section slot `Approach` vacates (**rejected**: the question does not hold; the set shrank rather than leaving a slot)

## Follow-ups carried forward

- G-0571 — Nothing enforces that an entity body carries its kind's required sections. Deliberately out of scope here: this epic made the surfaces agree on what the set *is*, not that anything refuses a body omitting one. ADR-0043 now decides how enforcement lands, and a successor epic implements it.
- G-0578 — Worktree-rituals hook test writes an executable without the ETXTBSY-safe helper
- G-0579 — D-0015's consequences cite a drift guard that no longer exists
- G-0580 — The skill-edit backstop reaches neither agent cards nor verb skills
- G-0581 — The backticked-verb resolution policy does not walk the guidance source
- G-0582 — ADR-0003 accepts a seventh entity kind the kernel does not carry

The last four were found by a prose-and-measurement preflight run before M-0307 would
have started. Each is a claim on a shipped or normative surface that no mechanism
checks — the same class this epic closed for body sections, in four other places.

## Doc findings

Scoped `wf-doc-lint` pass over the ten `docs/` files in the epic's change-set: no
findings. Every markdown link resolves. Six backticked `aiwf <verb>` invocations name
verbs that do not exist, and all six are correct prose — two say outright that no such
verb exists, four sit in the "deliberately out of scope" tables — and none was
introduced by this epic. The single `TODO` match is the word appearing inside an audit
table row, not a marker.

## Handoff

The per-kind section set has one owner and every surface follows it. What is
deliberately left open is enforcement: nothing yet refuses a body that omits a required
section, and the tree carries 217 gap bodies and 27 decision bodies that do.

ADR-0043 settles how that closes — a byte scan at every body-writing verb plus a gate on
the push range, never a tree-wide rule — and states its boundary with ADR-0042, which
governs the adjacent property of emptiness at the readiness transition and is itself
unimplemented. Whichever milestone implements either should weigh both together; they
touch the same rule surface and were designed apart.

A successor epic carries that work, and should carry a deletion obligation with it: once
a refusal names the missing sections, the prose that currently states them — the
`aiwf-add` skill's per-kind table, the body-sections table in `design-decisions.md`, and
`RequiredSections`' own "not a guarantee anything verifies" caveat — becomes deletable
rather than merely correct.
