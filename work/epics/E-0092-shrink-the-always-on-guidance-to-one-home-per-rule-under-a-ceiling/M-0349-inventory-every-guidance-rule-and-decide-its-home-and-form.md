---
id: M-0349
title: Inventory every guidance rule and decide its home and form
status: draft
parent: E-0092
depends_on:
    - M-0333
    - M-0334
tdd: advisory
acs:
    - id: AC-1
      title: The inventory lists every rule in the guidance set once, with its current homes
      status: open
    - id: AC-2
      title: Every rule carries one disposition from the closed set
      status: open
    - id: AC-3
      title: Every conflict is decided by the maintainer and recorded
      status: open
    - id: AC-4
      title: The on-demand documents and the router are specified
      status: open
    - id: AC-5
      title: Each host's handwritten primed ceiling is set from the primed rules
      status: open
---
## Goal

List every guidance rule in this repository once, with where it lives today, and decide its home and form: primed or on demand, kept, merged, tightened, reduced to a pointer, or deleted as a copy. Settle every conflict with the maintainer.

## Closes

- (none)

## Context

M-0333's fence is in place and M-0334 has recorded the baseline, so nothing here changes guidance text; this milestone produces the table the later milestones carry out. The guidance set is the one M-0333 defines — the root `CLAUDE.md` and `AGENTS.md`, `.guidance/project.md` and what it routes to — compared against the shipped aiwf fragment (`internal/skills/embedded-guidance/aiwf-guidance.md`) and the selected packs under `.guidance/packs/`, which are canonical homes for the rules they carry.

## Acceptance criteria

### AC-1 — The inventory lists every rule in the guidance set once, with its current homes

### AC-2 — Every rule carries one disposition from the closed set

### AC-3 — Every conflict is decided by the maintainer and recorded

### AC-4 — The on-demand documents and the router are specified

### AC-5 — Each host's handwritten primed ceiling is set from the primed rules

## Constraints

- No guidance text changes in this milestone.
- The primed test is E-0092's: nearly every task, or harm before the agent would think to look.
- One home per rule, chosen by audience: operating aiwf in any repository belongs to the shipped fragment; developing aiwf belongs to this repository.
- Conflicts are the maintainer's decisions, presented one at a time, with the evidence from both homes.

## Design notes

- A near-duplicate is one row whose disposition is `merge into` the surviving statement; the survivor's row records the merge.
- A `tighten` row states what the tightened rule must still carry — the obligation and its one-line reason — so review of M-0335 can check the rewrite against it.
- A check-backed rule is `pointer to` only if its diagnostic already states the remedy or M-0337's audit will make it; otherwise it stays `keep` or `tighten`.

## Surfaces touched

- This milestone's body: `## Inventory`, `## Decisions made during implementation`, `## Validation`.
- E-0092's open questions.

## Out of scope

- Any change to guidance text (M-0336, M-0335, M-0337, M-0339).
- The content of external language packs.

## Dependencies

- M-0333 — the guidance set and the measure
- M-0334 — the baseline, recorded before any disposition is carried out

## Coverage notes

- (none)

## References

- D-0089 — external language-content ownership
- D-0091 — prose-presence evidence restriction
- G-0676 — how the guidance grew

## Inventory

| Rule | Current homes | Words | Kind | Placement | Disposition |
|---|---|---|---|---|---|

---

## Release note

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
