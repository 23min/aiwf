---
id: E-0053
title: Make aiwf check and the policies test suite fast
status: active
---
## Goal

Cut the wall-time of `aiwf check` — the pre-push and CI chokepoint — and of
the `internal/policies` test suite, by eliminating redundant git-subprocess
fan-out and a redundant pre-push lint run, without weakening any guarantee
or moving any rule from pre-push to CI.

On the kernel's own repo (659 entities) `aiwf check` measures ~85s, almost
entirely git-subprocess overhead: a single run spawns ~895 git processes,
683 of them `git merge-base --is-ancestor` issued one-per-reflog-pair by the
orphaned-AI-commit walk. The check runs ~6 independent git-history passes
that never share a loaded history.

## Scope

In scope:

- Profile `aiwf check` and the policies suite to a per-rule wall-time
  baseline (CPU profile + subprocess attribution). Profile before optimizing.
- A shared per-check git-history context: load commits and the commit DAG
  once, then consume that across every history-walking rule, replacing the
  independent re-walks.
- Rewrite the orphaned-AI-commit walk to answer ancestry from an in-memory
  DAG (683 subprocess spawns collapse to one).
- A last-green-lint marker so the pre-push `golangci-lint` is skipped when
  HEAD is already linted green against a clean tree (gap `G-0318`).
- Drive the `internal/policies` suite below its residual ~9s parallel-bound
  floor (gap `G-0321`).

Out of scope:

- Rule *tiering* — changing which rules fire at pre-push versus CI. That
  alters a guarantee's timeliness and is a separate decision, not a
  performance refactor.
- Gap `G-0317` (section-level granularity of the M-0196 backstop) — not a
  performance concern.

## Constraints

- Behavior-preserving: byte-identical findings before and after. Every
  optimization is gated by the existing rule fixtures plus a before/after
  wall-time measurement.
- No guarantee silently relocated from pre-push to CI.
- The first milestone establishes the baseline the rest measure against; no
  optimization lands without a measured delta.

## Source

Surfaced while diagnosing wrap+push slowness during E-0048 / M-0196. The
underlying gaps are `G-0319` (the `aiwf check` cost), `G-0318` (pre-push
lint redundancy), and `G-0321` (the policies-suite floor); `G-0320` (the
policies-suite test fixture) already landed on trunk as the first cut.
