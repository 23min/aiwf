---
id: G-0325
title: Parallelize aiwf check independent history walks and blob reads
status: open
discovered_in: M-0216
---

## What's missing

M-0216 collapsed the big fan-outs — the FSM per-entity walk, the 683
per-pair `merge-base`, the 46 per-branch `rev-list --first-parent`, and the
five per-rule `git log HEAD` trailer walks. The remaining independent passes
(the shared `WalkHeadCommits` HEAD walk, the FSM `git log --all --raw` walk,
the `BuildCommitDAG` build, the orphan walk's per-ref `reflog show`) still run
serially. Run the independent read-only passes on a bounded worker pool.

## Why it matters

The FSM `--raw` walk (~9s) and the provenance/DAG passes are largely
independent work that serializes today. Composes with the rest: G-0322 makes
each walk cheaper, G-0323 shorter, this overlaps them. Determinism caveat: sort
at the aggregation boundary so findings stay byte-identical.
