---
id: M-0160
title: Operational pain — reallocate stress, trunk-collision regress, apply rollback
status: draft
parent: E-0030
tdd: required
acs:
    - id: AC-1
      title: Reallocate combinatorial real-git E2E coverage
      status: open
      tdd_phase: red
---
## Goal

Close evidence-backed operational-pain scenarios surfaced by the M-0159 history-mining audit. Three concrete classes with real in-repo incidents:

1. **Reallocate stress** — 26 reallocate commits in this repo's history confirm cross-branch ID collisions are recurring (CLAUDE.md §"Id-collision resolution at merge time" documents the operator-discipline gap). Verify the `aiwf reallocate` path holds under combinatorial verb-sequence scenarios.

2. **G-0167 trunk-collision regression** — retitle+body growth pushed git rename detection below 50% similarity (`8b56ba1c` "fix(gitops): trailer-driven rename detection"). Pin the regression class so it cannot recur.

3. **G-0170 apply-rollback data-preservation** — `ed0b5014` "fix(verb): apply rollback preserves uncommitted dirty files at touched paths" closed the original incident. Pin the contract via real-git E2E so a future refactor cannot regress the bless-mode data-preservation guarantee.

## Context

Per the M-0159 evidence-priority split, this milestone (Tier 2) addresses operational pain that has already bitten this repo. Distinguished from M-0161 (Tier 3 imagination-driven hardening) by in-history evidence. Distinguished from M-0159 (Tier 1) by being post-framework: M-0159 lands the combinatorial E2E framework (G-0211); M-0160 reuses it for these three scenarios.

## Dependencies

- **M-0159** (Tier 1) — must complete first; M-0160 reuses M-0159's E2E framework.
- **Existing fixes**: `8b56ba1c` (G-0167 trailer-driven rename), `ed0b5014` (G-0170 apply rollback). These are committed; M-0160 adds regression-pin tests, not new fixes.

## Out of scope

- Tier 3 imagination-driven scenarios (G-0200..G-0207, G-0209) — covered by M-0161.
- Data-loss scenarios crossing epic boundaries (G-0212) — future-epic.

## Acceptance criteria

<!--
AC seed set (to be allocated via `aiwf add ac` at start-milestone time):

1. Reallocate-stress combinatorial test: two parallel-branch operators reallocate the same id; merge; verify `aiwf reallocate` resolves cleanly across the matrix of {pre-push, post-merge, with cross-reference, without cross-reference}.

2. G-0167 trunk-collision regression test: retitle a long-bodied entity from a short title to a long title; verify rename detection finds the file via trailer-driven mechanism, not similarity.

3. G-0170 apply-rollback test: dirty an uncommitted file at a touched path; trigger a verb commit failure; verify the dirty content is preserved on rollback.

These three are the seed set; aiwfx-start-milestone refines and allocates them.
-->

### AC-1 — Reallocate combinatorial real-git E2E coverage

