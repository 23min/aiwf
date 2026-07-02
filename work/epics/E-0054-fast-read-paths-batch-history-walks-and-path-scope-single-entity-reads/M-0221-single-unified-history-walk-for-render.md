---
id: M-0221
title: Single unified history walk for render
status: draft
parent: E-0054
tdd: required
acs:
    - id: AC-1
      title: render resolves all entity histories from a single git-history pass
      status: open
      tdd_phase: red
    - id: AC-2
      title: provenance and scope views resolve from the shared pass, not per-milestone
      status: open
      tdd_phase: red
    - id: AC-3
      title: rendered site byte-identical before and after the refactor
      status: open
      tdd_phase: red
    - id: AC-4
      title: measured render wall-time delta recorded in Validation
      status: open
      tdd_phase: red
---
## Goal

Replace render's per-entity git-history fan-out with one shared single-pass walk,
covering **both** walk families the spike identified:

1. **Per-entity events** — `resolver.history(id)` → `history.ReadHistory`, one HEAD
   walk per epic / milestone / AC composite (`M-NNNN/AC-N`) / other-entity, cached
   per id in the resolver.
2. **Provenance/scopes** — `show.LoadEntityScopeViews(m.ID)`, run once per
   milestone, which *re-walks* the milestone's history uncached **and** runs a full
   `readAllAuthorizeOpeners` grep (an unbounded HEAD `git log`), plus per-scope
   `LoadEntityScopes` walks and per-SHA `git show` date lookups.

On the kernel tree that is ~1,860+ `git log` walks (~3,500 subprocesses, estimated)
and ~28 minutes. Feed the per-entity event lists (bucketed by `aiwf-entity` /
`aiwf-prior-entity`) and the authorize-opener / scope map from one shared HEAD-scoped
pass. The spike proved ~12.8s, byte-identical across all 657 pages.

## Notes

- **Reuse, don't reinvent.** Build on E-0053's HEAD-scoped `check.WalkHeadCommits`
  (extend it, or a shared helper, with author-date `%aI` — which also eliminates the
  per-SHA `git show` date lookups in the scope views). `resolver.go` already imports
  `internal/check`, so the dependency direction is sanctioned. The genuinely new code
  is the bucketing + authorize-opener map + scope-FSM replay layer on top of one
  pass — not a new walker.
- **Do NOT reuse `gitops.BulkRevwalk`.** It walks `--all` (would leak feature-branch
  commits and break AC-3 byte-identity) and its extracted trailer set omits
  `aiwf-scope-ends` / `aiwf-to` / `aiwf-prior-entity` (it collapses repeats to a
  last-value map). `WalkHeadCommits` already captures the full trailer block and
  preserves repeats — it lacks only `%aI`.
- **Share the authorize-opener/scope map with M-0223's guard — don't add a third
  copy.** Render, `history`, and `show` all build the same map today via two
  near-duplicate implementations; the single-pass version should be the shared
  source, not a fourth.
- **Correctness traps to preserve, all load-bearing:**
  - HEAD ref scope, not `--all` (matches `ReadHistoryChain`).
  - Fold `M-NNNN/AC-N` events into **both** the AC bucket and the parent milestone
    bucket (a bare `ReadHistory(m.ID)` folds AC events in today).
  - Canonicalize width on **both** the bucket key and the query id (`E-22` ↔
    `E-0022`) so narrow/wide commits don't split into two buckets.
  - Keep the full trailer slice (repeating `aiwf-scope-ends`), not a last-value map.
  - Per-bucket SHA dedup; oldest-first order (`--reverse`).
  - Drop bucketed commits with an `aiwf-entity` trailer but empty verb+actor (the
    prose-mention false-positive `ReadHistoryChain` already excludes).
  - Replay the scope FSM (authorize opened/paused/resumed + `scope-ends`) in-memory
    from the buckets; take open/end dates from the walk's `%aI`.
- **Decide the error semantic deliberately.** Render today swallows a per-entity
  history error into one blank tab (`resolver.go` best-effort). A single shared pass
  that errors must not silently blank *every* page — pick fail-loud or degrade, and
  pin it. Byte-identity (AC-3) is a *healthy-tree* claim; the error path is changed
  by this decision and pinned separately.
- The throwaway spike (`resolver_bulkspike.go`, reverted, env-gated) is the reference
  behavior only; productionize with tests — do **not** ship the env-gated form.

### AC-1 — render resolves all entity histories from a single git-history pass

### AC-2 — provenance and scope views resolve from the shared pass, not per-milestone

### AC-3 — rendered site byte-identical before and after the refactor

The mechanical test is a **synthetic golden-site fixture** — a small fictional
planning tree committed under `testdata/`, rendered via the new path, byte-diffed
(`diff -rq`) against a committed golden site. The fixture must exercise every
correctness trap or the diff is vacuous: a pathless acknowledge (`--allow-empty`)
commit, an archived entity, an entity with **repeating** `aiwf-scope-ends`, an
`M-NNNN/AC-N` composite, both narrow (`E-22`) and canonical (`E-0022`) id widths, and
an **active-scope opener**. The one-time real-kernel-tree `diff -rq` (the 28-min old
path vs the new path) is a dev sanity check only, not this AC's assertion (the
testdata rule requires synthetic goldens; the old path can't regenerate a reference).

### AC-4 — measured render wall-time delta recorded in Validation

Structural assertion: the milestone's Validation section is present and populated with
a before/after wall-time measurement taken by `performance.md`'s "How to measure"
recipe (`strace -f -c` subprocess attribution + byte-diff), naming the mechanism
measured. The absolute number is environment-specific and not a CI gate; the AC
asserts the record exists, not a threshold.
