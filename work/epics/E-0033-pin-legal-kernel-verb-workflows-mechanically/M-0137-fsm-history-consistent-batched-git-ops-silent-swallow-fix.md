---
id: M-0137
title: 'fsm-history-consistent: batched git ops + silent-swallow fix'
status: in_progress
parent: E-0033
depends_on:
    - M-0130
tdd: required
acs:
    - id: AC-1
      title: internal/gitops/ bulk-revwalk helper streams (commit, parent, paths, trailers)
      status: met
      tdd_phase: done
    - id: AC-2
      title: internal/gitops/ cat-file --batch content-reader pump
      status: met
      tdd_phase: done
    - id: AC-3
      title: 'fsm-history-consistent: no per-entity exec.Command — routes through helpers'
      status: met
      tdd_phase: done
    - id: AC-4
      title: history-walk-error subcode emits per failed entity (severity error)
      status: met
      tdd_phase: done
    - id: AC-5
      title: Walker continues past per-entity errors; partial findings preserved
      status: met
      tdd_phase: done
    - id: AC-6
      title: 'Negative test: per-entity walk failure surfaces history-walk-error'
      status: open
      tdd_phase: red
    - id: AC-7
      title: 'Perf regression test: kernel tree aiwf check completes within baseline budget'
      status: met
      tdd_phase: done
    - id: AC-8
      title: Audit catalog R-RULE-149 updated to list all four subcodes with severities
      status: open
      tdd_phase: red
    - id: AC-9
      title: 'G-0149 body updated: fsm-history slice closed; perf retrofits remain open'
      status: open
      tdd_phase: red
---
## Goal

Retrofit `internal/check/fsm_history_consistent.go` to use batched git operations and fix the silent-swallow correctness path that lets a single transient per-entity walker failure return zero findings. Lands two general-purpose helpers in `internal/gitops/` along the way.

## Background

M-0130 shipped `fsm-history-consistent` as the kernel chokepoint that makes the per-entity status FSM a tree-invariant. Two issues were discovered after wrap, recorded in G-0149:

1. **Subprocess fan-out.** The rule shells out per-entity: `git log --follow -m --name-only` per entity, plus `git show <commit>:<path>` per (commit, parent) pair. On a 331-entity tree that's ~3,000 fork/execs per `aiwf check`. Pre-push latency scales with consumer tree size; on macOS the cost is OS-resource-bound (see G-0125, archived).
2. **Silent-swallow on walker failure.** `FSMHistoryConsistent` returns `nil` when `walkStatusChanges` errors, and `walkStatusChanges` fail-fasts on the first per-entity error. One transient git subprocess failure under load wipes every finding from the rule — invisibly. The operator sees a green check; real FSM violations slip through. Diagnosed empirically in the M-0130 session: same binary, same content, intermittent "4 errors" vs "0 findings" on sibling worktrees under concurrent test load.

The silent-swallow violates CLAUDE.md §*Engineering principles* ("Errors are findings, not parse failures") and silently negates the kernel's "framework correctness must not depend on the LLM's behavior" commitment — the chokepoint can turn off under exactly the load it was designed to police.

## Approach

1. **Build two helpers in `internal/gitops/`** templated on `internal/cli/history/history.go`'s existing single-walk shape:
   - **Bulk-revwalk helper** — one `git log --all --name-status -M --pretty=...` subprocess that streams `(commit-sha, parent-sha, paths-touched, trailers)` tuples. Replaces the per-entity `git log --follow` loop.
   - **`cat-file --batch` content-reader pump** — one long-lived subprocess for entity-blob reads. Replaces N short-lived `git show <commit>:<path>` calls. `cat-file --batch-check` for trailer-only queries.
2. **Retrofit `fsm-history-consistent`** to route through the helpers. No more `exec.Command("git", ...)` in the rule's hot path; all reads go through the helpers' long-lived subprocesses.
3. **Replace the silent-swallow** at `FSMHistoryConsistent:71-77`:
   - Emit `fsm-history-consistent/history-walk-error` findings (severity `error`) per failed entity, naming the entity and the underlying error.
   - Stop fail-fast in `walkStatusChanges` (or its batched successor); accumulate partial observations + a per-entity error slice. Successful entities still produce findings; failed entities each produce one `history-walk-error` finding.
4. **Test the partial-failure path mechanically** — fixture that arranges a per-entity walk failure (e.g., delete a referenced blob, cancel mid-walk for one entity), asserts the rule still emits findings for healthy entities AND surfaces a `history-walk-error` for the broken one. This is the negative test that pins the new contract; without it, the swallow can return as a regression because every existing test is structured to succeed end-to-end.
5. **Measure perf before and after.** Baseline the kernel-tree `aiwf check` runtime; the retrofit should reduce wall time substantially (3,000 fork/execs → ~2 long-running subprocesses). Pin a regression budget the perf test asserts.
6. **Reconcile R-RULE-149 in `docs/pocv3/design/legal-workflows-audit.md`** to list four subcodes: `illegal-transition` (error), `forced-untrailered` (error), `manual-edit` (warning), `history-walk-error` (error). Note the partial-failure semantics.
7. **Update G-0149's body** to record that this milestone closes the fsm-history slice; reframe remaining scope as the two interactive-verb retrofits (`aiwf status` worktree views, `aiwf show` scope views).

## What this milestone does *not* do

- Does **not** retrofit `aiwf status` worktree views (G-0149's call site #1). Separate scope; perf-only; no kernel-chokepoint correctness angle.
- Does **not** retrofit `aiwf show` scope views (G-0149's call site #2). Same reason.
- Does **not** change the M-0130 ACs or audit catalog beyond the R-RULE-149 row reconciliation. M-0130 is `done`; this milestone is a follow-up retrofit, not a redo.
- Does **not** address M-0136's historical-error backlog. The `aiwf acknowledge-illegal` verb is M-0136's deliverable; this milestone only fixes how the rule reports errors, not how operators retroactively clear them.

## Inserted between M-0130 and M-0136

M-0136 (`aiwf acknowledge-illegal`) ships the verb that clears the 4 historical `illegal-transition` errors from `f4ea7329`. Its correctness depends on `fsm-history-consistent` firing reliably — i.e., the silent-swallow being gone. Sequencing this milestone before M-0136 means M-0136's tests run against a rule that doesn't intermittently lie.

## At wrap

Promote G-0149 body to record the partial close; G-0149 itself stays `open` because the two interactive-verb retrofits remain. The `aiwf-tests:` metric for the perf AC names a number (chosen at AC-7 design time) so future regressions are detectable.

## Related

- **G-0149** — the gap this milestone partial-closes (the fsm-history-consistent slice). Filed on main as G-0148; reallocated to G-0149 on epic/E-0033 after the merge id-collision.
- **M-0130** — the milestone whose deliverable this retrofits.
- **D-0008 / D-0010** — the per-subcode disjointness + merge-skip decisions that constrain the predicate logic; preserved unchanged.
- **CLAUDE.md §Engineering principles** — *"Errors are findings, not parse failures."* The silent-swallow is the exact pattern that principle forbids.
- **G-0125** (archived) — first surfaced the macOS subprocess-fan-out angle that G-0149 inherits.
- **`internal/cli/history/history.go:283, :515`** — single-walk template the new helpers should mirror.

## Work log

Per-AC outcome notes. Phase + status timeline lives in `aiwf history M-0137/AC-<N>` — not duplicated here.

### AC-1 — internal/gitops/ bulk-revwalk helper streams (commit, parent, paths, trailers)

`BulkRevwalk(ctx, root, fn)` streams `CommitRecord{Commit, Parents, Paths, Trailers}` via one `git log --all --name-status -M -m --pretty=...` subprocess; printable `===AIWF-REC===` / `===AIWF-PATHS===` record markers + `\x1f` field separators; `bufio.Scanner`-backed parsing; helper-test coverage at 100% on splitOnMarker / parseBulkChunk / parseBulkTrailers / parsePathsBlock. · commit `d83a1d30` · 35 tests + subtests passing

### AC-2 — internal/gitops/ cat-file --batch content-reader pump

`BlobReader` wraps a long-lived `git cat-file --batch` subprocess (one StdinPipe + StdoutPipe pair); `Read(commit, path)` writes one `<commit>:<path>\n` request, parses the `<sha> <type> <size>` (or `<input> missing`) header, reads exactly `size` bytes of content + the trailing LF. `ErrBlobMissing` sentinel for the not-found branch; binary content roundtrips exactly (NUL bytes + mid-content newlines preserved). `parseBatchHeader` at 100% coverage (table-driven over found / missing / wrong-field-count / non-integer-size / negative-size). · commit `54c7c24d` · 9 BlobReader tests + 8 parseBatchHeader subtests passing

### AC-3 — fsm-history-consistent: no per-entity exec.Command — routes through helpers

`batchedWalkStatusChanges` in `internal/check/fsm_history_walker.go` consumes one `gitops.BulkRevwalk` stream (whole-repo commit log) and one `gitops.BlobReader` cat-file pump for all status reads. Deleted from the rule: `walkOneEntity`, `listCommitPathPairs`, `commitParents`, `statusAtCommitPath`, `commitTrailers` — the five M-0130 per-entity helpers that each fanned out one or more `exec.Command` calls per entity. `walkStatusChanges` retained as a thin adapter so the existing M-0130 test fixtures continue to drive the same observation shape. Mechanical evidence via `internal/policies/m0137_ac3_batched_walker.go` — source-check policy asserting both files reference the batched helpers and no longer define the per-entity helpers. · commit `5a31e6e7` · policy passes

### AC-4 — history-walk-error subcode emits per failed entity (severity error)

`fsm-history-consistent/history-walk-error` finding (severity error) emits per failed (entity, commit, side) read. Source: `historyWalkErrorFindings` in `internal/check/fsm_history_walker.go`; deduped per (entity, commit, side) so a multi-parent merge with the same parent-side read failing doesn't inflate the count. Test: `TestFSMHistoryConsistent_AC4_CancelledContext_EmitsWalkError` (pre-cancelled context → walker fails → finding emerges). Hint table entry + SKILL.md row landed for AI-discoverability. · commit `5a31e6e7` · 1 RED→GREEN test

### AC-5 — Walker continues past per-entity errors; partial findings preserved

Partial-failure preservation pinned by `TestFSMHistoryConsistent_AC5_PartialFailure_PreservesGoodFindings`: a fake blobReader errors on E-0002's paths while delegating to a real BlobReader for E-0001's. The walker emits `illegal-transition` for E-0001 (good portion preserved) AND `history-walk-error` for E-0002 (failed portion surfaced) — proving the M-0130 fail-fast + entry-point swallow is gone. Closes the silent-swallow load-bearing correctness issue G-0149 flagged. · commit `5a31e6e7` · 1 RED→GREEN test

### AC-6 — Negative test: per-entity walk failure surfaces history-walk-error

### AC-7 — Perf regression test: kernel tree aiwf check completes within baseline budget

### AC-8 — Audit catalog R-RULE-149 updated to list all four subcodes with severities

### AC-9 — G-0149 body updated: fsm-history slice closed; perf retrofits remain open
