---
id: G-0340
title: Path-scoped single-entity history acceleration with bloom-filter maintenance
status: open
discovered_in: E-0054
---
## What's missing

`aiwf history` / `aiwf show` resolve an entity's timeline with a full-history
trailer grep (`git log --grep 'aiwf-entity: <id>'`), ~1.5s on the kernel tree.
A path-scoped `git log -- <path>` with changed-path bloom filters is ~65ms
(measured; ~20× the base commit-graph, which git writes without bloom filters by
default). Applying that lever to single-entity reads — plus maintaining the bloom
filters via `aiwf update` — is an attractive standalone perf win, deferred out of
E-0054 because single-entity history at ~1–2s is not the 28-minute render pain and
the equivalence is not free.

## Why it's deferred, not done

Path-scoping is a **different query**, not a faster grep. The trailer grep must
remain the authoritative oracle; path-scoping is a *verified accelerator only*, and
these constraints gate any implementation:

- **Pathless trailer commits are invisible to a path query.** `aiwf
  acknowledge-illegal` / `acknowledge-mistag` write `--allow-empty` commits carrying
  `aiwf-entity:` but touching no file; `git log -- <path>` cannot see them (6 live
  entities already have such events). A path-scoped result must be *unioned* with a
  bounded trailer query for these.
- **The path set is not fully tracked in frontmatter.** `prior_ids` records only
  `reallocate` id-lineage — not `rename` slug changes (no frontmatter trace),
  `archive` moves (~533 entities; pre-archive path derivable only by the archive
  convention), or transitive parent-dir moves (archiving/renaming an epic moves
  every child milestone's path with no trace in the child). A naive current-path
  query returns a fraction of an archived entity's history (measured: 1 of 3 events).
- **History simplification.** `git log -- <path>` prunes merge commits (TREESAME)
  that `--grep` retains; matching grep semantics needs `--full-history` / `-m`.
- **Bloom maintenance is net-new.** aiwf has zero commit-graph maintenance today;
  git's default `gc.writeCommitGraph` writes the base graph *without* bloom filters,
  so the lever is not self-maintaining. `aiwf update` would need to write
  `--changed-paths` (routed through the consent-gated / marker-managed artifact
  conventions; ADR-0015 for any settings touch). Filters are SHA-keyed and shared via
  the common object store — stale only ever means slower, never wrong (verified).

## Notes

- Reopening the M-0219 / G-0322 disposition is justified but on accurate grounds:
  M-0219 *did* evaluate `--changed-paths`, against `aiwf check`'s full-DAG walk (where
  it correctly does nothing) — it never evaluated single-entity path-scoped reads,
  a different query shape where the win is real.
- Any equivalence test must include an *acknowledged* and an *archived* entity in its
  fixture, or it passes vacuously while the field breaks.
