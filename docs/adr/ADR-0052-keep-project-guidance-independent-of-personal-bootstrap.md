---
id: ADR-0052
title: Keep project guidance independent of personal bootstrap
status: accepted
supersedes:
    - ADR-0051
---
> **Date:** 2026-09-22 · **Decided by:** Peter Bruinsma

## Context

Project engineering policy must be available to contributors without depending on
their personal assistant setup. Personal collaboration preferences also apply to
non-coding tasks and have a different owner and lifetime from project policy.

Adopting aiwf project guidance can overlap with an existing personal-bootstrap
source. aiwf needs a clear handover contract so two sources do not remain active.
It does not need to own how another tool configures assistants or maintains
repositories that have not adopted aiwf delivery.

## Decision

Deliver aiwf-selected engineering guidance as tracked repository-local policy.
Keep canonical engineering content outside aiwf. Keep global Claude and Codex
instructions personal-only; do not rely on global engineering imports, skills or
an instruction-based engineering router for project delivery.

Treat an aiwf-managed `.guidance/index.md` as ownership of project engineering
guidance. Legacy delivery must respect that ownership, including an empty
installed selection and disabled maintenance. Check compatibility before handover
and remove only recognized legacy routes after validating their replacements.
An incompatible installation leaves guidance handover incomplete with an
actionable diagnostic; unrelated aiwf update work can proceed.

Leave personal-bootstrap configuration and legacy-repository synchronization to
the personal-bootstrap maintainer. aiwf does not change personal settings or
prepare sibling repositories. The bootstrap's own documentation owns its delivery
mechanism and migration procedure.

## Consequences

- Contributors can read installed project guidance without aiwf, ai-dotfiles or
  VS Code synchronization. Project maintainers select and update that policy.
- Legacy consumers need no aiwf upgrade merely to retain guidance. Maintaining
  their delivery is the personal-bootstrap maintainer's responsibility.
- Compatibility checks establish the handover conditions they inspect; they do
  not certify delivery in other environments or prove an assistant read guidance.
- E-0094 and M-0348 verify aiwf's side of the integration with ai-dotfiles:
  that its synchronization respects aiwf ownership, and that an incomplete
  handover leaves legacy delivery intact. Delivery to repositories aiwf does not
  own, and the bootstrap's hook configuration, startup behavior and trust
  controls, are verified and documented in that repository.

## Validation

Exercise handover with compatible and incompatible personal installations,
aiwf-owned repositories, empty selections, disabled maintenance and retries.
Check exclusive project ownership and preservation of personal settings and
unrelated repositories.

Record fresh-host observations separately from installation tests. Verify that
project guidance is usable without ai-dotfiles, and test coexistence against the
bootstrap's documented behavior rather than prescribing its implementation here.

## References

- ADR-0051 — the personal-only boundary retained by this replacement.
- E-0094 — external project guidance delivery.
- M-0348 — migration and coexistence verification.
- D-0089 — proposed external content ownership and tracked project policy.
