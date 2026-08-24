---
id: M-0322
title: Decide the rendered legality reference and ship it if taken
status: draft
parent: E-0089
depends_on:
    - M-0321
tdd: advisory
acs:
    - id: AC-1
      title: The render decision lands before any render code
      status: open
    - id: AC-2
      title: If taken, the committed render matches a fresh render, enforced
      status: open
---
## Goal

Decide whether a human-readable legality reference is generated from the table, and
ship it if the decision says so.

## Context

A reader wanting to know what aiwf permits has two surfaces today: `docs/workflows.md`,
which is narrative and measured three of seven steps working when walked literally,
and the two catalogs under `docs/design/`, which are working papers whose structure
is pinned but whose claims are held by nothing.

Once the table is total and target-bearing it can answer the question directly, and
a generated document cannot drift from it by construction. `aiwf render roadmap
--write` already establishes that shape: derived view, regenerated rather than
maintained, committed and compared in CI.

D-0077 deliberately leaves shipping the render open. That is why the decision is
this milestone's first criterion rather than an assumption in its goal — a render
is a new shipped surface with its own maintenance and its own failure modes, and
the argument for it is not settled by the table being total.

## Acceptance criteria

### AC-1 — The render decision lands before any render code

A decision record states whether the legality reference is generated, and its
consequences name what happens to `docs/workflows.md` and to the two catalogs under
`docs/design/` either way. No render code is committed before that record is
accepted.

### AC-2 — If taken, the committed render matches a fresh render, enforced

Where the decision is to ship: a committed document is produced from the table, and
a check fails when the committed bytes differ from a fresh render. Editing the
document by hand reddens the suite. Where the decision is to decline: this criterion
is closed as not applicable, with the decision cited.

## Constraints

- **The decision governs the code, not the reverse.** Building the render first and
  recording the decision after inverts the order this criterion exists to hold.
- **Generated means generated.** A document that is rendered once and then hand-
  maintained is the drift this milestone is meant to end; the comparison check is
  what makes the claim true rather than intended.
- **Do not absorb the catalogs' disposition.** What happens to the two `docs/design/`
  documents is named by the decision, not executed here.

## Design notes

The render can only carry what the table carries. Reasoning — why a contract may go
straight to rejected, why scope reach is a three-edge tree — lives in ratified
decisions and does not become renderable by this work. A render that implies
otherwise would be worse than none, so the shape of what it omits belongs in the
decision.

## Out of scope

- Retiring or archiving the two `docs/design/` catalogs.
- Rewriting `docs/workflows.md`.
- Any render surface beyond the legality table.

## Dependencies

- M-0321 — a render of a table with holes would publish the holes.

## References

- D-0077 — leaves this decision open by design
- `aiwf render roadmap` — the derived-view precedent
- `docs/workflows.md`, `docs/design/legal-workflows-audit.md`,
  `docs/design/legal-workflows-first-principles.md` — the surfaces the decision must
  account for
