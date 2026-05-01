---
id: M-003
title: Build the violet widget under TDD
status: in_progress
parent: E-01
tdd: required
acs:
  - id: AC-1
    title: Violet widgets render at 60 fps
    status: open
    tdd_phase: red
  - id: AC-2
    title: Pack receives canonical OpResult
    status: met
    tdd_phase: done
  - id: AC-3
    title: Reviewer notes are exported
    status: deferred
    tdd_phase: red
---

## Goal

Demonstrate the I2 surface on a fixture with the full vocabulary
exercised: a `tdd: required` milestone, three ACs across three statuses
(open, met, deferred), each with a phase value, and matching body
headings using the em-dash separator.

## Acceptance criteria

### AC-1 — Violet widgets render at 60 fps

The violet widget renders at sustained 60 fps under the standard
fixture load.

### AC-2 — Pack receives canonical OpResult

Pack target receives an OpResult shaped per ADR-0001.

### AC-3 — Reviewer notes are exported

Deferred to the next milestone; see body of M-004 when it lands.

## Work Log

### AC-2 — met
The pack-receiving path landed; tests pass. Phase advanced to done.

## Decisions made during implementation

- (none)

## Validation

- `go test ./tools/...` — passes.

## Deferrals

- AC-3 deferred; will pick up in M-004.

## Reviewer notes

- (none)
