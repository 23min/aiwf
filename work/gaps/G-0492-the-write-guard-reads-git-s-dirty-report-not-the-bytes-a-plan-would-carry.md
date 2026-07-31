---
id: G-0492
title: The write guard reads git's dirty report, not the bytes a plan would carry
status: open
discovered_in: M-0283
---
## What's missing

The commit-side write guard (ADR-0038, M-0283) asks git what the operator has
changed, then checks whether the plan covers those paths. What the commit
actually carries is a filesystem walk of every path under a move. Those are
different sets, and every path in the second but not the first is committed
unexamined.

Three known instances, reached by different routes:

- Ignored files. Both halves of the dirty set exclude them by construction.
  Measured before the fix: a `.gitignore`d file inside an epic directory was
  committed by `aiwf rename` with git reporting a clean tree, becoming tracked
  from that commit onward. M-0283 closes this one by querying ignored paths
  beneath a move's prefixes.
- G-0487's set — `assume-unchanged`, `skip-worktree`, sparse checkout. Still
  open; measured intact after the guard shipped.
- Any future mechanism that makes git stop reporting a path.

Each was found separately, and the first two were each treated as their own
limit. They are one property.

## Why it matters

The guard's purpose is that no verb commits content it did not compute. Reading
the dirty set means it can only be as truthful as git's reporting, and a caller
asking 'what has the operator changed' is not asking the question the guard
needs answered — 'what will this commit carry that differs from the record'.

Closing instances one at a time leaves the next one undiscovered until someone
measures it, which is how both of the first two were found.

## Scope

The guard's input, not its verdict or its seam. Out of scope: what the guard
does once it knows a path diverges, and the commit-construction path itself.

## Resolution options

1. Compare HEAD's blobs against disk for every path a plan would carry. One
   `git ls-tree -r` over the plan's prefixes plus one `git hash-object`
   pass — the same subprocess count the guard pays today. Closes all three
   instances as one class, since none of them can hide a blob hash. Costs more
   bytes moved, and a decision about what to do with a path present on disk but
   absent from HEAD.
2. Keep adding per-mechanism queries as instances surface. Cheapest per step,
   and the record above suggests the steps keep coming.
3. Accept and document the bound as inherent, stating in the guard's own error
   message that it reflects git's view of the tree rather than the tree.