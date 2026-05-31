---
id: G-0189
title: CI statusline shows stale result after push
status: open
---
## Problem

After a push, the CI segment (`✓ ci`) shows the previous push's result
because `gh run list --branch <b> --limit 1` returns the most recent run,
which is still the old one until GitHub creates the new run. The 45-second
cache TTL makes it worse — even after the new run starts, the cache serves
the stale green.

This is misleading: it looks like instant CI confirmation when the new run
hasn't even started.

## Fix

Compare the CI run's `headSha` with the local `git rev-parse HEAD`:

1. Add `headSha` to the `gh run list --json` fields.
2. If `headSha` != local HEAD → stale result → show `… ci` (gray, pending).
3. Include HEAD in the cache key so a push auto-invalidates.

Glyphs after fix:
- `✓ ci` green — CI passed *for this commit*
- `✗ ci` red — CI failed *for this commit*
- `→ ci` yellow — CI running *for this commit*
- `… ci` gray — no CI result for this commit yet
- `? ci` — couldn't determine CI state

## References

- `.claude/statusline.sh` lines 280–331 (CI section)
