---
title: 'aiwf check performance: an immutable, per-commit-sha cache for the full-history revwalks'
status: captured
date: 2026-07-05
---

# aiwf check performance: an immutable, per-commit-sha cache for the full-history revwalks

## Classifier note

This is an initiative document, following the precedent of
[`id-lifecycle.md`](id-lifecycle.md) and
[`agent-agnostic-execution-topology.md`](agent-agnostic-execution-topology.md):
`initiative` is not yet an official aiwf entity kind, so this lives under
`docs/initiatives/` as a capture point for a design that's concrete enough to
scope but not yet built. Unlike `id-lifecycle.md` (three unreconciled designs,
deliberately unresolved), this document proposes one specific mechanism,
backed by measurement and a correctness-verified prototype — closer to a
pre-ADR design spec than an open survey.

## Why this exists

This surfaced as a **prerequisite**, not a tangent, to the id-lifecycle /
branch-strategy discussion in [`id-lifecycle.md`](id-lifecycle.md) (the EMB —
"ephemeral mutation branch" — proposal, and G-0281's opt-in gaps-inbox side
channel). Both of those designs assume pushing is cheap enough to retry on
contention. It isn't, on this repo, today: `aiwf check` costs **~22 seconds,
unconditionally, on every single push**, regardless of whether the push is
one gap filing or a hundred-commit epic wrap. No branch/id-allocation
strategy is worth adopting until that's addressed — a design that assumes
frequent, cheap pushes but pays a flat 22s tax on each one just relocates
where the pain shows up.

## Problem, quantified

Measured directly on this repo (719 entity files, 6,213 commits reachable
from `--all`, 43 refs) via `./bin/aiwf-diag check` (a worktree-scoped build,
per this repo's own binary-discipline convention) and cross-checked with
`strace`-free direct timing (strace itself was tried first and inflated
absolute numbers ~5-7x via ptrace overhead on syscall-heavy git subprocesses;
relative proportions held, absolute numbers below are all strace-free):

| Component | Real cost | Source |
|---|---|---|
| `gitops.BulkRevwalk` (`fsm-history-consistent`, `git log --all --raw --no-abbrev -M -m`) | 12.2s | `internal/gitops/revwalk.go:144`, sole caller `internal/check/fsm_history_walker.go:136` |
| ↳ of which, `-m` per-parent merge-diff fan-out | 3.8s | see "Finding 1" below |
| `check.WalkHeadCommits` (`git log --reverse HEAD`, full metadata) | 2.4s | `internal/check/head_history.go:105` |
| `area_mistag.WalkAcknowledgedMistags` — independent `git log ... HEAD` walk | 1.7s | `internal/check/area_mistag.go:159`, see "Finding 2" below |
| `orphan_dag`'s `git rev-list --all --reflog --parents` | 1.2s | `internal/check/orphan_dag.go:34` — genuinely distinct (needs reflog data), not a folding candidate |
| Long tail (per-ack `rev-parse`, `ls-tree` ×4, reflog `log -1` ×20, misc) | ~3-4s | not fully attributed; diminishing returns to chase further |
| **Total** | **~22s** | |

### Finding 1 — `-m`'s merge fan-out is provably discarded

`-m` forces `git log --raw` to emit a separate diff record per parent for
every merge commit (193 merges in this repo's reachable history). All three
known consumers of `BulkRevwalk`'s output discard merge-commit observations
unconditionally, as their first line:

```go
// internal/check/fsm_history_consistent.go
func illegalTransitionFindings(...) { if o.IsMergeCommit { continue } ... }  // line 361
func manualEditFindings(...)        { if o.IsMergeCommit { continue } ... }  // line 476
func forcedUntraileredFindings(...) { if o.IsMergeCommit { continue } ... }  // line 689
```

Measured: dropping `-m` cuts this call from 12.2s to 8.4s (saves 3.8s, ~17%
of the total 22s budget). A scoped, low-risk fix, independent of everything
else in this document — pending one more pass confirming no other consumer
of `BulkRevwalk`'s output needs the per-parent expansion.

### Finding 2 — a fourth, avoidable full-HEAD walk

`internal/check/area_mistag.go`'s `WalkAcknowledgedMistags` runs its own,
fully independent `git log --pretty=... HEAD` walk (measured 1.7s),
extracting SHA + trailers — a strict subset of what `check.WalkHeadCommits`
already computed moments earlier in the same `aiwf check` invocation
(M-0216/AC-5 explicitly consolidated five *other* rules onto that shared
`head []HeadCommit` slice; this one was missed). Fixable by threading `head`
through instead of re-walking — eliminates the call entirely, no design
change needed.

### The deeper pattern

Findings 1 and 2 are worth ~5.5s (~25%) combined, with no architectural
change — but they don't address the structural cause: **every rule that
walks git history walks the *entire* reachable history from scratch, on
every invocation, with no notion of "already verified as of last time."**
`BulkRevwalk` and `WalkHeadCommits` are themselves *already* the product of
real consolidation work (E-0053/M-0216 collapsed ~3,000 and 5 per-entity
subprocess fan-outs, respectively, into one call each) — this proposal is
the next turn of the same crank: collapsing "walk everything, every time"
into "walk everything once, then walk only what's new."

## The proposed mechanism

### The core insight: git commits are immutable, so this cache is exact, not approximate

A commit's diff against its parent(s) never changes once committed — content
and parent set are fixed at creation. That means a commit's *derived*
observations (its `statusChange` records, in `fsm_history_walker.go`'s
terms) are valid **forever**, for as long as that commit stays reachable.
This isn't a heuristic cache like "skip files that look unchanged" (which
can be wrong) — it's an exact memoization keyed by a content-addressed,
immutable identity. The only thing that can make a cached entry wrong is the
commit becoming *unreachable* (handled by filtering against current
reachability, not by recomputing), or aiwf's own extraction logic changing
(handled by a schema version stamp).

### Design

- **Cache contents:** commit sha → its derived observations (the
  `statusChange` records `batchedWalkStatusChanges` currently produces
  in-memory and discards after each run).
- **Storage location: shared across worktrees, not worktree-scoped.** The
  golangci-lint cache's precedent (worktree-scoped, under
  `.git/worktrees/<name>/`) doesn't transfer here: that cache is
  worktree-scoped specifically because it stores *absolute paths* baked in
  from whichever checkout produced it — a hazard with no equivalent in
  this design. This cache is `commit-sha → observations`, and observations
  carry only *relative* repo paths; a commit's content is identical
  regardless of which worktree walks it. So the cache lives once, in the
  shared `.git/` directory (not per-worktree), and every worktree benefits
  from every other worktree's walks — a
  meaningfully bigger win than the original per-worktree scoping, given
  this repo routinely runs many concurrent epic/milestone worktrees.
- **Concurrent writers: atomic rename, no locking.** Two worktrees'
  `aiwf check` runs finishing near-simultaneously could both try to update
  the shared cache file. Resolution: read, compute, write via a sibling
  temp file then atomic rename (this repo's own code-health atomic-write
  principle) — no lock needed. Neither writer's result is wrong; the
  "loser" of the rename race just means the next check re-walks a handful
  of commits that were already cached elsewhere. No correctness cost, only
  a trivial, self-correcting efficiency cost.
- **Per-ref watermarks, reconciled via `merge-base`, in one combined
  subprocess call.** Store each ref's last-seen tip sha. On each check,
  compute `merge-base(stored_tip, current_tip)` per ref — **not** the raw
  stored tip — and issue one single `git log --all ^<merge-base_1>
  ^<merge-base_2> ... <current_tip_1> <current_tip_2> ...` covering every
  ref at once (matching this repo's own "one bulk call, not a per-ref
  fan-out" instinct from E-0053/M-0216, rather than N per-ref calls). Using
  merge-base instead of the raw stored tip is what makes one rule cover
  both cases without a special-case branch:
  - **Ordinary fast-forward:** merge-base equals the stored tip, so the
    walk behaves exactly as a naive `stored..current` would.
  - **Force-push / rewrite:** merge-base finds the actual common ancestor,
    so the walk correctly re-covers only the genuinely-new commits on the
    rewritten history — no separate "is this ref rewound" branch, no risk
    of over- or under-excluding.
  - **New ref** (no stored watermark): nothing to exclude for it; its tip
    is simply one of the positive refs in the same combined call, and it
    still hits the cache for any commits it shares with already-walked
    refs (e.g., a feature branch cut from recently-checked main pays only
    for its own new commits).
  - **Deleted ref** (e.g., a merged-and-removed epic branch): drop its
    entry from the watermark map — no special eviction logic needed, since
    the reachability-filtering step below already excludes its commits'
    observations if they're no longer reachable from anywhere.
- **Reachability filtering:** each check still computes the current
  reachable set (`git rev-list --all`, ~1.2s, already cheap and already
  measured) and filters the cache down to it — this is what makes a
  rewound-away commit's stale cached observations silently drop out, with
  no explicit eviction logic needed.
- **Schema/logic version stamp:** a version constant bumped whenever the
  observation-extraction code changes, forcing one full rebuild — the
  cache is invalidated by *either* a git-history change *or* an
  aiwf-logic change, not just the former.
- **Fail-safe posture:** any cache read/parse/version-mismatch error →
  treat as absent, do a full walk. Matches this repo's existing instinct
  elsewhere (the pre-push hook's own gitleaks "conservative fallback:
  scan_all=1" when a range isn't locally resolvable) — never let cache
  corruption or ambiguity silently produce an incomplete finding set.

### Generalizing: one primitive, not two bespoke caches

`check.WalkHeadCommits` (~2.4s) has the identical shape to `BulkRevwalk`
(~12.2s) — a full walk from scratch, every check, over immutable,
content-addressed history. The watermark/reconciliation/reachability/
versioning machinery above is not specific to `statusChange` records; it's
generic over "what to extract per commit." Building it as one shared
primitive, parameterized by the per-commit extraction the caller needs,
fixes *both* cost centers (~14.6s of the ~22s total) for one engineering
investment, rather than a bespoke cache for `BulkRevwalk` alone with
`WalkHeadCommits` left as a second, later, separately-designed effort.

## Prototype evidence

A throwaway, read-only bash prototype (`incremental_revwalk_prototype.sh`,
run from `/tmp`, deleted after — no production code touched, nothing
committed, no writes to this repo) validated the core mechanism directly
against this repo's own history, deliberately scoped to `HEAD` only (not
`--all`) since multi-ref reconciliation is a separate, tractable
implementation concern the per-commit-cache argument doesn't depend on:

1. **"Cached baseline" walk** — everything up to a simulated watermark
   25 commits back from HEAD (`git log HEAD~25 --raw --no-abbrev -M -m
   ...`): **9.38s**, 6,090 commits.
2. **Incremental walk** — only commits new since that watermark
   (`git log HEAD~25..HEAD --raw --no-abbrev -M -m ...`): **0.28s**, 107
   commits (more than 25 because merges in that range bring in
   already-merged-elsewhere history — an expected, real effect, not an
   error).
3. **Fresh full walk** (ground truth — what production pays today, on
   every single invocation): **9.46s**, 6,197 commits.
4. **Correctness check:** the sha set from (1) unioned with the sha set
   from (2) was compared against the sha set from (3) — **byte-identical,
   PASS.** No commit missed, none double-counted.

Result: a **~33x speedup** (9.46s → 0.28s) for a realistic "a handful of
commits landed since the last check" increment, with the union-equals-fresh
property formally verified, not assumed. The one-time cost of the initial
"cached baseline" walk (~9.4s here) is paid once, ever, per cache lifetime —
after that, steady-state cost is proportional to *new* commits since the
last check, not total repository history, which only grows over time under
the current design.

### Honest caveat, from an earlier, noisier measurement

An earlier ad hoc timing (last-1000-commits range: 13.4s, *more* than the
full 6,213-commit walk's 12.2s) showed the relationship isn't perfectly
linear — `-M` rename-detection cost tracks how many files a commit touches,
not just commit count, so a single large commit (an archive sweep touching
hundreds of files, a rewidth migration) can cost disproportionately
regardless of recency. This doesn't undermine the design: because the cache
is keyed per-commit-sha with the watermark always advancing past whatever
was just walked, an expensive one-off commit is paid for **exactly once**
across the cache's entire lifetime, then amortizes to zero forever after —
it just means the *first* check that happens to encounter it (post-cache
build, or after a rebuild) may spike, not every subsequent one.

## Relationship to the id-lifecycle initiative

This is the named prerequisite from `id-lifecycle.md`'s EMB discussion: any
design assuming "push often, retry on contention is cheap" (EMB, G-0281, or
even plain tight push-cadence discipline) needs this fix — or an equivalent
— to actually deliver on that assumption at this repo's current and future
scale. It does not resolve which of E-0052 / ADR-0001 / G-0281 / EMB should
be adopted; it removes a shared cost that makes evaluating them under real
conditions (rather than an idealized "pushes are free" assumption) possible.

## Before this is trusted: the testing bar

This cache sits underneath `fsm-history-consistent` — the one rule that
guarantees this repo's entity-status history was never illegally mutated.
A silent false negative here is exactly the class of failure this repo is
built to prevent, so "thoroughly and completely" is the bar, not "the
happy path passes":

- **Property/generative tests, not example-based tests alone**, asserting
  the one invariant this whole design rests on: *incremental result
  (cached ∪ newly-walked, reachability-filtered) always equals a fresh
  full walk*, across synthetic git histories covering every scenario known
  to carry risk:
  - ordinary fast-forward (the common case)
  - force-push / history rewrite on one ref
  - a brand-new ref with no prior watermark
  - a deleted ref whose commits become unreachable
  - merge commits (with and without `-m`, once Finding 1 lands)
  - a ref rewound to a point *before* the last stored watermark, then
    fast-forwarded past it again (oscillation, not just one-directional
    rewrite)
  - multiple of the above compounding in one check invocation (e.g., one
    ref force-pushed while another is brand new)
- **Concurrent-writer tests, not just reasoned-about safety.** Actually
  spawn multiple processes writing the shared cache at once against a
  synthetic repo and assert: no corruption, no lost correctness (a
  "losing" writer's redundant re-walk next time is acceptable; a wrong
  finding is not), the atomic-rename behavior holds under real contention,
  not just under the sequential reasoning above.
- **Fail-safe/fallback tests**: a corrupted cache file, a version-mismatched
  cache, a cache referencing a sha the object store no longer has (partial
  clone / GC'd) — each must fall back to a full walk, not error out or
  silently under-report.
- **Full branch-coverage discipline** on every conditional this introduces
  (fast-forward vs. rewrite vs. new vs. deleted ref; version-match vs.
  mismatch; corrupt vs. valid cache; race won vs. lost) — this repo's own
  hard rule, not a suggestion, and especially not optional for code
  underneath a provenance-integrity guarantee.

## Formal methods — sighting loom, not drawing on it yet

Too early to bring formal verification into this design — the mechanism
above hasn't been built, let alone stabilized, and per `id-lifecycle.md`'s
own "Formal methods fit" section, TLA+/Dafny-class tools earn their keep
once a protocol is settled enough to be worth exhaustively checking, not
while it's still being shaped. Naming it now anyway, since `id-lifecycle.md`
already did the legwork of assessing `loom` (github.com/23min/loom) as a
live, usable-today tool, not a future one: if this cache's watermark/
reconciliation protocol (`Ref`, `stored`/`current` tips, `merge-base`-based
exclusion, the reachability filter) ever gets formalized, it's a strong
candidate for the same `knows`/`relates`/`proves` umbrella treatment
`id-lifecycle.md` recommends for the entity-id protocol — plausibly the
*same* umbrella, since both protocols share the `Ref`/`ConfirmAgainstRef`-
shaped vocabulary already spelled out there. Not a decision to make now;
a marker so it isn't rediscovered from scratch later.

## Open questions

- **Cache storage format** — a flat sha-keyed file is almost certainly
  adequate at today's scale (6,213 commits); worth confirming it stays
  adequate as history keeps growing, rather than assuming indefinitely.
- **Interaction with `aiwf check --fast`.** `--fast` already exists and
  already skips this entire tier (sub-second, used today for the
  statusline glyph and CI pre-flight) but isn't wired into the pre-push
  hook. Wiring `--fast` into pre-push is a free, available-today, *interim*
  option — but a real trade, not a neutral one: it defers FSM-history /
  provenance / orphan-dag findings to CI-only, and this repo has direct,
  lived precedent (G-0179) for that failure shape — golangci-lint was
  once CI-only, and debt piled up invisibly across three milestone wraps
  before G-0179 added the local gate. This document's proposed cache is
  the fix that avoids that trade entirely, by making the local, blocking,
  every-push check itself cheap rather than removing it.
- **Sequencing: after the quick fixes land and their real impact is
  measured, not before.** Findings 1 and 2 (a `wf-patch`, in progress) get
  `aiwf check` from ~22s to roughly ~16.5s — real, but a stopgap: the
  underlying "cost scales with total history" problem doesn't go away at
  16.5s, it only buys time, and history only grows (986 `aiwf add` events
  in this repo's first 2.3 months alone). If 16.5s is still felt as painful
  once measured for real, that's the evidence-based trigger to scope this
  as an epic promptly — with the refinements in this document (shared
  cache, `merge-base`-based reconciliation, the unified primitive, the
  testing bar above) already folded in rather than rediscovered mid-build.

## Desired future property

`aiwf check`'s cost on this repo should scale with **how much changed since
the last check**, not with **how much history exists** — the same property
`git status`, `git diff`, and (already) the golangci-lint pre-push gate all
have. A repository with ten years of history and a one-line pending change
should check about as fast as a repository with ten commits and the same
one-line change.

## Provenance

Emerged from a design conversation (2026-07-05) that started by pressure-
testing the EMB branch-strategy proposal in `id-lifecycle.md`, surfaced that
pushing isn't cheap on this repo today, and traced the cost to its actual
source via direct measurement (`strace`, targeted `git` timing, and source
reading) rather than assumption. The two scoped fixes (Finding 1, Finding 2)
and the cache proposal all came out of that same investigation. The storage,
reconciliation, generalization, and testing-bar refinements (shared not
worktree-scoped cache, `merge-base`-based per-ref exclusion in one combined
call, the unified `BulkRevwalk`/`WalkHeadCommits` primitive, the property/
concurrent-writer/fail-safe testing requirements, and sighting `loom`) were
added in a follow-on pass the same day, before any of this was built —
tightening the design while it was still cheap to change, ahead of the
`wf-patch` for Findings 1/2 that's scoped to land first.
