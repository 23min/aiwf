# Performance — how aiwf scales with git history

**Status:** living document. This is the *ground doc* for performance in aiwf —
where the cost model, the measured baselines, the levers, and the one load-bearing
caching invariant live. Prior perf knowledge was scattered across an archived
epic's wrap (E-0053), a research paper ([`docs/research/00-fighting-git.md`](../research/00-fighting-git.md)),
and a lone perf test; this doc consolidates it. Update it when you measure, ship a
lever, or change the cost model.

Companion reading: [`design-decisions.md`](design-decisions.md) (why history *is*
git log), [`00-fighting-git.md`](../research/00-fighting-git.md) (the branch/merge
theory that makes the caching invariant below the safe one).

---

## TL;DR

1. **aiwf's storage design is correct; its *read strategy* is what scales with
   history.** History = `git log` + trailers is the right source of truth. What was
   missing is a derived read layer over it — the same move git itself makes with
   `commit-graph`. Keep the storage design; fix the reads.
2. **The one invariant that keeps any cache safe under worktrees/merges/rebases:
   cache only immutable facts, keyed by the git object SHA they derive from — never
   by a moving ref (HEAD, a branch, "trunk").** A SHA-keyed cache can only ever be
   *incomplete* (costs a recompute), never *wrong*. A pointer-keyed cache (the naive
   "trunk watermark") is invalidated by exactly the branch/merge operations aiwf does
   constantly. This is why git's own commit-graph survives everything.
3. **Three distinct cost classes, not one** (details below): per-entity O(total
   commits) trailer greps; subprocess-spawn overhead (brutal on Docker/linuxkit
   devcontainers); and O(refs) fan-out over local + remote branches.

---

## Measured baseline

Recorded on the kernel repo itself (`github.com/23min/aiwf`), 2026-07-01:

| dimension | value |
|---|---|
| commits (HEAD) | 5,510 |
| refs total / local branches / remote refs | 90 / 48 / 10 |
| entity markdown files | 652 |
| `.git` size | 217 MB |
| git version | 2.54.0 |
| commit-graph present | yes (base chunks only — **no** changed-path bloom filters) |

Verb wall-times (warm OS cache, devcontainer on Docker/linuxkit):

| verb | wall | dominant cost |
|---|---|---|
| `aiwf list` | 0.24s | pure filesystem — **the floor, and it's fine** |
| `aiwf status` | 1.0s | one bounded `git log -n 20` — already correct |
| `aiwf history M-0091` | 2.3s | one `git log --grep` over all 5,510 commits |
| `aiwf show M-0091` | 3.4s | history + scopes + provenance = several full walks |
| `aiwf render --format=html` | **28 min** (657 pages) | ~N+2 full-history greps *per milestone*, ×2 walk families |
| `aiwf check` | **78s** | subprocess-bound: 11s user + 29s system |

> **Since shipped (E-0054):** the `render` and read-verb rows above are the *before*
> baseline. E-0054 landed two of the levers below — `aiwf render --format=html` is now
> **~4.5s** (688 pages, ~466×; M-0221's single unified history walk) and `aiwf history` /
> `aiwf show` shed ~44% / ~32% on scopeless entities (M-0223's grep guard). See the
> "Recommended sequence" section for the shipped-vs-deferred status of each lever.

> The devcontainer runs on a Docker/linuxkit VM where `fork`/`exec`/filesystem
> syscalls are far slower than native Linux or CI. The 29s of *system* time in
> `aiwf check` is that tax. **Felt latency is dominated by subprocess count**, more
> than CI numbers (E-0053 measured ~21s for the same check on faster hardware)
> suggest. Optimizations that cut subprocess count help the local experience
> disproportionately.

---

## Why reads scale with history

aiwf deliberately has **no separate event log and no projection file**
([`design-decisions.md`](design-decisions.md)): markdown frontmatter is canonical
state, `git log` is the history, and structured trailers (`aiwf-verb:`,
`aiwf-entity:`, `aiwf-actor:`) make the log queryable. This is what makes aiwf
merge-safe and drift-free — it is *right* and must stay.

The cost of that purity: any operation that needs historical or provenance data
re-derives it by walking git history, per invocation, with no materialized view.
That couples read latency to *total repo history size*. The fix is **not** to add a
second source of truth — it is to add a *derived, rebuildable read layer*, exactly
as git's `commit-graph` is a disposable materialized view over the immutable object
store. This distinction is the crux: **"no event log as source of truth" ≠ "no
derived cache."** The repo's own C1 code-health principle explicitly blesses caches
that name their invalidation rule.

Is aiwf "designed incorrectly"? No — it is designed correctly but was missing the
read-optimization layer every git-backed tool eventually adds. That layer was
explicitly deferred until a real consumer felt the friction. The friction has
arrived.

---

## The three cost classes

### 1. Per-entity O(total commits) trailer grep

`aiwf history`, `aiwf show`, and especially `aiwf render` resolve an entity's
history via `git log --grep '^aiwf-entity: <id>$'` (`internal/cli/history/history.go`
`ReadHistoryChain`). `--grep` is only a pre-filter — git still reads *every* commit
message. So each call is O(5,510 commits) regardless of how few touched the entity.

`render` multiplies this: `internal/cli/render/resolver.go` resolves history
**N+2 times per milestone** — once per AC via the `M-NNNN/AC-N` composite, plus the
shared `m.ID` walk, plus a second *uncached* `m.ID` walk inside
`show.LoadEntityScopeViews` (the commits table and provenance timeline reuse the
resolver cache, so they add no walks). Across all entities that is ~1,860+
full-history walks in one render — and the authorize-opener grep family on top (see
the spike result below). This is the 28-minute render.

**The lever git already offers but aiwf doesn't use: query by *path*, not by grep.**
Each entity is a file at a known path. With changed-path bloom filters, `git log --
<path>` skips the commits that never touched the file in microseconds. Measured on
this repo:

| one entity's history | no bloom filters | with bloom filters |
|---|---|---|
| `git log -- <exact path>` | ~1.3s | **~65ms** typical (~14ms best case) |
| `git log --grep=aiwf-entity:` | ~0.9s | ~0.9s (grep can't use bloom) |

**Path-scoping is a fast *accelerator*, not a drop-in for the grep — it is a
different query.** Three gaps make `git log -- <path>` ≠ `git log --grep=aiwf-entity:`,
so the trailer grep must stay the authoritative oracle and path-scoping a *verified*
fast path (deferred to G-0340):

- **Pathless trailer commits are invisible to a path query.** Any `--allow-empty`
  commit carrying `aiwf-entity:` but touching no file is missed — a whole class:
  `acknowledge-illegal` / `acknowledge-mistag`, `authorize` openers/lifecycle, and
  `audit-only`. Six entities already have empty acknowledge events alone (five live,
  one archived). A path query must be unioned with a bounded trailer query.
- **The path set is only partly tracked.** `prior_ids` records `aiwf reallocate`
  lineage only (26 entities) — *not* `aiwf rename` slug changes (30 commits, no
  frontmatter trace), `archive` moves (~508 entities; pre-archive path derivable by
  convention, not frontmatter), or transitive parent-dir moves (archiving an epic
  moves every child milestone's path). A naive current-path query returned 1 of 3
  events for an archived entity (for archived G-0103, the grep and path sets are
  entirely disjoint).
- **History simplification.** `git log -- <path>` prunes merge commits (TREESAME) that
  `--grep` retains; matching grep semantics needs `--full-history` / `-m`.

On a typical entity the lever is ~20× (path-scoped ~65ms vs ~1.3s over the base
commit-graph; best case ~14ms on a rarely-touched path) — worth building once the
equivalence above is handled, not before.

> **Spike result (2026-07-01, this repo).** A throwaway spike routed `render`'s
> per-entity history through one shared single-pass walk (env-gated, one binary doing
> both). It also surfaced a *second* per-entity O(commits) walk family: `provenanceFor`
> → `show.LoadEntityScopeViews` does **two** more full greps per milestone
> (`ReadHistory` + `readAllAuthorizeOpeners`). Batching *both* families (one history
> walk + one memoized authorize-opener walk) gave:
>
> | render `--format=html` (657 pages) | wall (this devcontainer) |
> |---|---|
> | baseline (per-entity greps) | **28 min 03s** |
> | bulk (both walk families batched) | **12.8s** (~130×) |
>
> Clean, sequential, uncontended runs; output **byte-identical across all 657 pages**
> (full-tree `diff -rq`). Note the baseline's 43% CPU / 462s *system* time — it is
> subprocess-*wait* bound (the Docker/linuxkit `fork`/`exec` tax), not compute bound,
> so cutting subprocess count is the lever that pays here. The lesson for the
> production change: render needs *two* things
> from history (per-entity events + the authorize-opener map) and both come from **one**
> pass — the fix is a single unified walk feeding both, not two separate batchings.

### 2. Subprocess-spawn overhead

`aiwf check` (pre-push hook — runs constantly) historically spawned ~895 git
processes; E-0053 cut that to ~8–10 by building an in-memory commit DAG and a shared
HEAD walk. `aiwf add` still spawns ~13–14, scaling O(local branches) because it runs
`git ls-tree` per branch (see class 3). On the devcontainer each `fork`/`exec` is
expensive, so subprocess count is the felt-latency driver.

The structural fix is partly in the codebase, but the reusable piece is the
HEAD-scoped walker, **not** `BulkRevwalk`. `internal/gitops/revwalk.go` `BulkRevwalk`
runs **one** `git log --all --raw` subprocess for `check`, but it walks `--all` and
collapses repeating trailers to a last-value map — unsafe for the read verbs, whose
output is HEAD-scoped and needs `aiwf-scope-ends` / `aiwf-to` / `aiwf-prior-entity`.
E-0053's `check.WalkHeadCommits` is the HEAD-scoped, full-trailer precedent to build
on. Routing `render` through one such shared HEAD pass collapses ~1,860 walks → 1
(see the render spike below); `history` / `show` instead drop the redundant
authorize-opener grep (M-0223), not via a batched walker.

### 3. O(refs) fan-out

48 local branches. The allocator (`internal/trunk`) enumerates `refs/heads/*` and
`refs/remotes/*` and runs a `git ls-tree` **per ref** on every `aiwf add`
(ADR-0025). ADR-0030 (E-0060) widened this same per-ref scan to two more
consumers: `aiwf check` now runs it eagerly (once per invocation, via
`LoadTreeWithTrunk`), and `aiwf show`/`aiwf list` run it lazily — `show`
only on a local-tree miss for the one queried id, `list` only from within
a filtered listing, never the no-args counts path — so neither read verb's
common case (local resolution) pays the O(refs) cost. `aiwf check`'s reflog /
isolation oracle walks per ritual head. Many of those 48 branches are almost
certainly merged ritual/epic branches. **Branch hygiene** (prune merged branches;
have the oracle skip merged refs — G-0324) is a cheap partial win, and a per-ref-SHA
`ls-tree` cache (a ref that hasn't moved yields the same ids) removes the rest.

---

## What E-0053 already did (and what it left)

Epic **E-0053** — "Make aiwf check and the policies test suite fast" (done,
2026-06-30). Delivered:

- **M-0216** — shared per-check git-history context: replaced 683
  `merge-base --is-ancestor` calls with one `git rev-list --all --reflog --parents`
  DAG; blob-object-id dedup via `BulkRevwalk`; collapsed 5 HEAD walks into one.
  Result: `aiwf check` ~48.8s → ~21.7s (with base commit-graph) on CI hardware,
  behavior byte-identical (31 findings before/after).
- **M-0220** — re-fixtured the heaviest real-tree check integration test to a
  synthetic fixture; full `go test` suite ~93s → ~70s.

Cancelled after measurement (recorded here so they aren't re-litigated):

- **M-0217** — skip redundant pre-push lint: adversarially judged risky for marginal
  win.
- **M-0218** — drive the policies suite below its 9s floor: it runs fully overlapped
  behind the integration package, so optimizing it changes no wall-clock.
- **M-0219** — wire commit-graph maintenance into init/update. **Dropped on a
  workload-scoped measurement** (G-0322 wontfix): it *did* evaluate `--changed-paths`,
  but against `aiwf check`'s full-DAG `--raw` walk — where bloom filters correctly do
  nothing (check reads every path, so nothing is skippable) — and found ~1.5s over the
  base commit-graph. It never measured **single-entity path-scoped reads**, a different
  query shape where the bloom lever is real (class 1 above). Reopening for that shape
  is justified (deferred to **G-0340**); conflating the two workloads is the trap.

Deferred levers carried forward (profiled, unstarted): **G-0323** incremental
`aiwf check` via a validated watermark; **G-0324** branch hygiene; **G-0325**
parallelize independent history walks + blob reads; **G-0327** harden the fsm-history
blob read; **G-0328** golden-fixture byte-identity comparator for `aiwf check`.

---

## The caching invariant (read this before adding any cache)

> **Cache only immutable, content-addressed facts. Key every cache by the git object
> SHA the fact derives from, never by a moving ref (HEAD / a branch / "trunk").**

Then no worktree switch, merge, or rebase can make the cache *wrong* — only
*incomplete*, which costs a recompute, never a wrong answer. This is precisely why
git's own commit-graph is safe across all of it: a commit's tree-diff, generation
number, and bloom filter are functions of an immutable, content-addressed commit
object.

**Why the "trunk watermark" (naive incremental check) is fragile.** A scalar "last
validated trunk position W; only check `W..HEAD`" is *pointer*-addressed and breaks
on exactly aiwf's workflow:

- *Whose trunk?* On an epic-branch worktree, `main` is behind and the branch carries
  off-trunk commits. One shared scalar can't describe 48 branches at different
  positions.
- *The merge is where the watermark is actively dangerous, not just stale.* A merge
  commit is precisely where new violations appear — two branches independently
  allocating the same id is the `ids-unique/trunk-collision` case. A watermark that
  says "these commits were already seen, skip them" can skip validating the
  integration point itself. That is a correctness hole, not just lost speed.

**The salvage: memoize a pure function on an immutable input; never "skip a range."**
Split the checks into two rule classes:

- **Historical rules** whose verdict for commit X depends only on X + its ancestors
  (fsm-history-consistency, provenance-trailered, post-cutoff). Memoize these per SHA
  — a merge can't invalidate them, it only creates new keys to fill.
- **Tree-state rules** (ids-unique-vs-trunk, shape, refs, contracts) that depend on
  the current working tree, already O(tree) not O(history). These **always run
  fresh**, so the merge-collision check is never skipped. Hole closed.

**Where a persistent read-model cache goes and how it's keyed.** "The event
projection as of commit X" is immutable, so cache it under key = X:

- worktree switch → load the entry for that worktree's HEAD SHA; miss → build it. No
  invalidation, ever.
- merge → new commit Y → miss → build incrementally from the first-parent's cached
  projection (union of parents + Y's own trailers). Cheap and correct.
- rebase / reallocate / rewrite → new SHAs → new keys → miss → rebuild; old entries
  orphan and LRU-evict. Never reused wrongly, because the SHA *is* the ancestry hash.

Store SHA-keyed entries in the **common** git dir (`git rev-parse --git-common-dir`),
gitignored, rebuildable from scratch at any time. Because entries are keyed by
immutable SHA, worktrees on branches with shared history *reuse each other's warm
entries* — the shared object store becomes an advantage, not a hazard. Concurrency
uses the repo's mandated write-temp-rename (C3) + a lockfile, same as git.

Which levers touch persistent state at all:

| lever | persists state? | worktree/merge safe? |
|---|---|---|
| commit-graph + bloom filters | git-managed | ✅ SHA-keyed, immutable |
| shared HEAD-walk batching (render); grep guard (history/show) | **no** | ✅ pure recompute |
| path-scoped history queries | **no** | ✅ a query, not a cache |
| read-model projection cache | yes | ✅ **only if** SHA-keyed (above) |
| incremental check "watermark" | yes | ⚠️ only as per-SHA memoization (above) |

The three risk-free levers (batching, path-scoping, bloom filters) deliver most of
the read-path win with *no* persistent state to corrupt. Do them first.

---

## commit-graph + changed-path bloom filters

The repo's base commit-graph originally carried **no bloom filters** — verified by
inspecting the chunk table (`OIDF/OIDL/CDAT/GDA2`, no `BIDX/BDAT`). git's default gc
writes the base graph but **not** bloom filters; you opt in explicitly:

```sh
git commit-graph write --reachable --changed-paths
```

On this repo that write took ~11s once and added the `BIDX/BDAT` chunks, after which
a single-entity path-scoped history dropped from ~1.3s (base graph) to ~65ms (best
case ~14ms). Bloom filters are keyed by commit SHA (immutable) and shared across
worktrees via the common object store, so they are safe by construction — stale only
ever means slower. Verified: git **preserves** existing filters across `gc` and plain
`commit-graph write` but never **creates** them by default, and commits not yet in the
graph still return correct (slower) results — so a fresh clone has none until the
explicit write runs, which is why the maintenance below is net-new.

Maintenance options (pick one; deferred to G-0340): run
`git commit-graph write --reachable --changed-paths` opportunistically after a
mutating verb (aiwf makes one commit per mutation), or configure `git maintenance`
with the changed-paths task and register it in `aiwf init` / `aiwf update`. The base
graph alone (git's default) does **not** enable the path-scoping lever — bloom
filters are the point.

---

## Is "snappy regardless of repo size" achievable?

Largely yes:

- Reads (`history` / `show` / `render`) → O(commits-touching-the-entity) with
  path-scoping + bloom filters, or O(Δ-since-cache) with the SHA-keyed read model —
  both effectively flat for practical repos.
- `check` → O(new commits since the last validated SHA) with per-SHA memoization of
  the historical rules + always-fresh (already cheap) tree rules.
- The honest floor is `tree.Load` (O(entities), pure filesystem, already fast) plus
  one bounded git touch. The one unavoidable cost is a cold-cache / bloom rebuild
  after a large fetch — paid once, and backgroundable.

---

## Recommended sequence (bang-for-buck)

0. **Branch hygiene** (G-0324) — prune the merged local branches. Free; shrinks
   allocator + check fan-out immediately.
1. **Route `render` through a single unified history walk** — ✅ **shipped (E-0054 /
   M-0221).** Built on E-0053's HEAD-scoped `check.WalkHeadCommits` (extended additively
   with `%aI` + `%s`); **not** `gitops.BulkRevwalk`, which walks `--all` (leaks branch
   commits, breaks byte-identity) and collapses repeating trailers. The genuinely new
   code is `render`'s in-memory `historyIndex`: one HEAD pass bucketed into per-entity
   events + the authorize-opener map + a scope-FSM replay, so every resolver read is a
   map lookup. Batches **both** render walk families (per-entity events *and* the
   authorize-opener/provenance map). Measured **~35 min → ~4.5s** (688 pages, ~466×),
   byte-identical (data differential vs the untouched `ReadHistoryChain` /
   `LoadEntityScopeViews` oracles + a real-tree `diff -rq`). Biggest single render win.
1b. **Guard the unconditional authorize grep across the read verbs** — ✅ **shipped
   (E-0054 / M-0223).** The same grep ran unconditionally in `history` text and `show`
   (`~3.4s`); it now runs only when the entity's loaded events carry scope data, and the
   two near-duplicate private impls were consolidated into one shared helper,
   `cliutil.AuthorizeOpeners` (reused by the M-0221 render pass — no third copy).
   Measured **~44%** off `aiwf history` and **~32%** off `aiwf show` on a scopeless
   entity; low risk (verified), gated on a non-vacuous fixture.
2. **Path-scoped history + maintained changed-path bloom filters** (deferred, G-0340)
   — per-entity history ~1.3s → ~65ms, but only once the query-equivalence gaps
   (pathless commits, path-set derivation, history simplification) are handled; the
   trailer grep stays the oracle.
3. **SHA-keyed read-model cache** + **per-SHA incremental check** (G-0323, reframed
   as memoization) — the "flat regardless of size" tier; larger effort; safe only
   under the caching invariant above.
4. **Slim the allocator's ref fan-out** — per-ref-SHA `ls-tree` cache or an id
   high-water-mark with the ref scan as a collision backstop only.

Do 0–1 first and re-measure; they may shrink how much of tier 3 is worth building
(YAGNI).

---

## How to measure (so future work stays honest)

- **Attribute subprocesses, don't guess.** E-0053's key finding (subprocess-bound,
  not CPU-bound) came from `strace -f -c` counting `waitid`/`clone`. A CPU profile
  alone would have mis-pointed at Go code.
- **Prove behavior-identical when optimizing a read path.** Byte-diff the rendered
  output (or the JSON envelope) before/after — E-0053/M-0216 pinned "31 findings
  before/after, byte-identical." An optimization that changes output is a bug, not a
  win.
- **Measure the right endpoint for cached upstreams.** The base commit-graph "win"
  and the bloom-filter win are *different mechanisms*; measuring one says nothing
  about the other (the M-0219 trap). Name which mechanism you're measuring.
- **Build a worktree-scoped binary** (`make diag-aiwf`) when diagnosing against
  uncommitted kernel source — the PATH `aiwf` was built from stale code (see
  CLAUDE.md "Worktree binary discipline").
