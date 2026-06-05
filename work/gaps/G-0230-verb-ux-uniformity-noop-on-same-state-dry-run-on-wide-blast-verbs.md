---
id: G-0230
title: 'Verb UX uniformity: NoOp on same-state + dry-run on wide-blast verbs'
status: open
---
## What's missing

Two parallel uniformity gaps in the mutating-verb surface, plus the policy test that pins the first:

1. **`verb.Result.NoOp` on same-state inputs across every verb.** Today `archive`, `rewidth`, `contract bind`, `contract recipe install`, `init`, `update`, statusline-scaffold all return `NoOp` with a descriptive message when the input already equals current state. But `aiwf rename`, `aiwf retitle`, `aiwf promote`, `aiwf cancel`, `aiwf move`, `aiwf acknowledge-illegal` return a Go error instead. The discipline is half-rolled-out. Specific changes:
   - `rename` to same slug → NoOp ("already named X")
   - `retitle` to same title → NoOp
   - `promote` to current status → NoOp
   - `cancel` of already-cancelled → NoOp
   - `move` to current parent → NoOp
   - `acknowledge-illegal` against an already-acknowledged SHA → NoOp (avoids appending duplicate empty audit commits — the limited "re-running creates duplicates" C2 smell)
2. **Dry-run / `--apply` on wide-blast-radius rewrites.** `archive` and `rewidth` are dry-run-by-default with `--apply` flipping the Plan into execution. Two more verbs have the same blast-radius shape but no dry-run: `aiwf reallocate` (rewrites every cross-reference to the renumbered id) and `aiwf rename` (today only mutates the file's slug, but if it ever grows cross-ref rewrites it inherits the same shape). Extend the pattern; share the `--apply` semantics.
3. **`internal/policies/verb_result_noop_invariant.go`** — AST-level policy test asserting every mutating verb in `internal/verb/` has at least one test case that drives it with same-state inputs and asserts `Result.NoOp == true`. Allowlist the by-design-additive verbs (`add`, `authorize-open`, `edit-body --body-file`) with a one-line rationale.

## Why it matters

C2's verdict was Strong but flagged "no-change-returns-error" as a real UX smell: an operator who runs `aiwf promote M-0090 done` twice (e.g., once interactively, once from a forgotten script) gets a confusing error the second time instead of a clean "already done" no-op. The kernel's other guarantees (single-commit, atomic-apply, FSM-policed) make state convergence safe; the operator-facing message should match that safety. The policy test is the load-bearing piece — without it, the discipline rots back to one-of as new verbs land.

## Source

`docs/pocv3/health-scorecard-2026-06-04.md` §C2 (all three recommended moves; refuting evidence list).
