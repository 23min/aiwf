---
id: M-0223
title: Guard the unconditional authorize-opener grep in the read verbs
status: draft
parent: E-0054
tdd: required
acs:
    - id: AC-1
      title: guard predicate returns skip for events with no scope data
      status: open
      tdd_phase: red
    - id: AC-2
      title: history and show output identical for scoped and scopeless entities
      status: open
      tdd_phase: red
    - id: AC-3
      title: measured read-verb wall-time delta recorded in Validation
      status: open
      tdd_phase: red
---
## Goal

Stop `aiwf history` (default text) from running a repo-wide `git log` grep on every
invocation. `BuildScopeEntityMap` (`internal/cli/history/history.go`) greps the whole
tree for `aiwf-verb: authorize` openers to build a global SHA→scope map, and it is
called **unconditionally** — even when the entity has no authorization and the whole
repo holds only a handful of authorize openers. On a milestone with zero scopes the
text path measured ~2.1s vs ~1.2s for `--format=json` (which skips the grep): ~0.9s
of pure waste per call, ~40% of the default command's wall time.

Guard it: skip `BuildScopeEntityMap` when the entity carries no authorization/scope
data (no `authorized_by`, no `aiwf-scope-ends`); when scope data is present, bound
the grep to the referenced SHAs rather than the whole history. Zero correctness
change — an entity with no scopes renders identical output, an entity with scopes
renders identical scope info. The shared scope map benefits `render` too.

## Notes

- Correctness oracle: the guarded path must produce byte-identical `aiwf history`
  output (text and JSON) to the unguarded path for both an entity **with** and an
  entity **without** authorization scopes — that pairing is the AC fixture.
- This is orthogonal to the render single-pass milestone but shares the theme
  (remove a whole-history grep that shouldn't run); land it independently.

## Acceptance criteria

### AC-1 — guard predicate returns skip for events with no scope data

### AC-2 — history and show output identical for scoped and scopeless entities

### AC-3 — measured read-verb wall-time delta recorded in Validation

