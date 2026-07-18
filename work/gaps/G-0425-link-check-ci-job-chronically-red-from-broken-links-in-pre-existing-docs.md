---
id: G-0425
title: link-check CI job chronically red from broken links in pre-existing docs
status: open
priority: medium
discovered_in: E-0067
---
## What's missing

The `link-check` CI workflow (lychee) has failed on every `main` push since ~2026-07-17
— at least six consecutive failures — reporting 12 broken relative links in pre-existing
documentation: `docs/adr/archive/ADR-0009-…`, `docs/adr/pocv3/design/agent-orchestration.md`,
`docs/adr/pocv3/design/parallel-tdd-subagents.md`, `docs/CLAUDE.md`, and
`docs/work/gaps/G-0099-…`. The links point at files that no longer exist at the referenced
paths (e.g. `docs/adr/archive/ADR-0001-…`, `docs/adr/archive/ADR-0003-…`), likely orphaned
by an archive move or a docs reorganization. None of the affected files were touched by the
epic during which this was surfaced.

## Why it matters

A CI job that is chronically red is a broken smoke detector: once maintainers learn the
`link-check` job "always fails," a genuinely-new broken link — a dead ADR cross-reference,
a moved design doc — sails through unnoticed. Restoring the job to green (fix or remove the
12 dead links, or repoint them at the entities' current paths) makes the signal
trustworthy again. Discovered during E-0067's epic-wrap CI run while confirming the wrap's
own docs were link-clean.
