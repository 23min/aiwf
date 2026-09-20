---
id: M-0346
title: Deliver explicitly selected project guidance through update
status: draft
parent: E-0094
depends_on:
    - M-0344
    - M-0345
tdd: required
---
## Goal

Ship a complete init/update workflow for packs explicitly listed in `aiwf.yaml`, including tracked outputs, host routing, safe refresh/removal, and legacy handover.

## Closes

- (none)

## Context

The external corpus and compatible ai-dotfiles delivery exist. This milestone serves explicit configuration directly; automatic detection and interactive selection follow separately. It must have a real CLI caller when it lands.

## Acceptance criteria



## Constraints

Use required TDD and repository build/lint/race/selfcheck gates. Resolve write ordering and recovery before shipping. No persistent cache, lockfile, package solver, extra upgrade command, or automatic personal-file rewrites. Do not add an index that disables legacy guidance merely because defaults enable maintenance.

## Design notes

E-0094 fixes user-visible behavior. The configuration's representation of unadopted versus explicitly empty selection is an implementation choice to settle against existing YAML handling. This distinction adds no new user-facing mode.

## Surfaces touched

Configuration/schema; shared init refresh; update and upgrade integration; ownership/rendering; doctor and user documentation.

## Out of scope

Automatic applicability detection, selection prompts, live migration of aiwf itself, and language content embedded in the binary.

## Dependencies

- M-0344 — Publish the external engineering guidance corpus.
- M-0345 — Preserve legacy guidance through repository-aware routing.

Corpus and compatible ai-dotfiles routing, including the ownership/preflight boundary. Host behavior is inherited from E-0093.

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
