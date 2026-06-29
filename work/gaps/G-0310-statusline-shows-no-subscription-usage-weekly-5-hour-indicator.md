---
id: G-0310
title: statusline shows no subscription-usage (weekly / 5-hour) indicator
status: open
---
## What's missing

The statusline renders a context-window dot (token fill vs the model's context
limit) but nothing about **subscription-limit** consumption. Claude Code exposes
the same figures the `/usage` command shows directly in the statusline's stdin
JSON (official docs, `code.claude.com/docs/en/statusline.md`):

- `rate_limits.seven_day.used_percentage` — the weekly (7-day) limit, 0–100.
- `rate_limits.five_hour.used_percentage` — the 5-hour rolling limit, 0–100.
- each also carries `resets_at` (unix epoch).

Surface these as colored dots (green / yellow / red by threshold) so the
operator can pace usage at a glance. The field appears only for Claude.ai
Pro/Max subscribers and only after the first API response in a session, and each
window can be independently absent — so a missing window must render **nothing**
(no `?`, no `0%`), exactly as the health glyph degrades.

## Why it matters

The weekly limit is the one that strands you for days when you hit it; a
glanceable indicator lets you slow down before you're blocked, instead of
discovering it interactively. Unlike the CI and tree-health dots, this needs **no
extra process or cache** — the data is already a field in the stdin JSON the
statusline parses every render, so it is a near-free live read.

## Fix direction

A `usage_dots` segment near the context ball: for each present window, a dot
colored green / yellow / red on `used_percentage` (thresholds tuned for quota,
not context — the weekly is costly to hit, so a tighter red). Absent field →
emit nothing for that window. The 5-hour window is lower value (it self-heals in
hours) but symmetric and cheap. A later extension could append "resets in Nd" on
a red weekly from `resets_at`. Behavioral coverage via the existing statusline
harness: stub `rate_limits` in the stdin JSON and assert the dots; assert blank
when the field is absent. Touches `.claude/statusline.sh` and its byte-identical
embedded mirror.
