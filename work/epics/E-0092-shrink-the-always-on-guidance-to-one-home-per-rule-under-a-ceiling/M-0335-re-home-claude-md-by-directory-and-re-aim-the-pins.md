---
id: M-0335
title: Re-home CLAUDE.md by directory and re-aim the pins
status: draft
parent: E-0092
depends_on:
    - M-0333
    - M-0334
tdd: required
acs:
    - id: AC-1
      title: Root CLAUDE.md imports only the shipped fragment
      status: open
    - id: AC-2
      title: Nested CLAUDE.md files exist under internal, cmd, docs, and work
      status: open
    - id: AC-3
      title: Every pin on a moved passage is re-aimed or retired with a recorded reason
      status: open
    - id: AC-4
      title: The ceiling constant steps down to the re-homed size
      status: open
---

## Goal

Thin the root `CLAUDE.md` to what every session needs and move the rest under the directories it governs, where the harness loads it only when a session reads files there; re-aim the tests that pin moved passages.

## Closes

- (none)

## Context

A subdirectory `CLAUDE.md` loads lazily and `@` imports resolve relative to the importing file, up to four hops (Claude Code memory documentation). Nearly all Go lives under `internal/`, with a handful of files under `cmd/`. The tests that pin root passages are the ones G-0676's floor command lists; each reads the root by path and must be re-aimed when its passage moves. The imported Go module belongs with the Go conventions, under `internal/`, and the Python and TypeScript imports load languages this repository does not use.

## Acceptance criteria

### AC-1 — Root CLAUDE.md imports only the shipped fragment

Root `CLAUDE.md`'s `@` import lines resolve to exactly one path, the shipped fragment. **Pass criterion**: a structural test over the root's import lines finds the fragment import and nothing else; no home-relative import remains at root. **Edge cases**: an `@` inside a code span is not an import. **Code references**: the ceiling policy's import resolver, reused.

### AC-2 — Nested CLAUDE.md files exist under internal, cmd, docs, and work

`internal/CLAUDE.md`, `cmd/CLAUDE.md`, `docs/CLAUDE.md`, and `work/CLAUDE.md` exist; `cmd/CLAUDE.md` imports `internal/CLAUDE.md` by relative path; `internal/CLAUDE.md` imports the operator's Go module. **Pass criterion**: a structural test over the four files' existence and their import lines. **Edge cases**: the relative import resolves from `cmd/`, not from the repository root. **Code references**: the same test.

### AC-3 — Every pin on a moved passage is re-aimed or retired with a recorded reason

Every test in the list G-0676's floor command produces, re-run at this milestone's start and recorded here, either passes against the passage's new file or is removed with its reason in this body's table. **Pass criterion**: the policy suite is green and the table accounts for every listed test; the list and the command sit in Validation. The mechanical half is the green suite; the accounting is held at review.

### AC-4 — The ceiling constant steps down to the re-homed size

The ceiling constant is lowered to the count the policy reports after the move. **Pass criterion**: the policy passes at the new constant and fails at the previous size; Validation records the command and both figures.

## Constraints

- Content moves; nothing is reworded, and nothing is deleted except the two language imports, each with a `deleted` disposition.
- Every `CLAUDE.md` commit is its own, with a disposition block per moved passage naming the file it moved to.
- A pin is re-aimed, not rewritten; a retired pin's reason is recorded here.
- What stays at root is what every session needs regardless of directory: the engineering principles, the commitments as a pointer, the worktree default, the validation cadence, the commit conventions, and the pointers the fence added.
- One row appended to the iteration log in `docs/design/growth.md` when this lands.

## Design notes

- The partition: `internal/` takes the Go conventions, test discipline, coverage, CLI conventions, type design, verb design, and the enforcement map; `cmd/` re-imports it; `docs/` takes the documentation hierarchy and ADR authoring; `work/` takes the entity-authoring and id-collision material. Anything the reviewer cannot place stays at root.
- The ceiling counts root plus repo-resolved imports, so the nested files are outside it by construction; their size is reported by the growth script, not capped.
- `aiwf update`'s behaviour on a nested `CLAUDE.md` is verified here and recorded in Validation; it maintains only the root import marker today.

## Surfaces touched

- `CLAUDE.md`, `internal/CLAUDE.md`, `cmd/CLAUDE.md`, `docs/CLAUDE.md`, `work/CLAUDE.md`
- the pinning tests under `internal/policies/`

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
- D-0089 — why the Go module import moves with the Go conventions

## Release note

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
