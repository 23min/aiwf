---
id: M-0335
title: Re-home project guidance for Claude and Codex
status: draft
parent: E-0092
depends_on:
    - M-0333
    - M-0334
tdd: required
acs:
    - id: AC-1
      title: Both root entry points route to project-local guidance
      status: open
    - id: AC-2
      title: Both hosts read relevant guidance before root-started edits
      status: open
    - id: AC-3
      title: Every pin on a moved passage is re-aimed or retired with a recorded reason
      status: open
    - id: AC-4
      title: The ceiling constant steps down to the re-homed size
      status: open
    - id: AC-5
      title: No instruction file exists below the repository root
      status: open
    - id: AC-6
      title: Merge and tighten dispositions are applied as the inventory records
      status: open
---

## Goal

Thin both host entry points and move task-specific repository development rules into canonical project-local documents, with explicit routing that works from a root-started session.

## Closes

- (none)

## Context

E-0092's delivery prerequisite supplies selected language guidance in the repository. This milestone relocates aiwf-specific development rules. It must not assume Claude's nested-file loading is also Codex's loading mechanism. Go code, Python scripts and TypeScript tests each retain a route to their selected guidance.

## Acceptance criteria

### AC-1 — Both root entry points route to project-local guidance

The root Claude entry point retains its operating-fragment import and concise project-local task routing, with no home-directory language imports. Codex has equivalent native routing without treating Claude import syntax as an automatic read. **Pass criterion**: structural checks resolve each route and reject a missing selected target; observed task-specific loading is covered by AC-2. Changes to generated blocks use their owning updater.

### AC-2 — Both hosts read relevant guidance before root-started edits

Place concise directory entry files under internal, cmd, docs and work for Claude, referencing canonical project-local development guidance. Give Codex explicit routes to those same canonical documents from its root entry point; do not depend on automatic discovery of descendants. Shared Go-development rules serve both internal and cmd without duplicating their prose. **Pass criterion**: check route resolution mechanically and record fresh root-started sessions for both hosts that read the relevant instructions before their first edit, including a new file. No project-language route depends on a home directory. File existence alone does not establish delivery.

### AC-3 — Every pin on a moved passage is re-aimed or retired with a recorded reason

Every test in the list G-0676's floor command produces, re-run at this milestone's start and recorded here, either passes against the passage's new file or is removed with its reason in this body's table. **Pass criterion**: the policy suite is green and the table accounts for every listed test; the list and the command sit in Validation. The mechanical half is the green suite; the accounting is held at review.

### AC-4 — The ceiling constant steps down to the re-homed size

Lower each host's ceiling to its measured upfront size after relocation. **Pass criterion**: the relocated tree passes at the new ceilings; fixtures restoring the old upfront payload fail when it is larger. Record commands, before/after counts and conditional-task loads. Files required upfront remain counted even when nested.

### AC-5 — No instruction file exists below the repository root

### AC-6 — Merge and tighten dispositions are applied as the inventory records

## Constraints

- Relocate rules without changing their meaning; language delivery has already migrated in the prerequisite.
- Preserve source/output ownership and commit each logical guidance move with its dispositions.
- Re-aim pins or retire them with reasons. Do not create duplicate prose merely to satisfy an old pin.
- Keep universal collaboration and repository-entry instructions reachable before work.
- Record the growth delta and both hosts' loading observations.

## Design notes

- Partition aiwf-specific development material by task: Go implementation, CLI work, documentation, and entity authoring. Share a canonical document where directories need the same rules.
- Directory placement alone does not remove a document from the upfront count; its required loading behavior determines that.
- Exercise `aiwf update` in a disposable checkout and verify it preserves handwritten development guidance and regenerates the expected routes without restoring obsolete home-directory imports.

## Surfaces touched

- Both root host entry points, directory entry files, and canonical development guidance.
- Existing pinning tests and guidance routing checks.

## Out of scope

- Deleting copies (M-0336) and the pointer cut (M-0337).
- Any change to what a pinned passage says.

## Dependencies

- M-0333 — the fence and the ceiling
- M-0334 — the baseline, taken before anything moves

## Coverage notes

- (none)

## References

- G-0676 — the floor command that lists the pins
- D-0089 — selected language conventions have an external source and tracked project delivery

## Release note

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
