---
id: M-0342
title: Enable host detection across setup refresh and diagnosis
status: in_progress
parent: E-0093
depends_on:
    - M-0341
tdd: required
acs:
    - id: AC-1
      title: Host resolution handles detection overrides and empty configuration consistently
      status: met
      tdd_phase: done
    - id: AC-2
      title: Init update and worktree refresh materialize the complete resolved host set
      status: met
      tdd_phase: done
    - id: AC-3
      title: Unselected host settings and artifacts remain untouched
      status: met
      tdd_phase: done
    - id: AC-4
      title: Doctor detects selected-host absence drift and guidance conflicts
      status: met
      tdd_phase: done
    - id: AC-5
      title: Dry-run and upgrade refresh honor host selection without hidden writes
      status: met
      tdd_phase: done
    - id: AC-6
      title: Host configuration and supported capabilities are discoverable
      status: met
      tdd_phase: done
---
## Goal

Expose complete automatic Claude and Codex setup consistently through configuration, init, update, upgrade refresh, doctor, and aiwf-created worktrees.

## Closes

- (none)

## Context

Both artifact adapters and safe guidance operations are available internally. Public orchestration still contains unconditional Claude paths in skill refresh, guidance, hooks, and statusline handling. This milestone exposes host selection only after those paths can honor the same effective host set.

## Acceptance criteria

The criteria below define the observable completion contract.

### AC-1 — Host resolution handles detection overrides and empty configuration consistently

Cover neither executable, Claude only, Codex only, and both on controlled PATHs. Non-executable files are not detected as installed commands. An absent hosts field enables detection; an explicit supported list overrides it even when tools are absent; an explicit empty list selects no host. Reject unknown values and deduplicate repeated valid hosts deterministically. Round-trip relevant configuration mutations to prove that empty does not become absent through serialization. Missing, malformed, and invalid configuration must not cause partial host materialization.

### AC-2 — Init update and worktree refresh materialize the complete resolved host set

Run each local lifecycle entry point across the supported host combinations in temporary repositories and real temporary Git worktrees. A single invocation installs the full selected surface, preserves existing selected-host consent behavior, and reports whether selection came from detection or configuration. Add Codex to a previously Claude-only environment and prove the next refresh adds its complete surface while preserving Claude output. A no-host selection performs core aiwf setup and explicitly reports that no host artifacts were selected.

### AC-3 — Unselected host settings and artifacts remain untouched

Under Codex-only and no-host resolution, exercise secondary hook-sync, statusline, settings, guidance, and installation-health paths and verify that they neither create nor rewrite Claude-only files. Existing Claude hook-consent flags and stored decisions must not silently enable Claude against an explicit host override; report an incompatible request. Core Git hooks retain their independent behavior. When a previously selected host disappears or is deselected, preserve its artifacts and distinguish retained files from current refreshed output.

### AC-4 — Doctor detects selected-host absence drift and guidance conflicts

Materialize each host, remove an artifact, and separately mutate bytes in skills, templates, and guidance. Verify that doctor names the affected host and artifact family with remediation using the same rendered expectations as materialization. User bytes outside managed guidance blocks and unrelated host files must not be misclassified as generated drift. Report unavailable tools, retained unselected installations, wiring opt-outs, and skipped symlink/conflict updates distinctly. Existing family severity conventions remain explicit, and the report must not imply that disk health guarantees delivery into a model's context.

### AC-5 — Dry-run and upgrade refresh honor host selection without hidden writes

Snapshot the consumer filesystem and configuration before supported dry-run paths and verify they remain unchanged while the ledger describes the actual resolved hosts and intended guidance writes or skips. Exercise upgrade's re-executed update boundary with a fake installer/process boundary so unit tests need neither a download nor a real binary replacement. The new update path receives the correct checkout context and resolves hosts consistently. Prove failure propagation when re-execution or configuration resolution fails.

### AC-6 — Host configuration and supported capabilities are discoverable

Verify the hosts field, supported values, absent-versus-empty semantics, guidance opt-out, and effective-selection reporting through schema, generated examples, and CLI help/structured output where those surfaces exist. Cover observable parsing and example round trips, not prose phrase presence. Review the operator documentation and generated skill references for automatic setup, explicit overrides, retained installations, unsupported capabilities, and the existing configurable worktree path.

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

- G-0504 — Selected-host drift reporting advances this gap. Its additional requirement for planning rituals to refresh templates before reading them remains outside this milestone, so this milestone does not claim to close the whole gap.

## Reviewer notes

- (none)
