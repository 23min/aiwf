---
id: G-0372
title: aiwf check's full-history git revwalks run unconditionally on every push
status: open
priority: medium
---
## What's missing

`aiwf check`'s git-history-dependent rules (`fsm-history-consistent` via
`gitops.BulkRevwalk`, `check.WalkHeadCommits`, `orphan_dag`) walk the
entire reachable commit history from scratch on every invocation, with no
notion of "already verified as of the last check." That history-scaling
term is what this gap tracks. It is one of three independent terms in
`aiwf check`'s total cost, and no longer the largest (measured 2026-07-18,
this repo: ~860 entities, ~7,650 commits):

1. **Environment — controlled, not code.** The devcontainer bind mount
   multiplies every git object access ~7× (identical repo: ~56s on the
   mount vs ~7.7s on native fs, before maintenance). Object-store health
   compounds it: with `maintenance.auto` off, ~38k loose objects had
   accumulated; `git repack -ad` plus a fresh commit-graph with
   changed-path Bloom filters cut the check from ~7.7s to ~3.6s native
   and ~56s to ~15–20s on the mount, findings byte-identical.
   `maintenance.auto=true` is now set, so aiwf's one-commit-per-mutation
   churn can no longer silently re-bloat the store. The mount multiplier
   itself is a workstation/devcontainer-layout decision, out of aiwf's
   hands.
2. **The cross-branch collision scan** — work proportional to
   entities × refs, discarded for every locally-present id. E-0067's
   lazy-scan epic addresses it; not this gap.
3. **The history-scaling walk itself — this gap.** Post-maintenance on
   native fs, the residual is ~3s: `BulkRevwalk`'s git side is ~0.26s;
   the dominant remainder is the FSM tier's ~6,900 `git cat-file --batch`
   blob reads (one per distinct entity-file blob observed across history)
   issued as strictly serial request-response round-trips
   (`internal/gitops/catfile.go`), plus the Go-side parse of every
   commit record, on a rule pipeline that runs its independent walks
   strictly serially.

Mechanisms for the structural fix, in rising order of ambition:

- **A persistent blob-SHA → parsed-frontmatter-status cache.** The FSM
  walker's per-blob status reads are a pure function of content-addressed,
  immutable blob content. Caching below the observation layer sidesteps
  the acknowledgment constraint entirely — verdicts are still recomputed
  fresh each run from observations plus the HEAD-reachable acknowledgment
  set — so a stale entry is dead weight, never a wrong finding. No
  watermarks, no merge-base reconciliation, no reachability filtering.
- **Pipelining the `BlobReader` protocol** — batch the ~7k requests
  instead of one write/read round-trip per blob.
- **Parallelizing the independent rule walks** (`BulkRevwalk`, the HEAD
  walk, orphan-dag, the cross-branch scan) — they are independent until
  findings assembly and currently run one after another.
- **Per-commit observation caching with ref watermarks** — the full
  E-0058-class design, warranted only if the smaller mechanisms prove
  insufficient. Any design that memoizes *verdicts* (rather than
  content-pure observations) must satisfy the correctness constraint
  recorded in
  `docs/initiatives/check-performance-incremental-revwalk-cache.md`:
  `fsm-history-consistent`'s verdict depends on the HEAD-relative
  acknowledgment set, not just commit content. Two prior designs (E-0058
  and a ref-tip-watermark alternative) were set aside on exactly that
  ground and on complexity; the initiative doc preserves both
  post-mortems and the concrete counterexample.

## Why it matters

This gap is the concrete blocker on the branch/id-allocation strategy
discussion in `docs/initiatives/id-lifecycle.md` (the EMB — "ephemeral
mutation branch" — proposal, and G-0281's gaps-inbox side channel): both
assume pushing is cheap enough to retry on contention. The operational
fixes above plus E-0067 close most of the distance, but the remaining
check cost still scales with total repository history, which only grows
(~1,400 commits landed in the first half of July alone). The structural
property this gap asks for — check cost proportional to what changed
since the last check, not to how much history exists — remains unbuilt;
the blob-SHA status cache above is the smallest mechanism that delivers
a large share of it while respecting the acknowledgment constraint that
defeated the two prior designs.
