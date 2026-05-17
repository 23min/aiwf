---
id: E-0032
title: Idiomatic-Go cleanup completion and enum-adoption chokepoint
status: proposed
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

- [ ] `internal/gitops/gitops.go` exports `ParseTrailers`; `cliutil/scopes.go` no longer defines `ParseTrailerLines`; all 3 call sites consume the gitops export.
- [ ] `internal/cli/cliutil/completion.go` carries the lifted completion helpers; `cmd/aiwf/main.go` no longer defines them.
- [ ] `cmd/aiwf/verbs_cmd.go` no longer exists; the 8 verbs live in their `internal/cli/<verb>/` subpackages with per-package `_test.go`.
- [ ] All 16 single-command verbs live in `internal/cli/<verb>/` subpackages.
- [ ] `contract`, `doctor`, `milestone` and their subcommands live in `internal/cli/<verb>/` subpackages.
- [ ] The 7 supporting files (`selfcheck.go`, `render_resolver.go`, `rituals.go`, `show_scopes.go`, `tests_metrics_check.go`, `provenance_check.go`) live under `internal/cli/`.
- [ ] `cmd/aiwf/main.go` ≤ 50 lines; `cmd/aiwf/` directory contains `main.go` only (plus possibly a small `doc.go`).
- [ ] `internal/policies/enum_literal_adoption.go` exists, runs as a Go test, fires green on the tree at landing, and catches the `transition.go:198`-shape regression if reintroduced.
- [ ] `entity/transition.go:198`, `entity/entity.go:100`, `:477` use the constants.
- [ ] CLAUDE.md's "What's enforced and where" table gains a row for the new policy.
- [ ] **G-0107 status: `addressed`. G-0126 status: `addressed`.**

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
