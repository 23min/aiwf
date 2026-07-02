---
id: E-0054
title: 'Fast read paths: batch history walks and path-scope single-entity reads'
status: proposed
---
## Goal

Make aiwf's read verbs — `render` and `history` — fast in the devcontainer,
where they are subprocess-*wait* bound (the Docker/linuxkit `fork`/`exec` tax), by
cutting per-invocation git-subprocess count from O(entities × commits) to a single
shared history pass, and by removing a repo-wide grep that `aiwf history` runs
unconditionally.

Measured on the kernel tree (this devcontainer):

- `aiwf render --format=html` takes **~28 minutes** because it issues **~1,860+
  per-entity `git log` walks (~3,500 subprocesses)** across **two** walk families:
  per-entity history (`resolver.history` → `history.ReadHistory`, one walk per
  epic/milestone/AC/other-entity) and per-milestone provenance/scopes
  (`show.LoadEntityScopeViews`, which re-walks the milestone's history *uncached*
  and runs a full `readAllAuthorizeOpeners` grep — once **per milestone**). A
  throwaway single-pass spike rendered **byte-identical** output in **~12.8s**
  (~130×).
- `aiwf history <id>` (default text) is **~2×** slower than it needs to be: it runs
  `BuildScopeEntityMap` — a repo-wide `git log --grep 'aiwf-verb: authorize'` — on
  **every** invocation, even though the entity has no authorization and the whole
  tree holds only a handful of authorize openers. On a milestone with zero scopes
  the text path measured ~2.1s vs ~1.2s for `--format=json` (which skips that grep):
  ~0.9s of pure waste per call.

This epic adds a derived *read strategy*, not a second source of truth. `git log` +
trailers stays canonical (per `design-decisions.md`); the design and per-lever
worktree/merge safety analysis live in
[`docs/pocv3/design/performance.md`](../../../docs/pocv3/design/performance.md)
(which needs the corrections below folded in — see Source).

## Scope

In:

- **Render single unified history walk.** Feed render's per-entity event lists
  *and* the authorize-opener/scope map from one shared HEAD-scoped pass, covering
  **both** walk families (batching history alone still timed out in the spike, so
  the provenance/scope family must be collapsed in the same change).
  - Build on E-0053's HEAD-scoped `check.WalkHeadCommits` (extend it, or a shared
    helper, with author-date `%aI` — which also removes the per-SHA `git show` date
    lookups). Do **not** reuse `gitops.BulkRevwalk`: it walks `--all` (would leak
    feature-branch commits and break byte-identity) and collapses repeating
    trailers to a last value (would drop multi-scope `aiwf-scope-ends`). The new
    code is the bucketing + authorize-opener map + scope-FSM replay layer on top of
    one HEAD pass, not a new walker.
  - Preserve exactly: HEAD ref scope (not `--all`); `M-NNNN/AC-N` events folded
    into both the AC bucket and the milestone bucket; width canonicalization
    (`E-22` ↔ `E-0022`) on both bucket key and query id; the full trailer slice
    (repeating `aiwf-scope-ends`); SHA dedup + oldest-first order. Decide the
    error semantic deliberately — today render swallows a per-entity history error
    into one blank tab; a shared pass that errors must not silently blank *every*
    page.
- **Guard the unconditional authorize-opener grep.** Skip `BuildScopeEntityMap`
  when the entity carries no authorization/scope data (no `authorized_by`, no
  `aiwf-scope-ends`); bound it to the referenced SHAs otherwise. Roughly halves the
  default `aiwf history` text command with zero correctness risk, and the shared
  scope map benefits render too.

Out (deferred, tracked separately as a gap):

- **Path-scoped single-entity history + changed-path bloom-filter maintenance.**
  Attractive on raw numbers (a path-scoped `git log -- <path>` with changed-path
  bloom filters measured ~65ms vs ~1.3s for the base commit-graph and ~1.5s for the
  trailer grep — a ~20× lever on *path* queries), but **path-scoping is a different
  query, not a faster grep**, and single-entity history at ~1–2s is not the 28-minute
  render pain. Deferred until it is a felt pain, and gated on the correctness
  constraints below. If picked up, the trailer grep stays the authoritative oracle
  and path-scoping is a *verified accelerator only*:
  - **Pathless trailer commits are invisible to a path query.** `aiwf
    acknowledge-illegal`/`acknowledge-mistag` write `--allow-empty` commits carrying
    `aiwf-entity:` but touching no file; `git log -- <path>` cannot see them. Six
    live entities already have such events. A path-scoped result must be *unioned*
    with a bounded trailer query for these.
  - **The path set is not fully tracked in frontmatter.** `prior_ids` records only
    `reallocate` lineage — not `rename` slug changes (no frontmatter trace),
    `archive` moves (~533 entities; pre-archive path derivable only by convention),
    or transitive parent-dir moves (archiving/renaming an epic moves every child
    milestone's path with no trace in the child). A naive current-path query returns
    a fraction of an archived entity's history.
  - **History simplification.** `git log -- <path>` prunes merge commits (TREESAME)
    that `--grep` retains; matching grep semantics needs `--full-history`/`-m`.
  - Any equivalence test must include an *acknowledged* and an *archived* entity in
    its fixture, or it passes vacuously while the field breaks.
- Incremental `aiwf check` via a validated watermark (G-0323) — the persistent-cache
  tier; must obey the SHA-keyed (never pointer-keyed) invariant in the perf doc; a
  separate epic.
- Branch hygiene (G-0324) and allocator ref fan-out — cheap follow-ups.

## Constraints

- **Behavior-preserving.** Rendered output byte-identical to the per-entity path
  (assert via full-tree `diff -rq` against a real fixture).
- **The trailer grep is the authoritative oracle for entity history.** Any
  path/id-scoped acceleration (deferred) must provably equal it — including
  pathless acknowledge commits and full archive/rename/parent-dir path sets.
- **No persistent cache in this epic.** Both in-scope milestones are pure recompute
  over git's own commit-graph, so they carry zero worktree/merge risk. Any future
  cache obeys the SHA-keyed invariant in the perf doc.
- No rule moves from pre-push to CI; no guarantee weakened.

## Source

- [`docs/pocv3/design/performance.md`](../../../docs/pocv3/design/performance.md)
  — needs a correction pass: (1) it frames path-scoped history as equivalent to the
  trailer grep, which is false (pathless commits, path-set gaps, history
  simplification); (2) the M-0219 reopen should state accurately that M-0219 *did*
  evaluate `--changed-paths` but against `aiwf check`'s full-DAG walk (where it
  correctly does nothing) and never against single-entity path-scoped reads; (3) the
  "reuses the BulkRevwalk shape; no new architecture" line understates that render
  needs a HEAD-scoped walker (BulkRevwalk is `--all`).
- E-0053 (prior perf epic; check-side subprocess collapse; source of the HEAD-scoped
  `WalkHeadCommits`) and its deferred gaps G-0323 / G-0324 / G-0325.
- The render single-pass spike (28 min → 12.8s, byte-identical across 657 pages;
  2026-07-01).
