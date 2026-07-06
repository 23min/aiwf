---
id: G-0379
title: trunk-collision hint always suggests reallocate, wrong for stale-branch renames
status: open
---
## What's missing

`internal/check/hint.go`'s `hintTable` has no entry for
`ids-unique/trunk-collision`. `HintFor` falls back to the bare `"ids-unique"`
hint — *"run `aiwf reallocate <path>` on one of the duplicates to renumber
it"* — for every `ids-unique` finding regardless of subcode, including
`trunk-collision`.

That generic hint is correct for a same-tree duplicate (two entities
independently allocated the same id, no trunk involved) but actively wrong
for the common `trunk-collision` trigger: a rename landed on trunk (`aiwf
retitle`/`aiwf rename`) after a branch forked, invisible to the branch's
stale copy (`G-0378`). Running `aiwf reallocate` against that case does not
resolve anything — it renames the branch's stale-but-otherwise-correct copy
to a new id, producing a genuine duplicate entity that then has to be
reverted by hand once the real cause is understood.

## Why it matters

Confirmed live: three independent sessions hit the same `trunk-collision`
finding in one day, all following the hint's single recommended remediation,
`aiwf reallocate`. One session followed through and created a spurious
duplicate entity (`G-0376`) requiring a manual `git revert` to undo. The
hint is the first (and often only) thing an operator or an LLM assistant
reads when this finding fires — a wrong default remediation there gets
followed literally, repeatedly, by design (that's what a hint is for).

## Direction

Add a subcode-specific entry for `ids-unique/trunk-collision` (the lookup
already checks `code+"/"+subcode` before falling back to the bare `code`,
per `HintFor` — no new lookup mechanism needed) that leads with checking
whether this looks like a stale branch relative to a trunk-side rename
before reallocating:

- Name a concrete check: whether the trunk-side path was produced by a
  rename since the branch's fork point (e.g. `git log --diff-filter=R
  --follow -- <trunk-path>` against the configured trunk ref, or simply
  attempting `git merge`/`git pull` from trunk and observing whether it
  resolves the divergence cleanly, which it will for a genuine same-entity
  rename).
- If so: merge/rebase trunk into the branch — never reallocate.
- Only if the two paths are genuinely unrelated entities (no rename
  relationship, confirmed): `aiwf reallocate <path>` remains correct, as
  today.

Leave the bare `"ids-unique"` hint (for a same-tree, non-trunk collision)
unchanged — it's correct for that case.

This is a wording-only change with no logic change, so it does not need to
wait for `G-0378`'s detection fix — and stays relevant after that fix
ships, since `ADR-0031` deliberately keeps a manual, non-kernel-verb rename
on trunk as a residual (safe) false positive that this hint will still
need to cover correctly.

## Scope

- `internal/check/hint.go`: add the `ids-unique/trunk-collision` entry.
- `internal/check/hint_test.go`'s `finding-hints-name-command` chokepoint
  requires every hint to name a concrete command — the drafted wording
  above names `git log`/`git merge`/`aiwf reallocate` explicitly.
- Consider whether CLAUDE.md's §"Id-collision resolution at merge time"
  needs the same stale-branch-vs-trunk-rename caveat alongside its existing
  `git mv`-vs-`aiwf reallocate` guidance.
- Small enough for a `wf-patch`.

## Related

- `G-0378` — the detection gap this hint currently papers over.
- `ADR-0031` — records that a manual rename on trunk stays a deliberate,
  accepted residual false positive even after `G-0378` ships, which is why
  this hint fix has lasting value beyond that gap's resolution.
