---
id: G-0201
title: authorize preflight carve-out accepts cross-rung ritual mismatches
status: addressed
discovered_in: M-0105
addressed_by:
    - M-0161
---
M-0105/AC-6 extended the authorize preflight's future-branch carve-out
([`internal/verb/authorize.go`](../../internal/verb/authorize.go)) from
"main + ritual --branch" to "main-or-ritual current + ritual --branch".
The extension uses a flat union — no hierarchical check between
`CurrentBranch` shape and `--branch` shape.

Consequence: the carve-out accepts syntactically valid but
ritual-incoherent combinations:

- `CurrentBranch="epic/E-0001-foo"` + `--branch="epic/E-0002-bar"`
  (different epic, same rung)
- `CurrentBranch="milestone/M-0007-cache"` + `--branch="epic/E-0009-y"`
  (up-the-tree from milestone to epic)
- `CurrentBranch="patch/g-0042-fix"` + `--branch="milestone/M-0008-z"`
  (up-the-tree from patch to milestone)
- `CurrentBranch="epic/E-0001-foo"` + `--branch="patch/g-0042-fix"`
  (epic → patch, skipping milestone)

Each combination produces an authorize commit whose `aiwf-branch:`
trailer names a branch that doesn't follow the parent-child
relationship implied by the operator's current checkout. The verb
accepts; the trailer records the operator's stated intent.

## Why parked

YAGNI. The looser check covers every legitimate ritual invocation
(`aiwfx-start-epic` step 7, `aiwfx-start-milestone` step 4) and
refuses the loudest mistakes (non-ritual current, non-ritual
--branch). A hierarchical check would be more code and require
parsing the entity hierarchy at preflight time for a narrower
window — to catch operator typos that the implicit-ritual-current
path and `--force --reason` escape valve already handle.

Reviewer flagged the looseness during M-0105 Cycle 1 review;
deliberate trade-off documented inline at the carve-out site.

## When to address

When cross-rung typos become a real incident class. The fix shape:

1. Add a `branchparse.RungOf(branch) string` helper returning
   `epic`/`milestone`/`patch`/`""`.
2. Tighten the carve-out: accept only when the (CurrentBranch
   rung, branchExplicit rung) pair is one of {(main, epic),
   (epic, milestone), (milestone, patch), (epic, patch)}.
3. Add cells to the consolidation milestone (M-0158) covering
   the rejected cases.

Until that incident class shows up, the looser check is the
right trade-off.
