---
id: D-0089
title: aiwf ships no language-specific guidance
status: proposed
relates_to:
    - E-0092
---
> **Date:** 2026-09-13 · **Decided by:** Peter Bruinsma (human), while planning E-0092

## Question

Where do language conventions (a formatter and lint set, test idioms, error wrapping) live for a downstream project that uses aiwf: in aiwf's shipped surfaces, or elsewhere? Non-obvious because the shipped guidance fragment is the broadest-reach always-on surface a consumer has, and this repo's own `CLAUDE.md` restates every topic of the operator's Go module, which reads as if aiwf owned them.

## Decision

aiwf owns no language-specific conventions in its embedded guidance. Language and engineering packs are authored in an external repository and become project policy through explicit selection. aiwf may detect languages using externally supplied patterns and deliver the selected packs as tracked project files. Consumers need neither aiwf nor ai-dotfiles merely to read those files.

The shared aiwf operating fragment describes operating aiwf. Repository development guidance describes what is specific to developing this repository. Personal collaboration and machine/session preferences remain in ai-dotfiles.

## Reasoning

- Language content and workflow machinery change for different reasons. Adding a language must require no aiwf code change or release.
- Project selection makes opinionated conventions explicit; detecting Python does not select a package manager on the project's behalf.
- Tracked project guidance reaches contributors, containers and worktrees without a home-directory import or editor synchronization.
- A shared source can have generated host outputs without creating another authored copy. Readable installed files and observed model use remain separate claims.

## Consequences

- E-0092 removes generic language restatements after the external-delivery follow-up to E-0093 has migrated this repository.
- The delivery follow-up owns the small catalogue, simple detection patterns, explicit selection and upstream checks during `aiwf update`; the detailed design belongs there.
- The language corpus is not embedded in aiwf. Its updater may materialize project-selected copies without owning their conventions.
- Personal ai-dotfiles delivery must not be required for project language guidance. Project-specific exceptions remain project-authored material, separate from generated files.
