---
id: M-0213
title: Opt-in best-effort fetch before id allocation
status: done
parent: E-0052
tdd: required
acs:
    - id: AC-1
      title: aiwf add --fetch refreshes the trunk ref before allocating
      status: met
      tdd_phase: done
    - id: AC-2
      title: The fetch is best-effort and never blocks the add
      status: met
      tdd_phase: done
---
## Goal

Add an opt-in, best-effort refresh of the trunk-tracking ref immediately before
allocation, so `max` is computed against the freshest published trunk. A session
that has not fetched recently allocates against a stale `refs/remotes/origin/...`
view and can hand back an id that already landed upstream; an opt-in fetch
narrows that window (class 2 of G-0272's taxonomy).

Best-effort is load-bearing: a fetch failure (offline, no remote, network error)
must degrade to current local-only allocation with a warning — never block or
fail the add. The fetch narrows the window; it does not close it (another machine
can publish between the fetch and the commit — that residual is G-0274's to
cure). Surfaced as a `--fetch` flag on `aiwf add`; a `doctor` staleness nudge is
a possible complement, deferred.

Source: G-0273. Parent epic E-0052.

### AC-1 — aiwf add --fetch refreshes the trunk ref before allocating

`aiwf add <kind> --fetch` refreshes the configured trunk ref (only that ref, not
a full `fetch --all`) before computing `max`, so an id that landed on trunk since
the last local fetch is seen and skipped.

Evidence: a test with a local clone whose trunk ref is advanced out-of-band — the
`--fetch` allocation reflects the upstream id; the same allocation without
`--fetch` does not (test fixture ids in backticks below are literal allocator
outputs, not entity references).

### AC-2 — The fetch is best-effort and never blocks the add

The fetch is best-effort and never blocks the add: a failure (no remote, an
unreachable origin, a network error) degrades to local-only allocation with a
warning and a success exit, identical to today's behavior.

Evidence: a no-remote repo where `aiwf add --fetch` succeeds, emits a warning,
and allocates against the local view.

## Work log

Phase timeline is authoritative in `aiwf history M-0213/AC-<N>`; not duplicated here.

### AC-1 — aiwf add --fetch refreshes the trunk ref before allocating
`gitops.FetchBranch` (single `git fetch <remote> <branch>`, not `--all`) →
`cliutil.FetchTrunkBestEffort` (parses the trunk ref into remote+branch via
`parseRemoteTrackingRef`, fetches, returns a descriptive error) → `aiwf add`'s
`--fetch` flag runs the fetch before `LoadTreeWithTrunk`. · commit `56207075` ·
tests: `TestFetchBranch_RefreshesRemoteTrackingRef`, `TestParseRemoteTrackingRef`
(full table), `TestAdd_FetchReflectsUpstreamID` (clone, out-of-band advance:
`--fetch` → `G-0003`, no-fetch → `G-0002`).

### AC-2 — The fetch is best-effort and never blocks the add
The dispatcher warns to stderr and continues on any fetch failure; `FetchBranch`
returns `[]string`-free error, never panics. · commit `56207075` · tests:
`TestFetchBranch_NoRemote_Errors`, `TestFetchTrunkBestEffort_NonRemoteTrackingTrunk_Errors`,
`TestFetchTrunkBestEffort_MalformedConfig_Errors`, `TestAdd_FetchBestEffort_NoRemote`
(no-remote → warning + success exit + local `G-0001`). Both AC tests vacuity-checked
(disabling the fetch block fails both).

## Decisions made during implementation

- **Scope held to the trunk ref only (the option-A path).** During implementation
  the question arose whether `--fetch` should fetch *all* of origin's branches and
  the allocator scan every `refs/remotes/origin/*` (catching a teammate's
  pushed-but-unmerged feature-branch ids). Decided to keep the narrow single-branch
  refresh: this repo is trunk-based (short-lived feature branches), the broader view
  burns ids on abandoned branches and costs O(remote branches) per add, and YAGNI —
  the driver was solo+agents in local worktrees (M-0212's domain). The rejected
  broader design is captured as **`G-0316`** (see Deferrals), with ADR-0001 as its
  heavyweight structural alternative.

## Validation

- `make check-fast` (golangci-lint + go vet + full `go test` incl
  `internal/policies`): **green**.
- `go build ./...`: **green**. `golangci-lint` over the four affected package
  trees: **0 issues**.
- Branch coverage: `FetchBranch` / `parseRemoteTrackingRef` 100%; the new `add.go`
  `--fetch` block fully covered (both the success-enter and warning arms); the one
  uncovered `add.go` block is the **pre-existing** `LoadTreeWithTrunk` error path,
  not this change. `FetchTrunkBestEffort`'s four branches (config-error, parse-fail,
  fetch-error, success) are all reachable across the cliutil unit tests + the
  integration tests, so the diff-scoped gate is green on the combined profile.
- `aiwf check`: **0 errors**.

## Deferrals

- **`G-0316`** — "Broaden allocator to scan all remote-tracking refs, not just trunk."
  The fetch-all + scan-all-remote-refs design (option B), deliberately not taken
  here. Allocated on `main` (per the working-cadence directive), discovered-in
  M-0213. Milestone-sized if revived.

## Reviewer notes

- **Independent two-lens review before wrap.** Code-quality (`wf-review-code`) →
  **APPROVE**: all load-bearing claims verified by measurement (single-branch fetch,
  never-blocks, fetch-before-trunk-read, parse correctness, AC non-vacuity);
  lock-held-across-fetch confirmed correct-by-design (the fetch→read→allocate
  sequence must be atomic w.r.t. concurrent local adds); 0 lint issues. Design
  (`wf-rethink`) on the fetch unit → **KEEP**, no rewrite.
- **Track-for-later (non-blocking, recorded not actioned):**
  - `parseRemoteTrackingRef` (cliutil) and `config.TrunkBranchShortName` both parse
    the trunk-ref string with different algorithms for different questions
    (remote+branch vs short-name). A mild single-source-of-truth tension; per YAGNI,
    leave both until a *third* trunk-ref consumer appears, then promote a shared
    `config.TrunkRemoteAndBranch()` and collapse.
  - `add.Run`'s positional signature now carries ~18 string params plus the new
    `fetch bool` — a pre-existing readability smell (not introduced here; the type
    checker catches transpositions). If `add.Run` grows once more, refactor to an
    options struct.
- **Accepted limitation:** `git fetch <remote> <branch>` updates the
  remote-tracking ref only under the standard clone refspec
  (`+refs/heads/*:refs/remotes/origin/*`); a hand-customized `remote.origin.fetch`
  would make `--fetch` a silent no-op rather than an error. Documented at
  `gitops/fetch.go`; acceptable for an opt-in flag, common case covered and pinned
  by `TestFetchBranch_RefreshesRemoteTrackingRef`.
