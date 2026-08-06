---
id: E-0081
title: One table owns entity body sections; the check fires on an absent one
status: active
---
## Goal

Give "which body sections an entity carries" a single owner, so the surfaces that
state it today cannot disagree — and repair the rule that names the requirement but
stays silent when a section is missing.

## Context

Five surfaces state the per-kind body section set. Measured 2026-08-06, for milestone:

| surface | says |
|---|---|
| `requiredSectionsByKind` (`internal/check/entity_body.go`) | Goal, Approach, Acceptance criteria |
| `entity.BodyTemplate`, exposed as `aiwf template` — what `aiwf add` writes | Goal, Acceptance criteria |
| the prose milestone template under the embedded rituals | Goal, Context, Acceptance criteria, +12 |
| the `show --format=json` body map | `goal`, `acceptance_criteria` |
| the root command's help text describing that map | `goal/acceptance_criteria` |

Four of the five omit `Approach`, and the one that requires it never fires.
`EmptyRequiredSections` reports only a section that is present and empty; a heading
absent outright is skipped, a stance its own doc comment takes deliberately. The map
named as the requirement therefore enforces non-emptiness of whatever happens to be
present, and nothing enforces membership at all.

Verified end to end in a scratch consumer repo: `aiwf add milestone` writes a body
missing a section the kernel declares required, and `aiwf check` reports nothing on
that axis.

A second instance sits in a different kind. The prose epic template places
out-of-scope one heading level below the flat `## Out of scope` that the check, the
`aiwf add` scaffold, and the JSON body map all name, so epics drafted by following the
ritual carry no `out_of_scope` key in `aiwf show --format=json`. Two instances across
two kinds is what makes this structural rather than two point fixes. A third sits on
the same artifacts: the prose epic template mandates a title H1 that the `aiwf add`
scaffold does not write and that no epic in this tree carries.

Separately, the always-on guidance instructs an assistant to fill a body from a
per-kind template path that resolves for two of six kinds. Gap and contract have no
such file and are born-complete, so `aiwf add` hard-refuses an empty body for exactly
the two kinds with no scaffold to work from.

Absorbs G-0482, G-0479 and G-0541.

## Scope

- One owner for the per-kind section set; every other surface derives from it or is
  mechanically checked against it, rather than restating it.
- The requirement fires when a required section is absent, not only when it is present
  and empty — reached through the helper the verb gate and check rule already share,
  so the two cannot drift on what "missing" means.
- Membership corrections the existing evidence already settles: `Approach` leaves the
  milestone set; the epic prose template's out-of-scope heading moves to top level so
  the existing requirement starts holding.
- The always-on guidance names `aiwf template <kind>`, which covers all six kinds, in
  place of a per-kind file path that resolves for two.

## Out of scope

- Whether the four milestone-spec sections that duplicate structured data should exist
  at all (G-0530). That asks whether a section is worth carrying, where this epic asks
  only that the surfaces agree; one of the four is load-bearing for the wrap ritual's
  commit-SHA trail, so it needs its own evidence.
- `aiwf doctor`'s byte-check of materialized ritual and guidance artifacts (G-0504) —
  the arrival axis, wider than templates.
- Acceptance-criterion sub-element bodies, which the AC rules govern separately.

## Constraints

- No new chokepoint. The absent-section case flows through `EmptyRequiredSections`,
  the helper `internal/verb/add.go` and the `entity-body-empty` rule already share.
- The prose templates survive as a superset, not a second source: a test proves each
  contains the derived required set. Collapsing them is not in this epic — four
  rituals read them, and the milestone template's work-log section is where per-AC
  commit SHAs are written and read back at wrap.
- D-0015 is preserved: the prose templates keep materializing where they do. Only the
  guidance line naming a per-kind path changes.
- Retiring `Approach` is a deletion from the required set, not a grandfather clause —
  no allowlist survives the epic.

## Success criteria

- [ ] The per-kind section set is stated once; a policy test fails if any surface
      listed in the *Context* table restates it instead of deriving from or being
      checked against it.
- [ ] A milestone body created by `aiwf add` and left unedited produces no
      `entity-body-empty` finding attributable to a section the template never wrote.
- [ ] An entity body missing a required section produces a finding naming that section.
- [ ] An epic drafted from the prose template carries `out_of_scope` in
      `aiwf show --format=json`.
- [ ] Every kind resolves through the body-scaffold route the always-on guidance names.
- [ ] Each gap listed in *References* as absorbed is `addressed`.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Does the section set live in `internal/check` and get read by `entity`, or move to `entity` and get read by `check`? | no | Layering direction is already policed; settled in the first milestone against the existing layering rule. |
| Does the prose-template superset check cover optional sections, or only the required set? | no | Only the required set unless a second case appears; the wider check is the speculative half. |
| Does the title H1 the prose templates mandate join the owned set, or leave the templates? | no | Settled with the template reconciliation; the tree's own convention is the evidence. |

## Milestones

- `M-0305` — one owner for the per-kind section set; the rule fires on an absent
  section and `Approach` leaves the milestone set · depends on: —
- `M-0306` — the shipped prose templates become a checked superset of that set, and
  the epic template's out-of-scope heading moves to top level · depends on: `M-0305`
- `M-0307` — the always-on guidance's body-scaffold instruction routes through the
  verb that covers every kind · depends on: —

## References

- G-0482 — `Approach` exists on no shipped surface; argues for a detector over point fixes
- G-0479 — epic template nests out-of-scope below the level three surfaces require
- G-0541 — the guidance's template path resolves for two of six kinds
- G-0530 — the adjacent, out-of-scope question of section membership
- D-0015 — ritual templates materialize to the templates dir
- `internal/check/entity_body.go`, `internal/entity/serialize.go`, `internal/verb/add.go`
