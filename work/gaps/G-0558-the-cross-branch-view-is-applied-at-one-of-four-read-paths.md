---
id: G-0558
title: The cross-branch view is applied at one of four read paths
status: open
priority: medium
---
## What's missing

Four read paths load the planning tree, through two different loaders, and only
one of them builds the cross-branch view. Nothing declares which answer is
canonical, and the surfaces disagree.

`cliutil.LoadTreeWithTrunk` runs `trunk.ScanCrossBranch` and populates
`CrossBranchHits`, which is the tier ADR-0030 added so a reference to an id
living on an unmerged sibling branch resolves as pending rather than unresolved.
Bare `tree.Load` does not.

| path | loader | cross-branch view |
|---|---|---|
| `aiwf check` | `LoadTreeWithTrunk` (`internal/cli/check/check.go:98`) | yes |
| `aiwf check --fast` | `tree.Load` (`check.go:370`) | no |
| `aiwf check --shape-only` | `tree.Load` (`check.go:444`) | no |
| `aiwf status` | `tree.Load` (`internal/cli/status/status.go:336`) | no |

Measured 2026-08-06 against this repo's own tree, same binary, same working
copy: `aiwf check` reports 0 errors and 16 warnings; `aiwf check --fast` reports
2 errors and 14 warnings; `aiwf status` agrees with `--fast`. The two errors are
`body-prose-id/unresolved` findings that the full check classifies as
`cross-branch-pending` warnings instead, because it can see the sibling branch
and the others cannot.

`--shape-only` is what the pre-commit hook runs, and `--fast` is documented in
its own comment as a cheaper approximation of the authoritative pre-push gate.

## Why it matters

A fast approximation that is *stricter* than the gate it approximates produces
false alarms, and false alarms train an operator to stop reading the surface.
`--fast` reports blocking errors the pre-push check will not raise on the same
bytes; someone who runs it before committing is told to go fix prose that no
gate will ever object to.

`aiwf status` inherits the same verdict and ends its Health line by directing
the reader to `aiwf check` for details — the one surface that will tell them
nothing is wrong. Two local surfaces render opposite verdicts and the pointer
between them leads from the strict one to the permissive one.

The divergence is invisible. No surface reports that it loaded without the
cross-branch view, so the difference reads as a finding appearing or
disappearing rather than as two questions being asked.

## Resolution shape

The decision is which loader is canonical for a read path, and it has a real
cost on one side: `trunk.ScanCrossBranch` walks git refs, `--fast` exists to
skip cost, and G-0157 already tracks status's git subprocess fan-out as a
performance concern. Converging everything on `LoadTreeWithTrunk` removes the
divergence and spends that cost at three more call sites.

The cheaper shape worth weighing against it: keep the loaders as they are and
have any path without the cross-branch view say so, and decline to raise
`unresolved` at error severity — a rule whose blocking verdict depends on a view
the caller did not build is reporting a question it cannot answer.

## Independence

This does not wait on G-0556. That gap asks which answer is authoritative for a
tree about to leave the machine — a policy question about the push boundary.
This one holds under every answer to it: four read paths in one binary should
not disagree about the verdict, whichever verdict is chosen. Fixing this first
also makes G-0556's options easier to state, since each currently has to be
written without assuming which loader a surface used.

G-0556's own body describes this divergence as `aiwf status` disagreeing with
`aiwf check`. That is narrower than what is measured above: the split runs
through `aiwf check` itself, and `--shape-only` puts it in the pre-commit hook.

## Related

- G-0556 — which answer is authoritative for a departing tree; the policy
  question this one is independent of
- G-0536 — no CI backstop for `aiwf check`; a CI position becomes a fifth read
  path and inherits whichever loader it is given
- G-0157 — the git subprocess fan-out in status, which is the cost side of
  converging on the trunk-aware loader
