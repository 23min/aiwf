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

## Validation

Measured on the kernel tree in this devcontainer (Docker/linuxkit) with `performance.md`'s
"How to measure" recipe. The **before** binary is the pre-M-0221 per-entity render (built
from commit `129d6d19`, the M-0221 `in_progress` promote); the **after** binary is the
single-pass build. Both render the same on-disk repo (same HEAD), into separate output
dirs, timed best-of-run via the render envelope's `elapsed_ms`.

| render `--format=html` | wall (elapsed_ms) | pages | git-history mechanism |
|---|---|---|---|
| before (per-entity fan-out) | **~35 min** (2,096,996 ms measured) | 688 | ~N+2 `git log --grep aiwf-entity` per milestone + the authorize-opener grep + per-SHA `git show` dates |
| after (single unified pass) | **~4.5s** (best of 3: 4464 / 4474 / 4700 ms) | 688 | ONE `git log --reverse HEAD` (`check.WalkHeadCommits` + `%aI`/`%s`) → in-memory buckets |

That is a **~466× wall-time cut** (2,096,996 ms → 4,464 ms) on the same 688-page tree — the
before/after each measured directly (the render envelope's `elapsed_ms`), not extrapolated
from the E-0053 spike.

Output is **byte-identical** before/after for the history and provenance content — proven
at two layers. (1) The committed data differential (`TestSinglePass_*`) asserts every
per-entity bucket equals `ReadHistoryChain([id])` and every scope view equals
`LoadEntityScopeViews`, over a fixture exercising every trap — this is the rigorous,
permanent guard (both oracles live on for the read verbs). (2) A one-time real-tree
`diff -rq` of the two rendered sites (dev sanity check, not a committed test — the testdata
rule requires synthetic goldens): **687 of the 688 pages were byte-identical**, `status.html`
included. The sole differing page was `M-0221.html` itself — its spec body (this Validation
section) was being authored during the 35-minute *before* render, so the two binaries read
the milestone's own markdown at different states; the diff there is body prose, not a
history/provenance rendering difference (same tab set, no scope/commit-row changes).

The subprocess mechanism is pinned mechanically by the AC-1/AC-2 git-trace count seam
(`TestRenderSinglePass_OneHeadWalkZeroPerEntityGreps`): zero per-entity greps, zero
authorize-opener greps, zero per-SHA scope-date `git show`s, exactly one HEAD walk.
Absolute numbers are devcontainer-specific and not a CI gate.

## Work log

### AC-1 — render resolves all entity histories from a single git-history pass
Extended `check.WalkHeadCommits` with `%aI` + `%s` (additive; check consumers unaffected)
and added `render/singlepass.go`: `buildHistoryIndex` buckets one HEAD walk into per-entity
`history.HistoryEvent` slices via the new pure `history.EventFromCommit`, folding
`M-NNNN/AC-N` into both the AC and milestone buckets, width-canonicalizing keys, deduping
per bucket, skipping prose-mentions. `resolver.history` is now a canonicalized in-memory
lookup. Pinned by `TestSinglePass_EventsMatchReadHistory` (bucket == `ReadHistoryChain`
oracle, every trap), `TestSinglePass_MilestoneFoldsACEvents`, `TestHistoryBucketKeys`, and
the runtime count seam `TestRenderSinglePass_OneHeadWalkZeroPerEntityGreps` (one HEAD walk,
zero per-entity greps).

### AC-2 — provenance and scope views resolve from the shared pass, not per-milestone
Consolidated the scope FSM replay (`cliutil.ReplayScopes`) and the authorize-opener map
(`cliutil.OpenersFrom`) into pure functions shared with M-0223's `LoadEntityScopes` /
`AuthorizeOpeners` (no fourth copy), and split `show.LoadEntityScopeViews` into a
git-gather part (keeping the M-0223 cost gates) and a pure `show.AssembleScopeViews`.
`resolver.provenanceFor` derives scope views from the index (opener map + replayed scopes +
`%aI` dates) through the same `AssembleScopeViews`. Pinned by
`TestSinglePass_ScopeViewsMatchLoadEntityScopeViews` (index views == `LoadEntityScopeViews`
oracle) and the count seam's zero-authorize-grep / zero-`git show` assertions.

### AC-3 — rendered site byte-identical before and after the refactor
The committed differential runs both projections over the synthetic trap fixture and
asserts equal (buckets == `ReadHistoryChain`; scope views == `LoadEntityScopeViews`); the
existing render integration suite (`render_templates_test.go`,
`TestBinary_RenderHTML_EndToEnd`) renders real trees with scopes and asserts the
history/provenance HTML — unchanged and green — proving the wiring end-to-end. The real
tree `diff -rq` (Validation) is the one-time sanity check.

### AC-4 — measured render wall-time delta recorded in Validation
See `## Validation`. Structural evidence: `TestM0221_AC4_ValidationRecordsRenderMeasurement`
(section-scoped assertion that Validation names both before/after, both mechanisms, and
carries wall-time values).

The authoritative per-AC phase/status timeline lives in `aiwf history M-0221/AC-N`.

## Deferrals

None originate in this milestone. The persistent SHA-keyed read-model cache that would make
even a single-entity read O(Δ-since-cache) (and would benefit the read verbs, not just
render) stays deferred as `G-0323` — it carries the worktree/merge-safety risk this
batching lever deliberately avoids (`performance.md` §"The caching invariant"). Path-scoped
history + bloom filters remain `G-0340`.

## Reviewer notes

- **The single-pass index is batching, not caching.** It lives only for one render
  invocation and is discarded at exit; the win is collapsing ~1,860 per-entity `git log`
  walks into one *within* a single render (688 pages), not amortizing across renders.
  Because it persists nothing, worktree/branch switching has zero new failure modes — every
  render reads the current worktree's HEAD fresh, and the HEAD scope (not `--all`) keeps
  output branch-consistent, matching `aiwf history` on the same checkout.
- **This is `aiwf check`'s model (E-0053), extended to render** — the two surfaces that fan
  one process over many entities. The single-entity read verbs (`history`/`show`) correctly
  got the *opposite* lever (M-0223's grep-guard); a whole-HEAD index would pessimize a
  one-entity query.
- **Error semantic: fail-loud, deliberate.** A shared-walk failure would silently blank the
  history/provenance of *every* page — strictly worse than the old per-entity best-effort
  that dropped one tab — so `RunSite` exits `ExitInternal`. Pinned by
  `TestRenderSinglePass_FailsLoudOnUnreadableHistory` (corrupt-history repo).
- **Differential blind spot, covered elsewhere.** `TestSinglePass_ScopeViews...` is a
  *source-equivalence* differential: both the index path and `LoadEntityScopeViews` now run
  through the shared `AssembleScopeViews`, so it cannot catch a bug *inside* Assemble. That
  logic (e.g. the `ent != id` self-scope guard) stays guarded by M-0223's independent
  `unguardedScopeViews` oracle in `scope_grep_guard_test.go` — verified by mutation probe.
- **`%B`→`%b` body derivation.** `EventFromCommit` derives the prose body from the full
  message by dropping everything up to the first blank line, matching git's own subject/body
  split. The trap fixture carries a commit with a real multi-line prose body plus the
  force / audit-only / principal / on-behalf-of / reason trailer set, so the differential's
  full-struct equality against `ReadHistory` (which reads `%b` directly) exercises the body
  derivation *and* those five otherwise-untested fields — not just trailer-only commits.
  Same latent `unfold=true` vs `valueonly` trailer-parse caveat E-0053 accepted for
  `aiwf check` — a healthy-tree claim, with the real-tree `diff -rq` as the one-time
  confirmation.
- **Committed golden site: deliberately not added.** The old render path is *removed* (render
  has only the single-pass path), so an old-vs-new *site* diff isn't binary-runnable. The
  data differential compares against the *live* `ReadHistoryChain` / `LoadEntityScopeViews`
  oracles (which live on for the read verbs) — a permanent regression guard stronger than a
  frozen golden, and it avoids the `status.html` timestamp brittleness a golden would carry.
- **Design review (`wf-rethink`): KEEP, no rework.** A fresh-context design pass over the
  three units (`historyIndex`/`buildHistoryIndex`, the `check.HeadCommit` reuse, the shared
  primitive seams) found no blocking issue: the event/exact bucket split is intrinsic (the
  two oracles have different membership); extending `check.HeadCommit` with two additive
  fields beats duplicating an ~80-line walker; `AssembleScopeViews`' closures encode a real
  strategy difference (Assemble decides *what* is needed, the caller supplies *how*); and
  keeping `EventFromCommit` separate from the oracle preserves the differential's
  independence. One micro-simplification was taken in-context: the exact-entity bucket dedup
  is a per-commit local set (duplication is only possible within one commit's trailers, since
  `WalkHeadCommits` visits each SHA once), matching `historyBucketKeys`' own pattern. Two
  YAGNI toss-ups (a `ScopeViewSources` struct for the 6-arg signature; relocating the walker
  to `gitops` if a third fan-out surface appears) were noted and left.
