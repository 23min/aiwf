---
id: E-0066
title: Add a priority field to gaps and decisions for a filterable backlog
status: done
---

# E-0066 — Add a priority field to gaps and decisions for a filterable backlog

## Goal

Give aiwf a kernel-supported `priority` field on gaps and decisions, so the backlog can be filtered and surfaced by importance instead of read end-to-end in id order. It replaces the ad-hoc inline `Severity:` prose that nothing can query with structured state that `aiwf check`, `aiwf list`, the JSON envelope, and the HTML render all understand.

## Context

Entities today carry no field expressing importance or urgency. `aiwf list --kind gap` lists in id order, and a handful of gaps encode importance as inline `Severity: …` prose that no filter, sort, or envelope can read — the foot-tracks of an expressed need with nowhere to land. The kernel's closed-set enums (kinds, statuses, `tdd_phase`) all express *what state an entity is in*; none expresses *how much it matters*. With 30+ open gaps, "what do I work next" means reading every body.

This is the implementation epic for **G-0078**, whose five design decisions this epic executes verbatim. The design deliberately mirrors the existing `area` feature: an optional, per-kind, closed-set frontmatter field carried on the shared `Entity` struct, with legality enforced by `aiwf check` rules rather than by the type system.

## Scope

### In scope

- A new optional `priority` frontmatter field, legal on **gap and decision only**, closed set `urgent | high | medium | low`, default unset — added to those kinds' `Schema.OptionalFields` and backed by a closed-set value predicate hardcoded in Go alongside kinds and statuses.
- `aiwf check` validation on two axes: the value must be in the closed set, and a new `priority-not-applicable` rule rejects `priority` present on epic, milestone, ADR, or contract (the mechanical backstop for the gap/decision-only scope).
- Writing it: `aiwf add --priority <level>` at creation (gap/decision, gated the way `--area` already is) and a dedicated `aiwf set-priority <id> <level>` verb for later changes — a deliberate second member of a `set-X` family alongside `set-area`, not a general-purpose edit verb.
- Reading it: a `--priority <level>` filter on `aiwf list` and `aiwf status`; the value on the JSON envelope entity payload and in `aiwf show`; a badge in the HTML render.
- Drift protection: extend the two literal-value chokepoints (`enum_literal_adoption.go`, `closed_set_status_constants.go`) so `priority` literals get the same treatment status/phase literals get.
- Discoverability: a new `aiwf-set-priority` skill and a `--priority` line added to the `aiwf-add` skill.

### Out of scope

- **Sort-by-priority ordering** — grouping by status then breaking ties by priority. Deferred to **G-0420**; this epic ships filtering only. No group-then-tiebreak sort infrastructure exists today and none is built here.
- `priority` on epic, milestone, ADR, or contract.
- Any second axis (severity + urgency, RICE/WSJF composites) — additive later if real friction appears.
- A finding rule that keys off a *specific* priority value; value enforcement stays advisory (shape-validation only).
- A general-purpose `aiwf set <id> --field=value` verb.

## Constraints

- Follow the `area` precedent: `Priority` sits on the shared `Entity` struct; per-kind legality is a `CarriesOwnPriority`-style predicate consulted by check rules, not a per-kind struct or a decode-time gate.
- The closed set is hardcoded in Go (like kinds and statuses) — no `aiwf.yaml` config knob, because the set is genuinely closed (unlike `area`'s operator-declared members).
- Value validation is advisory; scope validation is mechanical. "Gap and decision only" must be an enforced fact, not prose.
- AC-mechanical-evidence holds throughout: every AC needs a test that fails if its claim breaks. The HTML badge additionally requires a human-verified render against the kernel's own tree — a passing test does not stand in for the look.
- No verb sprawl beyond the intentional two-member `set-X` family.

## Success criteria

- [ ] A gap or decision can carry a `priority` set at creation (`aiwf add --priority`) or changed later (`aiwf set-priority`), and `aiwf show` surfaces it.
- [ ] `aiwf list --kind gap --priority urgent` returns exactly the urgent gaps and nothing else.
- [ ] Setting `priority` on an epic, milestone, ADR, or contract produces an `aiwf check` finding.
- [ ] The HTML render displays each gap/decision's priority as a badge, verified by eye against the kernel's own planning tree.
- [ ] Every milestone listed under *Milestones* is `done`; the deferred sort-ordering work is tracked in G-0420, not left implicit in this epic.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Warning vs error severity for `priority-not-applicable`? | no | Milestone planning; leaning warning, consistent with `area_unknown`'s advisory posture. |
| Does an existing render/list contract pin the entity JSON payload shape (needing a contract bump when `priority` is added)? | no | Check `docs/pocv3/plans/contracts-plan.md` and registered contracts during `aiwfx-plan-milestones`, before the read-surface milestone. |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| `priority-not-applicable` is net-new check logic — the `area` precedent only ever gates *requiredness*, never *presence*, so there's no rule today that rejects an out-of-scope field being present. | med | Small rule; structure it off `area_unknown.go` and pair it with a firing fixture (required anyway by `firing_fixture_presence.go`). |
| The HTML render has no reusable column/badge abstraction — the priority badge is bespoke template work. | low | Keep the badge minimal; human-verify the render. |

## Milestones

- `M-0261` — Field, validation, and drift chokepoints: the `priority` field on gap/decision (shared struct + `OptionalFields` + `CarriesOwnPriority` predicate + closed-set value predicate), the `priority-not-applicable` check rule with its firing fixture, and the two literal-drift chokepoint extensions. The foundation everything else reads and writes. · `tdd: required` · depends on: —
- `M-0262` — Write surface: `aiwf set-priority <id> <level>` (new verb: cobra wiring, completion, trailers) and `aiwf add --priority` (gap/decision gate); skills `aiwf-set-priority` (new) and `aiwf-add` (updated). · `tdd: required` · depends on: `M-0261`
- `M-0263` — Read surface: `--priority <level>` filter on `aiwf list` and `aiwf status`, the value on the JSON envelope entity payload, and `aiwf show` surfacing it. · `tdd: required` · depends on: `M-0261`
- `M-0264` — HTML render badge: priority badge in `aiwf render`, human-verified against the kernel's own tree. · `tdd: advisory` · depends on: `M-0261`

`M-0262`, `M-0263`, and `M-0264` each depend only on `M-0261` and are independent of one another — they can run in any order or in parallel once `M-0261` lands.

## ADRs produced

None committed up front. One candidate to weigh at milestone planning: `priority-not-applicable` introduces *presence-scope* field enforcement (reject a field on a kind that shouldn't carry it), a check class the kernel doesn't have today. If it reads as a reusable pattern for future per-kind optional fields, an ADR establishing "per-kind field applicability is enforced by a presence-scope check rule, not the type system" may be warranted.

## References

- G-0078 — the gap this epic implements; carries the full evidence, considered-alternatives, and the five ratified design decisions.
- G-0420 — the deferred sort-by-priority tiebreaker follow-up.
- The `area` feature — the design precedent (see the `aiwf-area` skill and `internal/check/area_unknown.go` / `area_required.go`).
- [`docs/pocv3/design/design-decisions.md`](../../docs/pocv3/design/design-decisions.md) — the closed-set-hardcoded-in-Go rule this field follows.
