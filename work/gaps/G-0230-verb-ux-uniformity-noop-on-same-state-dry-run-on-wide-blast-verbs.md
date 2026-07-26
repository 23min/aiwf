---
id: G-0230
title: 'Verb UX uniformity: NoOp on same-state + dry-run on wide-blast verbs'
status: open
priority: medium
---
## What's missing

Scoped to E-0073 / M-0281, narrowed to the NoOp-convergence half (the dry-run
half is deferred — see Out of scope). Two pieces:

1. **`verb.Result.NoOp` on same-state inputs across every mutating verb.** Today `archive`, `rewidth`, `contract bind`, `contract recipe install`, `init`, `update`, statusline-scaffold all return `NoOp` with a descriptive message when the input already equals current state. But `aiwf rename`, `aiwf retitle`, `aiwf promote`, `aiwf cancel`, `aiwf move`, `aiwf acknowledge-illegal` return a Go error instead. The discipline is half-rolled-out. Specific changes:
   - `rename` to same slug → NoOp ("already named X")
   - `retitle` to same title → NoOp
   - `promote` to current status → NoOp (guard fires only when nothing else is changing, so a same-status resolver-pointer write still mutates)
   - `cancel` of already-cancelled → NoOp
   - `move` to current parent → NoOp
   - `acknowledge-illegal` against an already-acknowledged SHA → NoOp (avoids appending duplicate empty audit commits — the limited "re-running creates duplicates" C2 smell; a correctness fix, not only UX)
2. **`internal/policies/verb_result_noop_invariant.go`** — AST-level policy test asserting every mutating verb in `internal/verb/` has at least one test case that drives it with same-state inputs and asserts `Result.NoOp == true`. Allowlist the by-design-additive verbs (`add`, `authorize-open`, `edit-body --body-file`) with a one-line rationale.

## Out of scope

Dry-run / `--apply` on wide-blast-radius rewrites (`archive`/`rewidth` have it; `aiwf reallocate` and `aiwf rename` do not) was the original second half of this gap. Deferred by decision, recorded in E-0073's Out-of-scope section: `reallocate` dry-run is YAGNI (collision-only, rare, no incident — refile if one occurs), and `rename` dry-run is rejected outright (dry-run-by-default is a regression for a hot, interactive, single-entity verb, unlike the batch sweeps that earned it).

## Why it matters

C2's verdict was Strong but flagged "no-change-returns-error" as a real UX smell: an operator who runs `aiwf promote M-0090 done` twice (e.g., once interactively, once from a forgotten script) gets a confusing error the second time instead of a clean "already done" no-op. The kernel's other guarantees (single-commit, atomic-apply, FSM-policed) make state convergence safe; the operator-facing message should match that safety. The policy test is the load-bearing piece — without it, the discipline rots back to one-of as new verbs land.

## Source

`docs/archive/pocv3/health-scorecard-2026-06-04.md` §C2 (all three recommended moves; refuting evidence list).
