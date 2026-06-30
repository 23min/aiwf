---
id: G-0324
title: 'Branch hygiene: prune merged ritual branches; oracle skips merged refs'
status: open
discovered_in: M-0216
---

## What's missing

The isolation-escape oracle and `--all` revwalks traverse every ref, including
already-merged ritual branches. Prune merged `milestone/*`, `epic/*`,
`patch/*` branches, and make the oracle skip trunk-ancestor refs.

## Why it matters

Oracle and orphan-walk cost scale with ref count; stale merged branches
inflate the provenance layer (and the orphan walk's ~46 `reflog show`) for
zero validation benefit (already validated on trunk). Cheap, complementary to
G-0323. Note: M-0216 AC-6 already moved the oracle's first-parent index
in-memory, but the per-ref reflog reads in the orphan walk still scale with
ref count.
