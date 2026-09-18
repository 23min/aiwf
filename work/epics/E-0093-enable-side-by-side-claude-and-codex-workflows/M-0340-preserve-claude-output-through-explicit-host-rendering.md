---
id: M-0340
title: Preserve Claude output through explicit host rendering
status: draft
parent: E-0093
tdd: required
---
## Goal

Introduce a tested rendering boundary for shared workflow sources while preserving the existing Claude artifact output and public host-selection behavior.

## Closes

- (none)

## Context

The current materializer copies embedded Markdown and routes paths through a Target value. E-0093 requires explicit placeholders and named fragments before Codex output is enabled. A planning probe has demonstrated deterministic paths, modes, and bytes for the existing materializer; permanent regression checks remain to be added.

## Acceptance criteria

The criteria below define the observable completion contract.

## Constraints

- Author shared workflow facts once; use explicit typed bindings and small named fragments only for host differences.
- Do not expose Codex selection or change the default host behavior in this milestone.
- Preserve Claude output for ordinary owned artifacts. Later milestones carry the explicit host-selection and symlink policy changes.
- Test filesystem and rendering behavior under D-0070; do not add literal prose-presence tests or rely on proposed D-0072 as accepted policy.
- No new runtime dependency or extensible plugin framework. Exercise all reachable new rendering branches with deterministic inputs.

## Design notes

- E-0093 and docs/initiatives/agent-host-artifact-adapters.md define the approved scope and selected host contracts.
- D-0073 keeps planning on main and implementation in an isolated worktree.

## Surfaces touched

- internal/skills
- internal/initrepo
- internal/skills/embedded
- internal/skills/embedded-rituals
- internal/skills/embedded-guidance

## Out of scope

- Public Codex enablement, automatic host detection, and new host configuration.
- Symlink policy changes, custom-role conversion, and live host compatibility claims.

## Dependencies

- None.

## References

- E-0093
- ADR-0014
- ADR-0016
- D-0070

## Release note



## Decisions made during implementation

- (none)

## Validation



## Deferrals

- (none)

## Reviewer notes

- (none)
