---
id: M-0221
title: Single unified history walk for render
status: in_progress
parent: E-0054
tdd: required
acs:
    - id: AC-1
      title: render resolves all entity histories from a single git-history pass
      status: met
      tdd_phase: done
    - id: AC-2
      title: provenance and scope views resolve from the shared pass, not per-milestone
      status: met
      tdd_phase: done
    - id: AC-3
      title: rendered site byte-identical before and after the refactor
      status: met
      tdd_phase: done
    - id: AC-4
      title: measured render wall-time delta recorded in Validation
      status: met
      tdd_phase: done
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
- **Share the authorize-opener/scope helper with M-0223 — don't add a third copy.**
  Render, `history`, and `show` all build the same map today via two near-duplicate
  implementations; the single-pass version should reuse M-0223's consolidated helper,
  not add a fourth.
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
    from the buckets, **including scopes opened on the milestone itself** (its own
    `authorize` commit is in its bucket); take open/end dates from the walk's `%aI`.
- **Decide the error semantic deliberately.** Render today swallows a per-entity
  history error into one blank tab (`resolver.go` best-effort). A single shared pass
  that errors must not silently blank *every* page — pick fail-loud or degrade, and
  pin it. Byte-identity (AC-3) is a *healthy-tree* claim; the error path is changed
  by this decision and pinned separately.
- The throwaway spike (`resolver_bulkspike.go`, reverted, env-gated) is the reference
  behavior only; productionize with tests — do **not** ship the env-gated form.

### AC-1 — render resolves all entity histories from a single git-history pass

Mechanical seam assertion (byte-identity alone doesn't prove the *mechanism* — you can
get identical output the slow way). Drive render over the synthetic fixture through an
injected/counted git seam and assert: exactly **one** HEAD history walk is issued, and
the render path makes **zero** per-entity `history.ReadHistory` / `resolver.history`
subprocess calls. The call count is the evidence.

### AC-2 — provenance and scope views resolve from the shared pass, not per-milestone

Same seam: assert render makes **zero** per-milestone `show.LoadEntityScopeViews`
calls and **zero** `readAllAuthorizeOpeners` invocations; the opener/scope map and the
scope FSM are derived from the shared pass (via M-0223's consolidated helper). Count,
don't infer.

### AC-3 — rendered site byte-identical before and after the refactor

**Differential test, not a bare golden.** While the old per-entity path still exists,
run both projections on the synthetic fixture — old (`ReadHistoryChain` +
`LoadEntityScopeViews`) vs new (bucketed single-pass) — and assert equal, and
`diff -rq` the full rendered site old-vs-new; delete the old path last. This proves
*new == old*, which a static golden (new == golden) does not. The synthetic fixture
must exercise every trap: a pathless acknowledge (`--allow-empty`) commit, an archived
entity, an entity with **repeating** `aiwf-scope-ends`, an active-scope opener, an
`M-NNNN/AC-N` composite, and both narrow (`E-22`) and canonical (`E-0022`) id widths. A
committed synthetic golden site remains as the post-deletion regression guard. The
one-time real-kernel-tree `diff -rq` (28-min old path vs new) is a dev sanity check
only, not this AC's assertion (the testdata rule requires synthetic goldens).

### AC-4 — measured render wall-time delta recorded in Validation

Structural assertion: the milestone's Validation section is present and populated with
a before/after wall-time measurement taken by `performance.md`'s "How to measure"
recipe (`strace -f -c` subprocess attribution + byte-diff), naming the mechanism
measured. The absolute number is environment-specific and not a CI gate; the AC
asserts the record exists, not a threshold.
