---
id: M-0161
title: 'Imagination-driven hardening: shallow, force-push, rename, detached, trunk'
status: draft
parent: E-0030
tdd: required
---
## Goal

Close the imagination-driven hardening gaps (G-0200, G-0201, G-0203, G-0204, G-0205, G-0206, G-0207, G-0209, G-0210) using the combinatorial real-git E2E framework M-0159 lands. These scenarios have no in-repo historical evidence — but per the user's M-0159 planning directive *"if we can imagine it, it will happen"*, coverage is mandatory because different operators have different workflows. The user has historically said N to force-push; other operators routinely say Y.

This is **Tier 3** in the E-0030 hardening evidence-priority split:

- **Tier 1 (M-0159)** — combinatorial framework + override convergence + seam coverage, all evidence-backed.
- **Tier 2 (M-0160)** — operational pain regressions, evidence-backed from this repo's incident history.
- **Tier 3 (M-0161, this milestone)** — imagination-driven coverage. No in-repo evidence, but the kernel principle "framework correctness must not depend on the LLM's behavior" requires mechanical pins for every scenario aiwf claims to handle.

## Context

The M-0158 honest-scope audit surfaced these as unmodeled real-world failure modes. The M-0159 history-mining investigation then categorized them as imagination-only (no in-repo evidence). The user's response on whether to drop them: *"don't remove the ones for which we don't have empirical evidence at this time. 'if we can imagine it, it will happen' (just because we didn't run into certain things may be because of the way I work, but other users work differently. I have often said N when asked to force push, for instance, but someone else might think it's OK)"*.

So all imagination-driven scenarios sequence into this milestone, fully covered via the M-0159 framework. The work is real even where the evidence is hypothetical.

## Scope

Gaps consumed (9 total):

- **G-0200** — preflight main-only carve-out hardcodes "main"; generalize to `aiwf.yaml.allocate.trunk`.
- **G-0201** — authorize preflight carve-out accepts cross-rung ritual mismatches; tighten hierarchical predicate.
- **G-0203** — BranchOracle.FirstParentBranches conflates lookup-failed with no-branches; typed errors + fail-shut decision.
- **G-0204** — BranchOracle silent on shallow clones (CI fetch-depth=1); detect + fail-shut or document fallback.
- **G-0205** — BranchOracle silent on force-pushed-away violating commits; reflog-walk or documented limitation.
- **G-0206** — BranchOracle false-positive on branch renames after authorize; reflog-walk for rename events.
- **G-0207** — Detached-HEAD handling untested in preflight and oracle; explicit error or supported path.
- **G-0209** — Ritual step ordering is advisory only; either kernel enforcement or remove the ordering claim from SKILL.md.
- **G-0210** — M-0158 spec table contains 9 documentation-only or duplicate cells; refactor catalog to mechanical-weight-only set.

## Dependencies

- **M-0159** (Tier 1) and **M-0160** (Tier 2) — both must complete first. M-0161 reuses M-0159's E2E framework and may depend on M-0160's reallocate-stress helpers.
- **G-0213** (cellcoverage landmine) — must be closed in M-0159 before any M-0161 rule reads `aiwf-branch` against a resolvability check.

## Out of scope

- New override paths beyond what M-0159 lands.
- Generalizing trunk config beyond named trunks (e.g., arbitrary "current ref is parent") — G-0200's scope is named-trunk only.
- Data-loss scenarios (G-0212) — future-epic.

## Acceptance criteria

<!--
AC seed set (to be allocated via `aiwf add ac` at start-milestone time, one per gap with combinatorial real-git E2E coverage required for each):

1. G-0200 — trunk-name configuration: hardcoded "main" → aiwf.yaml.allocate.trunk; real-git E2E with a non-default trunk name.
2. G-0201 — cross-rung carve-out hierarchical predicate; real-git E2E with epic-ritual + milestone-ritual interaction.
3. G-0203 — BranchOracle typed errors (lookup-failed vs no-branches); rule fails-closed on lookup error; real-git E2E.
4. G-0204 — shallow-clone handling: real-git E2E with git clone --depth=1; either detect + fail-shut or documented fallback path.
5. G-0205 — force-pushed history: real-git E2E with git push --force; either reflog-walk preserves the audit trail or documented limitation surfaces.
6. G-0206 — branch-rename handling: real-git E2E with git branch -m mid-scope; reflog-walk preserves the rename event.
7. G-0207 — detached-HEAD handling: real-git E2E with checkout <sha>; explicit error path or supported flow.
8. G-0209 — ritual step ordering: either kernel-side enforcement OR remove the ordering claim from SKILL.md. Per "no advisory-only floating" directive.
9. G-0210 — M-0158 spec-table catalog refactor: remove documentation-only cells, keep mechanical-weight cells only; structural meta-coverage redesign (branchcell.Pin registry with bijection enforcement) lands alongside.

These 9 are the seed set; aiwfx-start-milestone refines and allocates them.
-->

