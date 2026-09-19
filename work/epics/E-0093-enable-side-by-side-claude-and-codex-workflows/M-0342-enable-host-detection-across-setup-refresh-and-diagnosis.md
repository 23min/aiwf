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

`aiwf init`, `update`, upgrade refresh, `doctor`, and `worktree add` now use the same Claude/Codex host selection: detect executables on PATH, override with `hosts`, or select none with `hosts: []`. Setup materializes the selected hosts and preserves unselected installations; doctor reports missing, stale, and unsafe artifacts with remediation. Dry-run reports the planned setup without changing the checkout, and lifecycle help documents configuration, retained files, and capability limits. Codex uses aiwf-created worktrees without requiring its experimental managed-worktree feature; Claude-specific entry instructions are preserved.

## Decisions made during implementation

- (none)

## Validation

Validation environment: Linux amd64 devcontainer, Go 1.25.11, implementation branch `milestone/M-0342-host-lifecycle`, source through `89c9445cb`. Review the full milestone against `a6e54b8e3`; that range includes the lifecycle changes and the explicit worktree-guidance clarification.

Wrap gates on 2026-09-19:

- `make check-fast`: exit 0; vet with normal, stress, and testpins tags passed; `golangci-lint` reported `0 issues.`; the full test suite passed.
- `make diag-aiwf`: exit 0; built the implementation checkout's `bin/aiwf-diag` with `CGO_ENABLED=0`.
- `go test -race ./internal/config ./internal/initrepo ./internal/skills ./internal/cli/doctor ./internal/cli/initcmd ./internal/cli/update ./internal/cli/worktree ./internal/testsupport`: exit 0; all eight packages passed.
- `bin/aiwf-diag show M-0342`: all six ACs `met`, all six TDD phases `done`, no milestone findings.
- `bin/aiwf-diag check --since a6e54b8e3`: exit 0; `4 findings (0 errors, 4 warnings)`. The warnings are three terminal entities awaiting archive and the archive-sweep advisory. An unscoped check also reported the missing-upstream advisory; the explicit base enables provenance and body-section checks.

| Criterion | Re-runnable behavioral evidence |
| --- | --- |
| AC-1 | `go test ./internal/config ./internal/initrepo -run 'TestResolveHosts_|TestHosts_|TestArtifactEntryPoints_' -count=1`: controlled executable PATHs, explicit/null/empty selection, validation before writes, and configuration round trips. |
| AC-2 | `go test ./internal/cli/integration -run 'TestHostLifecycle_InitUpdateAndWorktreeUseResolvedHosts|TestHostLifecycle_AddingCodexPreservesClaudeArtifacts|TestHostLifecycle_WorktreeArtifactFailuresRollBackCreation|TestHostLifecycle_SelfCheckWorksWithoutAssistantCommands' -count=1`: real CLI and Git worktrees, all host sets, rollback, and assistant-free self-check. |
| AC-3 | `go test ./internal/cli/integration ./internal/initrepo -run 'TestHostLifecycle_DeselectedAndDisappearedHostsAreRetained|TestHostLifecycle_UnselectedClaudeSecondaryPathsStayUntouched|TestHostLifecycle_ExplicitClaudeRequestsPreserveUnselectedState|TestRetainedHostSteps_' -count=1`: retained file bytes, modes, timestamps, link handling, and untouched unselected secondary paths. |
| AC-4 | `go test ./internal/skills ./internal/initrepo ./internal/cli/doctor ./internal/cli/integration -run 'TestInspect|TestDoctor_|TestWriteHealth_|TestHostLifecycle_Doctor' -count=1`: missing/drifted/blocked artifacts, family severity, guidance conflicts, read-only diagnosis, and health-write selection. |
| AC-5 | `go test ./internal/initrepo ./internal/cli/integration -run 'TestHostDryRun_|TestHostLifecycle_DryRun|TestHostLifecycle_Upgrade' -count=1`: filesystem snapshots and preview/apply ledger parity; actual process re-execution through a fake installer boundary, checkout resolution, and failure propagation. No download or global binary replacement. |
| AC-6 | `go test ./internal/config ./internal/cli/integration ./internal/skills -run 'TestHostExamples_|TestHostDiscovery_|TestHostFragments_|TestClaudeArtifacts_MatchBaseline|TestMaterializeTo_Codex' -count=1`: executable help examples, configuration semantics, native fragment selection, and six frozen Claude inventories. |

The Claude inventory exceptions are documented in `internal/cli/integration/testdata/claude-baseline/README.md`. The shared worktree-skill correction changes its recorded hash only; Claude-specific entry fragments remain unchanged. Disposable two-host materialization confirmed that the new Codex launch and experimental-feature instructions do not appear in generated Claude markdown.

Per-AC branch audits and bounded manual mutation probes covered the changed logic and its observable contracts. AC-1 through AC-6 caught 10, 10, 11, 17, 7, and 7 compiled probes respectively: 62 caught, no survivors in that bounded set. Each probe failed a behavioral assertion, and source bytes were restored before subsequent gates. `make mutate-diff` reported Gremlins unavailable; this is not a complete mechanical mutation score or proof of branch completeness. The final shared-skill prose correction adds no runtime branches.

Live assistant discovery, Claude-to-Codex-to-Claude handoff, concurrent assistant sessions, and devcontainer rebuild/persistence are not established by these filesystem and subprocess tests. M-0343 owns those observations, including the explicit `--disable worktrees` condition. Full `make ci` remains the epic-to-main/push gate; it is not claimed as executed for this milestone wrap.

## Deferrals

- G-0504 — Selected-host drift reporting advances this gap. Its additional requirement for planning rituals to refresh templates before reading them remains outside this milestone, so this milestone does not claim to close the whole gap.

## Reviewer notes

- (none)
