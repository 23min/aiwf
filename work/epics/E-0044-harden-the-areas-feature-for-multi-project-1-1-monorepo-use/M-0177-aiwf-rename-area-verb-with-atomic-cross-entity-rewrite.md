---
id: M-0177
title: aiwf rename-area verb with atomic cross-entity rewrite
status: in_progress
parent: E-0044
tdd: required
acs:
    - id: AC-1
      title: rename-area rewrites the member and all referencing entities atomically
      status: open
      tdd_phase: red
    - id: AC-2
      title: the rename commit carries rename-area trailers and aiwf history renders it
      status: open
      tdd_phase: red
    - id: AC-3
      title: rename-area refuses undeclared old or already-declared new; no partial write
      status: open
      tdd_phase: red
    - id: AC-4
      title: rename-area <new> <old> reverses a prior rename
      status: open
      tdd_phase: red
    - id: AC-5
      title: rename-area ships tab-completion for old, a skill, and --help
      status: open
      tdd_phase: red
---
## Goal

Make renaming a declared area safe: `aiwf rename-area <old> <new>` renames the `aiwf.yaml` member and atomically rewrites every entity that references it, in one trailered commit — the same referential-integrity discipline `aiwf reallocate` applies to ids.

## Context

Today, renaming an area in `aiwf.yaml` (or removing one) leaves every entity that still carries the old value orphaned: `area-unknown` flags them at warning, and the grouping view silently buckets them into the complement. No verb rewrites the references. This milestone adds it, closing the Tier-0 referential-integrity hole on the area closed set.

## Acceptance criteria

<!-- Candidate ACs, formalized via `aiwf add ac <id> --title "..."` at start-milestone. -->

Candidate behaviors to formalize at start-milestone:

- `aiwf rename-area <old> <new>` renames the member in `aiwf.yaml` and rewrites the `area` frontmatter of every referencing entity in a single commit.
- The commit carries `aiwf-verb: rename-area` + entity/actor trailers; `aiwf history` renders the rename.
- Refuses when `<new>` already names a declared member, or `<old>` is not declared — clear error, no partial write.
- The rename reverses via the same verb (`rename-area <new> <old>`).
- Tab-completion offers declared members for `<old>`; skill coverage + `--help` ship with it.

## Constraints

- Atomic: either the `aiwf.yaml` member and all entity rewrites land, or nothing does — one commit, abort-before-commit on any failure.
- Single source of truth: the member set in `aiwf.yaml` is the authority; the verb never invents members.
- "What undoes this?" — the same verb with swapped args; documented at design.

## Design notes

- Mirror `aiwf reallocate`'s tree-walk-and-rewrite + trailer-stamp shape.

## Out of scope

- `paths:` (Tier 1) — rename-area operates on the label; the keystone milestone owns carrying any paths along.
- Renaming the display-only `areas.default` label (not a member).

## Dependencies

- None. Independent Tier-0; parallel with the other Tier-0 milestones.

## References

- `internal/config/config.go` — the `Areas` member set rewritten.
- `aiwf reallocate` — the precedent for atomic cross-tree reference rewrite + trailers.
- ADR-0006 — skills policy (the new verb needs a skill or allowlist entry).

### AC-1 — rename-area rewrites the member and all referencing entities atomically

### AC-2 — the rename commit carries rename-area trailers and aiwf history renders it

### AC-3 — rename-area refuses undeclared old or already-declared new; no partial write

### AC-4 — rename-area <new> <old> reverses a prior rename

### AC-5 — rename-area ships tab-completion for old, a skill, and --help

