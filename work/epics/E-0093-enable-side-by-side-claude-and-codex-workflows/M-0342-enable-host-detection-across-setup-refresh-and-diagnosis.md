---
id: M-0342
title: Enable host detection across setup refresh and diagnosis
status: draft
parent: E-0093
depends_on:
    - M-0341
tdd: required
---
## Goal

Expose complete automatic Claude and Codex setup consistently through configuration, init, update, upgrade refresh, doctor, and aiwf-created worktrees.

## Closes

- (none)

## Context

Both artifact adapters and safe guidance operations are available internally. Public orchestration still contains unconditional Claude paths in skill refresh, guidance, hooks, and statusline handling. This milestone exposes host selection only after those paths can honor the same effective host set.

## Acceptance criteria

The criteria below define the observable completion contract.

## Constraints

- Detect executables on PATH without launching the hosts, checking authentication, or contacting services.
- Keep detected machine state out of shared project configuration. Preserve explicit empty lists through configuration rewrites.
- A host disappearing or being deselected is not authorization to delete its files.
- Core Git hooks are host-independent; host lifecycle hooks and settings are capability-specific and preserve existing consent rules.
- Keep existing default worktree placement and explicit path overrides. Test with isolated process/filesystem boundaries, not the developer's installed assistants.
- Use the established CLI error and finding conventions, and report selected-host guidance conflicts without describing skipped updates as healthy.

## Design notes

- E-0093 and docs/initiatives/agent-host-artifact-adapters.md define the approved scope and selected host contracts.
- D-0073 keeps planning on main and implementation in an isolated worktree.

## Surfaces touched

- internal/config and schema/example generation
- internal/initrepo
- internal/cli/initcmd, update, upgrade, worktree
- internal/cli/doctor and cliutil
- operator documentation and generated verb guidance

## Out of scope

- Automatic host removal and general downgrade protection.
- New workflow kernel semantics or host-specific runtime services.
- Codex-managed worktrees and a migration of worktree.dir.

## Dependencies

- Implement Codex artifacts with safe guidance ownership

## References

- E-0093
- G-0504
- G-0600
- ADR-0018
- D-0070

## Release note



## Decisions made during implementation

- (none)

## Validation



## Deferrals

- (none)

## Reviewer notes

- (none)
