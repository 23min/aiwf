---
id: E-0054
title: 'Fast read paths: single-pass render walk and read-verb grep guard'
status: active
---
## Goal

Make aiwf's read verbs — `render`, `history`, and `show` — fast in the devcontainer,
where they are subprocess-*wait* bound (the Docker/linuxkit `fork`/`exec` tax), by
cutting per-invocation git-subprocess count from O(entities × commits) to a single
shared history pass, and by removing a repo-wide authorize grep that `aiwf history`
and `aiwf show` run unconditionally.

Measured on the kernel tree (this devcontainer):

- `aiwf render --format=html` takes **~28 minutes** because it issues **~1,860+
  per-entity `git log` walks** (~3,500 subprocesses, estimated) across **two** walk
  families: per-entity history (`resolver.history` → `history.ReadHistory`, one walk
  per epic/milestone/AC/other-entity) and per-milestone provenance/scopes
  (`show.LoadEntityScopeViews`, which re-walks the milestone's history *uncached*
  and runs a full `readAllAuthorizeOpeners` grep — once **per milestone**). A
  throwaway single-pass spike rendered **byte-identical** output in **~12.8s**
  (~130×).
- `aiwf history <id>` (default text) is **~2×** slower than it needs to be: it runs
  `BuildScopeEntityMap` — a repo-wide `git log --grep 'aiwf-verb: authorize'` — on
  **every** invocation, even though the entity has no authorization and the whole
  tree holds only a handful of authorize openers (4). On a milestone with zero scopes
  the text path measured ~2.2s vs ~1.2s for `--format=json` (which skips that grep):
  ~1.0s of pure waste per call.
- `aiwf show <id>` pays the **identical** grep by a different route
  (`LoadEntityScopeViews` → `readAllAuthorizeOpeners`, run before it knows the entity
  has any scope) and measured ~3.4s. Same waste, a *second* implementation.

This epic adds a derived *read strategy*, not a second source of truth. `git log` +
trailers stays canonical (per `design-decisions.md`); the design and per-lever
worktree/merge safety analysis live in
[`docs/pocv3/design/performance.md`](../../../docs/pocv3/design/performance.md).

## Scope

In:

- **Render single unified history walk (M-0221).** Feed render's per-entity event
  lists *and* the authorize-opener/scope map from one shared HEAD-scoped pass,
  covering **both** walk families (batching history alone still timed out in the
  spike, so the provenance/scope family must be collapsed in the same change).
  - Build on E-0053's HEAD-scoped `check.WalkHeadCommits` (extend it, or a shared
    helper, with author-date `%aI` — which also removes the per-SHA `git show` date
    lookups). Do **not** reuse `gitops.BulkRevwalk`: it walks `--all` (would leak
    feature-branch commits and break byte-identity) and its extracted trailer set
    omits `aiwf-scope-ends` / `aiwf-to` / `aiwf-prior-entity` (collapsing repeats to
    a last-value map). The new code is the bucketing + authorize-opener map +
    scope-FSM replay layer on top of one HEAD pass, not a new walker.
  - Preserve exactly: HEAD ref scope (not `--all`); `M-NNNN/AC-N` events folded
    into both the AC bucket and the milestone bucket; width canonicalization
    (`E-22` ↔ `E-0022`) on both bucket key and query id; the full trailer slice
    (repeating `aiwf-scope-ends`); SHA dedup + oldest-first order. Decide the
    error semantic deliberately — today render swallows a per-entity history error
    into one blank tab; a shared pass that errors must not silently blank *every*
    page.
- **Guard the unconditional authorize-opener grep across the read verbs (M-0223).**
  The same wasted grep has two near-duplicate implementations —
  `BuildScopeEntityMap` (`history` text) and `readAllAuthorizeOpeners` via
  `LoadEntityScopeViews` (`show`, and render). Guard **both**: skip the grep when the
  entity's *loaded events* carry no scope data (no `AuthorizedBy`, no
  `aiwf-scope-ends`); bound it to the referenced SHAs otherwise. The predicate must
  key off the loaded event slice, not entity frontmatter (`aiwf-scope-ends` is a
  commit trailer with no frontmatter counterpart). Consolidate the duplicate
  implementations rather than add a third in the render pass (single source of
  truth). **Low** correctness risk — verified that the scope map is consumed only via
  `AuthorizedBy`/`ScopeEnds` — gated on a non-vacuous fixture that includes an
  active-scope opener.

Out (deferred to G-0340):

- **Path-scoped single-entity history + changed-path bloom-filter maintenance.**
  Attractive on raw numbers (a path-scoped `git log -- <path>` with changed-path
  bloom filters measured ~65ms vs ~1.3s over the base commit-graph — a ~20× bloom
  lever — vs ~0.9s for the trailer grep), but **path-scoping is a different query,
  not a faster grep**, and single-entity history at ~1–2s is not the 28-minute render
  pain. Deferred until it is a felt pain, and gated on the correctness constraints in
  G-0340 (pathless `--allow-empty` commits from `acknowledge`/`authorize`/`audit-only`
  are invisible to a path query; `prior_ids` tracks only `reallocate` lineage, not
  `rename`/`archive`/parent-dir moves; history simplification prunes merges). The
  trailer grep stays the authoritative oracle.
- Incremental `aiwf check` via a validated watermark (G-0323) — the persistent-cache
  tier; must obey the SHA-keyed (never pointer-keyed) invariant in the perf doc; a
  separate epic.
- Branch hygiene (G-0324) and allocator ref fan-out — cheap follow-ups.

## Constraints

- **Behavior-preserving on a healthy tree.** Rendered output byte-identical to the
  per-entity path, asserted via a **synthetic golden-site fixture** enumerating the
  trap entities (see M-0221 AC-3) — not the real kernel tree (the repo's testdata
  rule requires synthetic goldens, and the 28-min old path cannot regenerate a
  reference). The one-time real-tree `diff -rq` is a dev sanity check, not the CI
  assertion. The error path is intentionally changed (see the render error semantic)
  and pinned separately, not covered by "byte-identical".
- **The trailer grep is the authoritative oracle for entity history.** Any
  path/id-scoped acceleration (deferred, G-0340) must provably equal it — including
  pathless commits and full archive/rename/parent-dir path sets.
- **No persistent cache in this epic.** Both in-scope milestones are pure recompute
  over git's own commit-graph, so they carry zero worktree/merge risk. Any future
  cache obeys the SHA-keyed invariant in the perf doc.
- No rule moves from pre-push to CI; no guarantee weakened.

## Source

- [`docs/pocv3/design/performance.md`](../../../docs/pocv3/design/performance.md) —
  the design, measurements, caching invariant, and per-lever safety analysis
  (corrected for this epic's premises in commit `af717d17`).
- E-0053 (prior perf epic; check-side subprocess collapse; source of the HEAD-scoped
  `WalkHeadCommits`) and its deferred gaps G-0323 / G-0324 / G-0325.
- The render single-pass spike (28 min → 12.8s, byte-identical across 657 pages;
  2026-07-01).
