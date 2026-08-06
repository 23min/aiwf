---
id: G-0556
title: A cross-branch reference passes the pre-push hook and fails CI
status: open
---
## What's missing

Surfaces that read the planning tree disagree about whether it is valid,
because they resolve references against different sets of git refs, and nothing
reconciles them or reports that they diverged.

Prose reference resolution is tiered. The working tree resolves first. An id
that misses it but sits on the configured trunk ref resolves **silently** —
trunk is authoritative. A miss there consults the cross-branch view (built by
ADR-0025 for allocation, extended to resolution by ADR-0030): every local
`refs/heads/*` and every remote-tracking `refs/remotes/*`. A hit there fires
`body-prose-id/cross-branch-pending` at **warning** severity — deliberately
non-blocking, on the reasoning that it resolves on its own once that branch
merges. Only a miss at every tier is `body-prose-id/unresolved`, at **error**
severity. `refs-resolve` has the same tiers minus the silent trunk one.

The tier that decides this is the one made of refs the asking machine happens
to hold, so the verdict is a property of the machine rather than of the tree.
An id minted on a local branch that has never been pushed is reachable from
exactly one working copy on earth. Everywhere else — a teammate's clone, a CI
checkout, any repo built from the tree alone — it resolves at no tier and is an
error. `cross-branch-pending`'s premise, that the reference resolves once the
branch merges, is true; what it does not say is that until then the tree is
valid only here.

Two places observe that today, and neither is reachable by fetching harder.
`TestBinary_ArchiveKernelMigration_LeavesCheckClean`
(`internal/cli/integration/archive_kernel_migration_test.go`) copies `work/`,
`docs/adr/` and `aiwf.yaml` into a temp dir and runs `git init` there; that
repo has exactly one ref, so the cross-branch view adds nothing the working
tree does not already hold, and a reference that was a warning on the
developer's machine hardens to an error. A clone of the repo behaves the same
way for the same reason: the branch is absent from the remote, so no checkout
depth reaches it.

Neither surface is wrong on its own terms. The check is answering "will this
resolve for me, eventually"; the other two incidentally measure whether the
tree is self-contained right now. Both are legitimate questions. What is
missing is that no surface tells the operator the two answers have diverged.

## Why it matters

The pre-push hook is the chokepoint the framework's guarantees rest on, and
this is a state it passes while CI fails on identical bytes. `aiwf check` — the
hook's whole content — sees the sibling branch, reports a warning, and exits 0.

`aiwf status` disagrees with it. Status loads the tree without the cross-branch
view, so it counts the same references as errors in its Health line while
`aiwf check` counts zero — and that line ends by pointing the operator at
`aiwf check` for details, which is the surface that will tell them nothing is
wrong. Two local surfaces render opposite verdicts on the same bytes, neither
says so, and the one the hook runs is the permissive one.

That split is wider than status-versus-check, and runs through `aiwf check`
itself. Measured 2026-08-06: five paths load the tree through two loaders. Every
mutating verb and the full `aiwf check` build the cross-branch view;
`aiwf check --fast`, `aiwf check --shape-only` — the pre-commit hook — and
`aiwf status` do not, and on this repo's own tree the first group reports zero
errors while the second reports two. So the write path and the authoritative
read path agree, and three secondary read paths dissent. That half is tracked
separately as G-0558, is fixable ahead of this gap, and turns on a different
argument: `unresolved` is a claim about every tier, and a loader that built one
tier cannot substantiate it.

The condition is reachable through ordinary, sanctioned use. Filing a gap
mid-flight on whatever branch is checked out is the workflow ADR-0030 exists to
support; referring to it from a later gap on mainline is the friction that ADR
set out to remove. Two gaps on mainline cite an id allocated on the E-0079 epic
branch, filed exactly that way, each passing every gate at the moment it
landed.

And it accumulates unseen. The test suite catches it; nothing an
entity-authoring commit fires does. The pre-commit hook checks shape only. The
pre-push chain's Go-gated gates skip a planning-tree push entirely, and the
`aiwf check` it ends on exits 0, so the push itself succeeds — discovery is CI,
after the commits have landed. The local validation cadence names entity
markdown as a non-build-input that leaves a green run green, so the run that
would report it is exactly the run the cadence says to skip. On a trunk several
commits ahead of its remote, every commit since the last push carries the
fault.

The failure also surfaces far from its cause: an archive-migration test fails
on account of a body-prose reference that has nothing to do with archiving,
because that test is simply a place where a ref-less copy of the tree gets
checked.

## Direction

The choice is which answer is authoritative for a tree that is about to leave
the machine, and it is a real decision rather than a bug to patch:

- **Give the hook the test's eyes.** Have `aiwf check` also evaluate a ref-less
  view before a push, so a reference that only resolves cross-branch blocks at
  the boundary where the tree stops being local. Closes the divergence
  directly, but narrows a leniency ADR-0030 chose on purpose, and the push
  becomes the moment an operator is told to go fix prose they wrote days ago.
- **Give the test the hook's eyes.** Seeding the temp repo's refs makes the
  verdict depend on which branches the developer happens to have — green where
  the sibling branch exists, red in a fresh clone and in CI. Relaxing the
  assertion instead means relaxing the `unresolved`-at-error class the test
  exists to catch, since there is no `cross-branch-pending` finding in that
  repo to tolerate. Either way it retracts the mechanical evidence for
  `M-0085/AC-7`, which is this assertion.
- **Narrow the assertion to its subject.** The test is about whether the first
  `--apply` sweep leaves the tree valid; a global zero-errors assertion couples
  it to every rule whose verdict depends on refs the fixture deliberately does
  not stage. Excluding exactly those rules keeps the AC's claim, touches
  neither ADR-0030's leniency nor the hook, needs no refs, and stays
  deterministic — but reaches only this one site.
- **Report the divergence rather than resolving it.** Add a finding that fires
  when a reference resolving only off the working tree is reachable from trunk,
  so the operator learns the tree is not self-contained without either surface
  changing its verdict. Diagnosis, not a green build: CI stays red until the
  source branch merges. It also cannot see the silent tier — a trunk-only id
  produces no finding to escalate.

Worth naming to reject: *don't cite an unmerged id in the first place*.
ADR-0030 exists to make that citation legitimate and its Context rejects the
wait-for-merge alternative by name, so tightening authoring practice would undo
the decision rather than implement it.

Whichever wins should also settle scope, which is wider than one test.
`refs-resolve` carries the same severities off the same cross-branch view, so a
structured `depends_on` to an unmerged id diverges exactly as prose does — and
it has a second observation site the prose rule does not.
`TestPolicy_ThisRepoTreeIsClean` runs the check rules against the **live** tree
loaded without the cross-branch view, and fails on an error-severity
`refs-resolve`. That site is blind by loader choice rather than by repo
topology, so it diverges even on a machine where the sibling branch is present
and no ref-seeding reaches it. `body-prose-id` escapes it today only because
that test's code filter does not list it.

G-0536 waits on this. It proposes a CI position for `aiwf check`, which is the
third observer of the same divergence and the one that would report it as a
push-blocking failure of the check itself rather than of a test. Its resolution
is sound only once this decision is made.

## Decided — ADR-0041

The answer is none of the four above; it is a fifth shape they all miss, because
each of them accepts the premise that a cross-branch reference is one condition.
It is two. A reference resolvable from a remote-tracking ref is published —
anyone who fetches can follow it. A reference resolvable only from a local branch
exists on one working copy on earth. ADR-0030 fires the same non-blocking subcode
for both, and the distinction is already computed and discarded: `trunk.RefHit`
carries the ref that answered, and the resolver groups on the union.

ADR-0041 splits them. Remote-visible stays a warning; local-only becomes an error
that blocks at the push boundary; absent from every tier stays `unresolved`. The
remedy the error names is push the branch — neither of the two remedies ADR-0030
rejected, and what the standing guidance on allocating and pushing promptly
already asks for.

What decided it against the four: option 1 taxes the published case to protect
against the unpublished one; option 2 cannot generalize, since ref-seeding does
not reach `TestPolicy_ThisRepoTreeIsClean`, which is blind by loader choice
rather than by repo topology; option 3 is right on its own terms and now
uncontroversial, but scopes one test rather than answering the question; option 4
is satisfied locally once G-0558 and the agreement invariants land. And the
measured frequency — instances with a window long enough to span a push run at
about one per three weeks — makes option 1's standing cost the wrong trade for
the protection it buys.

Sequencing that follows: G-0558 first, then the agreement invariants D-0063
records, then ADR-0041's classification, then the fixture that drives its
lifecycle. G-0536's CI position is what finally makes CI's verdict converge with
the operator's, and it is the last step rather than a precondition.

## Provenance

Found on 2026-08-05 while wrapping G-0547, when reconciling a patch branch with
mainline turned `make check-fast` red on a test the patch does not touch. The
same test fails identically on mainline without the patch, so the condition
predates that work and is unrelated to it.
