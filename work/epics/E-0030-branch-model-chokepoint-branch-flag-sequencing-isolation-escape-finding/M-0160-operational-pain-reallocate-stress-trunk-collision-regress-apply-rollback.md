---
id: M-0160
title: Operational pain — reallocate stress, trunk-collision regress, apply rollback
status: in_progress
parent: E-0030
tdd: required
acs:
    - id: AC-1
      title: Reallocate combinatorial real-git E2E coverage
      status: met
      tdd_phase: done
    - id: AC-2
      title: G-0167 trunk-collision regression binary-level E2E
      status: met
      tdd_phase: done
    - id: AC-3
      title: G-0170 apply-rollback data-preservation binary-level E2E
      status: met
      tdd_phase: done
    - id: AC-4
      title: 'Kernel chokepoint: id-rename without reallocate trailer'
      status: open
      tdd_phase: green
---
## Goal

Close evidence-backed operational-pain scenarios surfaced by the M-0159 history-mining audit. Three concrete classes with real in-repo incidents:

1. **Reallocate stress** — 26 reallocate commits in this repo's history confirm cross-branch ID collisions are recurring (CLAUDE.md §"Id-collision resolution at merge time" documents the operator-discipline gap). Verify the `aiwf reallocate` path holds under combinatorial verb-sequence scenarios.

2. **G-0167 trunk-collision regression** — retitle+body growth pushed git rename detection below 50% similarity (`8b56ba1c` "fix(gitops): trailer-driven rename detection"). Pin the regression class so it cannot recur.

3. **G-0170 apply-rollback data-preservation** — `ed0b5014` "fix(verb): apply rollback preserves uncommitted dirty files at touched paths" closed the original incident. Pin the contract via real-git E2E so a future refactor cannot regress the bless-mode data-preservation guarantee.

## Context

Per the M-0159 evidence-priority split, this milestone (Tier 2) addresses operational pain that has already bitten this repo. Distinguished from M-0161 (Tier 3 imagination-driven hardening) by in-history evidence. Distinguished from M-0159 (Tier 1) by being post-framework: M-0159 lands the combinatorial E2E framework (G-0211); M-0160 reuses it for these three scenarios.

## Dependencies

- **M-0159** (Tier 1) — must complete first; M-0160 reuses M-0159's E2E framework.
- **Existing fixes**: `8b56ba1c` (G-0167 trailer-driven rename), `ed0b5014` (G-0170 apply rollback). These are committed; M-0160 adds regression-pin tests, not new fixes.

## Out of scope

- Tier 3 imagination-driven scenarios (G-0200..G-0207, G-0209) — covered by M-0161.
- Data-loss scenarios crossing epic boundaries (G-0212) — future-epic.

## Acceptance criteria

<!--
AC seed set (to be allocated via `aiwf add ac` at start-milestone time):

1. Reallocate-stress combinatorial test: two parallel-branch operators reallocate the same id; merge; verify `aiwf reallocate` resolves cleanly across the matrix of {pre-push, post-merge, with cross-reference, without cross-reference}.

2. G-0167 trunk-collision regression test: retitle a long-bodied entity from a short title to a long title; verify rename detection finds the file via trailer-driven mechanism, not similarity.

3. G-0170 apply-rollback test: dirty an uncommitted file at a touched path; trigger a verb commit failure; verify the dirty content is preserved on rollback.

These three are the seed set; aiwfx-start-milestone refines and allocates them.
-->

**Disclosure: AC body contracts back-filled post-hoc (G-0216).** The four contracts below were written during M-0160 REFACTOR after each AC's implementation had already landed — they describe what the implementation actually delivers, not a lock contract written before RED phase. This is the failure mode [G-0216](../../gaps/G-0216-empty-ac-body-blocks-milestone-draft-to-in-progress-promote.md) names. Future milestones starting after G-0216's fix lands must write AC body contracts BEFORE `aiwf promote M-NNN in_progress`; the kernel will refuse the transition with empty AC bodies. M-0160 (and M-0159) are grandfathered.

### AC-1 — Reallocate combinatorial real-git E2E coverage

**Observable behavior.** A test suite under [`internal/cli/integration/reallocate_scenarios_test.go`](../../../internal/cli/integration/reallocate_scenarios_test.go) drives the worktree-built `aiwf` binary as subprocess against real-git fixtures, covering seven representative reallocate-verb invocation shapes drawn from the ~19 historical `aiwf-verb: reallocate` commits in this repo's history (`git log --grep="aiwf-verb: reallocate"`).

**The seven shapes:**

1. Single-step renumber (`G-X → G-Y`) preserves canonical frontmatter shape — `id:` field updated to new id, `prior_ids:` carries the old id, file at new slug path on disk.
2. Chained renumber (`G-X → G-Y → G-Z`) grows `prior_ids:` oldest-first across multiple reallocates. Pins G-0118's invariant — prior_ids must record the full chain, not just the most-recent jump.
3. Cross-branch merge collision (CLAUDE.md §"Id-collision resolution at merge time"). Trunk and feature branch each independently allocate the same id; `aiwf check` fires `ids-unique/trunk-collision`; `aiwf reallocate` on the feature side resolves it; subsequent check is silent.
4. Cross-reference body-prose rewritten atomically on reallocate (G-5 invariant). Entity A's body mentions G-X; reallocating G-X to G-Y rewrites A's body in the same commit. Pins the prose-grammar rewrite at [`internal/verb/reallocate.go`](../../../internal/verb/reallocate.go).
5. `aiwf-prior-entity` trailer present on the reallocate commit + `aiwf history G-old` bridges old → new. Audit-trail invariant: the renumber event is queryable via the kernel's trailer-driven history.
6. Reallocating an epic atomically moves the contained milestone. The milestone file's path AND its `parent:` frontmatter field both update in the reallocate commit. Pins the directory-rename branch at `reallocate.go::pathInside`.
7. Trunk-aware allocator skips trunk-side ids when allocating on a feature branch (positive baseline; complement to scenario 3). Establishes that scenario 3's collision shape is anomalous (parallel un-pushed branches), not the steady state.

**Edge cases:**

- Scenario 3 uses a hand-authored colliding file rather than time-traveling two operators — the kernel's observable is two files with the same `id:` value, which the hand-authored fixture produces deterministically; the parallel-allocation story is the upstream cause but not the kernel-side observable.
- Scenario 7's discrimination is filesystem-based (assert the trunk-side slug `trunk-gap` survives on `G-0001`, the feature-side `G-0002` exists on disk) rather than output-substring (the verb's subject line doesn't include slugs).
- Scenario 3 verifies the trunk-collision finding fires AND clears, via inline envelope-parse pinned to `code: ids-unique, subcode: trunk-collision`. The framework's Expectation's `NoFindingWithCode: check.CodeIDsUnique` covers the steady-state post-reallocate silence.
- The `aiwfx-start-epic` step-7 framework topology pattern (opener-first) is NOT used here — reallocate scenarios are single-branch state mutations, not cross-branch policy questions.

**References.**
- CLAUDE.md §"Id-collision resolution at merge time"
- M-0159 framework: [`branch_scenarios_helpers_test.go`](../../../internal/cli/integration/branch_scenarios_helpers_test.go)
- Production verb: [`internal/verb/reallocate.go`](../../../internal/verb/reallocate.go)
- G-0118: prior_ids population fix

### AC-2 — G-0167 trunk-collision regression binary-level E2E

**Observable behavior.** A binary-level real-git E2E test under [`internal/cli/integration/trunk_rename_g0167_test.go`](../../../internal/cli/integration/trunk_rename_g0167_test.go) reconstructs the M-0125/G-0139 retitle + body-enrichment failure shape against the worktree-built `aiwf` binary, drives `aiwf check` as subprocess (pre-push hook equivalent), and asserts no `ids-unique/trunk-collision` finding fires.

**The fixture must satisfy two similarity invariants:**

- **Per-commit retitle similarity > 50%.** The retitle commit's `git show -M --diff-filter=R` must produce a rename pair, so G-0167's trailer-driven detection at [`internal/gitops/refs.go`](../../../internal/gitops/refs.go) pass 1 can lift the rename via the `aiwf-verb: retitle` trailer.
- **Cumulative origin/main..HEAD similarity < 50%.** Default `git diff -M50` must NOT pair the old and new paths — so pass 2 (G-0109 fallback) cannot rescue, and the trailer-driven detection is genuinely the load-bearing path.

Both invariants are pinned by inline `t.Fatalf` sanity checks in the scenario Setup so the fixture cannot silently degrade into a false-pass shape.

**Edge cases:**

- Title size cap (80 chars per `entities.title_max_length`) constrains the retitle string — the test uses a 67-char title.
- The original-body size must be moderate (~25 lines) so the title-line change is a small fraction of the per-commit diff (per-commit similarity ~91%); the enriched body is ~5× larger to drop cumulative similarity below 50%.
- The trunk-collision rule's silencing depends on `gitops.RenamesFromRef` returning the rename map; the rule itself is unchanged from M-0106.
- Sabotage-verifiable: reverting pass 1 of `RenamesFromRef` (the trailer-walk arm at lines 247-253) makes the scenario fire with the trunk-collision finding.

**References.**
- Fix commit: `8b56ba1c` "fix(gitops): trailer-driven rename detection (G-0167)"
- Original failure surface: M-0125/G-0139 push to `epic/E-0033-...`
- Unit-level companion: [`TestRenamesFromRef_DetectsTrailerDrivenRenameAcrossBodyEdits`](../../../internal/gitops/refs_test.go) at `refs_test.go:344`

### AC-3 — G-0170 apply-rollback data-preservation binary-level E2E

**Observable behavior.** A binary-level real-git E2E test under [`internal/cli/integration/apply_rollback_g0170_test.go`](../../../internal/cli/integration/apply_rollback_g0170_test.go) drives `aiwf edit-body M-XXX` bless-mode as subprocess against a worktree carrying pre-existing uncommitted edits at the touched path, induces commit failure via empty git identity env vars (mirrors the unit-level pattern at [`internal/verb/apply_test.go`](../../../internal/verb/apply_test.go) `TestApply_RollbackPreservesPreExistingDirtyContent`), and asserts three load-bearing properties:

1. **HEAD SHA did NOT advance.** Commit failure preserves the ref state; rollback didn't accidentally move HEAD.
2. **Worktree bytes match the pre-Apply dirty state, NOT HEAD.** The operator's hand-edit survived — the G-0170 contract. Pre-G-0170 the rollback's `git restore --staged --worktree` reverted to HEAD and silently discarded the hand-authored prose.
3. **Error envelope does NOT contain the misleading "no changes to commit" message** that the pre-G-0170 retry path produced (when rollback reverted to HEAD and a subsequent bless attempt saw clean state).

**Why free-form (no `RunScenarios`).** The M-0159 framework's `Expectation` is designed for `aiwf check --format=json` envelope assertions. AC-3's assertions are filesystem state (worktree bytes) and git state (HEAD SHA) on a verb other than `check`. The test calls `newScenarioEnv(t)` directly to inherit the real-git fixture + worktree-built binary discipline, then does its own assertions.

**Edge cases:**

- Empty git identity env vars (`GIT_AUTHOR_NAME=""`, `GIT_AUTHOR_EMAIL=""`, `GIT_COMMITTER_NAME=""`, `GIT_COMMITTER_EMAIL=""`, `GIT_CONFIG_GLOBAL=/dev/null`, `GIT_CONFIG_SYSTEM=/dev/null`) force the commit to fail deterministically. Env vars must be appended LAST in `testutil.RunBin`'s composition (the AC-6/M-0159 last-wins precedence discovery).
- The "no changes to commit" substring check is intentionally fragile to future error-message rewording — the assertion's job is to fire if the misleading message reappears.
- The `handEditFixtureAC3` constant carries the synthetic operator prose; readability extraction per AC-3 reviewer N-1.
- Sabotage-verifiable: gutting `applyTx.rollback` step 2 (captured-bytes write-back loop at `apply.go:437-452`) with `_ = dedup` fires the worktree-bytes assertion.

**References.**
- Fix commit: `ed0b5014` "fix(verb): rollback restores pre-Apply worktree state, not HEAD (G-0170)"
- Unit-level companions: `TestApply_RollbackPreservesPreExistingDirtyContent`, `TestApply_RollbackIsFullyClean_G0170Regression`, `TestRollback_RemoveErrorIsCapturedWhenRestoreSucceeds`
- The misleading-message anti-pattern is named in G-0170's design notes ("a blind retry wrapper around bless mode is actively harmful")

### AC-4 — Kernel chokepoint: id-rename without reallocate trailer

**Observable behavior.** A new kernel rule fires `id-rename-untrailered` (warning severity, `ClassBranchChoreography`) when a commit between `merge-base(HEAD, trunk)` and `HEAD` renames an id-bearing entity file (path satisfies `entity.PathKind` + `entity.IDFromPath`) AND lacks an `aiwf-verb:` trailer in the rename-class closed set (`retitle` / `rename` / `reallocate` / `archive` / `move` per `gitops.IsRenameVerb`).

Catches the CLAUDE.md §"Id-collision resolution at merge time" operator-discipline failure mode: an operator resolves a trunk-collision via inline `git mv` instead of `aiwf reallocate <id-or-path>`. The immediate trunk-collision finding clears (gitops' rename detection paired the move), but the kernel trailer history misses the renumber event — `aiwf history G-old` doesn't bridge to the new id, cross-references in body prose aren't rewritten, and any future check rule keyed on `aiwf-verb: reallocate` doesn't see the rename.

**The rule's surface.** Production code lives at [`internal/check/id_rename_untrailered.go`](../../../internal/check/id_rename_untrailered.go):

- `CodeIDRenameUntrailered = codespkg.Code{ID: "id-rename-untrailered", Class: ClassBranchChoreography}` (typed Code shape, matches `CodeIsolationEscape`'s precedent)
- `UntrailedIDRename{SHA, OldPath, NewPath, OldID, NewID}` struct (pre-computed by the gather layer)
- `RunIDRenameUntrailered(renames, ackedSHAs) []Finding` (pure function — one warning per record minus ackedSHAs)
- `WalkUntrailedIDRenames(ctx, root, ref) []UntrailedIDRename` (gather-side walker — fail-shut on subprocess error per `WalkCherryPicks` precedent)

Wired in [`internal/cli/check/provenance.go`](../../../internal/cli/check/provenance.go) BEFORE `ResolveUntrailedRange` (the rule is independent of the `@{u}..HEAD` audit scope; uses the trunk ref directly, same as the trunk-collision rule).

**Edge cases:**

- **Acknowledged SHAs are exempted** via the existing `ackedSHAs map[string]bool` parameter (M-0159/AC-3 helper-lift contract; per-SHA closed-set scoping). This makes `id-rename-untrailered` the **fourth** consumer of the ack-helper-lift, alongside `fsm-history-consistent`, `isolation-escape`, and `trailer-verb-unknown`. PolicyAcksHelperLift extended to police the four-consumer wiring.
- **Non-entity file renames are filtered** at the walker via `entity.PathKind` + `entity.IDFromPath`. README.md → DOCS.md does not match any id-bearing path pattern; the walker emits no record.
- **Closed-set authority lives in gitops** (`IsRenameVerb` getter + `RenamesInCommit` exported function — REFACTOR per reviewer N-2). The check rule consumes via composition; no duplicated map.
- **Walker error handling is fail-shut.** Transient git subprocess failures (lock contention, permissions) degrade to "no records, no error" — consistent with `WalkCherryPicks`; the chokepoint is one rule among many and a transient git hiccup shouldn't block the check pass.
- **M-0158 cell `branch-cell-id-rename-untrailered`** registered in [`internal/workflows/spec/branch/rules.go`](../../../internal/workflows/spec/branch/rules.go) for drift policy compliance (every `ClassBranchChoreography` code requires a cell per M-0158/AC-6).
- **M-0158/AC-5 meta-coverage keyword** `IDRenameUntrailered` registered so the test-name discoverability policy recognizes the rule's tests.

**Hint table entry** at [`internal/check/hint.go`](../../../internal/check/hint.go) names the canonical resolution (`aiwf reallocate <new-id-or-path>` — rewrites cross-references and bridges `aiwf history`) AND the sovereign-human override (`aiwf acknowledge-illegal <sha> --reason "..."` — records a separate audit-trail commit without rewriting history).

**Sabotage-verified discrimination** at three layers:
- Unit-level (rule's ackedSHAs branch): dropping `if ackedSHAs[r.SHA] { continue }` fires `TestIDRenameUntrailered_AckedSHAExempted` (got 2 findings; want 1)
- Integration-level (ack-helper-lift wire-up): same sabotage fires `TestIDRenameUntrailered_AC4_AcknowledgeIllegalSilences`
- Hint-flow (rendered Hint contains `aiwf reallocate` AND `aiwf acknowledge-illegal`): pinned at `TestRunProvenanceCheck_IDRenameUntrailered_FindingCarriesHint` (M-0106/AC-12 + M-0159/AC-9 pattern)

**References.**
- CLAUDE.md §"Id-collision resolution at merge time" (the hint that landed as this rule)
- ADR-0011 layer-4 branch-choreography carve-out
- Pinned by: 5 unit tests (`TestIDRenameUntrailered_*`) + 3 integration scenarios (`TestIDRenameUntrailered_AC4_*`) + 1 hint-flow test (`TestRunProvenanceCheck_IDRenameUntrailered_FindingCarriesHint`) + 6 walker unit tests (`TestWalkUntrailedIDRenames_*`) + `TestEntityIDFromPath` + `TestCommitHasRenameClassVerb`

