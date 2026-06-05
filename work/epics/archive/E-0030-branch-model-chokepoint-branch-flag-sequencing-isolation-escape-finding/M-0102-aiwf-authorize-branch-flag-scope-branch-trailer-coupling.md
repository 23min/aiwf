---
id: M-0102
title: aiwf authorize --branch flag + scope-branch trailer coupling
status: done
parent: E-0030
tdd: required
acs:
    - id: AC-1
      title: --branch flag wired on aiwf authorize with Cobra completion
      status: met
      tdd_phase: done
    - id: AC-2
      title: aiwf-branch trailer constant plus git-ref-shape ValidateTrailer rule
      status: met
      tdd_phase: done
    - id: AC-3
      title: aiwf-branch trailer emitted iff --branch passed (backward-compatible)
      status: met
      tdd_phase: done
    - id: AC-4
      title: aiwf-branch sorts between aiwf-scope and aiwf-scope-ends in trailerOrder
      status: met
      tdd_phase: done
    - id: AC-5
      title: internal/branchparse/ package extracted from worktrees.go
      status: met
      tdd_phase: done
    - id: AC-6
      title: --branch completion returns ritual-shape local branches
      status: met
      tdd_phase: done
    - id: AC-7
      title: completion_drift_test passes on new flag without allowlist entry
      status: met
      tdd_phase: done
    - id: AC-8
      title: --branch against non-existent branch not refused at this milestone
      status: met
      tdd_phase: done
---

## Goal

Add `aiwf authorize --branch <name>` flag and the new `aiwf-branch:` commit trailer key recording the scope-branch coupling on the `authorize` commit. Lift `parseEntityFromBranch` and the ritual-shape regexes from `internal/cli/status/worktrees.go:485` into a new `internal/branchparse/` package so M-0103's preflight and the existing `aiwf status --worktrees` correlation share one regex set. Pure additive: optional flag, new trailer, no behavior change when the flag is absent.

## Context

[ADR-0010](../../../docs/adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md) says AI multi-commit work belongs on named ritual branches; this milestone adds the kernel surface that records *which branch* a scope is bound to. Foundation for M-0103's preflight (which refuses dispatch when the coupling is absent) and M-0106's kernel finding (which detects drift away from the recorded branch).

The flag wires through Cobra completion per CLAUDE.md's auto-completion-friendly rule. The trailer key is added to CLAUDE.md § "Commit conventions" so `aiwf history` consumers and downstream tooling see it.

## Pre-decided design

Per E-0030 §"Design decisions":

- **Trailer key:** `aiwf-branch:`. Constant lands in `internal/gitops/trailers.go` alongside `TrailerActor`, `TrailerTo`, `TrailerScope`; included in `trailerOrder` between `TrailerScope` and `TrailerScopeEnds` (the scope's branch is metadata *about* the scope opening). `ValidateTrailer` shape rule: git-ref-shape regex (`^[A-Za-z0-9._/-]+$`, no leading slash, no embedded `..`); a refusal at write time is preferable to a soft find later.
- **Behavior when `--branch` is absent:** backward-compatible no-op. The trailer is emitted *only when* `--branch <name>` is passed. M-0103's preflight is what enforces the chokepoint — this milestone keeps the surface additive.
- **Completion behavior:** `RegisterFlagCompletionFunc("branch", ...)` returns local branches matching the ritual-shape regexes from `internal/branchparse/`. Full-branch-list completion is a smaller hammer (better UX when the operator is intentionally naming a custom branch) but defeats the discoverability win; ritual-shape-only is the right default.
- **`internal/branchparse/` extraction:** lifts `parseEntityFromBranch` and the ritual-shape compiled regexes from `internal/cli/status/worktrees.go:485` plus the helper that maps `branch → (kind, entity-id)`. Both this milestone's flag-completion and M-0103's preflight detection consume it; `worktrees.go` rewires to consume from the new package. One source of truth — by construction, not by review.

## Out of scope

- Refusing the dispatch when `--branch` is absent (that's M-0103, the preflight).
- Auto-creating the branch if absent — default is "require the named branch already exists" per ADR-0010's promote-then-cut sequencing rule. Deferred unless friction surfaces.
- Updates to `aiwfx-start-epic` / `aiwfx-start-milestone` rituals (M-0104 / M-0105).
- Kernel finding for post-hoc detection (M-0106).
- Spec-cell registration in `internal/workflows/spec/branch/` — that's M-0158's consolidation.
- Changes to human-actor `aiwf authorize` flows (sovereignty preserved).

## Dependencies

None — foundational.

## Acceptance criteria

<!-- Drafted at `aiwfx-start-milestone M-0102` time. The catalog below is the AC seed set:
1. `--branch <name>` flag is wired on `aiwf authorize <id> --to ai/<agent>` and round-trips through Cobra completion.
2. `aiwf-branch:` trailer constant lands in `internal/gitops/trailers.go` with the git-ref-shape `ValidateTrailer` rule.
3. The trailer is emitted on the authorize commit iff `--branch` was passed; absent flag = no trailer (backward-compatible).
4. Trailer ordering: `aiwf-branch:` sorts between `aiwf-scope:` and `aiwf-scope-ends:` per `trailerOrder`.
5. `internal/branchparse/` package exists; `parseEntityFromBranch` and the ritual-shape regexes are lifted from `internal/cli/status/worktrees.go:485`; `worktrees.go` consumes from the new package.
6. The flag's completion returns local branches matching ritual-shape regexes from `internal/branchparse/`.
7. `internal/cli/integration/completion_drift_test.go` recognizes the new flag without an allowlist entry.
8. `--branch <name>` against a non-existent branch is *not* refused at this milestone — that's M-0103's job. This milestone's behavior is "record whatever name was passed, validated only against trailer-shape rules."
-->

### AC-1 — --branch flag wired on aiwf authorize with Cobra completion

`StringVar(&branch, "branch", "", ...)` on the `authorize` command at `internal/cli/authorize/authorize.go:71`, threaded through `Run()` to `verb.AuthorizeOptions{Branch: branch}` on the `--to` path. `RegisterFlagCompletionFunc("branch", completeBranchFlag)` wires the completion (real function via AC-6; cycle-3 shipped a `cobra.NoFileCompletions` placeholder that AC-6 replaced).

Test: `TestNewCmd_SmokeShape` at `internal/cli/authorize/authorize_test.go:9` enumerates the flag set and fails on a missing `--branch`. End-to-end: `aiwf authorize --help` carries the flag with its ADR-0010 reference.

### AC-2 — aiwf-branch trailer constant plus git-ref-shape ValidateTrailer rule

`TrailerBranch = "aiwf-branch"` lands in `internal/gitops/trailers.go` alongside the I2.5 provenance trailers. `ValidateTrailer` enforces a permissive git-ref-shape regex (`^[A-Za-z0-9._/-]+$`) plus targeted checks for leading slash and embedded `..`. The regex rejects empty, whitespace, tilde, colon in one pass; the prefix and substring checks produce targeted error messages for the structural cases.

Tests: 11 new cases under `TestValidateTrailer_KnownKeys` covering ritual shapes (epic/M/patch), plain names (`main`), dotted names (`release-1.2.3`), and each violation class (empty, whitespace, leading slash, embedded `..`, tilde, colon). The shape rule also fires through `verb.Authorize` via the standard `validateAuthorizeTrailers` pass — pinned by `TestAuthorize_Open_WithBranch_InvalidShapeRefused` at `internal/verb/authorize_test.go:175`.

### AC-3 — aiwf-branch trailer emitted iff --branch passed (backward-compatible)

The trailer is emitted only when `opts.Branch` is non-empty after `strings.TrimSpace` — matches the existing `aiwf-reason:` emission convention. Empty/whitespace `Branch` leaves the trailer absent, preserving the pre-M-0102 commit shape for every call site that doesn't opt in.

Tests: paired positive/negative at the verb layer (`TestAuthorize_Open_WithBranch_EmitsTrailer`, `TestAuthorize_Open_NoBranch_NoTrailer`). Cli seam test at `internal/cli/integration/authorize_cmd_test.go::TestRunAuthorize_WithBranch_EmitsTrailer` drives `aiwf authorize ... --branch ...` through cobra → Run → verb → commit → `HeadTrailers` and asserts the trailer lands; **sabotage-verified** that swapping `opts.Branch = branch` to `opts.Branch = ""` fails the seam test, closing the reviewer-flagged cli-layer gap.

### AC-4 — aiwf-branch sorts between aiwf-scope and aiwf-scope-ends in trailerOrder

`TrailerBranch` is slotted into `trailerOrder` between `TrailerScope` and `TrailerScopeEnds` (the scope's branch is metadata *about* the scope opening). `SortedTrailers` reads `trailerOrderIndex` for stable canonical-order output.

Test: `TestSortedTrailers_BranchPositionBetweenScopeAndScopeEnds` at `internal/gitops/trailers_test.go:124` constructs an 8-trailer input in deliberately-wrong order including all three neighbors and asserts the sorted output places `aiwf-branch` between `aiwf-scope` (immediately before) and `aiwf-scope-ends` (immediately after).

### AC-5 — internal/branchparse/ package extracted from worktrees.go

`ParseEntityFromBranch` and the ritual-shape regex (`^(?:epic|milestone|patch)/([EeMmGg]-\d+)(?:-|$)`) lifted byte-equivalently from `internal/cli/status/worktrees.go:485` to `internal/branchparse/branchparse.go`. Three call sites in `worktrees.go` updated to consume from the new package; the local function and regex var deleted along with the `regexp` import. M-0103's preflight (next milestone) and M-0102's own AC-6 completion both consume the same source of truth — drift between the two regex sets is structurally impossible.

Tests: 14-case `TestParseEntityFromBranch` in the new package covering positive ritual shapes (with and without slug, narrow and canonical widths, lowercase/uppercase patch id), and 6 negative cases (non-ritual prefixes, missing id segment, empty). The narrow-id allowlist entry in `internal/policies/narrow_id_sweep_test.go` moved with the test home. End-to-end binary smoke confirmed `aiwf status --worktrees` correlation on this branch returns `E-0030`.

### AC-6 — --branch completion returns ritual-shape local branches

`completeBranchFlag` at `internal/cli/authorize/authorize.go:91` returns `ritualLocalBranches(".")` with the `ShellCompDirectiveNoFileComp` directive. `ritualLocalBranches(rootDir)` shells out to `git for-each-ref refs/heads/ --format=%(refname:short)`, splits the output line-by-line, and keeps only names that `branchparse.ParseEntityFromBranch` recognizes as ritual-shaped. Non-ritual branches (`main`, `fix/*`, `chore/*`, `patch/<topic-without-id>`) are deliberately omitted so completion is the discoverability surface for the convention itself.

Tests: three unit tests under `internal/cli/authorize/authorize_test.go` cover the filter (4 ritual + 4 non-ritual mixed input), the best-effort failure path (non-git directory returns nil), and the empty-repo case (returns nil, no panic). The helper is exposed to the external test package via `internal/cli/authorize/export_test.go::RitualLocalBranchesForTest`. End-to-end seam: `TestRunAuthorize_BranchCompletion_ReturnsRitualBranches` drives `aiwf __complete authorize E-0001 --branch ""` via the built binary against a fixture repo with mixed branches; asserts ritual branches surface, non-ritual ones don't, and the directive line `:4` (NoFileComp) is present. **Sabotage-verified**: swapping the adapter's cwd from `"."` to `"/nonexistent/path"` fails this test.

### AC-7 — completion_drift_test passes on new flag without allowlist entry

`TestPolicy_FlagsHaveCompletion` at `internal/cli/integration/completion_drift_test.go:41` walks the Cobra command tree and asserts every value-taking flag either has a registered completion function or appears in `optOutFlags`. The `--branch` flag has `completeBranchFlag` registered (AC-6); no allowlist entry needed. Verified by direct test run after each cycle 3 (placeholder) and cycle 4 (real function) — passes both times. Cycle 3 satisfied the drift property via `cobra.NoFileCompletions`; cycle 4 confirms the swap to the real function keeps it green.

### AC-8 — --branch against non-existent branch not refused at this milestone

Per the E-0030 design, branch-existence checking is M-0103's preflight responsibility; M-0102 validates only against trailer-shape rules from AC-2 and records whatever name was passed. `verb.Authorize` does not enumerate refs, query git for branch presence, or otherwise constrain the value beyond `ValidateTrailer`. The "refusal lives in M-0103" boundary is the same shape as ADR-0009's substrate-vs-driver split.

Test: `TestAuthorize_Open_WithBranch_NonExistentBranchAccepted` at `internal/verb/authorize_test.go:108` passes `Branch: "epic/E-9999-not-a-real-branch"` against a test repo that never creates that branch, asserts the verb succeeds and the trailer lands. The complementary M-0103 milestone test will assert the preflight refusal for the same input when the chokepoint moves up the stack.

## Work log

- Pre-activation: stale-test fix landed on main via `wf-patch` (`4a9baf3c` — `TestBinary_CheckDefault_KernelTreeShortOutput` was written on the premise that the kernel tree always has chronic warnings; those have been resolved, so the test now handles the clean-tree state).
- Activated E-0030 (`c13cf672`) and cut `epic/E-0030-branch-model-chokepoint` from that point. Promoted M-0102 draft → in_progress (`335842ca`).
- **Cycle 1 (AC-5)**: extracted `internal/branchparse/` package (`c43c5dfe`). Lifted `ParseEntityFromBranch` and the ritual-shape regex byte-equivalently from `worktrees.go`; rewired the three call sites; moved the narrow-id allowlist entry. End-to-end smoke confirmed `aiwf status --worktrees` correlation unchanged.
- **Cycle 2 (AC-2 + AC-4)**: added `TrailerBranch` constant + `ValidateTrailer` rule + position in `trailerOrder` (`e1eece82`). Also synced `canonicalTrailerKeys` in the trailer-shape drift test so cycle 3's emission would be recognized. Added rows to `docs/pocv3/design/provenance-model.md` "Trailer set" and "Closed-set constraints" tables — the discoverability rule requires trailer keys to be reachable via a canonical channel.
- **Cycle 2 discovery**: noticed `TrailerForceFor` missing from `canonicalTrailerKeys` (pre-existing drift, unrelated to M-0102). Filed as G-0195 via `aiwf add gap --discovered-in M-0102` (`840b12f2`).
- **Cycle 3 (AC-1 + AC-3 + AC-8)**: wired the `--branch` flag, trailer emission iff passed, no refusal for non-existent branch (`aa1dbeb6`). Initial commit landed with verb-layer tests but the reviewer subagent found a cli-layer seam gap — the `opts.Branch = branch` propagation was unexercised by `go test`. Added `TestRunAuthorize_WithBranch_EmitsTrailer` integration test driving the binary end-to-end; sabotage-verified it catches the regression class. Also added a cli-layer gate refusing `--branch` with `--pause/--resume` (matches the existing `--reason` gate).
- **Cycle 4 (AC-6 + AC-7)**: replaced cycle-3's `cobra.NoFileCompletions` placeholder with `completeBranchFlag` backed by `ritualLocalBranches` (`aabc9028`). Reviewer subagent approved with one non-blocking nit on the cobra-adapter seam test (directive code unpinned); applied the one-line tightening before commit. Sabotage-verified the seam test catches cwd-regression.

Phase + status promotes for each AC followed each implementation commit in order; the full timeline is queryable via `aiwf history M-0102` and `aiwf history M-0102/AC-<N>`.

## Decisions made during implementation

- **CLI gate for `--branch` + `--pause/--resume`** (cycle 3). The verb only emits the trailer on the `--to` path; without a gate, passing `--branch` with `--pause` or `--resume` would silently drop the flag — a usability footgun. Added the matching usage-error gate (mirrors the existing `--reason` + `--pause/--resume` rule). Not in the AC catalog; small, defensible, matches convention. Two unit tests pin the refusal codes.
- **Trim-then-validate for `Branch`** (cycle 3). `strings.TrimSpace(opts.Branch)` runs before `ValidateTrailer` in the verb; leading/trailing whitespace is silently fixed rather than rejected. Matches the existing `Reason` / `Agent` convention in the same function (lines 96, 130 in `verb/authorize.go`). Documented in cycle 3's commit body.
- **Spec body references to the unexported `parseEntityFromBranch`** (cycles 1 and beyond). The M-0102 spec body (lines 44, 59, 81) references the lowercase symbol name — the source state pre-lift. Intentionally not rewritten at wrap: the spec describes the work plan, not the post-implementation surface. The post-lift symbol is `branchparse.ParseEntityFromBranch`; the spec's "Lift X from path Y" prose is accurate as historical narrative.

No decisions warranted a `D-NNN` entity or an `ADR-NNNN`. The CLI gate and the trim convention are local-to-the-function shape choices, not project-scoped policy.

## Validation

- `golangci-lint run`: 0 issues across all touched packages.
- `CGO_ENABLED=0 go build -o /tmp/aiwf ./cmd/aiwf`: clean.
- `go test ./internal/branchparse/ ./internal/gitops/ ./internal/verb/ ./internal/cli/authorize/ ./internal/cli/integration/ ./internal/policies/ ./internal/cli/status/`: green (target-scoped re-run; the relevant unit + integration slices are clean).
- `make test`: full suite passes modulo a pre-existing orthogonal flake class (`internal/contractverify`, `internal/check::TestFSMHistoryConsistent_PerfBudget`, `internal/policies::TestM0124_*`, `internal/cli/integration::TestRun_HistoryReadsAiwfToAndForce` and similar). Each fail reproduces on a different test per run, every one passes 3+ times in isolation, all in packages untouched by this milestone. Documented Linux-tmpfs/`TempDir` cleanup race; not introduced by M-0102.
- `aiwf check`: 0 errors. Pre-wrap warnings included 8 `entity-body-empty/ac` warnings on `M-0102/AC-1..AC-8` — addressed by this wrap commit (the AC body sections above are no longer empty). Remaining warning at wrap-prep: `provenance-untrailered-scope-undefined ×1` (env-shape: "no upstream configured and no --since"; not a code issue).
- Binary smokes (cycle-3 and cycle-4):
  - `aiwf authorize --help`: `--branch` flag surfaces with ADR-0010 reference.
  - `aiwf __complete authorize E-0001 --branch ""` against a fixture repo with mixed ritual and non-ritual branches: only ritual ones returned; directive code `:4` (NoFileComp) present.
- Two subagent reviewer dispatches (cycle 3 and cycle 4): both verdicts **approve**, both with non-blocking findings that were applied before commit (cycle 3 — cli seam test; cycle 4 — directive-code assertion).

## Deferrals

- **[G-0195](../../gaps/G-0195-canonicaltrailerkeys-drifts-from-trailerorder-no-mirror-validity-guard.md)** — `canonicalTrailerKeys` in `internal/cli/integration/trailer_shape_test.go` is a hand-maintained mirror of `gitops.trailerOrder`; `TrailerForceFor` (added in M-0136) is missing from the mirror, and there is no mechanical guard tying the two together. The drift is silent today because no test fixture exercises a verb that emits `aiwf-force-for:`. Filed with `--discovered-in M-0102` during AC-2 verification. Out of scope for M-0102; doesn't block any E-0030 milestone.

No AC was deferred or cancelled. All 8 met.

## Reviewer notes

- **Two subagent reviewer dispatches** (cycle 3 and cycle 4) both returned **approve** verdicts with non-blocking nits. Cycle 3's reviewer found the cli-layer seam gap (the `opts.Branch = branch` propagation line at `authorize.go:154` was unexercised by `go test`); I closed it with `TestRunAuthorize_WithBranch_EmitsTrailer` and sabotage-verified the test catches the regression class. Cycle 4's reviewer suggested pinning the cobra directive code (`:4` = NoFileComp) in the `__complete` integration test so a future refactor that flips the directive to `Default` would be caught; applied the one-line tightening before commit.
- **Sabotage-verifications** were performed twice during the milestone — once for the AC-3 cli-seam test (changed `opts.Branch = branch` to `opts.Branch = ""` → seam test failed as expected), once for the AC-6 cobra-adapter seam test (changed `ritualLocalBranches(".")` to `ritualLocalBranches("/nonexistent/path")` → integration test failed as expected). Each was reverted from `/tmp/*.bak` immediately. Both confirm the seam tests are mechanically catching their target regression class, not just passing because the implementation happens to be correct.
- **Spec body references to `parseEntityFromBranch`** at lines 44, 59, 81 of this file describe the pre-lift state of the work, not the current symbol. Intentional per the *Decisions made during implementation* note above; the post-lift symbol is `branchparse.ParseEntityFromBranch`.
- **`G-0195` drift discovery** is out of scope but worth a follow-up: replacing the hand-maintained `canonicalTrailerKeys` map with one derived from `trailerOrder` at test init eliminates the parallel-source-of-truth class. The gap body sketches the resolution shape.
- **Test discipline**: every AC has a Go test under `internal/policies/` (M-0102 didn't need one — its ACs are surface-level, not policy-shaped) or under the relevant package's unit/integration tests, per CLAUDE.md's "AC promotion requires mechanical evidence" rule. Branch-coverage audited per cycle; sabotage-verifications added on the load-bearing seams.
- **Pre-existing flakes** in `internal/contractverify`, `internal/check`, `internal/policies`, `internal/cli/integration` were observed during this milestone but are orthogonal Linux-tmpfs/`TempDir` cleanup races in packages untouched by M-0102. Documented in *Validation* above; not blocking. Tracked separately if recurrent.

