---
id: G-0323
title: Incremental aiwf check via validated trunk watermark (walk only new commits)
status: wontfix
discovered_in: M-0216
---

## What's missing

`check` re-validates the whole history every run. No incremental mode scoping
the expensive walks to `<watermark>..HEAD` (a validated last-green trunk
marker), with full-walk fallback when absent/invalid.

## Why it matters

M-0216's floor analysis pins the byte-identical floor at ~20-30s because the
full-history `git log` can't be eliminated. Incremental scoping is the way
under the floor — flagged as "the biggest architectural lever". The hard part
is sound watermark invalidation (rewrites, force-pushes, rule changes), not
speed.
