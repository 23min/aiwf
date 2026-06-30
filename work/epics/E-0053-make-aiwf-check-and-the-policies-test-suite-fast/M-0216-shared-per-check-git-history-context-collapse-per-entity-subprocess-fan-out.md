---
id: M-0216
title: Shared per-check git-history context; collapse per-entity subprocess fan-out
status: in_progress
parent: E-0053
tdd: required
acs:
    - id: AC-1
      title: Orphaned-AI-commit walk uses in-memory DAG ancestry, no per-pair merge-base
      status: met
      tdd_phase: done
    - id: AC-2
      title: Shared per-check git-history context consumed by the history-walking rules
      status: met
      tdd_phase: done
    - id: AC-3
      title: aiwf check findings byte-identical before and after the refactor
      status: met
      tdd_phase: done
    - id: AC-4
      title: Measured check wall-time delta recorded in Validation
      status: met
      tdd_phase: done
    - id: AC-5
      title: Shared HEAD-history walk replaces five independent per-rule git-log walks
      status: open
      tdd_phase: green
    - id: AC-6
      title: Isolation-escape oracle first-parent index built from the shared in-memory DAG
      status: open
      tdd_phase: red
---
## Goal

Eliminate the per-entity git-subprocess fan-out in `aiwf check` by loading
git history once per check and sharing it across every history-walking rule.

Deliverable: (1) a shared per-check history context — commits, trailers, and
the commit DAG read in a single pass — consumed by the rules that today each
re-walk history independently; (2) the orphaned-AI-commit walk rewritten to
answer ancestry from the in-memory DAG, collapsing its 683
`git merge-base --is-ancestor` subprocess spawns to a single bulk read.
Findings must be byte-identical before and after, pinned by the existing
rule fixtures, with a measured wall-time delta against the baseline.

## Notes

Behavior-preserving refactor; the ancestry and FSM-history semantics are the
correctness surface.

### AC-1 — Orphaned-AI-commit walk uses in-memory DAG ancestry, no per-pair merge-base

In-memory commit DAG (`git rev-list --all --reflog --parents`, one pass) answers
ancestry, replacing the 683 per-pair `git merge-base --is-ancestor` spawns. Fast
path + merge-base fallback for the corrupt-repo case keeps findings identical.
Commit 97090bff · phase done.

### AC-2 — Shared per-check git-history context consumed by the history-walking rules

`BulkRevwalk` now streams `git log --all --raw --no-abbrev`, making it the
single-pass shared context — commits + parents (DAG) + trailers + per-path blob
object ids — consumed by `fsm-history-consistent` (blob ids), `area-mistag`
(paths), and the orphaned-AI-commit walk (DAG). The FSM walk reads status by blob
object id (a direct object lookup) rather than resolving `<commit>:<path>` per
read; ids dedupe across the walk so each unique blob is read once. Commit
c7a00f3d · phase done.

### AC-3 — aiwf check findings byte-identical before and after the refactor

Pinned by the fixture suites (`TestBulkRevwalk_*`, `TestParsePathsBlock`,
`TestFSMHistoryConsistent_*`) — a behaviour change fails them — and confirmed by
the live-tree diff: 34 = 34 findings, sorted-identical, old binary vs new. Phase
done.

### AC-4 — Measured check wall-time delta recorded in Validation

See `## Validation`. Phase done.

## Validation

Full suite (`go test ./...`), `golangci-lint`, and `go vet` green; `go build`
clean.

**Byte-identical findings.** `aiwf check --format=json` on the live kernel tree,
old binary (pre-AC-2) vs new binary, sorted findings compared: **34 = 34,
identical** — the AC-3 mechanical pin.

**Measured wall-time delta (AC-4).** `aiwf check --format=json`, live kernel tree:

- Algorithm-only (no git `commit-graph` present), apples-to-apples old vs new
  binary: **48.8s to 37.3s** (~24%, ~11s) — the blob-object-id read path (AC-2)
  on top of the in-memory DAG ancestry (AC-1).
- With git `commit-graph` present (git's traversal-acceleration cache): **35.4s
  to 25.3s**. The commit-graph compounds the algorithm work; it is tracked
  separately (G-0322 / M-0219), not part of this milestone.

**Floor analysis (why the win caps here, not at the spec's aspirational ~4s).**
The dominant remaining costs are `git log` subprocesses over the ~5,500-commit
history, which cannot be eliminated byte-identically: the FSM `git log --all
--raw` subprocess alone is ~9s (path-filtering it is *slower* — history
simplification overhead), and the provenance layer is ~12s (the isolation-escape
oracle's 46 `rev-list --first-parent`, the orphan reflog walk, the HEAD-trailer
walks). The realistic byte-identical floor is ~28-30s; the deeper wins are
architectural and are captured as the perf backlog below.

## Deferrals

Increments **B** (isolation-escape oracle first-parent index built in-memory from
the DAG, killing 46 `rev-list --first-parent`) and **C** (collapse the ~5
redundant `git log HEAD` trailer walks — acks, ack-entities, audit-only,
cherry-picks, provenance-commits — into one shared pass) were scoped OUT after
the floor analysis: the architectural follow-ups subsume and dwarf them. Captured
as the perf backlog (all `--discovered-in M-0216`):

- G-0322 — maintain git `commit-graph` on `aiwf init`/`update` (near-free ~9s);
  addressed by the follow-up milestone M-0219 in this epic.
- G-0323 — incremental / delta-scoped `check` via a validated trunk watermark
  (`<watermark>..HEAD`), the biggest architectural lever.
- G-0324 — branch hygiene: prune merged ritual branches; oracle skips
  trunk-ancestor refs.
- G-0325 — parallelize the independent history walks / blob reads.

### AC-5 — Shared HEAD-history walk replaces five independent per-rule git-log walks

### AC-6 — Isolation-escape oracle first-parent index built from the shared in-memory DAG

