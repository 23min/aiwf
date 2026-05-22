---
id: G-0162
title: cancel verb missing cross-entity non-terminal child/AC check
status: open
---
## Problem

The `aiwf cancel` verb (`internal/verb/promote.go::Cancel`) routes through `entity.CancelTarget(kind, status)` to determine the cancel terminal, then applies the transition without checking whether the entity's children allow it. Per D-0003 (epic) and D-0004 (milestone), cancel is illegal when the entity carries non-terminal children:

- **Epic.{proposed,active}.cancel** when `any-child.status ∉ milestone-terminal-set` — `ExpectedErrorCode: "epic-cancel-non-terminal-children"`.
- **Milestone.{draft,in_progress}.cancel** when `any-child-ac.status == "open"` — `ExpectedErrorCode: "milestone-cancel-non-terminal-acs"`.

Both are spec'd as `RejectionLayer: RejectionLayerVerbTime` in `internal/workflows/spec/rules.go` (lines 91–113 and 171–192). M-0125's negative driver (`TestM0125_AC2_NegativeDriver_VerbTimeRejection`) drives the binary against fixtures that satisfy these preconditions and expects non-zero exit. Today the verb succeeds, the cancel commit lands, and only later check passes (`epic-active-no-drafted-milestones`, `acs-shape`, …) might surface a related finding — but NOT the verb-time guard the spec calls for.

## Why it matters

The spec's verb-time chokepoint exists because *the user wants the cancel rejected before it lands*. Once the cancel commit is in, downstream tooling has to deal with a terminal entity whose children are still in-flight — orphan milestones under a cancelled epic, orphan ACs under a cancelled milestone. The check-time-only enforcement model means the bad state is *recorded* in git history, which is exactly the failure mode "framework correctness must not depend on the LLM's behavior" was meant to prevent.

The 4 cells affected:

| Cell | ExpectedErrorCode | Decision |
|------|-------------------|----------|
| Epic.proposed.cancel + non-terminal child | epic-cancel-non-terminal-children | D-0003 |
| Epic.active.cancel + non-terminal child | epic-cancel-non-terminal-children | D-0003 |
| Milestone.draft.cancel + open AC | milestone-cancel-non-terminal-acs | D-0004 |
| Milestone.in_progress.cancel + open AC | milestone-cancel-non-terminal-acs | D-0004 |

## Fix outline

Extend `verb.Cancel` (`internal/verb/promote.go` around line 187 where CancelTarget is consulted) with a cross-entity precondition check:

```go
if err := validateCancelChildren(t, e); err != nil {
    return nil, err
}
```

Where `validateCancelChildren`:

- For `KindEpic`: enumerate child milestones via `tr.ChildrenOf(epicID)`; if any is non-terminal, return `fmt.Errorf("%s cannot be cancelled: child milestone %s has non-terminal status %q (epic-cancel-non-terminal-children, D-0003); promote or cancel the child first", e.ID, child.ID, child.Status)`.
- For `KindMilestone`: enumerate the milestone's ACs; if any has `Status == open`, return `fmt.Errorf("%s cannot be cancelled: AC %s has open status (milestone-cancel-non-terminal-acs, D-0004); promote or cancel the AC first", e.ID, ac.ID)`.

Add unit tests under `internal/verb/cancel_invariants_test.go` (or extend existing `internal/verb/promote_test.go`) covering both kinds + both spec scenarios. The M-0125/AC-2 entries for these cells un-skip automatically once the test passes (the `ac2KnownImplGaps` map's entries are removed in the same commit).

## Closing this gap

When the impl lands, the entries in `ac2KnownImplGaps` (in `internal/policies/m0125_negative_driver_test.go`) get removed in the same commit. CI passes → gap promotes to addressed.

## Discovered in

M-0125/AC-2 driver dry-run.

## Status

`open`.
