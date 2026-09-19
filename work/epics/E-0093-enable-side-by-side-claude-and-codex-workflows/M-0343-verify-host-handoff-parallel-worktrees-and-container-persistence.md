---
id: M-0343
title: Verify host handoff parallel worktrees and container persistence
status: draft
parent: E-0093
depends_on:
    - M-0342
tdd: advisory
acs:
    - id: AC-1
      title: Fresh sessions discover the supported skills and instruction sources
      status: open
    - id: AC-2
      title: Claude Codex and Claude can hand off one unfinished workflow
      status: open
    - id: AC-3
      title: Concurrent host sessions keep edits and Git state in their assigned worktrees
      status: open
    - id: AC-4
      title: An independent Codex review executes through the supported ritual path
      status: open
    - id: AC-5
      title: A rebuilt devcontainer provides Codex and retains the selected state
      status: open
    - id: AC-6
      title: Implementation passes repository gates and documents its support boundary
      status: open
---
## Goal

Establish recorded evidence that the supported Claude and Codex workflows are usable in fresh local sessions and survive the intended container lifecycle.

## Closes

- (none)

## Context

Public host detection and artifact lifecycle behavior are complete. This milestone tests actual host discovery and operator workflows, which filesystem tests alone cannot establish. Existing devcontainer edits are uncommitted and must be reviewed on their own merits before they are incorporated. The implementation worktree must receive approved planning commits before feature work starts.

## Acceptance criteria

The criteria below define the observable completion contract.

### AC-1 — Fresh sessions discover the supported skills and instruction sources

Record fresh Claude and Codex runs in a disposable consumer checkout after materialization. Verify each host's exposed skill inventory and instruction-source diagnostics where available, and execute representative planning and review steps using the generated artifacts. Repeat relevant root/nested instruction precedence and size-limit scenarios for Codex. Separately verify that Codex working on aiwf receives repository development guidance through a maintained source/reference arrangement. State what was mechanically observable and what was only demonstrated by behavior.

### AC-2 — Claude Codex and Claude can hand off one unfinished workflow

Run an approved handoff from Claude to Codex and back on the same task/worktree using current aiwf records, branch/diff state, and a concise handoff note. The receiving session identifies completed versus unfinished work and continues without changing host selection or reconstructing the workflow by translating another host's artifacts. Record how uncommitted work and outstanding approvals were represented. A host account limit that prevents a leg leaves this observation incomplete.

### AC-3 — Concurrent host sessions keep edits and Git state in their assigned worktrees

Use separate terminals and distinct branches/worktrees for a bounded pair of non-overlapping tasks. Record both repository roots and branches before work, then verify each change and staged/unstaged state in the intended checkout. Each worktree must have its complete selected host artifacts. Create these checkouts with `aiwf worktree add`. Start the fresh Codex session with `codex --disable worktrees -C "<path>"`, record the CLI version and explicit disabled-feature invocation, and verify observed skill discovery in that checkout. Codex-managed worktree creation is not part of this observation. Keep integration serial and separately approved; do not run concurrent refreshes using divergent binaries against shared hook state. This demonstrates human-operated parallel sessions, not an orchestration engine.

### AC-4 — An independent Codex review executes through the supported ritual path

Exercise a review-required ritual with a fresh reviewer context that did not author the change. Record the dispatch method, reviewed diff, findings, and how the parent handled them. Confirm that the instructions resolve to capabilities available in the supported Codex setup. If delegation is unavailable, demonstrate the actionable limitation and keep the successful independent-review observation outstanding; self-review cannot satisfy this criterion.

### AC-5 — A rebuilt devcontainer provides Codex and retains the selected state

After separate rebuild approval, exercise the reviewed install and mount setup in a fresh container build. Verify that the global npm Codex binary is available even if the editor supplies another binary, and that repeating initialization does not reinstall an existing npm binary. Check host-backed Codex configuration/session state and login status before and after without exposing credentials. Confirm Claude remains usable and that the install/mount documentation matches the exercised behavior. Record any steps not run.

### AC-6 — Implementation passes repository gates and documents its support boundary

Run the required build, formatting/lint, race-test, coverage, and selfcheck gates against the final implementation checkout and record results with failures preserved. Review setup, host override, guidance opt-out, symlink diagnostic, handoff, and parallel-worktree instructions against the demonstrated workflows. State the deferred capabilities and the distinction between materialization health and observed discovery. This criterion records actual gate execution; it must not be replaced with a proxy assertion that commands appear in documentation.

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
