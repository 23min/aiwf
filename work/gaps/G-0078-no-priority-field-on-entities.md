---
id: G-0078
title: No priority field on entities; backlog isn't filterable or sortable by importance
status: open
---

## Problem

Entities have no kernel-supported field expressing importance or urgency. `aiwf list --kind gap` lists open gaps in id order; there is no way to express "do this one first" in structured state, no way to filter (`aiwf list --kind gap --priority high`), and no way for the HTML renderer or `aiwf list` to surface a backlog ranked by what to work next.

The friction first surfaced on the gap kind — the kernel's primary backlog. `priority` applies to gap and decision: the two kinds where "which one do I work next" is an open question the kernel can't currently answer. Milestones are already ordered by dependency logic, and epics are scoped by the milestones they contain — neither needs a separate priority axis.

## Evidence

The kernel currently has 30+ open gaps under `work/gaps/`. Picking which one to work next requires reading every body to recover the implicit priority that lives only in prose or the planner's head.

A handful of existing gaps carry inline `Severity: Low.` / `Severity: High.` notes in their bodies (e.g. G-0022, G-0023, G-0024, G-0026). Those are foot-tracks of an expressed need with no kernel-supported field to land on, so the information ended up in prose where nothing can sort or filter on it.

Note: "severity" already has a kernel meaning — *finding* severity in `aiwf check` (warning vs error). That's a different axis (per-rule-emission) and shouldn't be conflated with entity importance, which is one reason the leading direction below picks `priority` rather than `severity` as the field name.

## Root cause

The PoC's closed-set enums (kinds, statuses, `tdd_phase`) all express *what state an entity is in*. There is no closed-set enum for *how much it matters*. The state-not-workflow framing made priority feel out of scope; in practice a single priority field is structured state, not workflow, and is the thing most consumers reach for first when ranking work.

## Direction

Leading proposal: a single `priority` frontmatter field on entities, kernel-wide, with a closed set hardcoded in Go alongside kinds and statuses:

```yaml
priority: urgent | high | medium | low   # default: unset (treated as "none")
```

Reasons:

1. Single dimension, single sort key. Matches Linear, Asana, GitHub-via-labels, Shortcut — the dominant modern PM convention. Jira's priority-vs-severity split is mostly used for incidents, which aiwf doesn't model.
2. Composes with existing patterns: closed-set enum, hardcoded in Go, validated by `aiwf check`, sortable in `aiwf status`, queryable via the JSON envelope.
3. Default unset means existing gaps don't need backfill before the field can ship.
4. Adding a second axis later (impact vs urgency, or RICE composite) is additive and non-breaking.

Implementation surface:

- New optional frontmatter field `priority`, legal on gap and decision only (added to their `Schema.OptionalFields`), validated by `aiwf check` against the closed set.
- New `priority-not-applicable` check rule: fires if `priority` is present on epic, milestone, ADR, or contract. The shared `Entity` struct backing all six kinds doesn't gate field legality per kind on its own, so this is the mechanical backstop for the Decision 1 scope restriction — without it, "gap and decision only" is prose, not an enforced fact. Mirrors the shape of the existing `area_unknown.go` / `area_required.go` check rules; needs its own firing fixture per `firing_fixture_presence.go`.
- `aiwf add --priority <level>` (gap/decision only, gated the same way `--area` already is at `add.go`) plus a dedicated `aiwf set-priority <id> <level>` verb for changing it later — the second instance of a `set-X` family alongside `set-area`, not a one-off.
- `aiwf list` / `aiwf status` gain a `--priority <level>` filter. Sort-as-tiebreaker is explicitly out of scope for this gap: no group-by-status-then-secondary-key sort infrastructure exists anywhere in the codebase today (both verbs sort flatly by id), and the Evidence above is entirely about filtering/discovery friction, not a felt need for a specific sort order. Revisit as a follow-up gap once priority has real data and filtering alone proves insufficient.
- HTML renderer surfaces the value as a column / badge — bespoke template work; there's no generic per-entity metadata/column abstraction to reuse (the `area` tag itself only reaches templates via a bespoke `data-area` construct).
- JSON envelope carries it on the entity payload.
- Extend two literal-value drift chokepoints to cover `priority`, matching existing status/phase coverage: `internal/policies/enum_literal_adoption.go` (harvest `Priority*`-prefixed constants, currently scoped to `Status*` only) and `internal/policies/closed_set_status_constants.go` (add `Priority:` / `.Priority ==` match patterns alongside the existing `Status:` / `TDDPhase:` ones).
- Skills: update `aiwf-add` to document `--priority`; add a new `aiwf-set-priority` skill (required by `skill_coverage.go` for any new top-level verb).

## Decisions

1. **Scope** — `priority` applies to gap and decision only, not epic, milestone, ADR, or contract. Enforced by the `priority-not-applicable` check rule (see Implementation surface), not left as unenforced prose.
2. **Enforcement** — the *value* is purely advisory: `aiwf check` validates it against the closed set (the same baseline shape validation every frontmatter field gets); no finding rule keys off a specific priority value. The *scope* (which kinds may carry it at all) is mechanically enforced — see Decision 1.
3. **Sort order** — out of scope for this gap. `aiwf list` / `aiwf status` ship a `--priority <level>` filter only; group-by-status-then-priority-tiebreak sorting is deferred to a follow-up gap, since it requires new sort infrastructure the codebase doesn't have today and no evidence yet shows the need beyond filtering.
4. **Naming** — `priority`, matching the Linear/Asana/GitHub/Shortcut convention.
5. **Verb** — a dedicated `aiwf set-priority <id> <level>`, not a flag on a general-purpose edit verb (no such verb exists to hang it on; see Considered alternatives). It's the second instance of a `set-X` family alongside `set-area`, deliberately, so this doesn't read as one-off verb sprawl.
6. **Creation-time setting** — `aiwf add` also gets `--priority` (gap/decision only), mirroring how `--area` is already settable both at `add` time and later via `set-area`. `set-priority` remains the path for changing it after creation.

## Considered alternatives

- **Two-axis severity + urgency.** Rejected as YAGNI: the conflation only hurts in incident-tracking contexts the kernel doesn't address. Reintroduce additively if real friction shows up.
- **RICE / WSJF composite scores** (reach × impact × confidence / effort). Rejected for YAGNI: requires four new fields the kernel doesn't have, and the resulting score is the kind of opinionated workflow decision the kernel deliberately stays out of.
- **Continue using inline `Severity: …` prose lines.** Rejected: invisible to filters, sorts, and the JSON envelope. Defeats the point of structured state.
- **Use status as a proxy for priority** (e.g., `urgent` as a sub-status). Rejected: status answers "where is this in its lifecycle"; priority answers "how much does it matter." Collapsing them loses information and conflicts with the closed-set hardcoded FSM rule.
- **A general-purpose `aiwf set <id> --field=value` verb** (e.g. retrofitting `set-area` into `--area`/`--priority` flags on one command) to avoid a new verb. Rejected: no such general verb exists today, and every scalar frontmatter field currently gets its own single-purpose verb (`retitle`, `rename`, `set-area`). Retrofitting a shipped verb's contract for one new field isn't worth the migration risk; a small, symmetric `set-X` family reads as intentional and matches house style.

## Relationship to other gaps

- **G-0061** (no `aiwf list <kind>` verb): once `aiwf list` exists, priority becomes the obvious filter / sort key it would otherwise lack.
- **G-0053 / G-0052** (kernel demands behavior the verb routes don't deliver): same shape — a structured-state surface the kernel doesn't yet expose. Different mechanism; similar lesson.
- **G-0420** (no sort-by-priority tiebreaker): the sort-ordering follow-up deliberately deferred out of this gap's Decision 3 — filtering ships here, group-then-tiebreak sorting is scoped there.
