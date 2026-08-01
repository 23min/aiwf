---
id: M-0289
title: Lint and sweep narrow ids from README and the workflows guide
status: draft
parent: E-0078
tdd: required
acs:
    - id: AC-1
      title: A narrow id in README or the workflows guide fails a gate
      status: open
    - id: AC-2
      title: Neither README nor the workflows guide carries a narrow id
      status: open
    - id: AC-3
      title: The deferred doc-residue gap exists naming its three paths and reason
      status: open
---

## Goal

Stop the two repo-facing docs an assistant reads to learn the workflow from
modelling narrow width as current, behind a width-shaped lint that keeps them
that way — and record the residue this milestone deliberately does not sweep.

## Context

The shipped-surface work is a real-id problem where width is incidental. This is
the opposite: `README.md` and `docs/workflows.md` are repo-facing, real ids in
them are entirely legitimate, and the defect is purely that ~104 of them are
written at a width no allocator has emitted since the migration. So this needs a
genuinely width-shaped rule over a different corpus with the opposite stance on
real ids — a sibling of the shipped-surface guard, not a mode of it.

The two files are in scope because they teach the workflow. The rest of the
active doc tree is not, for a reason worth recording rather than leaving implicit:
its narrow ids are mostly citations of entities that were genuinely real at narrow
width, so the correct fix there is widening to the real canonical id, not
placeholdering. That is a different edit at a lower payoff, and folding it in
would bloat this lint's allowlist.

## Acceptance criteria

### AC-1 — A narrow id in README or the workflows guide fails a gate

A below-canonical-width entity id introduced into either file produces a finding
naming the file and line. Real canonical-width ids do not fire — unlike the
shipped-surface rule, this corpus is where real ids belong.

Evidence: a fixture asserting fire on a narrow id and no-fire on the canonical
form of the same id, for each kind prefix.

### AC-2 — Neither README nor the workflows guide carries a narrow id

Both files are swept. Where the narrow id was a teaching example, it becomes the
canonical placeholder form; where it named a real entity, it becomes that
entity's canonical id.

Evidence: the rule from AC-1, run over the real files, reports zero findings.

### AC-3 — The deferred doc-residue gap exists naming its three paths and reason

The residue this milestone declines is captured as its own gap rather than left
as an informal intention — naming the three paths and the widen-rather-than-
placeholder reason, so the next reader does not re-derive the scoping decision or
mistake the omission for an oversight.

Evidence: a structural assertion that the gap resolves through the loader and its
body names all three paths.

## Constraints

- **This lint's corpus and polarity are both distinct from the shipped-surface
  guard's.** Real ids are correct here and defective there; only width is at
  issue. Sharing an implementation is fine, conflating the rules is not.
- **The lint is scoped to the two named files**, not the active doc tree, so its
  allowlist stays short enough to read.

## Design notes

- The epic leaves open whether this fires from `aiwf check` or from
  `internal/policies`. Decided here. The check tier fires pre-push and catches in
  context before the work leaves the machine; the policy tier is a CI backstop
  that lands after a trunk push. The corpus is repo-only either way, so the rule
  is inert for consumers under both — which makes the earlier-catch argument the
  deciding one absent a reason to prefer CI.

## Out of scope

- `docs/design/**`, `docs/overview.md`, `docs/architecture.md` — the residue AC-3
  files as its own gap.
- Everything frozen by convention: the doc archive, research and explorations
  trees, the changelog, and the migration ADR itself.

## Dependencies

- None. Independent of the shipped-surface milestones and of the retirement.

## References

- G-0481 — the tier split and the reason the residue is deferred rather than
  swept.
