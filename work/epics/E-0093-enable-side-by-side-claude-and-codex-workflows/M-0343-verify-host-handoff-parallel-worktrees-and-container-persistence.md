---
id: M-0343
title: Verify host handoff parallel worktrees and container persistence
status: draft
parent: E-0093
depends_on:
    - M-0342
tdd: advisory
---
## Goal

Establish recorded evidence that the supported Claude and Codex workflows are usable in fresh local sessions and survive the intended container lifecycle.

## Closes

- (none)

## Context

Public host detection and artifact lifecycle behavior are complete. This milestone tests actual host discovery and operator workflows, which filesystem tests alone cannot establish. Existing devcontainer edits are uncommitted and must be reviewed on their own merits before they are incorporated. The implementation worktree must receive approved planning commits before feature work starts.

## Acceptance criteria

The criteria below define the observable completion contract.

## Constraints

- Observations record command or prompt, expected outcome, actual outcome, host version, checkout, and relevant permission/configuration conditions. An unavailable run is outstanding, not a pass.
- Do not use a model's claim that it read a file as a mechanical delivery guarantee; use exposed discovery evidence and observed workflow behavior, with their limits stated.
- Live external host invocations, rebuilds, pushes, merges, and other outward or irreversible steps retain their individual approval gates.
- Separate worktrees isolate files and indexes but share repository hooks and may share a globally installed aiwf binary; use consistent source-built tooling.
- Advisory TDD applies to this observation/documentation milestone. Any new logic discovered here still follows the repository's normal TDD and coverage rules.
- No independent copy of the repo's engineering guidance, no automatic chat transfer, and no claim of full hook/custom-agent/cloud parity.

## Design notes

- E-0093 and docs/initiatives/agent-host-artifact-adapters.md define the approved scope and selected host contracts.
- D-0073 keeps planning on main and implementation in an isolated worktree.

## Surfaces touched

- repository instruction entry points and referenced guidance
- .devcontainer/init.sh, initialize.sh, devcontainer.json, README.md
- consumer setup and handoff documentation
- milestone validation records

## Out of scope

- Automated parallel TDD orchestration from E-0019.
- General guidance reduction from E-0092 or guaranteed context delivery from G-0523.
- Provider transcript migration, Codex-managed worktrees, and external deployment.

## Dependencies

- Enable host detection across setup refresh and diagnosis

## References

- E-0093
- D-0073
- D-0095
- G-0523
- G-0600
- E-0092
- E-0019

## Release note



## Decisions made during implementation

- (none)

## Validation



## Deferrals

- (none)

## Reviewer notes

- (none)
