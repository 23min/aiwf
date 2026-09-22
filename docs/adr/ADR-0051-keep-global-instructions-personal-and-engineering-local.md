---
id: ADR-0051
title: Keep global instructions personal and engineering local
status: proposed
---
> **Date:** 2026-09-22 · **Decided by:** Peter Bruinsma

## Context

Claude and Codex serve coding and non-coding tasks. Global instructions affect
all repositories and tasks using an assistant profile. Global engineering imports
and project/legacy routing therefore make a project-delivery concern part of the
user's general assistant configuration. Repository-local instructions make that
policy explicit and reviewable alongside the project.

Keeping a conditional engineering router globally avoids some legacy migration
work, but still places engineering discovery instructions in every session.
Removing global engineering delivery without preparing its consumers can withdraw
guidance from existing repositories. Neither outcome satisfies the boundary.

## Decision

Keep global Claude and Codex instructions limited to personal collaboration
preferences that apply across coding and non-coding tasks. Put engineering policy,
including code-health, language conventions, engineering skills and routing to
those materials, in the repository. Do not use global engineering imports or a
global project/legacy router as the delivery mechanism.

Use aiwf's tracked project guidance for repositories adopting that delivery.
Keep legacy repository delivery in ai-dotfiles' existing synchronization tooling;
legacy consumers must not need an aiwf upgrade just to retain their guidance.
Engineering content remains canonically owned outside aiwf.

## Consequences

- E-0094 must reconcile personal instruction generation, legacy repository delivery
  and handover checks with this boundary. Personal preferences remain preserved.
- M-0348 includes the implementation and isolated verification needed for that
  reconciliation, as well as migration and growth observations. Logic changes
  require regression tests; live assistant observations remain separate evidence.
- A machine-wide removal of global engineering guidance requires preparing the
  repositories that depend on it and explicit approval of the shared change.
  Untouched legacy consumers cannot be assumed unaffected by removing their global
  source. A personal-bootstrap update must not silently perform that handover.
- Existing installations, including the container's installed global router, are
  migration inputs rather than evidence of compliance with the target boundary.
- Engineering skills need the same treatment as instruction files: globally
  discoverable engineering skill content is not made local by moving one import.

## Validation

Test generated global instructions for preservation of personal preferences and
absence of engineering delivery. Exercise old-aiwf and non-aiwf repositories,
aiwf-owned repositories, empty selections, disabled maintenance and failed
handovers. Confirm their repository-local engineering inputs and ownership rules.

Observe fresh Claude and Codex sessions on coding and non-coding tasks, including
outside repositories. Record supplied instructions and observed reads; distinguish
installed files from actual assistant use. Confirm global instructions require no
engineering discovery or reads, including for non-coding tasks outside repositories.
Report project-local reads for prose tasks inside repositories separately. Live
observations and shared installation changes retain their separate approval gates.

## References

- E-0094 — external engineering guidance delivery and migration.
- M-0348 — migration, coexistence and growth verification.
- D-0089 — external content ownership and tracked project policy.
