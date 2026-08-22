---
id: G-0609
title: Wrap rituals assume a single checkout, breaking under the worktree default
status: open
discovered_in: E-0087
---
## What's missing

Both wrap rituals instruct the operator to check out the branch they merge
into. `aiwfx-wrap-epic` step 6 says `git checkout main`; `aiwfx-wrap-milestone`
step 12 says `git checkout epic/E-NNNN-<slug>`. Git permits one checkout per
branch, so under the worktree layout this repo mandates — CLAUDE.md §"Default to
a worktree for any branch work" — the target branch is already held elsewhere and
the documented command fails:

```
fatal: 'main' is already used by worktree at '/workspaces/aiwf'
```

The epic case is the one observed, during E-0087's wrap. The milestone case has
the same shape and is latent rather than theoretical: it only worked there
because the epic branch happened to be checked out in the same worktree as the
milestone. Cutting the milestone into its own worktree — which
`aiwfx-start-milestone` step 5 explicitly offers — reproduces it.

Neither ritual is naive about worktrees in general; `aiwfx-start-epic` and
`aiwfx-start-milestone` both handle placement carefully. It is specifically the
wrap path that assumes a single checkout, which suggests it predates the
convention.

### The root cause is smaller than the symptom

The checkout is not incidental to the sequence — the sequence is built around it,
and the thing that anchors it there is one line of the wrap artefact.

`wrap.md` carries a **Merge commit:** field. To fill it, the artefact must be
written after the merge; so its commit lands on the integration target; so
promote-done must follow it there; so the target must be checked out. Every
downstream constraint descends from that back-reference.

The back-reference is also redundant. The merge commit carries
`aiwf-verb: wrap-epic` and `aiwf-entity: E-NNNN`, so `aiwf history E-NNNN`
already resolves it — the field is a second copy of a fact git owns, which is the
shape the shipped guidance's *keep the reasoning, derive the facts* rule names.

### The fix that removes the friction rather than automating around it

Drop the merge-SHA field, and the sequence collapses onto the working branch:
scaffold the artefact, add the CHANGELOG entry, commit both, promote to `done`,
regenerate the roadmap — all on the epic (or milestone) branch — then merge once
into the target. The target then receives exactly one commit, the trailered
merge, and `git -C <path> merge` works whether or not it is checked out
elsewhere.

That ordering is simpler in the single-checkout case too: four commits on one
branch and one merge, rather than commits interleaved across two branches. It
also removes an ordering trap — merging before promoting leaves the target with
the merge but not the status flip, which happened during E-0087's wrap and
needed an unpushed `reset --keep` to correct.

The scope invariant the current ordering exists to protect still holds:
promote-done remains the last verb-driven commit, and the roadmap regen and the
merge after it are plain `git commit`, never routed through the CLI's
trailer-decoration path.

A worktree-detection helper (`git worktree list --porcelain` to locate the
target) would also work and is a smaller edit, but it automates around a sequence
that does not need to span two branches.

## Why it matters

A ritual whose documented steps cannot execute under the repo's own mandated
default is a defect in the ritual, not an operator problem. An assistant
following it either fails at the checkout and improvises — which is where the
ordering mistake came from — or silently abandons the worktree convention.

These are also shipped surfaces. Consumers materialize both rituals via
`aiwf init` / `aiwf update`, and any consumer using git worktrees meets the same
wall with no context for why.

## Pinning

The fix ships without a regression check, because the obvious one is refused by
the repo's own rules. A test reading the shipped ritual and asserting it no
longer instructs a bare checkout is a negative phrase assertion over a shipped
surface — the class D-0070 retires — and `shipped-prose-assertion` fires on it.
Measured rather than assumed: the candidate test was written, the policy raised
a violation naming it, and the candidate was removed.

The two routes that would evade that check are worse than no check. A regexp
over the same prose exploits the bypass G-0607 records. Writing it as a
production policy exploits the scope hole G-0606 records. Neither is a pin; both
are the ban being worked around by its own author.

So this is a second concrete instance of the class G-0608 names — a negative
regression pin over a shipped surface, with nowhere to live. The first was a
dead-CLI-form guard whose shipped site was dropped for the same reason. Two
instances in one epic is the evidence G-0608 was filed to wait for, and settling
that class would give this fix somewhere to put its check.

What stands in the meantime: the shell snippets in both rituals were executed
against the live worktree layout and verified to resolve the correct paths, and
to return empty for a branch checked out nowhere.
