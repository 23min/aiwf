---
id: M-0336
title: Delete the copies from CLAUDE.md
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

Remove from `CLAUDE.md` the text that already loads from another home: the rules the shipped fragment carries, the generic Go conventions, and the entity-id asides.

## Closes

- (none)

## Context

The fragment ships every consumer-operating rule and is imported by the root, so a restatement in `CLAUDE.md` is a second copy and the one nothing checks. Every topic of the imported Go module is restated in the Go conventions section. The entity-id asides narrate where a rule came from, which the entity and the commit trailers already carry. D-0091 governs what may be asserted about the file afterwards: absence, not presence.

## Acceptance criteria

### AC-1 — CLAUDE.md carries no rule the fragment also carries

For each anchor the operating-anchors policy pins, its trigger phrases are absent from every `CLAUDE.md` in the repository. **Pass criterion**: the anchors policy, extended, reports an anchor whose phrases appear in a `CLAUDE.md`; on the tree it reports none. **Edge cases**: a phrase inside a markdown link to the fragment is not a restatement. **Code references**: `internal/policies/m0211_guidance_operating_anchors.go`. An absence check holds no text in place, which is why D-0091 permits it; it is weak against rewording, and the ceiling is the stronger evidence.

### AC-2 — CLAUDE.md cites no gap, epic, milestone, or decision id outside a markdown link

No id-shaped token (gap, epic, milestone, decision, or ADR) appears in any `CLAUDE.md` outside a markdown link destination. **Pass criterion**: a structural scan in the shape of the `skill-body-id` check, with link carriers masked, reports each stray id; on the tree it reports none. **Edge cases**: a placeholder in the letter-N form inside backticks is syntax, not a citation; a command example cites a placeholder, never a real id. **Code references**: `internal/check/skill_body_id.go` for the masking; the new scan under `internal/policies/`.

### AC-3 — The generic Go conventions section is deleted

The generic Go conventions section is deleted from root `CLAUDE.md`, its source remaining the Go module imported by `internal/CLAUDE.md`. This is a one-time act, met by the record: Validation cites the commit, whose disposition block names the module as the copy's source. The ceiling step in AC-4 is the mechanical trace.

### AC-4 — The ceiling constant steps down to the post-deletion size

The ceiling constant is lowered to the count the policy reports after the deletions. **Pass criterion**: the policy passes at the new constant and fails at the previous one; Validation records the command and both figures.

## Constraints

- Deletion of copies only; what stays is not reworded.
- Every `CLAUDE.md` commit is its own, with a `copy of <path>` disposition per removed passage; an id aside whose reasoning is nowhere else goes to the entity that owns it first, and its block says `relocated to`.
- One row appended to the iteration log when this lands.

## Design notes

- D-0091 shapes AC-1 and AC-2 as absence and structure checks; neither can hold a sentence in place.
- AC-3 is observational, so this milestone runs under `tdd: advisory`; its other criteria carry tests regardless.

## Surfaces touched

- root `CLAUDE.md`
- `internal/policies/m0211_guidance_operating_anchors.go` and one new scan

## Out of scope

- The pointer cut (M-0337).
- Changing the fragment; that is M-0339.

## Dependencies

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
