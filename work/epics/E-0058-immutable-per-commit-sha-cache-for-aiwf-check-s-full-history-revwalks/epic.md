---
id: E-0058
title: Immutable per-commit-sha cache for aiwf check's full-history revwalks
status: proposed
---

# E-0058 — Immutable per-commit-sha cache for aiwf check's full-history revwalks

## Goal

Make `aiwf check`'s git-history-dependent rules cost proportional to how much
changed since the last check, not to total repository history — without
weakening the correctness guarantee those rules currently provide.

## Context

`aiwf check`'s history-walking rules (`gitops.BulkRevwalk`, `check.WalkHeadCommits`,
the area-mistag walker, `orphan_dag`) each walk the *entire* reachable commit
history from scratch on every invocation. G-0372 measured this at ~22s
unconditionally on every push on this repo, growing with history size regardless
of how small the actual change was. Two independently-verified-safe reductions
already shipped as a `wf-patch` against G-0372 (dropping a dead `-m` fan-out;
folding a duplicate HEAD walk into the shared pass) — G-0372 stays open because
those fixes reduce the cost without addressing its root cause: every rule still
walks from scratch, every time.

This epic is that root-cause fix, captured in full design detail in
`docs/initiatives/check-performance-incremental-revwalk-cache.md`. The core
insight: git commits are immutable, so a commit's derived observations (the
`statusChange` records `BulkRevwalk`'s consumers compute) are valid forever, for
as long as the commit stays reachable — an exact memoization, not a heuristic
one. A throwaway, read-only prototype already validated the core claim directly
against this repo's own history: a cached-baseline walk plus an incremental walk
from a simulated watermark, unioned, reproduced a fresh full walk's SHA set
byte-identically — a ~33x speedup for the realistic "a handful of commits landed
since the last check" case.

This cache sits directly underneath `fsm-history-consistent` — the rule that
guarantees this repo's entity-status history was never illegally mutated. A
silent false negative here is exactly the failure class this repo is built to
prevent, so this epic is held to a materially higher evidence bar than most:
generative/property tests proving the cache-equals-fresh-walk invariant across a
named scenario list, real concurrent-writer tests (not reasoned-about safety),
and fail-safe tests for every corruption/version-mismatch shape — not just
branch coverage.

This is also an explicit prerequisite for G-0281 (E-0045's plumbing epic's
second milestone, an opt-in gaps-inbox whose design assumes "pushing is cheap
enough to retry on contention") — but this epic is independent of, and not part
of, that epic. It removes a shared cost that makes evaluating G-0281 (and any
similar branch-strategy proposal in `docs/initiatives/id-lifecycle.md`) under
real conditions possible; it does not decide whether those proposals should be
adopted.

**Risk framing.** This epic is deliberately high-risk, high-reward, and is
structured so that risk is discovered cheaply, at the very first milestone,
before any of the harder (and more expensive) multi-ref, concurrency, or
fail-safe work is attempted. It is possible the core memoization invariant does
not hold under some scenario that has no clean workaround — see *Success
criteria* below for why that is a legitimate, valued closing shape for this
epic, not a failure of it. Independent of whether the cache ships, doing this
milestone's evidence work well — generative testing, mutation testing, and
concurrent-writer testing applied together against a single correctness-critical
caching layer — is a rigor exercise this codebase has not attempted at this
level before, and is worth doing carefully for its own sake.

## Design: the reconciliation mechanisms, and how they interact

A scenario list assembled by intuition, rather than derived from an explicitly
enumerated state space, is exactly how a real correctness gap hides. This
section exists so that doesn't happen here: it names the actual mechanisms
this design rests on, checks one of them for completeness on paper, and names
the interactions between them explicitly rather than leaving them implicit.

**1. The per-ref watermark reconciliation FSM.** For a single ref, at check
time:

| has a stored watermark? | is it an ancestor of the current tip? | action |
|---|---|---|
| No | n/a | full walk of this ref |
| Yes | Yes | incremental walk (`stored..current`) |
| Yes | No | full walk of this ref (fallback) |

Three reachable states, not four — the second column only applies
conditional on the first being true. Checked against the edge cases rather
than assumed exhaustive: `stored == current` collapses into the incremental
state with an empty range (a no-op, not a 4th state); a hard reset backward
is indistinguishable from a force-push at this predicate's level and is
handled the same safe way; cache corruption or a version mismatch collapses
into the "no watermark" state per the fail-safe constraint, rather than
needing its own state; a ref deleted and recreated under the same name is
likewise indistinguishable from force-push here — safe, if conservative.

This FSM is *local*: it only reasons about one ref against its own stored
state. Four other mechanisms interact with it, and none of them are states
of this FSM:

**2. Per-commit cache lookup (cross-ref dedup).** Whichever range the FSM
above hands to `git log`, every commit encountered in that range still gets
an individual cache-hit-or-miss check — the SHA-keyed cache is shared across
every ref, so a commit newly in-range for one ref may already be cached from
another ref's earlier processing (in this check or a prior one). This is the
mechanism that actually delivers cross-ref deduplication, and it means the
real work done for one ref depends on what other refs have already
contributed: the FSM's chosen range only bounds the *query* cost (how much
`git log` enumerates), not the *derivation* cost (how many commits actually
get freshly computed).

**3. The global reachability filter.** Runs once per check, across `--all`,
independent of any single ref's FSM state — decides which cached entries
survive based on current global reachability. This interacts with (1): if a
ref's watermark regresses (the fallback state, e.g. after a force-push or
hard reset), commits between the old and new watermark may become globally
unreachable — if nothing else holds them — and get evicted here, even though
they were legitimately cached under a now-superseded watermark. A ref that
later oscillates back past those commits does not get a cache hit for
them — not a correctness bug (redundant work, not a wrong answer), but a
real, non-obvious coupling between (1) and (3). The "watermark-oscillation"
scenario must test this as a *trajectory* through both mechanisms across
successive checks, not as a single FSM transition.

**4. The concurrent-writer protocol.** Multiple processes running (1)–(3)
concurrently against the same shared on-disk file. The stated tolerance — a
losing writer's redundant re-walk is acceptable, a wrong finding is not — is
a claim about the *interaction* of all three mechanisms under concurrent
execution, not about any one of them in isolation. Ref-processing order must
be proven order-independent (two processes racing on the same repo, or one
process working refs in a different order than another, must converge to
the same final cache content) — a property to test, not to assume.

**5. The version-stamp fail-safe override.** A global kill-switch: a schema-
version mismatch collapses all of (1)–(3) back to their empty/initial state,
for every ref, simultaneously. Under (4), this raises a question that must be
tested, not assumed: can a concurrent writer ever observe a *torn* state —
part of the cache reflecting one version, part another — mid-race? The
answer must be no, and that is a property test, not a reasoned-about
assumption.

**What this implies for the milestones below:** the single-ref milestone's
property-test suite must cover mechanism (1) *and* its interaction with (3)
for one ref, not (1) in isolation. The multi-ref milestone must prove
mechanism (2)'s cross-ref deduplication is order-independent, not merely
present. The concurrent-writer milestone must test (4)'s interaction with
(5) — a version mismatch appearing mid-race — not only plain concurrent
writes of a single, agreed-on cache version. The scenario list in *Scope*
below is retained, but is understood as exercising these five mechanisms and
their named interactions, not as a flat, intuited list.

## Scope

### In scope

- A SHA-keyed, per-ref-watermarked cache of `BulkRevwalk`'s derived
  per-commit observations, stored worktree-scoped under `.git/...` (mirroring
  the existing golangci-lint cache precedent).
- Proof, via generative/property tests, that the cache is exact: unioning
  cached and incrementally-walked observations, filtered to current
  reachability, always equals a fresh full walk — across fast-forward,
  force-push/rewrite, new-ref, deleted-ref, merge-commit, and watermark-
  oscillation scenarios, first for a single ref (`HEAD`) and then generalized
  to every ref `--all` reaches.
- Concurrent-writer safety: multiple processes/worktrees writing the shared
  cache at once must never corrupt it or produce a wrong finding (a redundant
  re-walk by a "losing" writer is an acceptable cost).
- Fail-safe behavior: any cache read/parse/version-mismatch/missing-blob
  anomaly falls back to a full walk — never errors out, never silently
  under-reports.
- Production cutover: `BulkRevwalk`'s consumers derive from the cache instead
  of an unconditional full walk, with a standing golden-fixture byte-identity
  guard (closing G-0328) proving `aiwf check --format=json` output is
  unchanged by the switch.

### Out of scope

- Multi-ref reconciliation, concurrency, and fail-safe work for
  `check.WalkHeadCommits`, the area-mistag walker, or `orphan_dag` — this
  epic's cache covers `BulkRevwalk`'s consumers only (the largest single cost
  per G-0372's measurement); the other rules are candidates for a follow-on
  epic once this one's mechanism is proven and shipped.
- Deciding between the branch-strategy proposals this epic is a prerequisite
  for (E-0052 / ADR-0001 / G-0281 / the EMB proposal in
  `docs/initiatives/id-lifecycle.md`) — this epic removes a shared cost;
  it does not choose among them.
- Formal verification (TLA+/Dafny/`loom`-class tooling) of the cache's
  watermark/reconciliation protocol. Per the initiative doc's own assessment,
  that's a strong candidate once the protocol is settled and stable, not
  while it's still being built — naming it as a future option, not
  attempting it now.
- Wiring `aiwf check --fast` into the pre-push hook as an interim mitigation
  — a real, already-available trade (skips FSM-history/provenance/orphan-dag
  entirely) with lived precedent for going badly (G-0179); this epic is the
  fix that avoids needing that trade, not a reason to take it meanwhile.

## Constraints

- The cache must be exact, never approximate: any state it cannot prove
  correct for (a corrupted file, a version mismatch, a reachability
  ambiguity) must fall back to a full walk rather than serve a possibly-wrong
  answer.
- No change to `aiwf check`'s reported findings for any given tree state —
  the cache is a performance mechanism, not a behavior change. The M5 golden-
  fixture guard is the standing mechanical proof of this.
- Worktree-scoped storage only (no cross-worktree or cross-machine cache
  sharing) — matches this repo's own golangci-lint-cache precedent and sidesteps
  a whole class of staleness questions a shared cache would raise.
- `tdd: required` on every milestone in this epic — the property-test /
  mutation-test / concurrent-writer-test evidence bar this epic sets for
  itself is exactly what `tdd: required` exists to enforce mechanically.

## Success criteria

This epic has two legitimate closing shapes, both valued:

- [ ] **(a) Shipped:** every milestone listed below is `done`; `BulkRevwalk`'s
      consumers derive from the cache in production; `aiwf check`'s wall-clock
      cost on this repo scales with commits-since-last-check, not total
      history size, with the golden-fixture guard (G-0328) proving reported
      findings are unchanged; the property-test suite proving cache-equals-
      fresh-walk passes across every scenario named in *Scope* above.
- [ ] **(b) Cancelled with a proven negative result:** the first milestone's
      property-test suite (or a scenario discovered during it) demonstrates
      the core memoization invariant does not hold and has no clean
      workaround; the epic is promoted to `cancelled` with the disproof and
      the scenario that broke it written up in the epic body or a linked
      gap, so the negative result is preserved rather than silently
      abandoned.

Reaching *either* (a) or (b) closes this epic successfully; stalling in
neither shape (an abandoned epic with no recorded resolution) is the only
actual failure mode.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Does the core per-ref memoization invariant hold across every named scenario, including compounding cases? | Yes — gates every milestone after the first | First milestone's property-test suite; a failure here triggers the (b) cancellation path above |
| What exact on-disk cache format (flat file vs. other) stays adequate as history keeps growing? | No | Decided during the first milestone; revisited if a later milestone's concurrency/fail-safe work finds it inadequate |
| Should `check.WalkHeadCommits` / the mistag walker / `orphan_dag` eventually share this cache mechanism? | No | Deferred to a follow-on epic once this one ships (see *Out of scope*) |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| The core memoization invariant doesn't hold under some scenario with no clean workaround | High — the whole epic's premise | First milestone is the explicit, cheap go/no-go checkpoint before any multi-ref/concurrency/fail-safe investment; see *Success criteria* (b) |
| A subtle cache bug silently produces a wrong `fsm-history-consistent` finding | High — undermines the provenance guarantee this repo exists to enforce | Property tests, mutation testing, and the golden-fixture byte-identity guard (G-0328) as standing mechanical evidence, not a one-time check |
| Concurrent-writer corruption in a shared worktree | Medium | Dedicated milestone with real multi-process tests against a synthetic repo, not reasoned-about safety |
| Scope creep into the other history-walking rules (`WalkHeadCommits`, mistag walker, `orphan_dag`) | Medium — epic never converges | Explicitly out of scope; named as a follow-on epic candidate |

## Milestones

- Single-ref (`HEAD`) core cache, proven exact via generative/property tests
  across fast-forward, force-push/rewrite, new-ref, deleted-ref, merge-commit,
  and watermark-oscillation scenarios. Not wired into production yet — the
  epic's go/no-go checkpoint. · depends on: —
- Multi-ref (`--all`) reconciliation: extends the proven single-ref algorithm
  to every reachable ref, with cross-ref dedup by SHA and force-push/rewrite
  invalidation scoped to the affected ref; property tests extend to
  compounding multi-ref scenarios. · depends on: the single-ref core milestone
- Concurrent-writer safety: real multi-process tests against a synthetic
  repo proving no corruption and no wrong finding under concurrent cache
  writes. · depends on: the multi-ref reconciliation milestone
- Fail-safe / corruption handling: every cache anomaly (corrupted file,
  version mismatch, missing blob) falls back to a full walk; full branch
  coverage on every new conditional. · depends on: the multi-ref
  reconciliation milestone (independent of the concurrent-writer milestone —
  either order, or parallel work, is fine)
- Production cutover: wires the proven cache into `BulkRevwalk`'s consumers,
  replacing the unconditional full walk; closes G-0328 via a golden-fixture
  byte-identity guard proving `aiwf check --format=json` output is unchanged.
  · depends on: the concurrent-writer safety and fail-safe milestones

## References

- `docs/initiatives/check-performance-incremental-revwalk-cache.md` — the full
  design doc this epic implements, including prototype evidence and the
  testing bar this epic's *Constraints* section draws from.
- G-0372 — the gap that surfaced this design; stays open, referencing this
  epic, until this epic's production-cutover milestone closes it.
- G-0328 — golden-fixture byte-identity comparator for `aiwf check`; closed by
  this epic's production-cutover milestone.
- `docs/initiatives/id-lifecycle.md` — the EMB / branch-strategy discussion
  this epic's cache is a named prerequisite for (see its "Relationship to the
  id-lifecycle initiative" section).
- E-0045 — plumbing-based commit construction for aiwf verbs; unrelated
  mechanism, but its second milestone (G-0281) is the consumer this epic's
  cache is a prerequisite for.
