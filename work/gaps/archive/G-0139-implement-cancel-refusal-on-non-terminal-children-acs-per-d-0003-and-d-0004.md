---
id: G-0139
title: Implement cancel refusal on non-terminal children/ACs per D-0003 and D-0004
status: addressed
discovered_in: M-0123
addressed_by:
    - M-0139
---
## What's missing

Two preconditioned-illegal cells in `internal/workflows/spec/rules.go` reference
verb-time finding codes that the impl does not yet emit:

- `epic-cancel-non-terminal-children` — fires when `aiwf cancel E-NNNN` is
  invoked while any child milestone is still non-terminal (`draft` or
  `in_progress`). Per **D-0003** (committed in M-0123 phase 1) the verb refuses
  with a listing of the non-terminal children.

- `milestone-cancel-non-terminal-acs` — fires when `aiwf cancel M-NNNN` is
  invoked while any AC is still in a non-terminal status (`open`). Per **D-0004**
  the verb refuses with a listing of the non-terminal ACs.

Today both verbs cancel without consulting the child state. The kernel's status
FSM allows the transition; the spec says it should be guarded.

Both codes are listed in `internal/policies/m0123_ac5_drift_test.go`'s
`deferredImplErrorCodes` allowlist with this gap as the tracking reason. When
the impl lands, the allowlist entries come out and the M-0123/AC-5 drift test
re-binds the spec cells to the impl-side `Code: "..."` literals.

## Why it matters

Without the guards, an operator can cancel an epic mid-flight and orphan
its in-flight milestones — same for a milestone with open ACs. The spec
already encodes the rule; the impl needs to catch up so the spec stops being
aspirational.

The spec's verb-time chokepoint exists because *the user wants the cancel
rejected before it lands*. Once the cancel commit is in, downstream
tooling has to deal with a terminal entity whose children are still
in-flight — orphan milestones under a cancelled epic, orphan ACs under a
cancelled milestone. The check-time-only enforcement model means the bad
state is *recorded* in git history, which is exactly the failure mode
"framework correctness must not depend on the LLM's behavior" was meant
to prevent.

## The 4 cells

| Cell | ExpectedErrorCode | Decision |
|------|-------------------|----------|
| Epic.proposed.cancel + non-terminal child | `epic-cancel-non-terminal-children` | D-0003 |
| Epic.active.cancel + non-terminal child | `epic-cancel-non-terminal-children` | D-0003 |
| Milestone.draft.cancel + open AC | `milestone-cancel-non-terminal-acs` | D-0004 |
| Milestone.in_progress.cancel + open AC | `milestone-cancel-non-terminal-acs` | D-0004 |

## Proposed fix shape

Extend `verb.Cancel` (`internal/verb/promote.go` around line 187 where
`entity.CancelTarget` is consulted) with a cross-entity precondition
check:

```go
if err := validateCancelChildren(t, e); err != nil {
    return nil, err
}
```

Where `validateCancelChildren`:

- For `KindEpic`: enumerate child milestones via `tr.ChildrenOf(epicID)`;
  if any is non-terminal, return
  ```go
  fmt.Errorf("%s cannot be cancelled: child milestone %s has non-terminal status %q (epic-cancel-non-terminal-children, D-0003); promote or cancel the child first", e.ID, child.ID, child.Status)
  ```
- For `KindMilestone`: enumerate the milestone's ACs; if any has
  `Status == open`, return
  ```go
  fmt.Errorf("%s cannot be cancelled: AC %s has open status (milestone-cancel-non-terminal-acs, D-0004); promote or cancel the AC first", e.ID, ac.ID)
  ```

The same caveat that applies to G-0141 applies here: emitting the error
code in `fmt.Errorf` text doesn't satisfy AC-5's `Code: "..."` literal
scanner. Closing G-0139 fully requires the same structured-emission
pattern that's been deferred for all the entries in
`deferredImplErrorCodes` (G-0141 Phase 2 — possibly a new umbrella
gap/epic).

## Test surface

- Kernel-level fixture trees with:
  - One epic + one non-terminal milestone child → `aiwf cancel E-…`
    refuses with the expected code.
  - One milestone + one open AC → `aiwf cancel M-…` refuses with the
    expected code.
- Tests live at `internal/verb/cancel_invariants_test.go` (new) or
  extend `internal/verb/promote_test.go`.
- The M-0125/AC-2 entries for these 4 cells un-skip automatically once
  the verb-time guard is in place — `ac2KnownImplGaps` entries can
  come out in the same commit.

## Closing this gap

When the impl lands:

1. Remove the two entries from `deferredImplErrorCodes` in
   `internal/policies/m0123_ac5_drift_test.go`:
   - `"epic-cancel-non-terminal-children"`
   - `"milestone-cancel-non-terminal-acs"`
   (Subject to the structured-emission caveat above — if Phase 2 from
   G-0141 hasn't landed yet, the AC-5 drift policy needs the
   verb-error-message path before the entries can come out.)
2. Remove the 4 entries from `ac2KnownImplGaps` in
   `internal/policies/m0125_negative_driver_test.go`:
   - `"epic-proposed-cancel-anychildstatusnotinmilestoneterminalset"`
   - `"epic-active-cancel-anychildstatusnotinmilestoneterminalset"`
   - `"milestone-draft-cancel-anychildacstatuseqopen"`
   - `"milestone-in_progress-cancel-anychildacstatuseqopen"`
3. Promote G-0139 to `addressed` with `--by M-NNNN` (whichever milestone
   carries the impl).

## History

- **M-0123 (filed):** G-0139 filed at M-0123 wrap as a follow-up to
  D-0003 + D-0004, citing `deferredImplErrorCodes`. Original title:
  *"Implement cancel-cascade per D-0003 and D-0004"*.
- **M-0125 (refined):** G-0162 was filed at M-0125 unaware that G-0139
  already existed (duplicate). G-0162's value-add — the 4-cell table,
  the `internal/verb/promote.go::Cancel` line-number reference, the
  `validateCancelChildren` helper name, the closing-this-gap checklist
  — was merged into this body and G-0162 was cancelled as a duplicate.
- **M-0125 (retitled):** the original "cancel-cascade" wording was
  misleading — the behavior the spec actually requires is *refusal*,
  not cascading the cancel to children. Retitled to *"Implement cancel
  refusal on non-terminal children/ACs per D-0003 and D-0004"* (see
  `aiwf history G-0139` for the retitle commit) so future readers
  aren't misdirected toward auto-cancel semantics.
- **Confirmed:** M-0125/AC-2's negative driver dry-run confirmed at
  the cell level that all 4 cells are unguarded today (verb succeeds
  when spec says it should refuse with the structured code).
