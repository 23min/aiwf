---
id: G-0563
title: Bare-loading verbs refuse a reference that resolves only on another branch
status: open
priority: low
---
## What's missing

Most mutating verbs load the planning tree with bare `tree.Load`, which builds
no cross-branch view. The verb layer then suppresses a mutation's plan when the
projection introduces an error-severity finding. So a verb that writes a
reference to an id living only on an unmerged branch refuses it — on a verdict
reached without consulting the tier that would have resolved it.

Measured 2026-08-06 against a binary built from unmodified mainline, in a repo
whose `ADR-0002` exists only on a branch named `sibling`:

```
$ aiwf promote ADR-0001 superseded --superseded-by ADR-0002
error refs-resolve/unresolved: adr field "superseded_by" references unknown id "ADR-0002"
$ echo $?
1
```

No commit lands; the entity keeps its prior status. Against the same working
copy, `aiwf check` classifies that reference as a warning:

```
refs-resolve (warning) — adr field "superseded_by" references "ADR-0002",
known only on refs/heads/sibling (not yet merged into this branch)
```

`aiwf add` is the exception that shows the fix is available: it loads through
`cliutil.LoadTreeWithTrunk`, so the same reference reaches it as
`cross-branch-pending` and the add proceeds. `promote`, `cancel`, `retitle`,
`rename`, `move`, `milestone`, and the `add ac` path all load bare.

## Why it matters

This is the write-side face of the divergence G-0558 addressed on the read side,
and it wants the opposite remedy. A reporting surface that cannot substantiate
`unresolved` should decline to claim it. A verb that *acts* on the claim cannot
decline — refusing is an action — so it has to build the evidence instead.

The refusal is fail-safe: nothing is written, nothing is corrupted. What it costs
is the operator's ability to record a decision the policy permits. ADR-0041
classifies a reference to a published branch as a validated pointer anyone can
follow, and ADR-0030 established the pending tier precisely so in-flight work
need not wait for a merge. A verb that refuses such a reference reintroduces the
wait those decisions removed, and its message names the id as unknown when the
tool can demonstrate otherwise.

Frequency is currently zero. No supersession has ever been recorded in this
repo, and `depends_on` edges run between milestones of one epic, which share a
branch. The reason to expect that to change is ADR-0041: it exists to make
cross-branch references routine, and this repo's default multi-worktree workflow
produces exactly the conditions.

## Resolution shape

Have the verbs that project findings load through `cliutil.LoadTreeWithTrunk`,
so a refusal rests on every tier having been consulted. The cost is the
cross-branch scan per invocation, on verbs an operator waits on — which is the
trade to weigh, and the reason this is not a one-line change.

Two narrower options exist if that cost proves unacceptable. Pay the scan only
when a verb is about to refuse, re-resolving before the plan is suppressed
rather than on every load. Or scope it to the verbs that actually write a
reference, leaving the rest bare.

Whichever lands, the mechanical evidence is a verb-level test that writes a
reference resolvable only on a sibling ref and asserts the mutation commits with
a warning rather than refusing — the case measured above, inverted.

## Related

- G-0558 — the same divergence on the read paths, resolved in the opposite
  direction because a reporting surface may decline a verdict a verb cannot
- ADR-0030 — establishes the cross-branch pending tier
- ADR-0041 — classifies a cross-branch reference by whether its branch is
  published, and anticipates more such references
