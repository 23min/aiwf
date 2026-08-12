---
id: E-0083
title: Retire tdd.strict; gate body completeness at the readiness promote
status: proposed
---

# E-0083 — Retire tdd.strict; gate body completeness at the readiness promote

## Goal

Ask for a complete entity body at the moment its status claims it is ready to be
worked on, instead of at the push boundary behind a configuration knob that has
never been switched on. An epic may be born empty; one promoted to `active` may
not.

## Scope

### In scope

- Retire `aiwf.yaml: tdd.strict` — the config field, its schema entry, the
  generated `aiwf.example.yaml` line, `check.ApplyTDDStrict`, and the two shipped
  skill tables that document the escalation.
- A status-gated body-completeness rule firing at the readiness transitions, epic
  to `active` and milestone to `in_progress`, unconditionally and with no knob.
- The migration answer for a repo that already carries the field: a consumer must
  get a clear statement, never silence.
- Close G-0573, whose reproduction stops existing once the knob does.

### Out of scope

- `milestone-tdd-undeclared`'s severity. ADR-0042 leaves it at warning, and
  `aiwf add milestone` already hard-refuses the state it guards.
- `acs-tdd-audit`, which enforces the AC TDD ladder and computes its own severity
  from the milestone's `tdd:` policy. `tdd.strict` never touched it, so retiring
  the knob cannot change it.
- What the per-kind required section set contains, and whether an absent heading
  counts as missing. That is E-0081's subject; this epic consumes its answer.
- The verb-time projection guard's own wiring. The severity policy reaches it
  already; this epic changes which passes exist, not how they are composed.

## Context

`tdd.strict` raises two finding codes from warning to error: `entity-body-empty`
on epic and milestone, and `milestone-tdd-undeclared`. ADR-0042 records four
measurements about it, each taken rather than inferred: it does not touch the
rule that actually enforces TDD; it escalates two findings that carry no evidence
claim, which is not what the proposal introducing it described; half of it guards
a state `aiwf add milestone` already refuses to produce; and the other half
inverts a deliberate design, since G-0326 settled that draft-phase kinds warn
precisely because they have a drafting phase.

That inversion has an observable cost, measured while closing G-0567 and G-0573.
With the knob set, `aiwf add epic` prints `ok — no findings`, commits, and leaves
a tree the pre-push hook refuses. Refusing at creation instead is worse than the
defect: the body would have to be written before the entity has an id, which the
allocate-scaffold-fill sequence the plan-epic ritual runs cannot do.

The transition is the moment that makes the question meaningful, and it is also
the only moment a verb can act on. The verb-time projection guard compares the
findings before a mutation with the findings after, and refuses what the mutation
introduced. `entity-body-empty` reads body bytes from disk, so its before and
after copies are identical and the guard never attributes it to the verb — no
severity setting changes that. A rule keyed on status is absent from the before
state and present in the after state, so the guard sees it and refuses. The
precedent is `epic-active-no-drafted-milestones`, which already fires on that
same transition for the same class of reason: readiness claimed before its
prerequisites exist.

The knob has never been enabled — not by this repo, whose `aiwf.yaml` sets
`docs.strict` and not `tdd.strict`, and not by any consumer. There is no
migration population to weigh, which is most of why retiring rather than
repairing is proportionate.

## Constraints

- **Both halves ship together.** Retiring the knob without the replacement rule
  is a capability removal, not a redesign: `entity-body-empty` on epic and
  milestone would become a warning with no path to blocking at all. No milestone
  in this epic lands the removal before the rule that replaces it.
- **The rule consumes E-0081's owner; it does not restate the section set.**
  E-0081 gives "which sections a kind carries" a single owner and makes absence
  count. This epic asks that owner and adds no second opinion about membership.
- **No new definition of "complete".** The readiness rule and the existing
  `entity-body-empty` rule agree by construction, through the same helper, for
  the same reason the verb gate and the check rule already share one.
- **The severity seam stays uniform.** Removing `ApplyTDDStrict` leaves three
  passes; `PolicySeverityPassComposition` must stay green without an exemption
  being added for the call sites the removal makes inert.
- **A consumer carrying the field is answered, not ignored.** Silence is the one
  outcome ruled out; which answer is an open question below.

## Success criteria

- [ ] `tdd.strict` is absent from the config struct, the schema, and the
      generated example file, and no shipped surface documents it.
- [ ] Promoting an epic to `active`, or a milestone to `in_progress`, with a
      required body section missing or empty is refused by the verb, naming the
      section.
- [ ] The same entity in its pre-readiness status produces no such refusal —
      creating and holding a draft with an empty body stays a supported workflow.
- [ ] `aiwf add epic` followed immediately by `aiwf check` reports no error on
      any tree, whatever the consumer's configuration.
- [ ] A repo whose `aiwf.yaml` still sets `tdd.strict` is told so, by the
      mechanism the open question below settles.
- [ ] G-0573 is `addressed`.
- [ ] Every constraint listed above is pinned by a test or a policy, not by
      review alone.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Does a lingering `tdd.strict` produce a deprecation finding, or a hard config-load rejection naming the replacement? | yes, for the removal milestone | Settled against the repo's own precedent for retired config surfaces; the never-used measurement argues the cheaper answer suffices. |
| Does the readiness rule get its own finding code, or extend `entity-body-empty` with a subcode? | no | Settled in the rule's own milestone; the finding-code discoverability policy constrains either shape equally. |
| Does the rule fire on the FSM transition alone, or on any tree whose entity sits in a readiness status with an incomplete body? | no | The guard needs the transition; a standing check rule is the wider option, decided once the transition case is measured. |
| Does `aiwf promote --force` bypass the readiness rule? | no | Follows whatever the projection guard already does for its other refusals, which is documented as unconditional. |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| E-0081 changes what "complete" means while this epic is in flight | med | The epic stays `proposed` until E-0081 is `done`, and its rule reads E-0081's owner rather than a copy. The ordering is operator-held: `depends_on` is validated for milestones only, so nothing mechanical enforces it between epics. |
| The readiness rule blocks a promote an operator considers legitimate, with no escape | med | The force question above is settled before the rule lands; `--body`/`edit-body` is the ordinary remedy and needs no escape. |
| Removing a documented config field surprises a consumer mid-upgrade | low | Never-enabled is measured, not assumed; the migration answer is a success criterion rather than a footnote. |

## Milestones

Candidates, to be refined by `aiwfx-plan-milestones`:

- the readiness rule: a status-gated body-completeness check firing at epic
  `active` and milestone `in_progress`, reading E-0081's section owner
- the retirement: `tdd.strict` leaves the config, schema, example file, severity
  passes and shipped skill tables, and a consumer still carrying it is answered
- closing out: G-0573 promoted with the evidence, and the severity-seam policy
  confirmed green on three passes rather than four

## References

- ADR-0042 — the decision this epic implements, and the measurements it rests on
- E-0081 — owns the per-kind section set this epic's rule consumes
- G-0573 — the verb-side reproduction that stops existing when the knob does
- G-0326 — why born-complete kinds error and draft-phase kinds warn
- G-0055 — makes `--tdd` mandatory at milestone creation, which is why the knob's
  other half guards an unreachable state
- M-0094 — `epic-active-no-drafted-milestones`, the precedent for a rule that
  fires on the readiness transition
