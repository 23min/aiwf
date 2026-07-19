---
id: G-0428
title: Scope events sort lexically by author-local timestamp, not chronologically
status: open
priority: low
---
## What's missing

`AssembleScopeViews` (`internal/cli/show/scopes.go`) sorts scope events by lexical comparison of `%aI` timestamps, which preserve each commit's author-local UTC offset. Two commits from authors in different timezones can therefore sort out of true chronological order (`...T23:00:00-07:00` sorts before `...T05:00:00+00:00` despite happening later). `show` and `render` share the one sort call and inherit the bug identically.

## Why it matters

The feature promises chronological ordering and delivers it only for same-offset histories — a real correctness bug, cosmetic in blast radius (row ordering in scope tables). Finding F14 of `docs/initiatives/verb-layer-cleanup.md`; the fix is normalizing to `time.Time` (or UTC) before sorting, once, fixing both consumers.
