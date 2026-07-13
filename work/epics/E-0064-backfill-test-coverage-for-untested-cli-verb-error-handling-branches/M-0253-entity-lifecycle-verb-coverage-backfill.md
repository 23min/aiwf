---
id: M-0253
title: Entity-lifecycle verb coverage backfill
status: in_progress
parent: E-0064
depends_on:
    - M-0252
tdd: required
acs:
    - id: AC-1
      title: Entity-lifecycle flagged branches are tested or documented
      status: open
      tdd_phase: done
    - id: AC-2
      title: Coverage gate is clean for the entity-lifecycle group
      status: open
      tdd_phase: red
---

## Goal

Clear every branch `branch-coverage-audit` currently flags in the
entity-lifecycle verb group — `add`, `promote`, `retitle`, `rename`,
`reallocate`, `cancel`, `milestone`, `update`, `rewidth`, `editbody` —
using the shared failure fixtures M-0252 builds.

## Context

M-0252 lands the reusable fixtures for the failure modes these guards
share (root-resolution, actor-resolution, lock contention, malformed tree,
bad `--format`). Entity-lifecycle verbs carry the largest concentration of
flagged sites among the four consumer groups, since every one of them
shares the same `ResolveRoot`/`ResolveActor`/`AcquireRepoLock`/`tree.Load`
guard shape ahead of its verb-specific logic.

## Acceptance criteria

### AC-1 — Entity-lifecycle flagged branches are tested or documented

Every branch `branch-coverage-audit` flags (base = the commit before
M-0238/AC-3's rename) within `internal/cli/{add,promote,retitle,rename,
reallocate,cancel,milestone,update,rewidth,editbody}` carries either a
passing test (reusing M-0252's fixtures where the failure mode matches)
or a `//coverage:ignore <reason>` naming why the branch isn't
triggerable, mirroring `internal/cli/archive/archive.go:120`.

### AC-2 — Coverage gate is clean for the entity-lifecycle group

`make coverage-gate`, run with `AIWF_COVERAGE_BASE` set to the
pre-M-0238 commit, reports zero findings for the files listed in AC-1.

## Constraints

- Reuse M-0252's fixtures for shared failure modes rather than
  reimplementing them per file; only build a new fixture for a
  verb-specific failure mode not already covered.
- Per-site judgment only: real test where triggerable, honest
  `//coverage:ignore <reason>` otherwise.

## Out of scope

- Contract, diagnostic, bulk-input, and non-CLI infra files — M-0252,
  M-0254, M-0255, and M-0256's job.
- Any change to error-handling behavior beyond what's needed to make a
  branch testable.

## Dependencies

- M-0252 — its shared fixtures must exist before this milestone starts.

## References

- **E-0064** — parent epic.
- **M-0252** — shared fixtures this milestone consumes.
