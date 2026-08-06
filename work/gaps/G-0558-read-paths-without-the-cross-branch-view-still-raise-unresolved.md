---
id: G-0558
title: Read paths without the cross-branch view still raise unresolved
status: addressed
priority: medium
addressed_by_commit:
    - b0975f2f6bc3ab73600e81ec8abf9c421d96d531
---
## What's missing

Read paths load the planning tree through two different loaders, and the ones
that skip the cross-branch view still raise a finding that only the view can
justify.

`cliutil.LoadTreeWithTrunk` runs `trunk.ScanCrossBranch` and populates
`CrossBranchHits`, which is the tier ADR-0030 added so a reference to an id
living on an unmerged sibling branch resolves as pending rather than unresolved.
Bare `tree.Load` does not.

| path | loader | cross-branch view | raises the finding |
|---|---|---|---|
| `aiwf check` | `LoadTreeWithTrunk` (`internal/cli/check/check.go:98`) | yes | yes |
| `aiwf add` | `LoadTreeWithTrunk` (`internal/cli/add/add.go:192`) | yes | yes |
| `aiwf check --fast` | `tree.Load` (`check.go:370`) | no | yes |
| `aiwf status` | `tree.Load` (`internal/cli/status/status.go:336`) | no | yes |
| `aiwf show` | `tree.Load` (`internal/cli/show/show.go:109`) | no | yes |
| `aiwf render` | `tree.Load` (`internal/cli/render/render.go:309`) | no | yes |
| `aiwf check --shape-only` | `tree.Load` (`check.go:449`) | no | no |
| `promote`, `cancel`, `retitle`, `rename`, `move`, `milestone`, `add ac` | `tree.Load` | no | yes |

`aiwf check --shape-only` is the row that does not belong to this gap: it runs
`check.TreeDiscipline`, which reports stray files and nothing else, so it never
reaches a reference rule. The pre-commit hook is therefore untouched by any of
this, and the cost of converging it is not a cost this decision pays.

The write path is not a settled counterexample either. `internal/verb/add.go:177`
builds `check.BodyProseIDIndex` and refuses the add when `ScanBodyProseID`
reports an error, against a tree loaded with the view — but `aiwf add` is the
exception among mutating verbs rather than the rule. `promote`, `cancel`,
`retitle`, `rename`, `move`, `milestone`, and the `add ac` path all load bare, and the
verb layer suppresses a mutation's plan when the projection introduces an
error-severity finding the tree did not already carry. So a verb whose own write
creates the reference refuses on a verdict reached without the view, which cuts
the opposite way from the read paths: not a false alarm, a false refusal.

Measured 2026-08-06 against this repo's own tree, same binary, same working
copy: `aiwf check` reports 0 errors and 16 warnings; `aiwf check --fast` reports
2 errors and 14 warnings; `aiwf status` agrees with `--fast`. The two errors are
`body-prose-id/unresolved` findings that the full check classifies as
`cross-branch-pending` warnings instead, because it can see the sibling branch
and the others cannot.

`--shape-only` is what the pre-commit hook runs, and `--fast` is documented in
its own comment as a cheaper approximation of the authoritative pre-push gate.

## Why it matters

`unresolved` is a claim about every tier — working tree, trunk, and the
cross-branch view. A loader that built only the first cannot substantiate it.
The three ref-less paths are asserting a finding they have no evidence for, and
that is the whole mechanism of the disagreement.

A fast approximation that is *stricter* than the gate it approximates produces
false alarms, and false alarms train an operator to stop reading the surface.
`--fast` reports blocking errors the pre-push check will not raise on the same
bytes; someone who runs it before committing is told to go fix prose that no
gate will ever object to, and that the verb layer accepted when it was written.

`aiwf status` inherits the same verdict and ends its Health line by directing
the reader to `aiwf check` for details — the one surface that will tell them
nothing is wrong. Two local surfaces render opposite verdicts and the pointer
between them leads from the strict one to the permissive one.

The divergence is invisible. No surface reports that it loaded without the
cross-branch view, so the difference reads as a finding appearing or
disappearing rather than as two questions being asked.

## Measured 2026-08-06 — what the view costs

Isolated by timing two verbs that load with `LoadTreeWithTrunk` and then refuse
on an unresolvable argument, against one that loads bare and formats output.
Mean of three warm runs, on a tree of 1023 entities across 5 refs:

| invocation | loader | |
|---|---|---|
| `aiwf list` | `tree.Load` | 467 ms |
| `aiwf rename-area` (refused) | `LoadTreeWithTrunk` | 955 ms |
| `aiwf acknowledge mistag` (refused) | `LoadTreeWithTrunk` | 914 ms |

The view costs roughly 470 ms here — about 3.5× what walking the refs alone
accounts for, so the weight is in reading blob content per entity per ref for
the collision classification rather than in the ref walk. Expect it to scale
with entities × refs.

Converging the three ref-less paths on `LoadTreeWithTrunk` would therefore cost
about 470 ms each: `--shape-only` 461 ms → ~930 ms, doubling the pre-commit
hook; `--fast` 810 ms → ~1280 ms; `status` 2187 ms → ~2660 ms.

Worth recording because it is counter-intuitive: full `aiwf check` takes 6786 ms,
and almost none of that is the view. `git log HEAD` over 9400 commits measures
122 ms, so the bulk is the provenance and history rules. Adding the view to a
read path does not make it anything like as slow as the full check.

## Resolution shape

**Have a read-only path without the cross-branch view decline to raise
`unresolved` at error severity.** It costs nothing, it removes the disagreement,
and it is what the evidence supports rather than merely what is cheap: the
finding claims something about tiers the caller never built. The paths keep
their current loaders and their current speed.

The downgrade belongs at the reporting surface, not in the resolution rules.
The rules are shared with the verb layer, where a newly-introduced
error-severity `unresolved` is what suppresses a mutation's plan — moving the
gate there would let a verb commit a reference resolving nowhere. A surface that only prints may decline to
make a claim it cannot support; a surface that acts on the claim has to build
the evidence instead. `check.MarkUnverifiedResolution` is that pass, applied by
`--fast`, `status`, `show` and `render` to their own findings, and it is inert on a tree whose
`CrossBranchScanned` is set — so a caller that later switches loaders stops
downgrading without a matching edit.

The false refusal on the write path is the other half of the same divergence
and is tracked separately: it wants the opposite remedy (build the view, do not
soften the verdict), and it changes how the bare-loading verbs load.

Convergence on `LoadTreeWithTrunk` is the alternative and is measured above. It
buys the same consistency for about 470 ms per path, doubling the pre-commit
hook to do it, and G-0157 already tracks status's git fan-out as a cost concern.
It is the more expensive way to reach the same place.

Declining to claim `unresolved` removes a signal as well as a false one, so the
signal wants a deliberate home rather than an accidental one: a policy test that
builds the ref-less view on purpose and asserts the tree is self-contained. That
pays the ~470 ms once, in a suite already measured in minutes, and off every
interactive path. It also retires an accident — `TestPolicy_ThisRepoTreeIsClean`
currently catches ref-less `refs-resolve` errors as a side effect of being a test
about id widths that happens to load bare, and would lose that coverage the
moment someone scopes it to its stated subject.

## Independence

This does not wait on G-0556. That gap asks which answer is authoritative for a
tree about to leave the machine — a policy question about the push boundary.
This one holds under every answer to it: paths in one binary should not
contradict each other about a verdict, whichever verdict is chosen. Fixing this
first also makes G-0556's options easier to state, since each currently has to
be written without assuming which loader a surface used.

G-0556's own body describes this divergence as `aiwf status` disagreeing with
`aiwf check`. That is narrower than what is measured above: the split runs
through `aiwf check` itself, `--shape-only` puts it in the pre-commit hook, and
the write path sits on the other side of it.

## Related

- G-0556 — which answer is authoritative for a departing tree; the policy
  question this one is independent of
- G-0536 — no CI backstop for `aiwf check`; a CI position becomes another read
  path and inherits whichever loader it is given
- G-0157 — the git subprocess fan-out in status, which is the cost side of the
  convergence option this gap declines
