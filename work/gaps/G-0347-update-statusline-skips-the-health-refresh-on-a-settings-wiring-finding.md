---
id: G-0347
title: update --statusline skips the health refresh on a settings-wiring finding
status: open
---
## What's missing

`aiwf update` refreshes `.claude/health.aiwf.json` last — after every artifact is
materialized — via `doctor.WriteHealth` in `internal/cli/update/update.go`, so the
statusline installation-health stoplight reflects the just-updated setup. But the
`--statusline` branch early-returns when `cliutil.RunStatuslineScaffold` returns a
non-OK code, *before* reaching that health write. `RunStatuslineScaffold` returns a
findings code when the target settings file already holds a *different* `statusLine`
command (a pre-existing or differently-formatted key) — even though the statusline
script itself was already written and version-marked earlier in the same call. On
that path `WriteHealth` never runs, so the health cache is left stale.

## Why it matters

The stoplight then lies: after `aiwf update --statusline` marks the statusline, the
stoplight keeps rendering the stale pre-mark warning ("statusline carries no aiwf
version marker") until the next plain `aiwf update` or `aiwf doctor --write-health`.
The very command meant to fix the marker leaves the stoplight yellow reporting the
marker absent. Observed on v0.22.0: a user ran `aiwf update --statusline` with a
settings file that already carried a `~/.claude/statusline.sh` command (differing
from aiwf's canonical `$HOME/...` form); the statusline was correctly marked, but the
stoplight stayed yellow on a frozen health cache. The fix: run `WriteHealth`
regardless of the scaffold's findings-level rc — capture the rc, refresh health,
then return the rc — since the artifacts (the statusline included) did refresh and
only the optional settings-wiring reported a finding.
