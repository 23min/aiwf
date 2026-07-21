---
id: G-0427
title: aiwf show silently swallows history and scope read errors
status: addressed
priority: medium
addressed_by_commit:
    - f3e7a0ee
---
## What's missing

`BuildShowView` and `BuildCompositeShowView` (`internal/cli/show/show.go`) guard history and scope reads with `if err == nil { ... }` and no else-branch: on a `git log` failure the `History`/`Scopes` fields silently stay nil, `Run` returns `ExitOK`, and the JSON envelope's `omitempty` makes "couldn't read history" indistinguishable from "entity has no history." No test exercises either error branch. `render` and `aiwf history` treat the identical failure class fail-loud, with explicit reasoning that silently blanking a section is worse than failing.

## Why it matters

A corrupt or partially-readable repo produces a clean-looking, silently incomplete `aiwf show` — the one verb operators and AI sessions use as the canonical per-entity view. Finding F13 of `docs/initiatives/verb-layer-cleanup.md` (surfaced by the F6 deep-dive read); the fix is failing loud (`ExitInternal`) to match the siblings' precedent, plus the missing error-branch tests.
