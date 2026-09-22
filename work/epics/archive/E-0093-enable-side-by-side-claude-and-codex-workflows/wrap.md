# Epic wrap — E-0093

**Date:** 2026-09-19
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0093-enable-side-by-side-claude-and-codex-workflows

## Milestones delivered

- M-0340 — Preserve Claude output through explicit host rendering (merged ae04041e3).
- M-0341 — Implement Codex artifacts with safe guidance ownership (merged e3a0b0bd1).
- M-0342 — Enable host detection across setup refresh and diagnosis (merged 282257db3).
- M-0343 — Verify host handoff parallel worktrees and container persistence (merged 26ea9b383; AC-5 deferred to G-0699).

## Changelog entry

### Added — E-0093: side-by-side Claude Code and Codex workflows

- `aiwf init`, `update`, upgrade refresh, `doctor` and `worktree add` share host
  selection: detect installed commands on PATH, override with `hosts`, or select
  no assistant artifacts with `hosts: []`. Unselected installations are retained.
- Codex receives native `AGENTS.md` guidance, `.agents/skills/` and shared
  templates. Claude retains its artifact layout and consent rules. Setup
  preserves linked or aliased instruction files, reports unsafe guidance and
  ownership conflicts, and diagnoses drift across selected artifact families.
- Handoff and parallel-worktree guidance supports both assistants; live checks
  demonstrate fresh-session discovery and independent Codex review. Codex-managed
  worktrees, hosted review and automatic transcript transfer remain outside scope.
- Devcontainer setup installs Codex independently of editor extensions and
  prepares host-backed state. Actual rebuild persistence remains unverified
  in G-0699. Skill frontmatter corrections preserve the original descriptions.

## Summary

aiwf supports Claude Code and Codex from canonical workflow sources, with shared
host selection and host-specific delivery. Separate-worktree sessions and serial
handoff have recorded live evidence. Container rebuild verification is explicitly
deferred; isolated installation and mount tests do not replace that observation.

## ADRs ratified

- None in this epic. Existing ADR-0014 and ADR-0016 govern embedded distribution;
  ADR-0015 governs Claude settings consent.

## Decisions captured

- D-0070 — existing accepted boundary: test behavior and relationships, not shipped prose wording.
- D-0073 — existing accepted planning/main and implementation/worktree separation.
- D-0095 — accepted worktree entry convention; host-specific entry guidance preserves Claude behavior.
- AC-5 rebuild deferral is recorded in M-0343 and G-0699 with explicit user approval.

## Follow-ups carried forward

- G-0504 — planning rituals still need to refresh present templates before reading them; drift detection is implemented.
- G-0698 — align planning rituals with the accepted planning-on-main workflow.
- G-0699 — observe Codex availability and state preservation across a separately approved rebuild.
- G-0523 and G-0600 — instruction delivery and stale-binary downgrade concerns remain outside this epic's guarantees.

## Doc findings

Scoped checks of the nine changed narrative documents found no broken local
links or anchors, heading drift, documentation markers, orphaned changed docs,
stale added CLI invocations or removed-feature references. Render bindings and
new code references resolve to the implementation. Embedded-source references
are covered by the repository policy and materialization tests.

## Validation

After merging main `75521dfdc` into the epic at `cf4ec51`,
`AIWF_COVERAGE_BASE=75521dfdc098dc8e6443e7c63bbbde4e7aa132df make ci`
exited 0: vet, lint (0 issues), race tests, 91.5% statement coverage,
profile-driven gates, static build and 29 CLI self-check steps passed.
Local log: `/tmp/aiwf-E-0093-integrated-ci.log`. Hosted CI and rebuild were not run.

## Handoff

All child milestones are done. G-0501 and G-0178 are addressed. Local Claude
and Codex workflows are ready for use after installing the feature build and
refreshing selected-host artifacts. Follow-ups above remain open; container
rebuild persistence is not established. Hosted review, transcript transfer and
Codex-managed worktree creation are outside this support boundary.
