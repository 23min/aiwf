---
id: M-0103
title: 'AI-side preflight: aiwf authorize refuses without ritual branch context'
status: done
parent: E-0030
depends_on:
    - M-0102
tdd: required
acs:
    - id: AC-1
      title: AI-actor authorize on main without --branch refuses (branch-context-required)
      status: met
      tdd_phase: done
    - id: AC-2
      title: AI-actor authorize with --branch <missing> refuses (branch-not-found)
      status: met
      tdd_phase: done
    - id: AC-3
      title: AI-actor authorize from ritual-shape checkout (no --branch) accepts
      status: met
      tdd_phase: done
    - id: AC-4
      title: AI-actor authorize with --branch <existing> accepts
      status: met
      tdd_phase: done
    - id: AC-5
      title: --force --reason bypasses preflight (override path)
      status: met
      tdd_phase: done
    - id: AC-6
      title: --force without --reason refuses (regression guard)
      status: met
      tdd_phase: done
    - id: AC-7
      title: Non-AI authorize is unaffected by the preflight
      status: met
      tdd_phase: done
---

## Goal

Make `aiwf authorize <id> --to ai/<agent>` refuse the dispatch when no ritual branch context is in play — either `--branch <name>` is passed naming an existing ritual-shape branch, or the current checkout is already on a recognized ritual-shape branch (matched via `internal/branchparse/` from M-0102). Refusal produces an actionable error pointing at the ritual surface to use and naming the override path explicitly.

## Context

M-0102 added the `--branch` flag, the `aiwf-branch:` trailer, and the `internal/branchparse/` package; this milestone wires the chokepoint behavior that makes [ADR-0010](../../../docs/adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md)'s AI-isolation rule enforceable at the verb level. Together with M-0106's post-hoc kernel finding, this is defense in depth — the preflight blocks the bad dispatch at the source; the kernel finding catches drift that slips through.

Human-actor `aiwf authorize` invocations are unaffected — the preflight only fires when `--to ai/<id>` is in play. Author sovereignty is preserved per ADR-0010.

## Pre-decided design

Per E-0030 §"Design decisions":

- **Branch-context detection:** accept either signal. The preflight passes when (a) `--branch <name>` was supplied and `git show-ref --verify refs/heads/<name>` succeeds, *or* (b) `git symbolic-ref --short HEAD` yields a branch whose name matches one of the ritual shapes recognized by `internal/branchparse/`. Both signals are checked; failure is "neither matched."
- **Error message text** (to be tightened during AC drafting; the seed is below):

  > *"`aiwf authorize <id> --to ai/<agent>` requires a ritual branch context. Either run `aiwfx-start-epic <id>` / `aiwfx-start-milestone <id>` first to land on a recognized ritual branch (`epic/E-NNNN-<slug>` / `milestone/M-NNNN-<slug>` / `patch/g-NNNN-<slug>`), or pass `--branch <name>` naming an existing branch. To override this preflight as a sovereign act, use `--force --reason \"<one-sentence justification>\"`."*

- **Sovereign override:** `--force --reason "..."` bypasses the preflight. The existing trailer-shape rule (`internal/gitops/trailers.go::ValidateTrailer`) refuses `--force` from an `ai/` actor and requires a non-empty `--reason` after trim — so the override is structurally human-sovereign by reuse, not by new code in this milestone.
- **Error code** (for spec-cell coverage in the consolidation milestone): `branch-context-required` (for case 1 in the epic's corner-case catalog) and `branch-not-found` (for case 2). Both surface as typed `Coded` errors per [ADR-0012](../../../docs/adr/ADR-0012-typed-coded-error-pattern-for-legality-pertinent-verb-refusals.md) so machine consumers see them in the JSON envelope.

## Out of scope

- Rituals reorder (M-0104 / M-0105).
- Kernel finding for post-hoc detection (M-0106).
- Spec-cell registration in `internal/workflows/spec/branch/` — that's the consolidation milestone's work.
- Branch *creation* — the preflight only checks existence; cutting the branch is the ritual's job.
- Any changes to the trailer key or flag itself (already shipped in M-0102).
- Changes to human-actor `aiwf authorize` flows (sovereignty preserved).

## Dependencies

- **M-0102** — provides the `--branch` flag, the trailer, and the `internal/branchparse/` helpers this milestone reads.

## Acceptance criteria

<!-- Drafted at `aiwfx-start-milestone M-0103` time. AC seed set:
1. `aiwf authorize <id> --to ai/<agent>` on main (no `--branch`) refuses with a `branch-context-required` Coded error; exit code 1; JSON envelope status=error.
2. `aiwf authorize <id> --to ai/<agent> --branch epic/E-NN-X` against a non-existent branch refuses with a `branch-not-found` Coded error.
3. `aiwf authorize <id> --to ai/<agent>` from a checkout on `epic/E-NNNN-<slug>` (no `--branch`) accepts; the trailer records the current branch.
4. `aiwf authorize <id> --to ai/<agent> --branch epic/E-NNNN-<slug>` against an existing branch accepts.
5. `aiwf authorize <id> --to ai/<agent> --force --reason "..."` bypasses the preflight (override path); commit carries `aiwf-force:` and `aiwf-reason:` trailers.
6. `aiwf authorize <id> --to ai/<agent> --force` without `--reason` refuses (existing rule, regression guard).
7. Human-actor `aiwf authorize <id> --to ai/<agent>` is *unaffected*: a human-actor invocation on main, no `--branch`, succeeds (this preflight only fires when the implicit-or-explicit actor is `ai/`, but in practice `--to ai/` is what triggers it — the preflight branches on the target agent's role, not on the verb's actor).

Note: AC-7 is the kernel-correctness guard — the chokepoint must not regress the existing legitimate human-driven AI-delegation path that doesn't pass --branch (because there isn't one yet today). Once M-0104/M-0105 land, no real-world ritual flow leaves --branch unset; this AC documents the back-compat seam during the migration window.
-->

### AC-1 — AI-actor authorize on main without --branch refuses (branch-context-required)

`aiwf authorize <id> --to ai/<agent>` with neither `--branch` passed nor a current checkout that matches a ritual shape refuses with `PreflightBranchContextRequiredError` carrying the `branch-context-required` typed kernel-code per [ADR-0012](../../../docs/adr/ADR-0012-typed-coded-error-pattern-for-legality-pertinent-verb-refusals.md). The error message quotes the current branch, names the override path (`--force --reason "..."`), and points at the ritual surfaces (`aiwfx-start-epic` / `aiwfx-start-milestone`).

**Reading-A note**: the AC's "AI-actor" wording reflects the chosen gate semantic — the preflight fires when the **target** of `--to` starts with `ai/`, not when the verb's actor does. (Today `verb.Authorize` refuses ai/* actors at the actor check, so "AI-actor" can only mean "AI-target" in practice.) See D-NNN below.

**Pinned by**:
- Verb-level: `TestAuthorize_Open_AITarget_NoBranch_NoRitualCurrent_Refuses` in `internal/verb/authorize_test.go` — asserts `entity.Code(err) == CodePreflightBranchContextRequired.ID` (structural) + override-path hint substring.
- CLI seam: `TestRunAuthorize_AITarget_OnNonRitualBranch_NoBranch_Refuses` in `internal/cli/integration/authorize_cmd_test.go` — drives the binary end-to-end on a fresh repo whose HEAD is master.

### AC-2 — AI-actor authorize with --branch <missing> refuses (branch-not-found)

`aiwf authorize <id> --to ai/<agent> --branch <name>` where `refs/heads/<name>` does not resolve refuses with `PreflightBranchNotFoundError` carrying the `branch-not-found` typed kernel-code. Branch-existence is checked via `git show-ref --verify --quiet refs/heads/<name>` at the CLI layer; the verb consumes the boolean via `AuthorizeOptions.BranchExists`. The error message quotes the typo'd branch name and names the override path.

**Pinned by**:
- Verb-level: `TestAuthorize_Open_AITarget_BranchMissing_Refuses` — asserts `entity.Code(err) == CodePreflightBranchNotFound.ID` plus message-content checks.
- CLI seam: `TestRunAuthorize_AITarget_BranchMissing_Refuses` — passes `--branch epic/E-9999-typo` against a repo without that branch.
- CLI helper unit test: `TestBranchExists_False` in `internal/cli/authorize/authorize_test.go`.

### AC-3 — AI-actor authorize from ritual-shape checkout (no --branch) accepts

`aiwf authorize <id> --to ai/<agent>` from a checkout on `epic/E-NNNN-<slug>`, `milestone/M-NNNN-<slug>`, or `patch/[Gg]-NNNN-<slug>` accepts without requiring `--branch`. The verb resolves the implicit signal — `branchparse.ParseEntityFromBranch(opts.CurrentBranch) != ""` — and **promotes the implicit binding to explicit** by writing `opts.Branch = opts.CurrentBranch`, so the authorize commit carries `aiwf-branch: <current-branch-name>`. The promotion makes the binding kernel-readable for downstream consumers (M-0106).

**Pinned by**:
- Verb-level: `TestAuthorize_Open_AITarget_ImplicitFromCurrent_AcceptsAndEmitsTrailer` — asserts `aiwf-branch` trailer is emitted with the current-branch value.
- CLI seam: `TestRunAuthorize_AITarget_ImplicitRitualBranch_AcceptsAndRecords` — drives binary on a `git checkout -b epic/E-0001-engine` repo; asserts the trailer.

### AC-4 — AI-actor authorize with --branch <existing> accepts

`aiwf authorize <id> --to ai/<agent> --branch <name>` where `refs/heads/<name>` resolves accepts, regardless of the current checkout. The explicit signal carries: `opts.Branch + opts.BranchExists=true` is sufficient on its own. The authorize commit carries `aiwf-branch: <name>` (the explicit value, not the current-branch value).

**Pinned by**:
- Verb-level: `TestAuthorize_Open_AITarget_ExplicitBranchExists_AcceptsAndEmitsTrailer` — sets `CurrentBranch: "main"` (deliberately non-ritual) and asserts the explicit signal alone passes the preflight.
- CLI seam: `TestRunAuthorize_WithBranch_EmitsTrailer` (extended from M-0102) — `git branch epic/E-0001-engine` without checkout; the explicit `--branch` flag carries.
- CLI helper unit test: `TestBranchExists_True` in `internal/cli/authorize/authorize_test.go`.

### AC-5 — --force --reason bypasses preflight (override path)

`aiwf authorize <id> --to ai/<agent> --force --reason "..."` bypasses the preflight as a sovereign act, even on a non-ritual current branch. The bypass is the `!opts.Force` short-circuit on the preflight gate at `internal/verb/authorize.go`. The authorize commit carries the full force paper trail: `aiwf-force: <reason>`, `aiwf-reason: <reason>`. Per [`docs/pocv3/design/provenance-model.md`](../../../docs/pocv3/design/provenance-model.md) and the existing trailer-coherence rules, `--force` is structurally human-sovereign (a non-human actor invoking `--force` is refused by the coherence pass, so the override path can only be exercised by a named human).

**Pinned by** (sabotage-verified — removing `!opts.Force` causes both tests to fail with `branch-context-required`):
- Verb-level: `TestAuthorize_Open_AITarget_ForceReasonBypassesPreflight` — pins success on non-ritual `CurrentBranch` + the `aiwf-force` / `aiwf-reason` trailers + absence of `aiwf-branch` (the implicit-promotion does not run under override).
- CLI seam: `TestRunAuthorize_AITarget_ForceReasonBypassesPreflight` — drives binary end-to-end on master.

### AC-6 — --force without --reason refuses (regression guard)

`aiwf authorize <id> --to ai/<agent> --force` without `--reason` refuses with an error message naming `--reason`. The pre-existing rule lives at both layers:
- CLI: `internal/cli/authorize/authorize.go` returns `cliutil.ExitUsage` early.
- Verb: `internal/verb/authorize.go` returns a plain `fmt.Errorf` after the terminal-status check.

The AC pins the **error-message-identity invariant** for the operator-visible surface: an operator with `(Force=true, Reason="")` sees `--reason` named in the error, not `branch-context-required`. The preflight's `!opts.Force` short-circuit guarantees this regardless of literal source order between the force-requires-reason check and the preflight. A reorder that preserved `!opts.Force` would NOT regress AC-6. What WOULD: dropping the `!opts.Force` clause from the preflight (the preflight then fires for ai/* + non-ritual branch and the operator sees `branch-context-required` instead of `--reason`).

**Whitespace-only reason**: extending the existing `TestAuthorize_Open_ForceRequiresReason` to table-driven shape with `empty-reason` AND `whitespace-only-reason` sub-cases — the original test's comment promised the whitespace case but the test omitted it.

**Pinned by** (sabotage-verified at both layers independently):
- Verb-level: `TestAuthorize_Open_ForceRequiresReason` (extended) + `TestAuthorize_Open_AITarget_ForceWithoutReason_RefusesWithReasonError` (the non-terminal variant that exercises the error-message identity under non-ritual branch conditions).
- CLI seam: `TestRunAuthorize_AITarget_ForceWithoutReason_RefusesWithReasonError` — asserts exit code value (`cliutil.ExitUsage`), presence of `--reason`, absence of `branch-context-required`.

**Honest limit on layer-attribution**: the CLI seam test asserts exit code is `ExitUsage`. Both layers' gates produce `ExitUsage` via `cliutil.FinishVerb`'s "any other verb error → ExitUsage" mapping, so the assertion does NOT distinguish which gate fired — it pins that *some* gate refused with the right exit-code shape. The verb-level test pins the verb-side gate independently.

### AC-7 — Non-AI authorize is unaffected by the preflight

The preflight is structurally gated to `AuthorizeOpen` (the verb's open-a-fresh-scope mode) with an `ai/*` target. Pause / resume modes (`AuthorizePause`, `AuthorizeResume`) route through `authorizeTransition` and never enter the preflight code path. Non-AI `--to` targets (e.g. `bot/dependabot`) on the open path also do not enter the preflight gate.

The protection is doubled at the CLI dispatcher: `opts.Agent` is populated only in the `case to != "":` arm — pause/resume invocations never carry an Agent value, which is the second of the two gates protecting them from the preflight. A refactor that filled `opts.Agent` for pause/resume (e.g., to thread scope-context into transitional commits) would, in combination with a verb-side leak of the preflight to non-Open modes, regress AC-7. The combined regression is caught by the AC-7 CLI seam test.

**Pinned by** (sabotage-verified at the structural seam — injecting a refusal into `authorizeTransition` fails both tests):
- Verb-level: `TestAuthorize_PauseResume_DoNotTriggerPreflight` — table-driven pause + resume sub-cases on non-ritual `CurrentBranch`; asserts success + `aiwf-scope: paused`/`resumed` trailer + absence of `aiwf-branch` (no implicit promotion) + absence of `aiwf-force` (no leakage of Force into the transition emission path).
- Non-AI target side already pinned by `TestAuthorize_Open_NonAITarget_NoBranch_NoTrailer` and `TestAuthorize_Open_NonAITarget_BranchMissing_Accepted` (added in Cycle 1).
- CLI seam: `TestRunAuthorize_PauseResume_NonRitualBranch_Accepts` — opens the initial scope under `--force --reason` so the test repo can stay on master throughout; then drives `--pause` and `--resume` on the same non-ritual branch.

## Work log

One TDD cycle per AC or AC-group, each landing as a single commit on `epic/E-0030-branch-model-chokepoint`. Phase timeline (red → green → done) is in `aiwf history M-0103/AC-<N>`; this section captures outcomes and SHAs.

### Cycle 1 — AC-1 + AC-2 + AC-3 + AC-4 — core preflight refusal/accept

Implementation + tests landed together (the four ACs share one structural change set). Verb-side: added `CodePreflightBranchContextRequired` + `CodePreflightBranchNotFound` typed codes (per [ADR-0012](../../../docs/adr/ADR-0012-typed-coded-error-pattern-for-legality-pertinent-verb-refusals.md)) + `PreflightBranchContextRequiredError` / `PreflightBranchNotFoundError` types + `CurrentBranch` and `BranchExists` fields on `AuthorizeOptions` + the preflight gate in `authorizeOpen` (gated on `strings.HasPrefix(agent, "ai/") && !opts.Force`). CLI-side: added `currentBranch()` (git `symbolic-ref --short HEAD`) and `branchExists()` (git `show-ref --verify --quiet refs/heads/<name>`) helpers + `Run` plumbing to populate `opts.CurrentBranch` and `opts.BranchExists` on the `--to` path. Spec-side: two new `GlobalRules()` entries naming the codes so the M-0123/AC-5 legality-codes-referenced drift arm stays satisfied (predicates are scaffold; M-0158 elaborates).

Test set: 4 verb-level AC tests + 3 CLI-seam integration tests + 5 CLI-helper unit tests. Plus regression-guard `TestAuthorize_Open_NonAITarget_BranchMissing_Accepted` (added during reviewer follow-up) pinning the M-0102 invariant for non-AI targets.

Side-effect: 15 existing integration tests using `aiwf authorize --to ai/...` updated to either `git checkout -b epic/E-NNNN-...` before the call or `--force --reason` (one case, `TestScenario_RepeatedPauseResumeCycle`, used the override path to preserve an unrelated `aiwf check` substring assertion).

· commit `7d5eefdd` · 20 files, +825/-54

### Cycle 2 — AC-5 — --force --reason bypasses preflight

Pure test addition. The `!opts.Force` short-circuit on the preflight gate landed as part of Cycle 1's implementation; Cycle 2 pins the override behavior with verb-level + CLI-seam regression tests. Sabotage-verified: removing the `!opts.Force` clause causes both tests to fail with `branch-context-required`.

· commit `6b8cfdc8` · 2 files, +110

### Cycle 3 — AC-6 — --force without --reason refuses (regression guard)

Pure test addition. Pre-existing CLI-side + verb-side force-requires-reason gates. Extended the existing `TestAuthorize_Open_ForceRequiresReason` with whitespace-only-reason sub-case (the original comment promised it; the test omitted it). Added a non-terminal variant `TestAuthorize_Open_AITarget_ForceWithoutReason_RefusesWithReasonError` that pins the error-message-identity invariant under non-ritual branch conditions. CLI-seam companion drives the binary; sabotage-verified at both layers independently.

· commit `c81a1511` · 2 files, +108/-5

### Cycle 4 — AC-7 — pause/resume preflight Open-only gating

Pure test addition. Pause and resume succeed on non-ritual branches because the preflight is structurally gated to `AuthorizeOpen`. Sabotage-verified by injecting a refusal into `authorizeTransition` itself (the first sabotage attempt — moving the preflight to top of `Authorize` keyed on `opts.Agent` — didn't fire because `opts.Agent` is empty for pause/resume; the second sabotage at the helper level fired correctly). Also tightened AC-6 commentary per reviewer feedback (changed "gate-ordering invariant" to "error-message-identity invariant" in production comment and test preamble), added an `aiwf-force` absence assertion to the AC-7 verb test (belt-and-suspenders), added an exit-code assertion to the AC-6 CLI seam, and documented the `opts.Agent` structural invariant at the CLI dispatcher (the second gate protecting pause/resume).

· commit `7ec1b950` · 4 files, +223/-24

## Decisions made during implementation

- **Reading-A locked at start of milestone**: the preflight gates on the **target** of `--to` starting with `ai/` (not on the verb's actor). The verb's actor is always `human/` per `verb.Authorize`'s line 108; reading-A is therefore the only semantically meaningful interpretation. AC-7's title was retitled from "Human-actor authorize is unaffected" to "Non-AI authorize is unaffected" (commit `3865902e`) and the AC body's contradictory "succeeds" sentence was replaced with the structural pause/resume + non-AI-target framing.

- **The implicit-current-branch signal is *promoted* to explicit**: when AC-3's condition holds (no `--branch` + ritual current checkout), the verb writes `opts.Branch = opts.CurrentBranch` so the authorize commit carries `aiwf-branch: <current-branch-name>`. Without this promotion, downstream consumers (M-0106 in particular) would have to walk back through history to learn the binding. The trailer-emission contract carries forward the kernel's "history is the audit trail" principle ([CLAUDE.md](../../../CLAUDE.md) §"What aiwf commits to").

- **Two new `GlobalRules()` entries are scaffold-quality**: the predicates (`target-agent-role`, `ritual-branch-context-present`, `force`, `branch-flag-resolves`) are not consumed by any current driver — they pin the codes for the M-0123/AC-5 legality-codes-referenced drift arm. M-0158 elaborates them into the full branch-choreography cell set per [ADR-0011](../../../docs/adr/ADR-0011-legal-workflow-spec-methodology.md) §"Scope". The doc comment at `internal/workflows/spec/rules.go`'s `GlobalRules()` is explicit about this.

- **Cellcoverage fixture stamps a fictional `aiwf-branch` value**: `internal/cellcoverage/AuthorizeScope` now sets `CurrentBranch: "epic/" + entityID + "-cellcoverage-fixture"` so cell-coverage tests pass the preflight. Trade-off recorded as [G-0197](../../gaps/G-0197-cellcoverage-fixture-stamps-fictional-aiwf-branch-trailer-value.md); M-0106 will address.

- **`m0147_global_rule_test.go`'s helper now selects by code, not slice index**: `GlobalRules()` no longer has length 1 (the M-0103 entries joined). `globalScopeReachRule(t)` updated to find the scope-reach rule by `ExpectedErrorCode == "provenance-authorization-out-of-scope"`.

## Validation

```
go test -race ./...            # clean
golangci-lint run              # 0 issues
go build -o /tmp/aiwf ./cmd/aiwf  # ok
aiwf check                     # 0 errors, 1 env-shape warning (provenance-untrailered-scope-undefined)
```

Two independent reviewer subagents ran against the cycle output:
- **Cycle 1 review**: approved with one recommended change (add `TestAuthorize_Open_NonAITarget_BranchMissing_Accepted` to pin the M-0102 non-AI invariant) — implemented + sabotage-verified before Cycle 1 commit.
- **Cycle 3+4 review**: approved with one comment correction (the AC-6 invariant is **error-message-identity**, not **gate-ordering** — a reorder that preserves `!opts.Force` does not regress AC-6) + two minor additions (AC-6 exit-code assertion, `opts.Agent` invariant documentation). All addressed in the Cycle 4 commit before commit gate.

## Deferrals

Two gaps filed during the milestone, both with `--discovered-in M-0103`:

- [G-0197](../../gaps/G-0197-cellcoverage-fixture-stamps-fictional-aiwf-branch-trailer-value.md) — cellcoverage fixture stamps a fictional `aiwf-branch` trailer value. Scope target: M-0106 (the fictional value will interact with the upcoming kernel `isolation-escape` finding's "scope's branch vs commit's branch" comparison).
- [G-0198](../../gaps/G-0198-branchparse-regex-accepts-prefix-id-mismatch-epic-m-milestone-e.md) — `branchparse` regex accepts prefix-id mismatch (`epic/M-...`, `milestone/E-...`). Out of E-0030 scope; the consequence today is silent miscorrelation in the `aiwf status --worktrees` view for hand-typo'd branches. Worth filing for hygiene; defer to a standalone post-E-0030 follow-up.

## Reviewer notes

- **Gate-ordering vs error-message-identity (AC-6)**: the production code's refusal order is terminal-status → force-requires-reason → preflight. The order is preserved for *readability* (cheapest-cause-first) but is NOT pinned by any test. What IS pinned: the operator-visible error message identity, guaranteed by the preflight's `!opts.Force` short-circuit independent of literal source order. The production comment at the preflight site is honest about this distinction. A refactor that reorders the gates is fine; one that drops `!opts.Force` breaks the AC.

- **`opts.Branch = opts.CurrentBranch` mutates by-value `opts`**: safe today because `opts` is passed by value into `authorizeOpen`. If `opts` is ever migrated to a pointer (e.g., to avoid copy cost on a future grown struct), the mutation would leak back to the CLI caller. Inline comment at the mutation site documents the constraint.

- **Layer-attribution in AC-6 CLI seam**: the test pins exit code value (`cliutil.ExitUsage`) + presence of `--reason` + absence of `branch-context-required`. Both CLI-side and verb-side force-requires-reason gates produce `ExitUsage` via `cliutil.FinishVerb`, so the seam test does NOT distinguish which gate fired. The verb-level test pins the verb-side gate independently; the CLI-level test pins the binary-level shape of the refusal.

- **Two new `GlobalRules()` entries are scaffold-quality**: their predicates (`target-agent-role`, `ritual-branch-context-present`, `force`, `branch-flag-resolves`) are not consumed by any current driver. They satisfy the M-0123/AC-5 legality-codes-referenced drift arm; M-0158 elaborates them. This is recorded in the comment at `GlobalRules()`'s top.

- **The AC body sections in this spec were populated at wrap time** per the M-0102 precedent. The pre-existing HTML-comment AC-seed at lines 77-87 of the spec body was preserved as planning archaeology (it describes the original "AC-7 succeeds" reading-B assumption that was later replaced by reading-A; the comment is now slightly stale relative to the AC-7 body section but kept for historical context).
