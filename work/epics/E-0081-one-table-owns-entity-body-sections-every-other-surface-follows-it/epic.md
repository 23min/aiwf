---
id: E-0081
title: One table owns entity body sections; every other surface follows it
status: active
---
## Goal

Give "which body sections an entity carries" a single owner, so the surfaces that
state it today cannot disagree.

## Context

Every surface below states the per-kind body section set. Measured 2026-08-06/08; the
`says` column reads the milestone entry, or names the kinds a surface covers when it
does not reach milestone at all:

| surface | says |
|---|---|
| `requiredSectionsByKind` (`internal/check/entity_body.go`) | Goal, Approach, Acceptance criteria |
| `entity.BodyTemplate`, exposed as `aiwf template` — what `aiwf add` writes | Goal, Acceptance criteria |
| the prose milestone template under the embedded rituals | Goal, Context, Acceptance criteria, +12 |
| the `show --format=json` body map | `goal`, `acceptance_criteria` |
| the root command's help text describing that map | `goal/acceptance_criteria` |
| the `aiwf-show` skill's body-key table | `goal`, `approach`, `acceptance_criteria`, +5 |
| the `aiwf-add` skill's required-body-sections table, and its per-kind prose | Goal, Approach, Acceptance criteria |
| the `entity-body-empty` rule's package doc comment | Goal, Approach, Acceptance criteria |
| `docs/design/design-decisions.md`'s body-sections table (Normative tier) | Goal, Acceptance criteria, +5 from the prose template |
| `selfCheck*Body` (`internal/cli/doctor/selfcheck.go`) | born-complete kinds only — full bodies for adr, gap, decision, contract |
| `bornCompleteFixtureBody` (`internal/cellcoverage/fixture.go`) | born-complete kinds only — the same four, again |

They disagree in both directions: four omit `Approach` and four name it, and one is
wrong about a key it does name — the `aiwf-show` skill gives gap's section as
`whats_missing`, where the slug `SectionSlug` derives is `what_s_missing`, so a
reader following it looks up a key no envelope carries.

The last three were found by review rather than by the original sweep, which is itself
evidence for the epic: a surface nobody thought to look at is exactly the one that
drifts. Two of them are Go literals rather than prose, and they fail in the worst
direction — if a kind's set gains a section, they build entities missing it, and
because nothing reports an absent heading the `aiwf doctor --self-check` that runs
them passes anyway.

The disagreements survive because nothing can see them. `EmptyRequiredSections`
reports only a section that is present and empty; a heading absent outright is
skipped, a stance its own doc comment takes deliberately. So the map named as the
requirement enforces non-emptiness of whatever happens to be present, and nothing
enforces membership at all — the hole G-0571 now carries.

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
- Membership corrections the existing evidence already settles: `Approach` leaves the
  milestone set; the epic prose template's out-of-scope heading moves to top level so
  the existing requirement starts holding.
- The always-on guidance names `aiwf template <kind>`, which covers all six kinds, in
  place of a per-kind file path that resolves for two.

## Out of scope

- Enforcing membership. This epic makes the surfaces agree on what the set *is*; it
  does not make any of them refuse a body that omits a section. Tracked as G-0571,
  which carries the measurement of what closing it tree-wide would cost, and the
  narrower create-time option worth weighing on its own evidence rather than as a
  rider here.
- Whether the four milestone-spec sections that duplicate structured data should exist
  at all (G-0530). That asks whether a section is worth carrying, where this epic asks
  only that the surfaces agree; one of the four is load-bearing for the wrap ritual's
  commit-SHA trail, so it needs its own evidence.
- `aiwf doctor`'s byte-check of materialized ritual and guidance artifacts (G-0504) —
  the arrival axis, wider than templates.
- Acceptance-criterion sub-element bodies, which the AC rules govern separately.

## Constraints

- No new chokepoint, and no change to what `aiwf check` reports. The epic reconciles
  the surfaces that state the set; the rule reading it keeps its existing behaviour.
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
- [ ] The scaffold's rendered bytes are asserted per kind, so a display-form edit to a
      section name cannot pass every suite.
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

- `M-0305` — one owner for the per-kind section set, with the help text and the show
  skill's key table checked against it, and `Approach` retired · depends on: —
- `M-0306` — the shipped prose templates become a checked superset of that set, and
  the epic template's out-of-scope heading moves to top level · depends on: `M-0305`
- `M-0307` — the always-on guidance's body-scaffold instruction routes through the
  verb that covers every kind · depends on: —

## References

- G-0482 — the milestone template and the required set disagreed about `Approach`;
  argues for a detector over point fixes
- G-0571 — nothing enforces that a body carries its kind's required sections
- G-0479 — epic template nests out-of-scope below the level three surfaces require
- G-0541 — the guidance's template path resolves for two of six kinds
- G-0530 — the adjacent, out-of-scope question of section membership
- D-0015 — ritual templates materialize to the templates dir
- `internal/check/entity_body.go`, `internal/entity/serialize.go`, `internal/verb/add.go`
