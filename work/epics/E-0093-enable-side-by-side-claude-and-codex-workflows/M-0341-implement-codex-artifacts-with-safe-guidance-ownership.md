---
id: M-0341
title: Implement Codex artifacts with safe guidance ownership
status: draft
parent: E-0093
depends_on:
    - M-0340
tdd: required
---
## Goal

Provide complete, tested Codex artifact operations and preserve user-owned guidance for both hosts before public automatic detection is enabled.

## Closes

- (none)

## Context

The preceding milestone establishes explicit rendering and Claude compatibility evidence. This milestone adds the Codex adapter and safe instruction-file handling behind the internal boundary. The existing AtomicWriteFile contract deliberately replaces a destination path; guidance callers must prevent unintended symlink replacement.

## Acceptance criteria

The criteria below define the observable completion contract.

## Constraints

- Codex output consists of skills, aiwf-owned template support files, and native guidance. Do not invent Codex role-agent directories or copy Claude settings.
- Keep canonical sources single-owned and the shared AtomicWriteFile contract unchanged.
- Guidance defaults to automatic maintenance when its host is selected, with a persistent opt-out that stops maintenance without silently deleting existing content.
- Required independent review cannot become self-review when delegation is unavailable.
- Use per-file atomic writes with idempotent recovery; do not claim an all-files transaction. Public Codex host enablement belongs to the lifecycle milestone.

## Design notes

- E-0093 and docs/initiatives/agent-host-artifact-adapters.md define the approved scope and selected host contracts.
- D-0073 keeps planning on main and implementation in an isolated worktree.

## Surfaces touched

- internal/skills
- internal/initrepo
- internal/config guidance options
- embedded host fragments and template references

## Out of scope

- Executable detection and public Codex selection.
- Codex custom-agent TOML, lifecycle hooks, cloud execution, and statusline parity.
- Changing the generic atomic writer or following instruction-file symlinks.

## Dependencies

- Preserve Claude output through explicit host rendering

## References

- E-0093
- ADR-0018
- G-0178
- G-0501
- D-0070
- D-0095

## Release note



## Decisions made during implementation

- (none)

## Validation



## Deferrals

- (none)

## Reviewer notes

- (none)
