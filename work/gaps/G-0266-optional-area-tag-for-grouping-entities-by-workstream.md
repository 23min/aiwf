---
id: G-0266
title: optional area tag for grouping entities by workstream
status: addressed
addressed_by_commit:
    - e3168394
---
## What's missing

When one repo holds more than one workstream — a product plus a co-developed
internal tool, or a monorepo of several packages — every entity shares one flat,
global id space (`E-0001`, `ADR-0001`, `G-0001`, …) with no first-class way to
partition entities by area. Today the only separators are *structural* (hang a
whole workstream under one long-running mega-epic — bad for gap management,
branching, and status churn) or *convention* (an ad-hoc title prefix like
`dev:` on the parent-less kinds — unenforced and invisible to tooling). Neither
scales to a monorepo or to two co-developed workstreams in one tree.

## Why it matters

The concrete trigger: an internal tool developed in parallel beside the product
it feeds, fast but real, not deserving its own repo, yet still needing its own
planning, epics, and ADRs. Without an area concept that work either inflates a
mega-epic or rides an unenforced prefix convention that no view, filter, or
check understands. A first-class area tag gives monorepos a cleanly partitioned
planning tree, lets a co-developed tool be tracked beside its product, and makes
roadmaps / status / checks scopeable per workstream — none of which the flat id
space supports today.

## Direction (converged)

Ship **only the grouping-field design** (proposal Option 1). `area` is a
validated grouping/filter tag; ids stay flat and globally unique, so the id
model, allocator, references, commit trailers, `aiwf history`, and `reallocate`
are all untouched. The KISS shape is broad-but-shallow — one frontmatter field,
one `aiwf.yaml` block, one check finding, plus filter/group plumbing on the read
surfaces.

Specifics we landed on:

- **Fully optional; absence is its own partition.** An entity with no `area`
  field is never flagged — no warning, ever — and is automatically "the rest"
  (the implicit complement). You tag only the carve-out (`area: internal-tool`);
  every untagged entity, including every entity that exists today, falls through
  to the default with zero edits and zero migration.
- **Present ⇒ declared.** The validation is foreign-key-allows-NULL: *if* `area`
  is present and non-empty it must appear in the set declared in `aiwf.yaml`;
  absence is never evaluated. A new `area-unknown` check finding is the
  chokepoint (typo protection). When the `areas` block is absent entirely the
  field is inert.
- **No auto-stamping.** Nothing writes a default `area` into entities. An
  optional `default:` key in `aiwf.yaml` is *purely a display label* for the
  untagged complement in grouped views — not required, not a member of the tag
  set, never written to an entity.
- **Milestones (and ACs) derive area from their parent epic** — not stored —
  so "milestone disagrees with its epic" is unrepresentable rather than policed.
  The five root kinds (epic, adr, gap, decision, contract) carry the explicit
  field; a gap may default its area from `discovered_in` at add time.
- **Tag, not a directory axis.** On-disk layout stays kind-partitioned; area
  never reshapes the tree, so the loader and the ADR-0004 archive convention are
  untouched.
- **Read surfaces:** `--area` filter on `list` / `show` / `status`, and
  area grouping (a section per area) in `status`, `render roadmap`, and
  `render --format=html`. Write path: `aiwf add --area <name>`, validated and
  tab-completed from config; changing an area reverses via the same verb.

## Out of scope / non-goals

- **Id namespacing (proposal Option 2)** — per-area id sequences/prefixes
  (`refpack/E-0001`). Rejected, not deferred: it either breaks id stability
  (commitment #2) when an entity moves between areas, or is decorative; its only
  real benefit is area-local numbering, a cosmetic win not worth rippling the id
  format through the parser, allocator, every reference field, trailers,
  `history`, `reallocate`, and render anchors. Revisit only if area-local
  numbering is ever demonstrably worth an id-format migration.
- **General multi-valued tags / labels** — `area` is single-valued and
  closed-set on purpose; a second axis or multi-valued labels is a different,
  less-KISS feature.
- **A strict "every entity must have an area" mode** — precluded by "absence is
  never flagged," deliberately. Add an opt-in later only if real friction shows.

## Forward-compat note

The frontmatter decoder is strict (`internal/entity/parse.go` →
`KnownFields(true)`), so an `aiwf` binary built before the `area` field existed
would reject a file that uses it. That is generic to every frontmatter field
aiwf has ever added — `area` is not special — and with a single binary plus
`aiwf upgrade` the "old binary reading new tree" window is narrow. Not worth
designing around.
