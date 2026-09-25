---
id: M-0335
title: Route on-demand guidance through the project router and tighten what moves
status: draft
parent: E-0092
depends_on:
    - M-0349
    - M-0336
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

Make `.guidance/project.md` a short task router, move every on-demand rule into the documents it names, and apply the inventory's merge and tighten dispositions, so the root `CLAUDE.md` holds only primed rules.

## Closes

- (none)

## Context

M-0349's inventory gives each rule a disposition and names the on-demand documents; M-0336 has already deleted the copies. The routing block both entry points carry sends a task to `.guidance/project.md` first, then the index and the packs it needs. Today `.guidance/project.md` says to read `CLAUDE.md` in full and that it takes precedence over the packs, and the `AGENTS.md` preamble requires the same full read for Codex. After this milestone `CLAUDE.md` is the one home of primed development rules, Codex's required read of it covers only those, and everything else is reached through the router.

## Acceptance criteria

### AC-1 — Both root entry points route to project-local guidance

`.guidance/project.md` names each on-demand document and the tasks that need it, and states which primed rules in `CLAUDE.md` override a pack. The `AGENTS.md` preamble requires only the primed `CLAUDE.md` content. No home-directory language import returns. **Pass criterion**: a relationship check resolves every route in `.guidance/project.md` and both entry points to an existing file and rejects a missing target. Generated blocks change only through their owning updater.

### AC-2 — Both hosts read relevant guidance before root-started edits

Record fresh sessions for both hosts, started at the repository root, for a Go change, a Python script, a TypeScript test, a new file and a prose-only task. **Pass criterion**: tool events show each host reading the relevant on-demand document or pack through the router before its first relevant edit, and the prose-only task reading no language pack. File existence alone does not establish delivery.

### AC-3 — Every pin on a moved passage is re-aimed or retired with a recorded reason

Every test in the list G-0676's floor command produces, re-run at this milestone's start and recorded here, is re-aimed at a relationship check or an absence check — never at the passage's new file, which D-0102 counts as guidance too — or retired, with its reason in this body's table; its entry in the guidance-reader list changes or goes with it. **Pass criterion**: the policy suite is green and the table accounts for every listed test; the list and the command sit in Validation. The mechanical half is the green suite; the accounting is held at review.

### AC-4 — The ceiling constant steps down to the re-homed size

Lower each host's handwritten primed ceiling to its measured size after relocation. **Pass criterion**: the relocated tree passes at the new ceilings; fixtures restoring the old primed payload fail when it is larger. Record commands, before/after counts and on-demand loads.

### AC-5 — No instruction file exists below the repository root

**Pass criterion**: a policy fails when a tracked `CLAUDE.md` or `AGENTS.md` exists anywhere but the repository root, naming the path; on the tree it reports none.

### AC-6 — Merge and tighten dispositions are applied as the inventory records

Every rule M-0349 marks "merge" or "tighten" is rewritten in its one home, and every rule marked "move on demand" is in the document the inventory names. **Pass criterion**: a relationship check over the inventory table resolves each "move on demand" row to its destination document; the rewording itself is held at review, and its effect is judged by M-0338's rubric.

## Constraints

- One instruction file per host, at the root; no directory entry files.
- Rewording is allowed where the inventory records it; each removed or rewritten passage carries its disposition block in the commit body.
- Preserve source/output ownership; generated blocks change through their owner.
- Re-aim pins or retire them with reasons. Do not create duplicate prose merely to satisfy an old pin.
- Record the growth delta and both hosts' loading observations.

## Design notes

- `CLAUDE.md` is the one home of primed development rules. Codex reaches them through the `AGENTS.md` preamble's required read, which is why that read counts toward Codex's primed load.
- The on-demand documents and their locations are the ones M-0349 names.
- Exercise `aiwf update` in a disposable checkout, with `HOME` pointed at a scratch directory, and verify it preserves the handwritten router and regenerates the expected routes without restoring obsolete home-directory imports.

## Surfaces touched

- `CLAUDE.md`, the `AGENTS.md` preamble, `.guidance/project.md` and the on-demand documents.
- Existing pinning tests and guidance routing checks.

## Out of scope

- Deleting copies (M-0336) and the pointer cut (M-0337).
- The shared fragment (M-0339).

## Dependencies

- M-0349 — the dispositions and the on-demand document set
- M-0336 — copies are gone before the remainder moves

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
