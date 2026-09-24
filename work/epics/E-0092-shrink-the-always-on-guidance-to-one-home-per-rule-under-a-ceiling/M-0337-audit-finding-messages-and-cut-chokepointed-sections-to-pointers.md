---
id: M-0337
title: Audit finding messages and cut chokepointed sections to pointers
status: draft
parent: E-0092
depends_on:
    - M-0335
tdd: required
acs:
    - id: AC-1
      title: The audit table lists every policy and finding code a chokepointed section names
      status: open
    - id: AC-2
      title: Every message the audit marks as not stating the fix is corrected and pinned
      status: open
    - id: AC-3
      title: Every guidance enforcement pointer resolves to a policy or finding code
      status: open
    - id: AC-4
      title: Every repository path cited by development guidance exists
      status: open
    - id: AC-5
      title: The ceiling constant reaches its target
      status: open
---

## Goal

Turn every section that documents a rule a check already enforces into a one-line pointer, once the check's message says what to do; close G-0436.

## Closes

- G-0436 — stale `cmd/aiwf/` paths cited by `CLAUDE.md`; AC-4 is the check that keeps it closed.

## Context

The development-guidance set is the one M-0333 defines: both root entry points, `.guidance/project.md` and the on-demand documents it routes to. Generated operating and language blocks are judged through their owning sources; expected rendered copies are not independent rule restatements. E-0092 defines the primed/on-demand split and the two ceilings.

Most of the root's remaining bulk documents rules that a policy, a check rule, or a hook enforces. The check does the work; the prose spares one failed-check round trip, which a pointer spares equally once the finding message names the fix. The audit decides, per message, whether it already does.

## Acceptance criteria

### AC-1 — The audit table lists every policy and finding code a chokepointed section names

This body's audit table lists every policy id and finding code named in the development-guidance set at the milestone's start, each with its current message and a verdict: states the fix, or does not. **Pass criterion**: a test derives the set of ids and codes from those guidance files and asserts each has a row in this milestone's table, read through the loader; on the tree it passes. **Code references**: the policy-id literals under `internal/policies/`, the kernel's code set under `internal/check/`.

### AC-2 — Every message the audit marks as not stating the fix is corrected and pinned

Every message the table marks as not stating the fix is changed to state it, and a test pins the changed message. **Pass criterion**: each such message has a test asserting its remediation clause, listed in this body at wrap; no check's condition changes, only its text. **Edge cases**: a message shared by several sites is changed once at its source.

### AC-3 — Every guidance enforcement pointer resolves to a policy or finding code

Every pointer in development guidance of the form "enforced by `<id>`" resolves to an existing policy id or finding code. **Pass criterion**: a relationship check derives the pointer set from the files and the id set from the code, and reports a pointer that resolves to neither; on the tree it reports none. **Code references**: a new policy under `internal/policies/`.

### AC-4 — Every repository path cited by development guidance exists

Every backticked repository-relative path cited in development guidance exists. **Pass criterion**: a relationship check reports a cited path absent from the tree; on the tree it reports none. **Edge cases**: a glob-shaped citation is matched as a glob; a placeholder in angle brackets is not a path; a path under a gitignored directory is checked on disk. **Code references**: the same policy; G-0436's two stale citations are the fixture.

### AC-5 — The ceiling constant reaches its target

Both hosts' handwritten primed words, as defined by E-0092 and measured by M-0333's policy, pass at the ceiling M-0349 set for each host. Record each command and result, with the fragment's words, task-loaded text and personal/global text reported separately. A document required in full before any task counts as primed and cannot be excluded to meet the target.

## Constraints

- A section becomes a pointer only after its row's verdict is "states the fix".
- Message edits change text only; a condition change is a kernel change and out of scope.
- Each guidance commit may include its related source and generated outputs together, with a `pointer to <id>` disposition per compressed section.
- One row appended to the iteration log when this lands.

## Design notes

- The pointer shape is fixed so AC-3 can parse it: the section heading, one sentence, and "enforced by `<id>`".
- A gotcha that is not a rule, found inside a compressed section, lands in the skill for its task before the section is cut; the disposition block says `relocated to`.

## Surfaces touched

- Both host entry points and canonical development guidance
- messages under `internal/policies/` and `internal/check/`

## Out of scope

- Any change to what a check enforces.
- The fragment.

## Dependencies

- M-0335 — on-demand rules have moved and the primed remainder is tightened before check-backed rules are compressed
- M-0349 — the ceiling this milestone reaches

## Coverage notes

- (none)

## References

- G-0436
- G-0676 — the mechanism the pointers no longer feed

## Release note

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
