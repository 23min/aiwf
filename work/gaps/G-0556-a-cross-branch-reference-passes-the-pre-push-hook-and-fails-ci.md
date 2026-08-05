---
id: G-0556
title: A cross-branch reference passes the pre-push hook and fails CI
status: open
---
## What's missing

`aiwf check` and the test suite judge the same tree differently, because they
see different sets of git refs, and nothing reconciles them.

Reference resolution consults the cross-branch view (ADR-0030): the working
tree, every local `refs/heads/*`, and every remote-tracking `refs/remotes/*`.
An id that exists only on an unmerged sibling branch therefore resolves, and
the finding is `body-prose-id/cross-branch-pending` at **warning** severity —
deliberately non-blocking, on the reasoning that it resolves on its own once
that branch merges.

`TestBinary_ArchiveKernelMigration_LeavesCheckClean`
(`internal/cli/integration/archive_kernel_migration_test.go`) copies `work/`,
`docs/adr/` and `aiwf.yaml` into a temp dir, runs `git init` there, and commits
once. That repo has exactly one ref. The cross-branch view is empty, so the same
token resolves against nothing and hardens to `body-prose-id/unresolved` at
**error** severity. The test asserts zero errors and fails.

Neither surface is wrong on its own terms. The check is answering "will this
resolve for me, eventually"; the test is answering "is this tree self-contained
right now". Both are legitimate questions. What is missing is that no surface
tells the operator the two answers have diverged.

## Why it matters

The pre-push hook is the chokepoint the framework's guarantees rest on, and
this is a state it passes while CI fails on identical bytes. The operator gets
no local signal, because every local surface — the hook, `aiwf check`, `aiwf
status` — is looking at a machine where the sibling branch exists.

The condition is also reachable through ordinary, sanctioned use. Filing a gap
mid-flight on whatever branch is checked out is the workflow ADR-0030 exists to
support; referring to it from a later gap on mainline is the friction that ADR
set out to remove. Two gaps on mainline now cite an id allocated on an unmerged
epic branch, filed exactly that way, each passing every gate at the moment it
landed.

Two properties make it expensive to diagnose. The failure surfaces far from its
cause — an archive-migration test fails on account of a body-prose reference
that has nothing to do with archiving, because that test is simply the one place
a ref-less copy of the tree gets checked. And it is invisible until push: on a
trunk that is several commits ahead of its remote, every commit since the last
push carries the fault without a single local run reporting it.

## Direction

The choice is which of the two answers is authoritative for a tree that is
about to leave the machine, and it is a real decision rather than a bug to
patch:

- **Give the hook the test's eyes.** Have `aiwf check` also evaluate a
  ref-less view before a push, so a reference that only resolves cross-branch
  blocks at the boundary where the tree stops being local. Closes the gap
  exactly, but narrows a leniency ADR-0030 chose on purpose, and the push
  becomes the moment an operator is told to go fix prose they wrote days ago.
- **Give the test the hook's eyes.** Seed the temp repo's refs, or teach the
  fixture to tolerate `cross-branch-pending` the way the live check does. Keeps
  ADR-0030 intact and turns CI green, at the cost of the test no longer
  asserting that the tree is self-contained — which is the property a fresh
  consumer clone actually has.
- **Report the divergence rather than resolving it.** Keep both severities and
  add a finding that fires when a cross-branch-pending reference is reachable
  from trunk, so the operator learns the tree is not self-contained without
  either surface changing its verdict.

Whichever wins should also settle scope: `refs-resolve` has the same two
severities on the same axis as `body-prose-id`, so it has the same exposure,
and the test is only the place the divergence happens to be observable today.

## Provenance

Found on 2026-08-05 while wrapping G-0547, when reconciling a patch branch with
mainline turned `make check-fast` red on a test the patch does not touch. The
same test fails identically on mainline without the patch, so the condition
predates that work and is unrelated to it.
