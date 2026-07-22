---
id: G-0439
title: Doc-relocation sweeps (e.g. E-0034) skip CHANGELOG.md cross-references
status: open
discovered_in: E-0034
---
## What's missing

E-0034's docs/pocv3/ relocation swept cross-references across the docs/ tree and source code, but not `CHANGELOG.md` — a pre-migration `## [X.Y.Z]` entry still linked to `docs/pocv3/archive/gaps-pre-migration.md`, a path that stopped existing once the relocation landed (the correct post-relocation path is `docs/archive/pocv3/gaps-pre-migration.md`). Nothing caught this until a later release cut's pre-release link-check ran.

## Why it matters

`CHANGELOG.md` is append-only/forget-by-default (per its own documented convention) specifically so past entries don't need to track a moving codebase — but that convention assumes referenced files don't move without the mover also fixing CHANGELOG's copy of the link. A doc-relocation epic like E-0034 has every incentive to sweep docs/ and source but easily forgets CHANGELOG.md, since it isn't under docs/ and doesn't read as "documentation" in the moment. This produced a genuine broken-link finding that blocked a release's pre-flight check until hand-fixed at tag time. The fix is either: (a) future relocation/rename sweeps explicitly include CHANGELOG.md in their cross-reference scan, or (b) CHANGELOG.md's historical links are exempted from link-check the same way its content is exempted from other doc-lint rules, so a moved target doesn't retroactively break an already-published release note.