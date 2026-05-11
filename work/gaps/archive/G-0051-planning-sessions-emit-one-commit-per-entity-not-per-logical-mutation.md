---
id: G-0051
title: Planning sessions emit one commit per entity, not per logical mutation
status: addressed
discovered_in: E-0014
addressed_by:
    - M-0056
---

## Problem

Planning a complete epic in one session — epic + N milestones + M acceptance criteria — currently produces 1 + N + M (+ a body-content commit) commits. Concrete data: planning E-0014 generated 42 commits (1 epic + 7 milestones + 1 body-content commit + 33 AC commits). For users — human or LLM — this is friction: history is dominated by allocation events, per-entity rationale dissolves into noise, and reviewing a planning session means scrolling through dozens of one-line commits.

## Root cause

`aiwf add` is invocation-per-entity. The kernel rule *"every mutating verb produces exactly one git commit"* is intended as an atomicity guarantee (no half-finished mutations get committed), but it's currently read by the verb implementation as "one entity per commit" — much stricter than the rule actually requires. There is no useful intermediate state where, for example, M-0049 has AC-1 and AC-2 but not AC-3; the whole milestone-with-its-ACs is a single logical unit. Splitting it across 7 commits dilutes git history without adding atomicity.

## Direction (not a commitment)

Extend `aiwf add` to accept batched inputs, in increasing order of ambition:

1. **Repeated `--ac-title`** when adding a milestone (or via `aiwf add ac M-NNN`) — N ACs in one commit.
2. **`--body-file`** on every `add` variant — fold body content into the create commit (also resolves G-0052).
3. **Inline milestone definitions** when adding an epic — one commit creates the epic and its milestone scaffolds.

Combining 1 + 2 + 3 brings a 42-commit planning session down to ~8 (one per milestone, body + ACs inline) without changing the kernel atomicity guarantee. A heavier-weight `aiwf plan apply <file>` verb is conceivable for "1 commit per epic" but is a separate decision.

## Why this is a gap, not an enhancement

The friction is observable now and the user explicitly flagged this shape as not acceptable for downstream users. If commit-cardinality blocks adoption, it's a gap.
