---
id: G-0078
title: No priority field on entities; backlog isn't filterable or sortable by importance
status: open
---

## Problem

Entities have no kernel-supported field expressing importance or urgency. `aiwf list --kind gap` lists open gaps in id order; there is no way to express "do this one first" in structured state, no way to filter (`aiwf list --kind gap --priority high`), and no way for the HTML renderer or `aiwf list` to surface a backlog ranked by what to work next.

The friction first surfaced on the gap kind — the kernel's primary backlog — but applies to every kind that accumulates open work: gap, milestone, decision, ADR. Filing this kernel-wide; the per-kind cut belongs to the implementing milestone.

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

Implementation surface (sketch — milestone-level decisions deferred):

- New optional frontmatter field `priority` validated by `aiwf check`.
- A verb route to set it: either a dedicated `aiwf set-priority <id> <level>` or a `--priority` flag on a more general edit verb. Symmetric pattern with status / phase mutations.
- `aiwf status` gains a `--priority <level>` filter and a sort that uses priority as a tiebreaker (or primary, see open question 3).
- HTML renderer surfaces the value as a column / badge.
- JSON envelope carries it on the entity payload.

## Open sub-questions

1. **Scope** — does `priority` apply to every kind, or only to backlog-shaped kinds (gap, milestone, decision)? Epic and ADR are arguably ranked by the milestones / decisions they contain, not directly. Filing kernel-wide deliberately leaves the per-kind cut to the implementing milestone.
2. **Enforcement** — should `aiwf check` enforce anything (e.g., "an `urgent` open gap that has no scope-entity is a finding"), or is `priority` purely advisory metadata: filterable / sortable but unenforced?
3. **Sort order in `aiwf status`** — priority-first then status, or status-first then priority? Affects what the user sees at the top of the screen by default.
4. **Naming** — `priority` (Linear/Asana convention) vs `importance` (more descriptive of a state-not-workflow read). Lean: `priority`, because the field's purpose is to drive ordering and that's what every other tool calls it.

## Considered alternatives

- **Two-axis severity + urgency.** Rejected as YAGNI: the conflation only hurts in incident-tracking contexts the kernel doesn't address. Reintroduce additively if real friction shows up.
- **RICE / WSJF composite scores** (reach × impact × confidence / effort). Rejected for YAGNI: requires four new fields the kernel doesn't have, and the resulting score is the kind of opinionated workflow decision the kernel deliberately stays out of.
- **Continue using inline `Severity: …` prose lines.** Rejected: invisible to filters, sorts, and the JSON envelope. Defeats the point of structured state.
- **Use status as a proxy for priority** (e.g., `urgent` as a sub-status). Rejected: status answers "where is this in its lifecycle"; priority answers "how much does it matter." Collapsing them loses information and conflicts with the closed-set hardcoded FSM rule.

## Relationship to other gaps

- **G-0061** (no `aiwf list <kind>` verb): once `aiwf list` exists, priority becomes the obvious filter / sort key it would otherwise lack.
- **G-0053 / G-0052** (kernel demands behavior the verb routes don't deliver): same shape — a structured-state surface the kernel doesn't yet expose. Different mechanism; similar lesson.
