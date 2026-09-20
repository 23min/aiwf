---
id: M-0336
title: Remove duplicated project guidance for both hosts
status: draft
parent: E-0092
depends_on:
    - M-0335
tdd: advisory
acs:
    - id: AC-1
      title: CLAUDE.md carries no rule the fragment also carries
      status: open
    - id: AC-2
      title: CLAUDE.md cites no gap, epic, milestone, or decision id outside a markdown link
      status: open
    - id: AC-3
      title: The generic Go conventions section is deleted
      status: open
    - id: AC-4
      title: The ceiling constant steps down to the post-deletion size
      status: open
---

## Goal

Remove from both hosts' development guidance the text that already loads from another home: the rules the shipped fragment carries, the generic Go conventions, and the entity-id asides.

## Closes

- (none)

## Context

The shared operating fragment and externally owned language documents are canonical sources. Root and relocated development prose must not restate them. Generated host copies are expected delivery outputs and are excluded from duplicate-authoring checks by ownership, not by ignoring an entire host file. D-0091 governs the evidence restriction.

## Acceptance criteria

### AC-1 — CLAUDE.md carries no rule the fragment also carries

For each operating anchor, no independently authored copy remains in either host's repository-development guidance. Exclude only the exact generated operating blocks; the generated Codex block legitimately carries the shared source's rules. **Pass criterion**: fixtures distinguish an expected rendered block from an extra handwritten copy in the same file, and the live tree passes. Phrase checks cover only known anchors; semantic duplication remains a review question.

### AC-2 — CLAUDE.md cites no gap, epic, milestone, or decision id outside a markdown link

No id-shaped token (gap, epic, milestone, decision, or ADR) appears in handwritten development guidance outside a markdown link destination. **Pass criterion**: a structural scan in the shape of the `skill-body-id` check, with link carriers masked, reports each stray id; on the tree it reports none. **Edge cases**: a placeholder in the letter-N form inside backticks is syntax, not a citation; a command example cites a placeholder, never a real id. **Code references**: `internal/check/skill_body_id.go` for the masking; the new scan under `internal/policies/`.

### AC-3 — The generic Go conventions section is deleted

Remove generic Go conventions from repository-development prose wherever M-0335 placed them, retaining the selected project-local Go document and its routes for both hosts. Record the removal commit and the canonical source in its disposition. Retain aiwf-specific conventions. This is observational evidence, supported by the measured reduction.

### AC-4 — The ceiling constant steps down to the post-deletion size

Lower each host's ceiling to its measured post-deletion upfront size. The current tree passes at the lowered ceiling; a fixture restoring the larger pre-deletion payload fails. Record commands and both hosts' figures.

## Constraints

- Deletion of copies only; what stays is not reworded.
- Each guidance commit may include its related source and generated outputs together, with a `copy of <path>` disposition per removed passage; an id aside whose reasoning is nowhere else goes to the entity that owns it first, and its block says `relocated to`.
- One row appended to the iteration log when this lands.

## Design notes

- D-0091 shapes AC-1 and AC-2 as absence and structure checks; neither can hold a sentence in place.
- AC-3 is observational, so this milestone runs under `tdd: advisory`; its other criteria carry tests regardless.

## Surfaces touched

- Both host entry points and relocated development guidance, respecting generated ownership.
- The existing anchor policy and structural reference scan.

## Out of scope

- The pointer cut (M-0337).
- Changing the fragment; that is M-0339.

## Dependencies

- E-0092's external delivery and repository migration prerequisite must be complete.
- M-0335 — the root is thin before its copies are judged

## Coverage notes

- (none)

## References

- D-0091, D-0089
- `internal/check/skill_body_id.go` — the link-masking shape

## Release note

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
