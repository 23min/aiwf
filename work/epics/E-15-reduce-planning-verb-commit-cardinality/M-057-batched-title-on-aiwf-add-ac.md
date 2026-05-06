---
id: M-057
title: Batched --title on aiwf add ac
status: draft
parent: E-15
acs:
    - id: AC-1
      title: Repeated --title flag accepted on aiwf add ac M-NNN
      status: open
    - id: AC-2
      title: All N ACs allocated atomically; failure reverts entire batch
      status: open
    - id: AC-3
      title: Commit trailers list every created AC composite id
      status: open
---

## Goal

Allow multiple `--title` flags on `aiwf add ac <milestone-id>` so N acceptance criteria can be created in a single atomic commit. Replaces the current shape of one commit per AC, which was the dominant cost in E-14's 42-commit planning session (33 of 42).

## Approach

Repeated-flag support is mechanical in stdlib `flag` (custom `Value`) and trivial in Cobra. The allocator already scans the milestone's `acs[]` for max+1; extending it to allocate N consecutive ids in one batch is straightforward. The full batch validates pre-projection per the existing rule ("validates the projected tree before touching disk") — if AC-N+i would violate the tree, the whole batch aborts with no commit. The single produced commit carries `aiwf-verb: add` and lists every created composite id in `aiwf-entity:`.

Single-`--title` invocation continues to work unchanged — backward-compatible by construction.

## Acceptance criteria

### AC-1 — Repeated --title flag accepted on aiwf add ac M-NNN

### AC-2 — All N ACs allocated atomically; failure reverts entire batch

### AC-3 — Commit trailers list every created AC composite id

