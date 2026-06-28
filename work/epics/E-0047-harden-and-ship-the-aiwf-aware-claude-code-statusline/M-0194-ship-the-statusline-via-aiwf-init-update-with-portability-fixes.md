---
id: M-0194
title: Ship the statusline via aiwf init/update with portability fixes
status: draft
parent: E-0047
depends_on:
    - M-0193
tdd: required
---
## Deliverable

`aiwf init` / `update` materializes `.claude/statusline.sh` to consumers, with portability fixes (G-0183) — today it's the one aiwf-aware artifact with no consumer install path.

- Materialize the statusline as a gitignored, marker-managed artifact alongside the other `.claude/` artifacts (verb/ritual skills, agents, templates); the materialized copy matches this repo's own.
- Fix the portability defects: the `tac` token-transcript walk (absent on stock macOS) and the literal-tab sync parse (`${counts%%<TAB>*}`, fragile under editors that normalize whitespace).
- `aiwf init/update --statusline` wires `"statusLine"` into the Claude Code settings file under the explicit per-invocation consent established by ADR-0015 (interactive `[y/N]` on a TTY, or the explicit `--wire-settings` flag) — the one sanctioned settings edit.

Capstone: ships only after the statusline is correct (M2) + health-aware (M3) and covered by the M1 harness. ACs at milestone start.
