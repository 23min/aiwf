---
id: M-0336
title: Remove duplicated project guidance for both hosts
status: draft
parent: E-0092
depends_on:
    - M-0349
    - M-0334
tdd: advisory
acs:
    - id: AC-1
      title: Development guidance has no independent copy of operating rules
      status: open
    - id: AC-2
      title: Development guidance cites entity ids only through links
      status: open
    - id: AC-3
      title: The generic Go conventions section is deleted
      status: open
    - id: AC-4
      title: The ceiling constant steps down to the post-deletion size
      status: open
    - id: AC-5
      title: Every removal commit traces to an inventory row and its decided home
      status: open
    - id: AC-6
      title: An audience move is one commit that adds the rule and removes the old copy
      status: open
---

## Goal

Remove from both hosts' development guidance the text that already loads from another home — the rules the shipped fragment carries, the generic Go conventions, and the entity-id asides — and move each rule the inventory places in the other home, so every rule ends once, where its audience says.

## Closes

- (none)

## Context

The shared operating fragment and the selected language packs under `.guidance/packs/` are canonical sources. Development guidance must not restate them. M-0349's inventory records which passages are copies and of what, and which rules belong to the other audience; this milestone deletes exactly those copies and makes exactly those moves, before M-0335 relocates what remains. A move into the shipped fragment changes what every consumer loads, so it carries a CHANGELOG entry under E-0092 and follows the shipped-surface rules: no entity ids, no repository paths, no development history. From this milestone on, a test holds every removal commit to the inventory. Generated host copies are expected delivery outputs and are excluded from duplicate-authoring checks by ownership, not by ignoring an entire host file. D-0102 governs the evidence restriction.

## Acceptance criteria

### AC-1 — Development guidance has no independent copy of operating rules

For each operating anchor, no independently authored copy remains in either host's repository-development guidance. Exclude only the exact generated operating blocks; the generated Codex block legitimately carries the shared source's rules. **Pass criterion**: fixtures distinguish an expected rendered block from an extra handwritten copy in the same file, and the live tree passes. Phrase checks cover only known anchors; semantic duplication remains a review question.

### AC-2 — Development guidance cites entity ids only through links

No id-shaped token (gap, epic, milestone, decision, or ADR) appears in handwritten development guidance outside a markdown link destination. **Pass criterion**: a structural scan in the shape of the `skill-body-id` check, with link carriers masked, reports each stray id; on the tree it reports none. **Edge cases**: a placeholder in the letter-N form inside backticks is syntax, not a citation; a command example cites a placeholder, never a real id. **Code references**: `internal/check/skill_body_id.go` for the masking; the new scan under `internal/policies/`.

### AC-3 — The generic Go conventions section is deleted

Remove from `CLAUDE.md` the Go conventions the selected Go pack (`.guidance/packs/go/cobra/guide.md`) already carries, as the inventory marks them, retaining the pack and its route for both hosts. Record the removal commit and the canonical source in its disposition. Retain aiwf-specific conventions, and any rule the inventory marks as a genuine project override of the pack. This is observational evidence, supported by the measured reduction.

### AC-4 — The ceiling constant steps down to the post-deletion size

Lower each host's ceiling to its measured post-deletion upfront size. The current tree passes at the lowered ceiling; a fixture restoring the larger pre-deletion payload fails. Record commands and both hosts' figures.

### AC-5 — Every removal commit traces to an inventory row and its decided home

Each commit that removes text from development guidance or the shipped fragment names, in its disposition block, the inventory row it carries out, and the block agrees with the row's decision: the same kind of disposition and the same destination. **Pass criterion**: a test reads each disposition block in E-0092's commit range and the inventory table in M-0349's body through the loader, and reports a removal that names no row, names a row that does not exist, disagrees with the row's decision, or names a destination that does not exist; fixtures cover each, and the tree passes. Whether the destination states the rule is held at review, since D-0070 and D-0102 bar pinning guidance wording. **Code references**: a new test under `internal/policies/`, reading commits through the existing commit-range machinery.

### AC-6 — An audience move is one commit that adds the rule and removes the old copy

A rule the inventory moves between homes — `move to the shipped fragment` or `move to repository guidance <document>` — moves in one commit that adds it to its new home and removes it from the old, so no commit leaves it in both homes or in neither. A move into the shipped fragment regenerates the managed `AGENTS.md` block in the same commit. **Pass criterion**: AC-5's test also reports a move whose commit does not change its destination file; a fixture covers it, and the tree passes.

## Constraints

- Deletion of copies and audience moves only; what stays or moves is not reworded.
- Each guidance commit may include its related source and generated outputs together, with a `copy of <path>` disposition per removed passage; an id aside whose reasoning is nowhere else goes to the entity that owns it first, and its block says `relocated to`; a move's block says `relocated to <path>`. Every block names its inventory row.
- One row appended to the iteration log when this lands.
- A pin on a passage this milestone deletes is retired with its reason, and its entry leaves the guidance-reader list.

## Design notes

- D-0102 shapes AC-1 and AC-2 as absence and structure checks; neither can hold a sentence in place, and each is listed as a guidance reader.
- AC-3 is observational, so this milestone runs under `tdd: advisory`; its other criteria carry tests regardless.
- AC-5's test maps each inventory decision to the block dispositions it allows: `delete as copy of` to `copy of`, the three moves to `relocated to`, `pointer to` to `pointer to`, and `merge into` and `tighten` to the rewording form the fence accepts. Until M-0338 AC-3, a row not yet carried out is not a finding.

## Surfaces touched

- Both host entry points and `.guidance/project.md`, respecting generated ownership.
- The existing anchor policy and structural reference scan.
- `internal/skills/embedded-guidance/aiwf-guidance.md` and `CHANGELOG.md`, for moves into and out of the shipped fragment.
- The reconciliation test under `internal/policies/`.

## Out of scope

- The pointer cut (M-0337).
- Rewording or shortening the shipped fragment's rules; the inventory records any such suggestion for a later epic.

## Dependencies

- M-0349 — the inventory records which passages are copies, where their canonical home is, and which rules move between audiences
- M-0334 — the baseline, recorded before any decision is carried out

## Coverage notes

- (none)

## References

- D-0102, ADR-0054
- `internal/check/skill_body_id.go` — the link-masking shape

## Release note

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
