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
      status: met
      tdd_phase: done
    - id: AC-6
      title: Isolation-escape oracle first-parent index built from the shared in-memory DAG
      status: met
      tdd_phase: done
---
## Goal

Eliminate the per-entity / per-pair / per-rule git-subprocess fan-out in
`aiwf check` by reading git history into shared in-memory structures once per
check and having each history-walking rule derive from them, instead of each
rule re-walking history independently.

Delivered: two shared per-check artifacts and their consumers — (1) a **commit
DAG** (`git rev-list --all --reflog --parents`, one pass) consumed by the
orphaned-AI-commit ancestry walk (AC-1) and the isolation-escape oracle's
first-parent index (AC-6); (2) a **HEAD-reachable commit walk** (one
`git log HEAD`) consumed by the acks, ack-entities, audit-only, cherry-pick, and
provenance-commits gathers (AC-5). Plus the `--raw`-enriched `BulkRevwalk` pass
whose per-path blob object ids let the FSM rule read status by object id (AC-2).
Findings byte-identical before and after, pinned by the existing rule fixtures,
with a measured wall-time delta.

## Notes

Behavior-preserving refactor; the ancestry, first-parent, FSM-history, and
trailer-gather semantics are the correctness surface.

### AC-1 — Orphaned-AI-commit walk uses in-memory DAG ancestry, no per-pair merge-base

In-memory commit DAG (`git rev-list --all --reflog --parents`, one pass) answers
ancestry, replacing the 683 per-pair `git merge-base --is-ancestor` spawns. Fast
path + merge-base fallback for the corrupt-repo case keeps findings identical.
Commit 97090bff · phase done.

### AC-2 — Shared per-check git-history context consumed by the history-walking rules

`BulkRevwalk` now streams `git log --all --raw --no-abbrev`, enriching each
`PathTouch` with the pre/post blob object ids. The `fsm-history-consistent` rule
reads status **by blob object id** (a direct object lookup via
`BlobReader.ReadObject`) rather than resolving `<commit>:<path>` per read; ids
dedupe across the walk so each unique blob is read once. (Scope note: AC-2 is the
FSM-side blob-id consumer of the `BulkRevwalk` pass; the broader "many rules, one
pass" sharing is AC-5/AC-6. `BulkRevwalk`'s consumer remains the FSM walk;
`area-mistag` deliberately keeps its own HEAD-scoped walk.) Commit c7a00f3d ·
phase done.

### AC-3 — aiwf check findings byte-identical before and after the refactor

Pinned by the fixture suites (`TestBulkRevwalk_*`, `TestParseRawPathLine`,
`TestFSMHistoryConsistent_*`, the oracle/acks/cherry/provenance suites) — a
behaviour change fails them — and confirmed at every increment by the live-tree
diff: **31 = 31** findings, sorted-identical, old binary (pre-AC-2) vs final
binary. Phase done.

### AC-4 — Measured check wall-time delta recorded in Validation

See `## Validation`. Phase done.

### AC-5 — Shared HEAD-history walk replaces five independent per-rule git-log walks

`WalkHeadCommits` walks HEAD's reachable history once into `[]HeadCommit` (SHA,
trailers, author/committer email, body); the acks, ack-entities, audit-only,
cherry-pick, and provenance-commits gathers each derive their result in-memory
(exact predicate preserved) instead of spawning their own `git log HEAD`. The
`WalkAcknowledgedSHAs`/`Entities` names are retained (the `acks_helper_lift`
single-compute policy pins them); they derive rather than walk now. Byte-identical
levers empirically pinned: `%(trailers:unfold=true)` ==
`%(trailers:only=true,unfold=true)`, and an in-memory `(?m)^aiwf-[a-z-]+:` body
regex selects the same commit set as the prior `git log -E --grep`. Phase done.

### AC-6 — Isolation-escape oracle first-parent index built from the shared in-memory DAG

The oracle derived each ritual branch's first-parent chain from one
`git rev-list --first-parent <branch>` (46 on the kernel tree). It now derives
them from the shared commit DAG (the AC-1 artifact, a superset of what the oracle
needs) via `CommitDAG.FirstParentChain`, which reproduces `rev-list
--first-parent` exactly. The DAG is built once (`BuildCommitDAG`) and shared with
the orphaned-AI-commit walk. A nil DAG (rev-list failed) falls back to the
per-branch rev-list, preserving the per-ref `OracleErr` contract. Phase done.

## Validation

Full suite (`go test ./...`), `golangci-lint`, and `go vet` green; `go build`
clean. Diff-scoped coverage gate green.

**Byte-identical findings (AC-3).** `aiwf check --format=json` on the live kernel
tree, old binary (pre-AC-2) vs the final B+C binary, sorted findings compared:
**31 = 31, identical** — re-verified after each increment (AC-1, AC-2, AC-5,
AC-6, and the review cleanups).

**Subprocess fan-out collapsed.** Per-check git spawns on the kernel tree:
683 `merge-base --is-ancestor` → **0** (AC-1, in-memory DAG); 46
`rev-list --first-parent` → **0** (AC-6, shared DAG); 5 `git log HEAD` trailer
walks → **1** (AC-5, `WalkHeadCommits`); the shared DAG built **once** and used
by two consumers.

**Measured wall-time delta (AC-4).** `aiwf check --format=json`, live kernel tree:

- Algorithm-only (no git `commit-graph`), apples-to-apples old vs new binary:
  **48.8s → 37.3s** for AC-1+AC-2 (the in-memory DAG ancestry + blob-object-id
  read path); AC-5/AC-6 collapse further subprocesses on top.
- With git `commit-graph` present (the realistic state once G-0322/M-0219 land):
  **~35s → ~21.7s** across AC-2 + AC-5 + AC-6. Against the M-0215 baseline
  (~79s, pre-AC-1, no commit-graph) the cumulative E-0053 reduction is ~3.6×.

**Floor analysis (why not the spec's aspirational ~4s).** After B+C the dominant
remaining cost is the FSM `git log --all --raw` subprocess over the
~5,500-commit history (~9s; path-filtering it is *slower* — history-simplification
overhead). The realistic byte-identical floor is ~20s with a commit-graph; going
below it needs the architectural levers in the backlog, not more in-process
collapse.

## Deferrals

Increments B (oracle in-memory first-parent) and C (collapse the HEAD-trailer
walks) — originally scoped out — were brought back in and **landed** as AC-6 and
AC-5 after review found the "shared context" framing required them. The remaining
perf backlog (all `--discovered-in M-0216`), in priority order:

- G-0322 — maintain git `commit-graph` on `aiwf init`/`update` (near-free ~9s);
  addressed by the follow-up milestone M-0219 in this epic.
- G-0323 — incremental / delta-scoped `check` via a validated trunk watermark
  (`<watermark>..HEAD`), the biggest architectural lever (under the floor).
- G-0324 — branch hygiene: prune merged ritual branches; oracle skips
  trunk-ancestor refs (also cuts the orphan walk's ~46 `reflog show`).
- G-0325 — parallelize the *remaining* independent passes (the shared HEAD walk,
  the FSM `--raw` walk, the DAG build), determinism preserved by sorting at the
  aggregation boundary.

## Reviewer notes

Two independent fresh-context review passes (the first subagent attempt hit a
weekly usage limit; an inline adversarial self-review substituted there and
caught the rename/copy parent-side edge case, fixed before AC-2 closed):

- **AC-1 / AC-2 — code-quality + design (wf-rethink).** Code-quality: APPROVE
  (byte-identical by construction — the blob-id fast path is a pure optimization
  over an always-correct fallback). Design: *sound-with-reservations* — its
  load-bearing finding was that the original "shared context consumed by the
  rules" framing overclaimed (only the FSM consumed the enriched pass). **That
  finding is what motivated bringing increments B and C back in** (AC-5/AC-6), so
  the claim is now delivered, not narrowed.
- **AC-5 / AC-6 (B+C) — code-quality.** APPROVE. The reviewer verified the three
  byte-identical equivalences **empirically against the live tree**: the trailer
  formats diffed identical across all ~5,471 HEAD commits; the in-memory
  `^aiwf-` regex selected the identical commit set as `git log -E --grep`
  (~4,614); `FirstParentChain` matched `git rev-list --first-parent` for all 46
  ritual branches, zero mismatches. Two non-blocking advisories, both documented
  in `head_history.go`: **A1** the provenance path is fail-open on catastrophic
  git failure (consistent with the other gathers, vs the old fail-loud); **A2**
  the trailer-format equivalence is a verified tree-shape assumption, not a
  structural invariant.

