---
id: ADR-0042
title: Retire tdd.strict; require a complete body at the readiness promote
status: accepted
---
# ADR-0042 — Retire tdd.strict; require a complete body at the readiness promote

> **Date:** 2026-08-09 · **Decided by:** Peter Bruinsma

## Context

`aiwf.yaml: tdd.strict` raises two finding codes from warning to error:
`entity-body-empty` (on epic and milestone only) and `milestone-tdd-undeclared`.
Four facts about it, each measured rather than inferred:

**It does not make TDD stricter.** `acs-tdd-audit` — the rule enforcing that an
AC cannot be `met` without `tdd_phase: done` — computes its own severity from
the milestone's own `tdd:` policy, and is absent from the escalated set. The
knob named for TDD leaves the TDD ladder untouched; what it escalates is body
emptiness and the presence of a frontmatter field.

**It drifted from its own proposal.** The exploratory
[TDD architecture proposal](../explorations/07-tdd-architecture-proposal.md)
§10.6 described a future `tdd.strict` as promoting the *evidence* findings to
errors. What shipped escalates two findings that carry no evidence claim. (That
document is Exploratory tier and binds nothing; it is cited as intent, not
authority.)

**Half of it guards a state no verb can produce.** G-0055 made `--tdd`
mandatory at `aiwf add milestone`, and `milestone-tdd-undeclared` (G-0268) is
documented as the defense-in-depth backstop for the paths that bypass the verb —
`aiwf import` and hand-edits. Escalating a backstop for an
unreachable-through-verbs state buys little.

**The other half contradicts a deliberate design.** G-0326 settled that
born-complete kinds error on an empty load-bearing section *because they have no
draft phase*, and that epic and milestone stay warnings *because they do*.
`tdd.strict` inverts that for the draft-phase kinds — it is not a strictness
dial but a different opinion about whether the draft phase exists.

That inversion has an observable cost, measured while closing G-0573: with the
knob set, `aiwf add epic` prints `ok — no findings`, commits, and leaves a tree
the pre-push hook refuses. The obvious repair — refuse at creation, as the
born-complete gate does — is worse than the defect, because it requires the body
to be written before the entity has an id, which the shipped plan-epic ritual's
allocate-scaffold-fill sequence cannot do.

The knob has never been enabled: not by this repo, whose `aiwf.yaml` sets
`docs.strict` and not `tdd.strict`, and not by any consumer.

## Decision

Retire `tdd.strict`.

Replace its body-completeness half with a status-gated rule that fires at the
readiness transition — epic to `active`, milestone to `in_progress` —
unconditionally, with no configuration knob. A draft is allowed to be empty; an
entity whose status asserts it is ready to be worked on is not.

Leave `milestone-tdd-undeclared` at warning severity.

Two properties decide the transition point. The precedent is exact:
`epic-active-no-drafted-milestones` (M-0094, G-0063) already fires on that same
transition for that same class of reason — readiness claimed before the
prerequisites exist. And a status-gated rule is the only shape the verb-time
projection guard can act on, because it is absent from the pre-state and present
in the post-state, so the guard reads it as introduced and refuses the promote.
`entity-body-empty` cannot be made to do this at any severity: it reads body
bytes from disk, so the pre-state and post-state copies are identical and the
diff never attributes it to the verb.

## Consequences

The severity policy loses one of its four passes. The shared severity seam and
`PolicySeverityPassComposition` are unaffected — they govern how passes are
composed across surfaces, not which passes exist.

The verb-time projection guard's severity application becomes inert. Of the
remaining passes, the area and doc codes are composed at the CLI layer and never
reach a bare `check.Run`, and `archive-sweep-pending` is excluded from the
projection diff as an instance of G-0574. The call stays wired for uniformity,
on the same footing as `aiwf doctor`'s, which reads only a code no pass touches.

G-0573's behavioural half becomes moot rather than fixed: with no knob able to
make an empty draft body an error, `aiwf add epic` has nothing to misreport. Its
structural half — seven `check.Run` call sites held in no relation to each other
— is already closed by the shared seam.

Removing a documented configuration field is consumer-visible. `tdd.strict`
appears in the generated `aiwf.example.yaml`, in the config schema, and in the
shipped `aiwf-add` and `aiwf-check` skill tables. A consumer who has set it must
get a clear answer rather than silence: either the field is accepted and ignored
behind a deprecation finding, or it is rejected with a message naming the
replacement rule. That choice is deliberately left to the implementing
milestone, which should also decide whether the replacement rule warrants its
own finding code or extends an existing one.

Retiring the knob without landing the replacement rule would be a straight
capability removal wearing a redesign's clothes: `entity-body-empty` on epic and
milestone would become a warning with no path to blocking at all. The two halves
ship together or not at all.
