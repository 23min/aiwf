---
id: G-0737
title: The builder's Sonnet tier has no observation under the reduced guidance
status: open
discovered_in: M-0338
---
## What's missing

`aiwf.yaml` sets the builder agent's card to Sonnet at effort `xhigh` (`agents: builder: model: sonnet, effort: xhigh`), and no observation shows how Sonnet follows this repository's guidance since E-0092 moved most development rules out of the primed text. E-0092's harness fixes the model and effort in `scripts/e0092-observe/run.sh` (`claude --print … --model claude-opus-5-5 --effort high`, at `5092de5fd`; E-0092's wrap deleted the harness, which `git show 5092de5fd:scripts/e0092-observe/run.sh` still reads), so M-0334's baseline, its pilot, M-0335's observation and M-0338's after-run observed Claude only as `claude-opus-5-5` at `high`. The one Sonnet observation, M-0348's, ran `claude-sonnet-5` on Claude Code 2.1.278 before the reduction, with read-only tools, so it shows neither how Sonnet reaches the routed documents nor how it edits under them.

Nothing to run: the gap is an absent observation. A reader can look at `aiwf.yaml`'s `agents:` block and the comment above it, which grounds the builder's tier in cost and in Opus reviewing the builder's work, and at the Claude host lines of M-0334's, M-0338's and M-0348's Validation.

## Why it matters

The builder writes milestone code and tests under this guidance, which now reaches most development rules only when a session opens `.guidance/project.md` and the document it routes to. M-0338's after-run shows Opus skipping that step in some runs; how often Sonnet skips it is unknown. Where it does, the rules it misses reach builder commits and are caught, if at all, by the Opus reviewer at wrap.
