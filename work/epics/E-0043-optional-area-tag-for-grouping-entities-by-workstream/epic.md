---
id: E-0043
title: Optional area tag for grouping entities by workstream
status: active
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

Both resolved at wrap (decisions recorded in the named milestone specs):

| Question | Blocking? | Resolution |
|---|---|---|
| Does `aiwf add gap --discovered-in <id>` auto-derive `area` from the discovered-in entity, or only set it on explicit `--area`? | no | **Resolved (M-0173):** derive-on-omit — a gap derives its area from the discovered-in entity's effective area when `--area` is omitted (epic direct, milestone two-hop via `ResolvedAreaByID`); an explicit `--area` always wins; an untagged source leaves the gap untagged. |
| When the `areas` block is present but a render surface has zero tagged entities, is the per-area grouping suppressed or shown empty? | no | **Resolved (M-0175):** suppress empty *declared* area sections; always render the untagged/default complement (with a "(none)" / "no epics" placeholder when empty), so a grouped view always surfaces where un-triaged work lives. |

## Milestones

Sequenced. The data + config core is foundational; the other four depend only on it and are mutually parallelizable:

1. [M-0171](M-0171-area-field-on-root-kinds-and-aiwf-yaml-areas-block-with-validation.md) — Area field on the five root kinds (+ parent-derivation) and the `aiwf.yaml: areas` block with validation. *(No deps — foundation.)*
2. [M-0172](M-0172-area-unknown-check-finding-for-undeclared-area-values.md) — `area-unknown` check finding (present ⇒ declared). *(Depends on M-0171.)*
3. [M-0173](M-0173-aiwf-add-area-write-path-with-completion-and-discovered-in-derivation.md) — `aiwf add --area` write path + completion + `discovered_in` derivation. *(Depends on M-0171.)*
4. [M-0174](M-0174-area-filter-on-list-show-and-status.md) — `--area` filter on `list` / `show` / `status`. *(Depends on M-0171.)*
5. [M-0175](M-0175-area-grouping-in-status-render-roadmap-and-render-html.md) — Area grouping in `status`, `render roadmap`, and `render --format=html`. *(Depends on M-0171.)*

## Supersedes

- [G-0266](../../gaps/G-0266-optional-area-tag-for-grouping-entities-by-workstream.md) — the converged gap this epic implements; promoted to `addressed` at wrap.

## References

- [ADR-0004](../../../docs/adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md) — archive convention left untouched (area is a tag, not a directory axis).
- Design commitment #2 (stable flat ids) in CLAUDE.md — the constraint the rejected id-namespacing option would have violated.
