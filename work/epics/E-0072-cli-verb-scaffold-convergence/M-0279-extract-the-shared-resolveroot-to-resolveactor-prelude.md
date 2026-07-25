---
id: M-0279
title: Extract the shared ResolveRoot to ResolveActor prelude
status: draft
parent: E-0072
depends_on:
    - M-0278
tdd: advisory
acs:
    - id: AC-1
      title: Shared ResolveRoot to ResolveActor prelude helper exists
      status: open
    - id: AC-2
      title: All prelude-duplicating verbs route through the shared helper
      status: open
    - id: AC-3
      title: Prelude behavior including the usage-error arm is unchanged
      status: open
---
# M-0279 — Extract the shared ResolveRoot to ResolveActor prelude

## Goal

Introduce one shared `ResolveRoot → ResolveActor` prelude helper (with its identical usage-error arm) and route the ~23 verbs that duplicate it through it.

## Context

After the diagnostic block is single-sourced, the second-largest per-verb scaffold is the `ResolveRoot → ResolveActor` prelude: resolve the repo root, resolve the actor, and bail with the same usage-error arm when either fails. It is duplicated across ~23 verbs. Builds on the BeginVerbDiag milestone (same `cliutil` surface, same migration shape).

## Acceptance criteria

<!-- ACs created via `aiwf add ac`; contracts filled below. -->

### AC-1 — Shared ResolveRoot to ResolveActor prelude helper exists

### AC-2 — All prelude-duplicating verbs route through the shared helper

### AC-3 — Prelude behavior including the usage-error arm is unchanged

## Constraints

- Behavior-preserving: the resolved root, the resolved actor, the usage-error message, and the exit path are unchanged per verb.
- Extraction lands in today's `cliutil`; no anticipatory restructuring for the G-0227 split.
- Verbs with a legitimately different prelude (if any surface during migration) are documented as intentional non-members rather than forced through the helper.

## Out of scope

- The diagnostic block (its own milestone, a prerequisite).
- The regression-guard structural test (its own milestone).

## Dependencies

- The BeginVerbDiag milestone — same package surface; sequenced after it to keep the two migrations reviewable in isolation.

## References

- E-0072 — parent epic.
- G-0447 — the convergent-duplication cleanup this seam completes.
