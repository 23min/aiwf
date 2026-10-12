---
id: M-0337
title: Audit finding messages and check that guidance paths resolve
status: in_progress
parent: E-0092
depends_on:
    - M-0335
tdd: advisory
acs:
    - id: AC-1
      title: The audit table lists every policy and finding code a chokepointed section names
      status: open
      tdd_phase: red
    - id: AC-2
      title: Every message the audit marks as not stating the fix is corrected and pinned
      status: open
    - id: AC-3
      title: Every guidance enforcement pointer resolves to a policy or finding code
      status: cancelled
    - id: AC-4
      title: Every repository path cited by development guidance exists
      status: open
    - id: AC-5
      title: The ceiling constant reaches its target
      status: open
---

## Goal

Audit the message of every policy and finding code the development guidance names, correct each one that does not say what to do, and check that every repository path the guidance cites exists; close G-0436.

## Closes

- G-0436 — stale `cmd/aiwf/` paths cited by guidance; both citations it names read correctly in the tree, and AC-4 is the check that keeps development guidance's paths resolving.

## Context

The development-guidance set is the one M-0333 defines: both root entry points, `.guidance/project.md` and the on-demand documents it routes to. Generated operating and language blocks are judged through their owning sources; expected rendered copies are not independent rule restatements. E-0092 defines the primed/on-demand split and the two ceilings.

Much of the development guidance documents rules that a policy, a check rule, or a hook enforces. The check does the work; the prose spares one failed-check round trip, which the finding message spares equally once it names the fix. The audit decides, per message, whether it already does, and the messages that do not are corrected.

## Acceptance criteria

### AC-1 — The audit table lists every policy and finding code a chokepointed section names

This body's audit table lists every policy id and finding code named in the development-guidance set at the milestone's start, each with its current message and a verdict: states the fix, or does not. **Pass criterion**: the inventory test (M-0349's design notes) derives the set of ids and codes from those guidance files as they stand at the milestone's start and asserts each has a row in this milestone's table, read through the loader; it passes. **Code references**: the policy-id literals under `internal/policies/`, the kernel's code set under `internal/check/`.

### AC-2 — Every message the audit marks as not stating the fix is corrected and pinned

Every message the table marks as not stating the fix is changed to state it, and a test pins the changed message. **Pass criterion**: each such message has a test asserting its remediation clause, listed in this body at wrap; no check's condition changes, only its text. **Edge cases**: a message shared by several sites is changed once at its source.

### AC-3 — Every guidance enforcement pointer resolves to a policy or finding code

Cancelled. No guidance section is cut to a pointer, so the check would guard only the pointers already written: the development guidance holds one of the form "enforced by `<id>`", and it names a policy that exists. AC-4's check covers the pointers written as paths. A pointer naming a policy id, a finding code or a Go function is not checked.

### AC-4 — Every repository path cited by development guidance exists

Every backticked repository-relative path cited in development guidance exists. **Pass criterion**: a relationship check reports a cited path absent from the tree; on the tree it reports none. **Edge cases**: a glob-shaped citation is matched as a glob; a placeholder in angle brackets is not a path; a path under a gitignored directory is checked on disk. **Code references**: a new policy under `internal/policies/`; G-0436's two stale citations are the fixture.

### AC-5 — The ceiling constant reaches its target

Both hosts' handwritten primed words, as defined by E-0092 and measured by M-0333's policy, pass at the ceiling M-0349 set for each host. Record each command and result, with the fragment's words reported separately. Personal/global words and task-loaded text are observed rather than measured here: M-0334 AC-3 records both at the baseline, and M-0338 AC-2 compares the task-loaded text after. A document required in full before any task counts as primed and cannot be excluded to meet the target.

## Constraints

- Message edits change text only; a condition change is a kernel change and out of scope.
- One row appended to the iteration log when this lands.

## Design notes

- The audit table's verdicts are what a later cut of a check-backed section starts from: a section can become a pointer once its message states the fix, which E-0092 sets as the precondition.

## Surfaces touched

- messages under `internal/policies/` and `internal/check/`, with the tests that pin them
- a new policy under `internal/policies/` for the paths development guidance cites
- the inventory test, for AC-1's coverage of the audit table

## Out of scope

- Any change to what a check enforces.
- The fragment.
- Cutting a check-backed section to a pointer: M-0349's inventory decides `pointer to` for no row, and its test refuses a removal that does not carry its row's decision.

## Dependencies

- M-0335 — the routed guidance the audit and the path check read
- M-0349 — the ceiling AC-5 measures against

## Coverage notes

- (none)

## References

- G-0436
- G-0676 — how root guidance grows

## Release note

## Decisions made during implementation

- No guidance section is cut to a pointer. M-0349's inventory decides `pointer to` for no row, its test refuses a removal that does not carry its row's decision, and the primed load already meets both targets, so a cut would override recorded decisions to save no primed words.
- AC-3 is cancelled, for the reason under its heading.
- The milestone runs `tdd: advisory`: AC-5 is met by the ceiling test, which already passes, so it has no red phase; AC-1, AC-2 and AC-4 record their phases as their tests are written.

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
