---
id: G-0305
title: aiwf writes health.aiwf.json; statusline globs+unions per-producer health files
status: open
priority: medium
---
## Problem

M-0193 shipped the statusline health glyph by *running* `aiwf check --fast` at
render and showing ⚠ on aiwf errors. Meanwhile ai-dotfiles evolved a
per-producer health model (its README + `dotfiles-doctor`) that tasks aiwf with
being a producer:

- One fenced file per producer: `<main-checkout>/.claude/health.<source>.json`,
  gitignored via `.claude/health.*.json`, atomic write, resolved to the main
  checkout even from a linked worktree.
- Schema: `{"generated_at":"<ISO8601 UTC>","findings":[{"source","severity","message"}]}`;
  `severity ∈ {info,warn,error}`; empty findings (or no file) = healthy.
- `dotfiles-doctor --write` owns only `health.dotfiles.json`; **aiwf writes
  `health.aiwf.json` itself**.
- A statusline globs `health.*.json`, unions the findings, and shows one
  far-left ⚠ at the max severity — and never runs checks.

M-0193's render-run-check approach does not fit: aiwf is not a producer yet, the
statusline surfaces only aiwf (not dotfiles) findings, and it runs a check
rather than reading the union.

## Direction

1. aiwf becomes a producer: write `.claude/health.aiwf.json` from `aiwf check
   --fast` findings, mapped to the schema (`source: "aiwf"`, error→error,
   warning→warn), into the main checkout's `.claude/`, atomic + gitignored. New
   surface — a `--write-health` flag on `aiwf check`, or an `aiwf health` verb.
2. Statusline reads the union: glob `.claude/health.*.json`, union findings,
   render one ⚠ at max severity (dotfiles + aiwf both surface), never running a
   check itself.
3. Open fork: the "live" refresh of `health.aiwf.json` — render-adjacent write
   vs hook-driven — and the cross-repo question of which statusline owns the
   glob-union reader (aiwf's shipped statusline vs ai-dotfiles').

Keep `aiwf check --fast` (M-0193) as the engine. Spans aiwf + ai-dotfiles;
likely plans into a milestone. Surfaced while shipping E-0047, confirmed by an
ai-dotfiles pull revealing the per-producer health-file design.
