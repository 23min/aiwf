---
id: E-0043
title: Optional area tag for grouping entities by workstream
status: proposed
---
## Goal

Let a single repo hold more than one workstream — a product plus a co-developed internal tool, or a monorepo of several packages — by tagging entities with a validated, optional `area`. Roadmaps, status, and checks become scopeable per workstream, while the flat, globally-unique id space stays exactly as it is today.

## Context

Today every entity shares one flat global id space (`E-0001`, `ADR-0001`, `G-0001`, …) with no first-class way to partition by workstream. The only separators are *structural* (hang a whole workstream under one long-running mega-epic — bad for gap management, branching, and status churn) or *convention* (an ad-hoc title prefix like `dev:` — unenforced and invisible to tooling). Neither scales to a monorepo or to two co-developed workstreams in one tree.

The concrete trigger: an internal tool developed in parallel beside the product it feeds — fast but real, not deserving its own repo, yet still needing its own planning, epics, and ADRs.

[G-0266](../../gaps/G-0266-optional-area-tag-for-grouping-entities-by-workstream.md) records the full problem and a *converged* design (its "Direction (converged)" section). This epic implements that design verbatim; there are no open design questions, only decomposition. The sibling gap [G-0078](../../gaps/G-0078-no-priority-field-on-entities.md) (a priority/sort axis) is deliberately **not** folded in — `area` is single-valued grouping; priority is a different feature.

## Scope

### In scope

- An optional `area` frontmatter field on the five root kinds (epic, ADR, gap, decision, contract). Milestones and ACs **derive** their area from the parent epic — it is not stored on them, so "milestone disagrees with its epic" is unrepresentable rather than policed.
- An `aiwf.yaml: areas` block declaring the closed member set, plus an optional `default:` key that is *purely a display label* for the untagged complement in grouped views (never written to an entity, never a member of the tag set).
- A new `area-unknown` check finding: present ⇒ declared. If `area` is present and non-empty it must appear in the declared set; absence is never evaluated and never flagged; the field is inert when no `areas` block exists.
- Write path: `aiwf add --area <name>`, validated and tab-completed from config; a gap may default its area from `discovered_in` at add time. Changing an area reverses via the same verb.
- Read filter: an `--area` flag on `list`, `show`, and `status`.
- Read grouping: a section per area (with the untagged complement under the `default:` label) in `status`, `render roadmap`, and `render --format=html`.

### Out of scope

- **Per-area id namespacing** (`refpack/E-0001`): rejected, not deferred. It breaks id stability (commitment #2) when an entity moves between areas, or is merely decorative; revisit only if area-local numbering is ever demonstrably worth an id-format migration.
- **Multi-valued tags / a second grouping axis**: `area` is single-valued and closed-set on purpose.
- **A mandatory "every entity must have an area" mode**: precluded by "absence is never flagged"; add an opt-in later only if real friction shows.
- **The priority/sort field** (G-0078): a separate sibling feature.

## Constraints

- **Id model untouched (commitment #2).** Ids stay flat and globally unique; the allocator, references, commit trailers, `aiwf history`, and `reallocate` are not touched. `area` is a grouping tag, never a directory axis — on-disk layout stays kind-partitioned, so the loader and the ADR-0004 archive convention are untouched.
- **Zero migration.** Every entity that exists today, having no `area`, falls through to the implicit complement with no edits and no warning. Absence is its own partition.
- **Single source of truth for the member set** is `aiwf.yaml: areas`; the field validates against it and nothing else. No parallel registry.
- **Forward-compat is the generic strict-decoder window.** A binary built before `area` existed rejects a file using it (`KnownFields(true)`), exactly as for every prior frontmatter field; with one binary plus `aiwf upgrade` the window is narrow — not designed around.
- **No half-finished implementations.** Each milestone lands tested; the field, finding, write path, filter, and grouping each ship with their `--help`, skill docs, and discoverability coverage.

## Success criteria

<!-- Observable outcomes at epic close, not tests. -->

- [ ] A carved-out workstream is taggable via `area:` on the five root kinds, and via `aiwf add --area`, validated against `aiwf.yaml: areas`.
- [ ] A mistyped or undeclared area surfaces the `area-unknown` check finding; an absent area never warns.
- [ ] Every read surface listed in *Scope → In scope* can filter (`--area`) and group by area; untagged entities appear under the configured default label.
- [ ] Today's entire tree validates and renders unchanged with zero edits (no entity gains an `area`, none warns).
- [ ] Milestones/ACs reflect their parent epic's area in grouped views without storing the field.
- [ ] G-0266 promoted to `addressed` and archived under this epic's wrap.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Does `aiwf add gap --discovered-in <id>` auto-derive `area` from the discovered-in entity, or only set it on explicit `--area`? | no | Decided at the write-path milestone; lean: derive when `--area` is omitted and the discovered-in entity has one, else leave unset. |
| When the `areas` block is present but a render surface has zero tagged entities, is the per-area grouping suppressed or shown empty? | no | Decided at the read-grouping milestone; lean: suppress empty area sections, always show the default complement. |

## Milestones

Candidate decomposition (sequenced; refined and id-allocated via `aiwfx-plan-milestones`):

1. **Data + config core** — the optional `area` field on the five root kinds (+ parent-derivation for milestones/ACs) and the `aiwf.yaml: areas` block with closed-set validation. Foundation for everything below.
2. **`area-unknown` check finding** — the present-⇒-declared chokepoint. Depends on the data + config core.
3. **Write path** — `aiwf add --area` with config-sourced completion and `discovered_in` derivation. Depends on the core.
4. **Read filter** — `--area` on `list` / `show` / `status`. Depends on the core; independent of the write path.
5. **Read grouping** — per-area sections in `status`, `render roadmap`, and `render --format=html`. Depends on the core.

## Supersedes

- [G-0266](../../gaps/G-0266-optional-area-tag-for-grouping-entities-by-workstream.md) — the converged gap this epic implements; promoted to `addressed` at wrap.

## References

- [ADR-0004](../../../docs/adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md) — archive convention left untouched (area is a tag, not a directory axis).
- Design commitment #2 (stable flat ids) in CLAUDE.md — the constraint the rejected id-namespacing option would have violated.
