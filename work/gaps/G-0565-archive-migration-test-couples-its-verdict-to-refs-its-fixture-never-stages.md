---
id: G-0565
title: Archive-migration test couples its verdict to refs its fixture never stages
status: open
priority: medium
---
## What's missing

`TestBinary_ArchiveKernelMigration_LeavesCheckClean` is M-0085/AC-7's
binary-level evidence: the first `aiwf archive --apply` against a faithful copy
of the kernel's planning tree produces exactly one commit, sweeps every
terminal-status entity into `archive/`, and leaves the tree valid. Its subject
is the sweep.

Its closing assertion is not scoped to the sweep. It requires `aiwf check` to
report zero error-severity findings across every rule, and it runs that check
against a temp-dir copy of `work/`, `docs/adr/` and `aiwf.yaml` under a fresh
`git init` — one ref, no remote, no configured trunk. So every rule whose
verdict depends on which git refs exist is evaluated against a repository that
deliberately stages none.

Two rules do. `refs-resolve` and `body-prose-id` resolve a reference through a
tier stack — working tree, then the trunk ref, then the cross-branch view — and
report `unresolved` at error severity only on a miss at every tier. In this
fixture the second and third tiers are empty by construction, so a reference to
an id the copied tree does not itself contain misses all three and fails the
test, whether or not that id exists.

The sweep's own effects are already pinned structurally, and independently of
this: the commit-count delta, the synthetic entity's absence from its
active-tree path, and its presence under `work/gaps/archive/`. The zero-errors
assertion is the only part coupled to the fixture's ref topology.

## Why it matters

It fires, and it fired for real. Measured 2026-08-06: two gaps on mainline cited
an id minted on an unpushed epic branch, and this test failed on
`body-prose-id/unresolved` — an archive-migration test red over a body-prose
reference, in a run where nothing about archiving was wrong. The failure
surfaces far enough from its cause that reading it costs a diagnosis before it
costs a fix.

The condition recurs by design rather than by accident. ADR-0030 exists to make
citing an id that lives on an unmerged branch legitimate, and ADR-0041 keeps it
legitimate for any branch that has been published. Every such citation on
mainline is a reference this fixture cannot resolve.

Two things that look like they would rescue it and do not. G-0558's
`unresolved-unverified` downgrade applies to surfaces that skipped the
cross-branch scan; this fixture's check runs the scan and finds nothing, so it
has consulted every tier it can build and the verdict stands at error.
ADR-0041 narrows the population — only published references now reach mainline,
and a full clone fetches the branch and resolves them — but this fixture has no
remote at all, so no amount of publishing reaches it.

## Resolution shape

**Scope the assertion to what the test is about.** The sweep can affect a
bounded set of rules; assert on those, or exclude the two whose verdict turns on
refs the fixture does not stage. That keeps AC-7's claim intact, touches neither
ADR-0030's leniency nor the pre-push hook, needs no refs, and stays
deterministic.

Rejected, and worth recording as rejected: **seeding refs into the fixture**.
It would make the verdict depend on which branches the developer happens to
hold — green where the sibling branch exists, red in a fresh clone and in CI —
which is the machine-dependence G-0556 exists to remove, reintroduced inside the
test meant to be immune to it.

Also rejected: **relaxing the assertion to tolerate errors generally**. The
`unresolved`-at-error class is a real signal about a swept tree, and there is no
`cross-branch-pending` finding in this repository to tolerate instead — the
fixture produces the hard subcode or nothing.

## Related

- G-0556 — the divergence this is the last piece of; it named this remedy and
  judged it right on its own terms, while noting it scopes one test rather than
  answering the policy question ADR-0041 settled.
- G-0106 — the same test, the same shape on a different axis: an assertion
  coupled to ambient tree state rather than to the test's subject, addressed by
  making the premise explicit instead of depending on what the tree happened to
  contain.
- G-0536 — the other half of what remains: `aiwf check` has no CI position, so
  nothing evaluates a departing tree the way a fresh checkout would.
- ADR-0041 — narrows which references can reach mainline, without giving this
  fixture the refs to follow them.
