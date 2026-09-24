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

The table in `## Inventory` below has one row per rule. A row names the rule, every place it is stated today, its words, and whether it is a judgment rule or a check-backed one. **Pass criterion**: a relationship check derives every `##` and `###` section of the handwritten guidance set and asserts each is cited as a home by at least one row, read through the loader; on the tree it passes.

### AC-2 — Every rule carries one disposition from the closed set

Each row carries a placement — primed or on demand — by E-0092's test, and one disposition: `keep`, `merge into <rule>`, `tighten`, `move on demand to <document>`, `pointer to <policy or finding code>`, `delete as copy of <home>`, or `conflict`. **Pass criterion**: a check over the table rejects a row with no placement, no disposition, or a disposition outside the set; on the table it reports none.

### AC-3 — Every conflict is decided by the maintainer and recorded

A rule that contradicts another, in the guidance set or against the fragment, a pack or a skill, is presented to the maintainer one decision at a time. The decision and the rule it leaves standing are recorded in `## Decisions made during implementation`, and the row's disposition is replaced by the outcome. A decision with reach beyond this repository is recorded as a decision entity. **Pass criterion**: no row ends the milestone marked `conflict`.

### AC-4 — The on-demand documents and the router are specified

The on-demand documents are named, with their paths and the tasks each serves, and the `.guidance/project.md` router entries that reach them are drafted here. Every `move on demand to <document>` row names one of them. **Pass criterion**: the check from AC-2 rejects a row whose destination is not in the named set.

### AC-5 — Each host's handwritten primed ceiling is set from the primed rules

Sum the words of the rules placed primed, after their recorded dispositions, for each host, and set that as the target ceiling M-0337 reaches. **Pass criterion**: the figures and the command that derives them from the table are recorded in Validation, and E-0092's open question on the ceiling is resolved by an edit that cites them.

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
