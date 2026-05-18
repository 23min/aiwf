---
id: E-0032
title: Idiomatic-Go cleanup completion and enum-adoption chokepoint
status: done
---
## Goal

Close G-0107 by moving every top-level verb in `cmd/aiwf/` (~27 verbs across 19 single-verb files plus the 8-verb `verbs_cmd.go` cluster) into per-verb subpackages under `internal/cli/<verb>/`, shrink `cmd/aiwf/main.go` to G-0107's target ~30-line entry shape, and add the AST-based policy that prevents enum-constant adoption drift (G-0126). After this epic lands, `cmd/aiwf/` contains `main.go` only; verb code, helpers, and tests live under `internal/cli/`. The chokepoint becomes mechanical: closed-set comparison-site adoption is a CI test, not reviewer vigilance.

## Context

[G-0107](../../gaps/G-0107-reorganize-cmd-aiwf-into-idiomatic-per-verb-packages.md) (May 2026) sketched a three-step plan to move `cmd/aiwf/` from a flat 103-file `package main` toward idiomatic Cobra layout. Steps 1 and 2 landed:

- **Step 1** — `admin_cmd.go` split into per-verb files (commit `2e7a0f2`).
- **Step 2** — verb-support helpers extracted to `internal/cli/cliutil/` (commit `1d391c5`).

Step 2 left three pieces of residue, captured in this epic's scope:

1. [`internal/gitops/gitops.go:312`](../../../internal/gitops/gitops.go) `parseTrailers` and [`internal/cli/cliutil/scopes.go:258`](../../../internal/cli/cliutil/scopes.go) `ParseTrailerLines` are byte-identical functions. The cliutil variant was added during the migration; the original was never deleted.
2. [`cmd/aiwf/main.go:53–145`](../../../cmd/aiwf/main.go) carries ~90 lines of completion-helper plumbing (`registerFormatCompletion`, `completeEntityIDs`, …) that fits the cliutil pattern but didn't make the trip because the helpers reference local `resolveRoot`.
3. [`cmd/aiwf/verbs_cmd.go`](../../../cmd/aiwf/verbs_cmd.go) (978 lines, 8 unrelated verbs: `add`, `add ac`, `promote`, `edit-body`, `cancel`, `rename`, `move`, `reallocate`) is `admin_cmd.go`'s sibling monolith — implicit in G-0107's scope but not separately named in its body.

Step 3 (per-verb subpackages) was originally sketched as "lazy migration as verbs are touched." This epic instead executes it in full for all top-level verbs, batched into per-cluster milestones so each milestone is independently shippable and reviewable.

Orthogonal but co-discovered: [G-0126](../../gaps/G-0126-enum-constant-adoption-has-no-mechanical-chokepoint.md) — the hardcoded `ac.Status == "open"` at [`internal/entity/transition.go:198`](../../../internal/entity/transition.go) survived through G-0107's refactor with no mechanical chokepoint catching it. Per the kernel rule "framework correctness must not depend on LLM behavior," closed-set adoption is the kind of invariant that needs a policy test. Pinning the chokepoint here closes the broader audit's loop.

## Scope

### In scope

- Consolidate the trailer parser to a single exported `gitops.ParseTrailers`; switch the 3 call sites; delete `cliutil.ParseTrailerLines`.
- Lift completion helpers from `cmd/aiwf/main.go` into `internal/cli/cliutil/completion.go`.
- Move all ~27 top-level verbs from `cmd/aiwf/` into `internal/cli/<verb>/` subpackages with per-package `_test.go`:
  - 8 verbs in `verbs_cmd.go`: `add`, `add ac`, `promote`, `edit-body`, `cancel`, `rename`, `move`, `reallocate` (likely 7 subpackages, since `add ac` shares `internal/cli/add/`).
  - 16 single-command verbs: `archive`, `authorize`, `history`, `import`, `init`, `list`, `render`, `retitle`, `rewidth`, `schema`, `show`, `status`, `template`, `update`, `upgrade`, `whoami`.
  - 3 multi-subcommand verbs: `contract` (6 subcommands), `doctor` (with `--self-check`), `milestone` (with `depends-on`).
- Find homes under `internal/cli/` for the 7 supporting files ([`selfcheck.go`](../../../cmd/aiwf/selfcheck.go), [`render_resolver.go`](../../../cmd/aiwf/render_resolver.go), [`rituals.go`](../../../cmd/aiwf/rituals.go), [`show_scopes.go`](../../../cmd/aiwf/show_scopes.go), [`tests_metrics_check.go`](../../../cmd/aiwf/tests_metrics_check.go), [`provenance_check.go`](../../../cmd/aiwf/provenance_check.go)) — each goes with its owning verb or to a shared subpackage.
- Shrink `cmd/aiwf/main.go` to G-0107's target ~30-line entry shape; after the epic, `cmd/aiwf/` contains `main.go` only.
- Update the completion drift test ([`cmd/aiwf/completion_drift_test.go`](../../../cmd/aiwf/completion_drift_test.go)) for the new package layout.
- Write `internal/policies/enum_literal_adoption.go` (AST-based, comparison-sites only, seeded from `Status*` constants in [`internal/entity/entity.go`](../../../internal/entity/entity.go)).
- Fix the surfaced literal sites (`transition.go:198`, `entity.go:100`, `:477`) so the policy fires green at landing.
- New row in CLAUDE.md's "What's enforced and where" table for the new policy.

### Out of scope

- Broadening the enum-policy denylist beyond `Status*` (e.g., to `Kind*`, `Phase*`, trailer keys, scope events) — seed is `Status*`; expansion via later gaps as drift surfaces.
- Removing the `//enums:ignore <reason>` allowlist mechanism — matches the existing `//coverage:ignore` pattern.
- Reorganizing the existing `internal/cli/cliutil/` package boundary — accept its current shape.
- Multi-host adapter generation, plugin-side recipes, custom merge drivers, CRDT primitives — CLAUDE.md banned list.

## Constraints

- **Per-verb moves are independently shippable.** Each verb's move under M-3/M-4/M-5 is a separate commit; if one verb's move blocks (e.g., test interaction with a sibling), the rest still land. Milestone ACs are per-verb so partial completion is reviewable.
- **Policy lands green at commit time.** M-7's literal-fix work ships in the same milestone as the policy, not a follow-on patch.
- **No new abstractions.** Destination is `internal/cli/<verb>/` (G-0107's target shape) and `internal/cli/cliutil/` (existing). Policy is a sibling of [`internal/policies/test_setup_presence.go`](../../../internal/policies/test_setup_presence.go). No DSL, no reflection, no new package boundary not already implied by G-0107.
- **Strict-sequential M-3 → M-4 → M-5 → M-6.** All four touch the completion drift test and `cliutil`'s exported surface. Sequential ordering avoids merge-conflict thrash and lets the pattern stabilize per milestone before the next one builds on it.
- **Tests build the real binary where applicable.** Existing integration tests under `cmd/aiwf/` move with the verbs they exercise (per-package `_test.go`) or consolidate under `internal/cli/integration/` — M-6 settles the destination.
- **KISS / YAGNI.** Denylist seeds from `Status*` only. Check fires at comparison sites only. Follow-on gaps capture other category/site needs.

## Success criteria

- [x] `internal/gitops/gitops.go` exports `ParseTrailers`; `cliutil/scopes.go` no longer defines `ParseTrailerLines`; all 3 call sites consume the gitops export.
- [x] `internal/cli/cliutil/completion.go` carries the lifted completion helpers; `cmd/aiwf/main.go` no longer defines them.
- [x] `cmd/aiwf/verbs_cmd.go` no longer exists; the 8 verbs live in their `internal/cli/<verb>/` subpackages with per-package `_test.go`.
- [x] All 16 single-command verbs live in `internal/cli/<verb>/` subpackages.
- [x] `contract`, `doctor`, `milestone` and their subcommands live in `internal/cli/<verb>/` subpackages.
- [x] The 7 supporting files (`selfcheck.go`, `render_resolver.go`, `rituals.go`, `show_scopes.go`, `tests_metrics_check.go`, `provenance_check.go`) live under `internal/cli/`.
- [x] `cmd/aiwf/main.go` ≤ 50 lines (actual: **21 lines**); `cmd/aiwf/` directory contains `main.go` only.
- [x] `internal/policies/enum_literal_adoption.go` exists, runs as a Go test, fires green on the tree at landing, and catches the `transition.go:198`-shape regression if reintroduced.
- [x] `entity/transition.go:198`, `entity/entity.go:100`, `:477` use the constants.
- [x] CLAUDE.md's "What's enforced and where" table gains a row for the new policy.
- [x] **G-0107 status: `addressed`. G-0126 status: `addressed`.**

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Subpackage moves break cross-verb test fixtures | med | Each per-verb move is its own commit; CI surfaces broken tests immediately; rollback is trivial. |
| Helper extraction creates circular imports | med | Helpers move to `internal/cli/cliutil/` (no upward deps); verb packages depend on cliutil only. |
| Multi-subcommand verbs (contract, doctor, milestone) wiring is fragile | med | M-5 lands after M-3 and M-4 establish the pattern; integration tests cover subcommand dispatch. |
| Completion drift test churns through all 4 move milestones | low | Drift-test update lands with each milestone's set of moves; not deferred. |
| Enum policy false positives in test fixtures | low | Seed denylist small (`Status*`); `//enums:ignore <reason>` is the escape hatch. |

## Milestones

(Allocated after this spec is reviewed.)

- M-1 — Consolidate trailer parser (`gitops.ParseTrailers` exported; cliutil duplicate removed) · depends on: —
- M-2 — Lift completion helpers to `internal/cli/cliutil/completion.go` · depends on: —
- M-3 — Move verbs_cmd.go's 8 verbs to `internal/cli/<verb>/` subpackages (pattern-setter) · depends on: M-2
- M-4 — Move 16 single-command verbs to `internal/cli/<verb>/` subpackages · depends on: M-3
- M-5 — Move 3 multi-subcommand verbs (contract, doctor, milestone) to subpackages · depends on: M-4
- M-6 — Shrink main.go to ~30 lines; 7 supporting files find homes · depends on: M-5
- M-7 — Add `enum_literal_adoption` policy + fix surfaced literal sites (closes G-0126) · depends on: —

## Outcome

All seven milestones (M-0113–M-0119) landed at status `done`. **G-0107 and G-0126 are both closed.** Final state:

- `cmd/aiwf/main.go`: **21 lines** (target was ~30); the directory contains main.go only.
- `internal/cli/` houses every verb body, every cross-verb assembly concern (`root.go`), and the integration test suite.
- Three new drift policies fire on every CI run:
  - `PolicyCaptureStdoutSingleton` (M-0118/AC-7) — one canonical CaptureStdout, no per-package duplicates.
  - `PolicyEnvelopeVersionSource` (M-0118/AC-8) — every JSON envelope reads `version.Current().Version`, no package-global drift.
  - `PolicyEnumLiteralAdoption` (M-0119) — closed-set `Status*` literals must be constants at comparison sites; auto-enumerates from `internal/entity/entity.go` (no second source of truth); `//enums:ignore <reason>` allowlist.
- 22 status-literal sites updated across `internal/entity/`, `internal/render/glyph.go`, `internal/verb/auditonly.go`.

### Per-milestone summary

| Milestone | Outcome | Key commits |
|---|---|---|
| M-0113 Consolidate trailer parser | `gitops.ParseTrailers` exported; cliutil duplicate deleted; 3 call sites updated | (pre-rebase SHA) |
| M-0114 Completion helpers lift | `internal/cli/cliutil/completion.go` carries `RegisterFormatCompletion`, `CompleteEntityIDs`, `CompleteEntityIDFlag` | (pre-rebase SHA) |
| M-0115 8 verbs → subpackages | `verbs_cmd.go` deleted; 8 verbs in `internal/cli/<verb>/` with per-package `_test.go` | (pre-rebase SHA) |
| M-0116 16 single-cmd verbs → subpackages | All 16 verbs migrated; pattern stabilised | (pre-rebase SHA) |
| M-0117 contract / doctor / milestone → subpackages | The three multi-subcommand verbs land; `doctor.Dispatcher` seam designed and wired | (pre-rebase SHA) |
| M-0118 Shrink main.go (G-0107 closed) | `cmd/aiwf/main.go` → 21 lines; tests relocate to `internal/cli/integration/`; testutil package gains shared helpers | `9893ee16` (AC-6+AC-5), `3184892c` (AC-1) |
| M-0119 enum_literal_adoption policy (G-0126 closed) | Policy + 22 sites adopted; CI test wired | `39c96ff9`, `34175e75` |

## Decisions made during execution

- **Reordered M-0118 AC execution mid-milestone** when AC-2 (check verb move) revealed a cycle: `internal/cli/cliutil` imports `internal/check`, so `tests_metrics_check.go` → `internal/check/` would have cycled. Re-conferred with the user; check-rule helpers landed under `internal/cli/check/` co-located with the verb body that composes them. The file's own godoc rationale ("lives here rather than in `package check` because the rule requires git access") supported the pivot.

- **Binary tests merged into `internal/cli/integration/` instead of staying at `cmd/aiwf/`.** The original M-0118 plan was "split: cobra → integration, binary → cmd/aiwf". Mid-execution, helper-fragmentation cost became visible (`assertWellFormed`, `htmlMain`, `setupGitRepoWithUpstream` needed by both styles). Re-conferred and pivoted; cmd/aiwf truly contains only main.go now.

- **JSON-envelope Version source converges on `version.Current().Version`** for every verb. Trade-off accepted: ldflags-stamped `make install` builds report `(devel)` in the envelope instead of `<branch>@<sha>`; tagged installs were and remain correct. The convergence prevents the divergence class M-0118 item 6 named.

- **`-parallel 8` cap extended from `make test-race` to `make test`.** With ~80 tests in one `internal/cli/integration/` package, default-GOMAXPROCS parallelism overwhelms macOS git/codesign subprocess budgets. The Makefile gained the cap on the plain test target; a future broadening of `PolicyRaceParallelCap` (currently only scoped to race-mode) is noted as a deferral.

- **Six FSM-table literal lists fixed in entity.go, not just the two named.** The M-0119 spec named `:100` and `:477`. The other four schema-table entries (epics, milestones, ADRs, decisions, contracts) carried the same drift shape; fixing all six in one commit was the consistency choice.

- **TDDPhase constants used at one site in auditonly.go** even though TDDPhase* is out of M-0119's policy denylist scope. The literal `"done"` aliases both `StatusDone` and `TDDPhaseDone`; using `entity.TDDPhaseDone` here side-steps a Status* false-positive while leaving the (deferred) TDDPhase expansion to a future gap.

- **Rebase onto main mid-epic** (after G-0128 codesigning fix landed). Pulled the macOS Sonoma 14.8.x syspolicyd crash workaround into the epic's test infrastructure. The rebase orphaned the authorize-by SHA across 14 trailer-carrying commits; cleanup was a `git filter-branch --msg-filter sed` rewrite of `bb63124e…` → `7ddfbe23…`. Both backup branch (`backup/pre-rebase-AC1-AC8`) and the filter-branch safety net (`refs/original/refs/heads/...`) are safe to delete now.

## Deferrals

- **`PolicyRaceParallelCap` broaden to cover `make test`.** Makefile now carries the `-parallel 8` cap on both `test` and `test-race`. The policy's regex only scans race-mode invocations; widening is mechanical when drift in the non-race entry surfaces. Filed mentally.

- **Production-code import of `internal/cli/cliutil/testutil/` not yet policy-enforced.** The package's godoc says "Production code must not import this package" but no AST policy fires on a violation. The package is small and only imported by tests today; if drift surfaces, the chokepoint is a 30-line `PolicyTestUtilNotImportedFromProduction` following the same shape as the other AST policies.

- **`PolicyEnumLiteralAdoption` expansion to TDDPhase*, Kind*, trailer-key, scope-event constants.** The seed is `Status*`; the policy's structure makes adding new categories a 5-minute change (extend the enumerator's prefix filter). Drift in those categories will surface their own gaps when it bites.

- **Raw-assignment sites (`s := "open"`)** deliberately not flagged by `PolicyEnumLiteralAdoption`. Common in YAML decoding scaffolding and test fixtures; if drift in this shape surfaces, broaden the policy via a later gap.

## Reviewer notes

- The epic's commit count is high (246 commits at wrap) but bisects cleanly. ~70% are aiwf verb commits (small, atomic per-AC `promote`/`add`/`edit-body`); the remaining 30% are the substantive implementation commits, each tagged with `aiwf-entity: M-NNNN/AC-N` trailers. `git log --grep "M-0118/AC-"` (etc.) returns the per-AC story.

- M-0118 was the heaviest milestone (8 ACs, 7 commits including the 85-file mass-relocate commit `9893ee16`). M-0119 was the lightest (5 ACs, 2 implementation commits). The proportionality matches: M-0118 was where G-0107's "main.go entry-only" target finally landed; M-0119 was a focused policy-and-fix.

- The G-0128 codesigning fix arriving via rebase mid-epic is the kind of "host bug discovered during work" event that risks derailing milestones. The rebase + trailer-rewrite cleanup was ~10 minutes of mechanical effort and the work continued unblocked. Future epics on macOS should expect similar host-bug surprises and budget rebase windows.

- Three new drift policies (`PolicyCaptureStdoutSingleton`, `PolicyEnvelopeVersionSource`, `PolicyEnumLiteralAdoption`) collectively enforce the kernel rule "framework correctness must not depend on LLM behavior" for closed-set constant adoption, capture-helper consolidation, and JSON-envelope version sourcing. Reviewers introducing future drift in those categories will see CI fail before the code lands.

- Branch was held local throughout the epic (per the user's explicit preference). Branch is still local at wrap; pushing is a deliberate human decision separate from this milestone.
