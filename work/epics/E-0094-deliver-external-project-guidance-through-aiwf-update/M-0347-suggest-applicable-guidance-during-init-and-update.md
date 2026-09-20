---
id: M-0347
title: Suggest applicable guidance during init and update
status: draft
parent: E-0094
depends_on:
    - M-0346
tdd: required
---
## Goal

Add explainable detection and explicit interactive selection on top of the complete delivery workflow.

## Closes

- (none)

## Context

Users can already maintain selected packs by editing configuration. This milestone improves selection without changing who owns installed policy or weakening refresh safety.

## Acceptance criteria



## Constraints

Detection suggests policy; it never chooses tools or dependencies. Reuse the delivery path so interactive and explicit configuration receive identical validation and recovery. No new guidance-selection command.

## Design notes

E-0094 defines select/not-now/ignore semantics. Cover each behavioral rule without enumerating redundant input spellings.

## Surfaces touched

Generic detection, init/update interaction, configuration writes, and user documentation.

## Out of scope

Framework dependency parsing, session-start scanning, new provider mechanisms, or automatic policy adoption.

## Dependencies

- M-0346 — Deliver explicitly selected project guidance through update.

Complete explicit-selection delivery.

## References

- E-0094 — delivery scope and compatibility requirements.

---

## Release note

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
