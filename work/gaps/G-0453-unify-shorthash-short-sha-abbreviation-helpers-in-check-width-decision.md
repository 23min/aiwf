---
id: G-0453
title: Unify shortHash/short SHA-abbreviation helpers in check (width decision)
status: open
priority: low
---
## What's missing

Two gratuitously-different SHA-abbreviation helpers coexist in `internal/check`: `shortHash` (8 chars, `fsm_history_consistent.go`) and `short` (7 chars, `provenance.go`). Both do the same job — truncate a git SHA for a finding message — but at different widths.

## Why it matters

One job with two implementations at two widths is convergent duplication of the kind `wf-structural-sweep` targets: a reader must reason about why two SHA abbreviators exist and which width is "right". Surfaced while landing the G-0447 seams; deliberately split out of that mechanical sweep because it is **not** behavior-preserving.

## Resolution shape

Unify onto one helper. Unlike the other G-0447 seams this carries a decision: picking a single width **changes the displayed SHA length** in one family of finding messages (7→8 or 8→7). Options: (a) standardize on 7 (git's default short-SHA width) and update the fsm-history messages; (b) keep 8 and widen the provenance messages; (c) leave them if the widths are intentional. Decide the width, then collapse to one helper. Any golden or message-asserting tests on the changed side need updating.

## Where to fix

- `internal/check/fsm_history_consistent.go` — `shortHash` (8-char).
- `internal/check/provenance.go` — `short` (7-char).

## Related

- G-0447 — the convergent-duplication cleanup this was split from (seam 4c).
