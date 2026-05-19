---
id: M-0131
title: 'State-aware CancelTarget for Contract: cancel deprecated targets retired'
status: draft
prior_ids:
    - M-0127
parent: E-0033
depends_on:
    - M-0123
tdd: required
---
## Goal

Make `CancelTarget(kind)` in `internal/entity/transition.go` state-aware so that `aiwf cancel C-NNNN` on a deprecated contract targets `retired` (the natural lifecycle terminal) instead of `rejected` (an FSM-illegal target from `deprecated`). Closes gap **G-0129** (filed during M-0121's audit). Addresses the catalog's R-RULE-021 endorsement.

## The bug

Today's `CancelTarget`:

```go
case KindContract:
    return "rejected"
```

But the Contract FSM has no `deprecated → rejected` edge. So `aiwf cancel C-NNNN` on a deprecated contract fails with *"contract status `deprecated` cannot transition to `rejected`"*. The operator is stuck — they can't cancel a deprecated contract through the `aiwf cancel` verb, even though that's a legitimate lifecycle move (the terminal from `deprecated` is `retired`).

## Acceptance criteria

(Added via `aiwf add ac` once M-0123's schema is settled. Likely shape: three ACs — signature change, state-aware mapping per kind, paired tests.)

## Approach

1. **Refactor `CancelTarget` signature** from `(kind) string` to `(kind, currentStatus) string`. Five of the six kinds ignore the new parameter; Contract uses it.
2. **Per-kind switch:**

   ```go
   case KindContract:
       switch currentStatus {
       case "proposed", "accepted":
           return "rejected"
       case "deprecated":
           return "retired"
       }
       return ""  // illegal current-state; caller surfaces error
   ```

3. **Update the cancel-verb call site** to pass `entity.Status` as the new argument.
4. **Tests** under `internal/entity/transition_test.go` (and the cancel-verb integration test) covering every (kind, current-state) → cancel-target mapping, plus a negative case for `CancelTarget(KindContract, "retired")` returning `""` (already terminal).
5. **Update audit catalog**: remove "code bug" qualifier from R-RULE-021's Notes column; remove the "G-0129" qualifier from the source line.

## What this milestone does *not* do

- Does not introduce other FSM changes; the FSM tables themselves are untouched.
- Does not generalize the state-aware pattern beyond Contract — the other five kinds genuinely have single-target cancels.

## At wrap

Promote G-0129 to `addressed`:

```
aiwf promote G-0129 addressed
```

Add `addressed_by: [M-0131]` to G-0129's frontmatter in the same wrap commit.

## Related

- **G-0129** — the gap this milestone closes
- **R-RULE-021** in `legal-workflows-audit.md` — the spec entry
- **R-AUDIT-0031/0032/0033** — the per-source rules in §1
- `internal/entity/transition.go::CancelTarget`
