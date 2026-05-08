---
id: M-057
title: Batched --title on aiwf add ac
status: done
parent: E-15
acs:
    - id: AC-1
      title: Repeated --title flag accepted on aiwf add ac M-NNN
      status: met
    - id: AC-2
      title: All N ACs allocated atomically; failure reverts entire batch
      status: met
    - id: AC-3
      title: Commit trailers list every created AC composite id
      status: met
    - id: AC-4
      title: Single commit produced regardless of N (one or many ACs)
      status: met
    - id: AC-5
      title: Single-title invocation continues to work unchanged
      status: met
---

## Goal

Allow multiple `--title` flags on `aiwf add ac <milestone-id>` so N acceptance criteria can be created in a single atomic commit. Replaces the current shape of one commit per AC, which was the dominant cost in E-14's 42-commit planning session (33 of 42).

## Approach

Repeated-flag support is mechanical in stdlib `flag` (custom `Value`) and trivial in Cobra. The allocator already scans the milestone's `acs[]` for max+1; extending it to allocate N consecutive ids in one batch is straightforward. The full batch validates pre-projection per the existing rule ("validates the projected tree before touching disk") — if AC-N+i would violate the tree, the whole batch aborts with no commit. The single produced commit carries `aiwf-verb: add` and lists every created composite id in `aiwf-entity:`.

Single-`--title` invocation continues to work unchanged — backward-compatible by construction.

## Acceptance criteria

### AC-1 — Repeated --title flag accepted on aiwf add ac M-NNN

_Grandfathered: this AC was met before M-066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-057/AC-1` for the actual implementation history._

### AC-2 — All N ACs allocated atomically; failure reverts entire batch

_Grandfathered: this AC was met before M-066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-057/AC-2` for the actual implementation history._

### AC-3 — Commit trailers list every created AC composite id

_Grandfathered: this AC was met before M-066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-057/AC-3` for the actual implementation history._

### AC-4 — Single commit produced regardless of N (one or many ACs)

_Grandfathered: this AC was met before M-066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-057/AC-4` for the actual implementation history._

### AC-5 — Single-title invocation continues to work unchanged

_Grandfathered: this AC was met before M-066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-057/AC-5` for the actual implementation history._
