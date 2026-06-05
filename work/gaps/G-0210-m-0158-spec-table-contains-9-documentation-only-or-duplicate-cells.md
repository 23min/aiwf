---
id: G-0210
title: M-0158 spec table contains 9 documentation-only or duplicate cells
status: addressed
discovered_in: M-0158
addressed_by:
    - M-0162
---
M-0158's AC-2 specifies: *"Each epic corner-case (1-12)
represented as named cell branch-cell-N"*. AC-3 specifies:
*"Each override-surface row represented as a named override cell"*.

Following the literal spec body produced 16 cells. The
post-third-pass honest-scope audit identified that **9 of 16
cells (56%) carry no mechanical weight**:

| Cells | Count | Reason no mechanical weight |
|---|---|---|
| `branch-cell-3, 5, 6, 9, 11` | 5 | Legal corner cases that aren't overrides; pure documentation that "this case is OK." No drift policy reads them. |
| `branch-cell-8, 10` | 2 | Legal corner cases that ARE overrides; semantic duplicates of `branch-cell-override-cherry-pick` / `branch-cell-override-force-amend`. |
| `branch-cell-override-cherry-pick, branch-cell-override-force-amend` | 2 | Override-named cells; semantic duplicates of `branch-cell-8` / `branch-cell-10`. |

Only **7 cells pull weight**:
- 5 Illegal cells (`branch-cell-1, 2, 4, 7, 12`) — drift policy (M-0158/AC-6) enforces them.
- 2 standalone override cells (`branch-cell-override-preflight, branch-cell-override-f-nnnn-waiver`) — document override mechanisms not duplicated by corner cases.

## Why the over-specification happened

The AC framing treated the corner-case enumeration and the
override-surface table as **parallel** cell enumerations. They are
not parallel; they are **overlapping**:

- Corner case 8 (cherry-pick) IS the override-cherry-pick mechanism.
- Corner case 10 (force-amend) IS the override-force-amend mechanism.
- Override-preflight is a NEW mechanism not in the corner-case list.
- Override-f-nnnn-waiver is a NEW mechanism not in the corner-case list.
- The 5 documentation-only Legal corner cases (3, 5, 6, 9, 11) are
  not overrides at all.

The literal AC reading inflated the cell count to satisfy the
spec body without questioning whether the framing was honest.

## What's needed

Refactor `branch.Rules()` to the mechanical-weight-only catalog:

```
branch-cell-1           — illegal: branch-context-required
branch-cell-2           — illegal: branch-not-found
branch-cell-4           — illegal: isolation-escape (commit on main)
branch-cell-7           — illegal: isolation-escape (different ritual)
branch-cell-12          — illegal: isolation-escape (worktree mismatch)
branch-cell-override-preflight        — legal: --force --reason override
branch-cell-override-f-nnnn-waiver    — legal: F-NNNN waiver pattern
```

= 7 cells. Drop the other 9. Retitle M-0158 AC-2 and AC-3 to match.

## Why parked

The honest audit happened during M-0158 wrap. The user chose to
ship M-0158 with the over-specified catalog and address the
refactor in a follow-up real-world hardening milestone. The
documentation cost is recorded here so the refactor's scope is
explicit.

## Why this matters

The over-specification isn't a runtime correctness bug — the
documentation-only cells don't break anything. The cost is:

1. **Test signal weakness**: AC-5's keyword set double-maps
   cell-8 and cell-override-cherry-pick to the same test. A test
   deletion fires AC-5 for both cells; the "which view broke?"
   signal is lost.
2. **Catalog noise**: Future readers see 16 cells and don't know
   which carry weight. The five documentation-only cells could be
   removed and the catalog would still serve its purpose.
3. **Methodology drift**: The ADR-0011 spec-table methodology
   exists to enumerate **kernel-enforced** behaviors. Including
   documentation-as-cell undermines the methodology's value.

## Out of scope

The 5 illegal cells and 2 standalone override cells are correct
and stay. This gap is only about removing the 9 documentation-only
and duplicate cells.
