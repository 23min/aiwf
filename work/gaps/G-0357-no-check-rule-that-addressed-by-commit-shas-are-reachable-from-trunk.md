---
id: G-0357
title: No check rule that addressed_by_commit SHAs are reachable from trunk
status: open
---
## Problem

G-0355 strengthened the *verb-time* `--by-commit` check to require reachability
from **HEAD** — "the history the promote commits onto." That is the load-bearing
guarantee for the normal wf-patch flow (the closure runs post-merge on trunk, so
HEAD *is* trunk). But by construction it cannot cover three cases, because at
verb time "trunk" is ambiguous and only "HEAD" is knowable:

1. **Branch-anchored closures.** A promote run on a branch that *contains* the
   commit but never reaches trunk (abandoned, reset, force-moved) passes the
   HEAD check yet trunk never receives the commit. The claim is locally
   coherent, globally false.
2. **Unreconciled `--force`.** The sanctioned escape records a closure against an
   unmerged-fixing-branch or cross-repo commit "to be merged soon." Nothing
   re-validates if that merge never lands.
3. **Pre-existing / legacy state.** Gaps closed before G-0355, via `--force`, or
   via a raw `git commit` that bypassed the verb, may already carry off-trunk
   SHAs. The verb-time check guards only *new* promotes; it never audits
   existing frontmatter.

Only a tree-scanning check rule — running over all entities at the pre-push
chokepoint — catches all three. The two are complementary by anchor: verb-time
uses HEAD (the history being written onto); check-time uses the trunk ref (where
the claim must ultimately be true).

## Why it matters

The point of `addressed_by_commit` is a truthful "this gap was closed by commit
X." An off-trunk X makes that authoritative-looking but false, and it leaves the
machine on push unchallenged. This is the belt-and-suspenders half of the G-0355
fix: the verb guards the entry, the check guards the exit.

## Direction

- **Anchor = the configured trunk ref** (`allocate.trunk`), the same ref
  `ids-unique/trunk-collision` compares against — not HEAD. On an in-flight patch
  branch the closure is normally recorded only post-merge, so the normal flow
  does not false-positive.
- **Severity + escape (the main fork).** Either a **warning** (advisory;
  tolerates a deliberate in-flight `--force` reference, surfaces drift) or an
  **error with an `aiwf acknowledge` escape** (strict; a sovereign exemption
  recorded in git). Lean: **warning** — the verb-time check already blocks the
  common corruption, so the mirror's job is surfacing edge/legacy drift, and
  blocking would snare legitimate `--force`-then-merge-later states.
- **Kernel check rule** in `internal/check` (ships to consumers, unlike the
  aiwf-repo policy tests), adjacent to and strengthening `gap-addressed-has-
  resolver` (that checks the resolver *exists*; this checks the resolver *commit
  is on trunk*). Candidate code: `gap-addressed-commit-off-trunk`.
- **Discoverability + reversal.** New finding code -> `aiwf-check` skill +
  `--help` entry. The finding self-clears when the commit lands on trunk, is
  acknowledged, or the closure is reverted.

## Scope

The finding rule + trunk-ref resolution + tests (a fixture where a gap's
`addressed_by_commit` points off-trunk fires it; on-trunk stays silent), plus
the discoverability docs. Deferred alongside G-0355's verb-time check, which is
the load-bearing guarantee; this is defense-in-depth.
