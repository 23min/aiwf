---
id: M-0348
title: Verify migration and establish the reduction prerequisite
status: in_progress
parent: E-0094
depends_on:
    - M-0344
    - M-0345
    - M-0346
    - M-0347
tdd: advisory
acs:
    - id: AC-1
      title: aiwf uses tracked guidance without superseded imports
      status: open
    - id: AC-2
      title: Both hosts demonstrate relevant project reads in fresh sessions
      status: open
    - id: AC-3
      title: Mixed repositories retain the correct guidance source
      status: open
    - id: AC-4
      title: Growth measurements make the reduction prerequisite reproducible
      status: open
---
## Goal

Migrate aiwf itself, verify both hosts and legacy coexistence, and leave E-0092 a measured post-delivery starting point.

## Closes

- (none)

## Context

Corpus, project installation, and selection are available. Reconcile ai-dotfiles with personal-only global instructions and repository-local engineering delivery before shared installation handover. Reuse the existing synchronization and aiwf materialization paths; do not introduce another delivery framework.

## Acceptance criteria

### AC-1 — aiwf uses tracked guidance without superseded imports

Select the packs required by its actual languages and engineering conventions, install through the new update path, preserve repository-specific rules, and remove superseded managed delivery. Verify tracked files in a clean checkout and that dotfiles synchronization does not restore legacy imports. References: `aiwf.yaml`, `.guidance/`, `AGENTS.md`, `CLAUDE.md`, and migration command records. Broad prose reduction remains E-0092's work.

### AC-2 — Both hosts demonstrate relevant project reads in fresh sessions

Observe root-started tasks touching nested files, new files, and unrelated prose, in a checkout without ai-dotfiles. Include non-coding tasks outside repositories and verify that personal-only globals require no engineering discovery or reads. Record project-override precedence, relevant reads, absence of legacy engineering reads, and limits of observable behavior. References: milestone observation record with tasks, expectations, actual observations, environment, host/model versions, and installed revision. Obtain separate approval for live service invocations.

### AC-3 — Mixed repositories retain the correct guidance source

Complete the ai-dotfiles integration and test personal-only global Claude/Codex outputs, repository-local legacy engineering delivery, and aiwf handover checks. The ai-dotfiles maintainer owns its configuration and synchronization mechanism; use its [installation documentation](https://github.com/23min/ai-dotfiles#readme) as the integration reference. Account for globally installed engineering skills as well as instruction files. Verify ai-dotfiles synchronization prepares legacy repositories on use; do not bulk-modify sibling repositories. Before any approved removal of shared global engineering delivery, verify startup setup in the environments using it, including native hook trust and enablement. Verify its diagnostics for missing helpers and failed synchronization. Test that installation and opening one repository leave unopened repositories untouched, and that sessions outside Git create no project files. Do not infer startup readiness from hook-file existence or the handover compatibility check. Exercise the released combination against old-aiwf and non-aiwf projects alongside migrated projects. Include an incompatible personal installation, failed handover, empty installed selection, disabled maintenance, and update retries; confirm legacy repository-local access or exclusive aiwf project access as applicable. Preserve personal rules; do not treat a global project/legacy router as a completed migration. References: compatibility/integration fixtures plus an observation record for actual installed setup. Do not substitute local fixture success for distribution availability.

### AC-4 — Growth measurements make the reduction prerequisite reproducible

Record before/after commits, installed guidance revision, commands, expected outputs, observed growth results, and environment. Update E-0092 to identify E-0094 and its completed migration as the prerequisite; E-0092 still owns its own frozen behavioral baseline and ceiling. References: `docs/design/growth.md`, `scripts/growth-report.py`, and E-0092. Report global/personal loading separately from upfront and task-loaded project instructions.

## Constraints

TDD is advisory for adoption and live observations; new synchronization, generation and compatibility logic uses test-first development. Do not mark observational criteria met with file-existence proxies. Keep normal approval gates for commits and publication.

## Design notes

E-0094 supplies delivery and migration. E-0092 supplies the subsequent reduction; moving guidance is not by itself proof of lower instruction load.

## Surfaces touched

ai-dotfiles personal instruction generation, repository synchronization and compatibility checks; aiwf project configuration and host files, installed guidance, growth records, and E-0092 prerequisite text.

## Out of scope

E-0092's compression, universal model-compliance guarantees, or changing personal collaboration preferences.

## Dependencies

- M-0344 — Publish the external engineering guidance corpus.
- M-0345 — Preserve legacy guidance through repository-aware routing.
- M-0346 — Deliver explicitly selected project guidance through update.
- M-0347 — Suggest applicable guidance during init and update.

All preceding deliveries, with the actual compatible distributions available for the observed environment.

## References

- E-0094 — delivery scope and compatibility requirements.

---

## Release note

## Decisions made during implementation

- ADR-0052 — project guidance ownership and the personal-bootstrap boundary.

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
