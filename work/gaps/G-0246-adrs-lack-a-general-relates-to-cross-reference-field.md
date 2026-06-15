---
id: G-0246
title: ADRs lack a general relates_to cross-reference field
status: open
---
## What's missing

The ADR relation schema expresses only **supersession**, and only
ADR→ADR. `supersedes` and `superseded_by` are both constrained to
`AllowedKinds: []Kind{KindADR}` (`internal/entity/refs.go:42-46`,
`internal/entity/entity.go:487-488`). An ADR has **no** general
cross-reference field.

By contrast, two other kinds already carry an any-kind relation field:

- `decision.relates_to` — `AllowedKinds` empty = any kind
  (`internal/entity/entity.go:514`, `refs.go:57`).
- `gap.addressed_by` — any kind (`entity.go:500`, `refs.go:53`).

So an ADR cannot express "this ADR amends `D-NNNN`" or "this ADR relates
to `E-NNNN`'s scope" — the relation model is uneven across kinds.

## Why it matters

Surfaced by a downstream consumer that wanted an ADR to record that it
**amends a project-scoped decision** (`D-NNNN`). Today the only ADR→other
edge is supersession, which is the wrong primitive: it is bound to the
ADR lifecycle FSM (`accepted → superseded`) and the single-target
`superseded_by` back-edge. Bending `supersedes` to point at a decision
or an epic would conflate "this decision replaces that one" with "this
decision references that one."

The reviewer offered two shapes: (a) widen `supersedes` to any kind, or
(b) add an `amends` (multi→any) field. Both are refined here:

- **Reject (a).** Supersession is a lifecycle concept; widening its
  target set pollutes the FSM-coupled semantics. `superseded_by` is
  written by the `aiwf promote <adr> superseded` transition — a
  cross-kind target there has no transition to ride on.
- **Refine (b).** Rather than a bespoke `amends` field, give ADRs the
  **same `relates_to` (any-kind, multi) field decisions already have.**
  That is the symmetric, minimal move: one new optional frontmatter
  field, the existing any-kind ref-resolution path, no new FSM
  coupling. "Amends" is then expressible as a `relates_to` edge whose
  meaning is carried in the ADR prose — consistent with how
  `decision.relates_to` already works.

## Fix shape

1. Add `relates_to []string` to the ADR kind in
   `internal/entity/entity.go` (Multi, any-kind, optional) and the
   corresponding `ForwardRef` emission in `refs.go`.
2. Wire `--relates-to` into `aiwf add adr` (mirrors
   `aiwf add decision --relates-to`).
3. A post-create editor follows whatever G-0168 settles for the
   relation-editor fork (per-kind subverb vs generic) — this gap and
   G-0168 share that decision.
4. `aiwf check` ref-resolution already covers any-kind fields; the new
   field inherits it. A fixture test pins that an ADR
   `relates_to: [D-NNNN]` resolves and a dangling ref is flagged.

## Decision needed first

This changes the kernel relation model, so it wants an ADR to ratify it:
does the relation model standardize on a single any-kind `relates_to`
field across decision/gap/adr? Does supersession stay the sole ADR→ADR
edge? The decision pairs with G-0168's generic-vs-per-kind
fork — both are facets of "what is aiwf's cross-reference model?"

Source: downstream consumer feedback, 2026-06-12.
