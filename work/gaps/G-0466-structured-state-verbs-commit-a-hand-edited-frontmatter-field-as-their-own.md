---
id: G-0466
title: Structured-state verbs commit a hand-edited frontmatter field as their own
status: open
priority: high
discovered_in: M-0281
---
## What's missing

Every structured-state verb reads its entity from the loaded tree, re-serializes
the whole frontmatter, and commits. Nothing compares that frontmatter against
HEAD first. So when the working copy already carries a hand-edited frontmatter
field, the verb commits that edit alongside its own change, under its own
`aiwf-verb:` trailer.

Measured on a disposable repo. A gap's priority was set to `high` through
`aiwf set-priority`, then hand-edited to `low` in the working copy, then
`aiwf retitle` was run:

```
-title: Probe gap for laundering measurement     <- retitle's own change
+title: Probe gap retitled by a legitimate verb
-priority: high                                  <- the hand-edit, carried along
+priority: low
```

Trailers on that commit: `aiwf-verb: retitle`. `aiwf check` reports no finding.
`aiwf history` lists `set-priority high` and then `retitle`, so a reader
concludes the priority is `high`; it is `low`. The verb that owns the field was
never involved, and the record does not say so.

Exactly one path in the kernel refuses a frontmatter-dirty working copy:
`aiwf edit-body` in bless mode, which points the operator at `aiwf promote` /
`rename` / `cancel` / `reallocate`. That is the mode whose whole job is
committing the working copy, so it is the path least in need of the guard. No
structured-state verb has one.

## Why it matters

Hand-editing frontmatter is forbidden by the shipped guidance, and
`provenance-untrailered-entity-commit` is the finding that catches it — but only
when the operator commits the edit themselves. Run any legitimate verb instead
and the edit arrives inside a properly trailered commit, so the finding never
fires. The rule has a hole shaped precisely like the sequence an operator or an
LLM actually produces: hand-edit a field, then run a real verb.

The cost is a commit that misattributes a structural change to a verb that did
not make it, and a `history` timeline that is confidently wrong about the current
value of a field. Routing edits through verbs is worth doing because the commit
then says what happened; here it says something false.

## Scope

Pre-existing and independent of the same-state convergence work that surfaced it
(M-0281). It generalizes G-0463 from `edit-body --body-file` to every
structured-state verb: `promote`, `cancel`, `move`, `retitle`, `rename`,
`reallocate`, `set-priority`, `set-area`, `milestone tdd`,
`milestone depends-on`.

Both paths are affected, in different ways. On the mutating path the hand-edit is
committed under the verb's trailer. On the converging path a same-state `retitle`
commits nothing and leaves the hand-edit on disk — but the same-state comparison
reads the loaded value rather than HEAD, so it also reports "already set" about a
value HEAD does not have, and commits an empty diff when asked for the value HEAD
does have. See the closing paragraph.

## Options

1. **A shared precondition before projection** — every structured-state verb
   compares the loaded entity's frontmatter against HEAD's and refuses when they
   differ, pointing at the owning verb or at `git checkout <path>`. One rule, one
   place, mirroring bless mode's existing refusal and its message. Needs a
   decision on the escape hatch (`--force`, or none) and on the
   never-committed-entity case, where there is no HEAD version to compare.
2. **Project onto HEAD's frontmatter** rather than the loader's, so a verb writes
   only its own field and cannot carry drift. Silently correct rather than
   instructive, and it discards an edit the operator may have meant to keep.
3. **Detect after the fact** — a check rule that flags a commit whose entity diff
   touches fields outside its `aiwf-verb:`'s ownership. Catches the history
   already in the tree, which options 1 and 2 do not, but fires after the push.

Option 1 is the lean, on the same reasoning that settled G-0463: refusing tells
the operator something true about their tree, and one precondition shared by
every verb beats a comparison duplicated into each. Option 3 is a plausible companion
rather than an alternative.

**Not the fix:** a HEAD conjunct inside each same-state NoOp guard. The
comparison belongs at one shared precondition ahead of the guards rather than
duplicated into each of them, and where that precondition sits relative to the
same-state check decides whether any conjunct is needed at all — run it in the
verb prelude and a guard can never be reached with HEAD-divergent frontmatter.

The converging path is not, however, harmless. With HEAD at `priority: high` and
the working copy hand-edited to `low`, asking for `low` reports "already set to
`low`; nothing to change" — false about the record, and the operator's requested
mutation is silently dropped — while asking for `high` commits a tree
byte-identical to its parent, an empty-diff commit of the class the same-state
convergence work existed to eliminate. So a loaded-only comparison yields both a
false negative and a false positive, and "harmless" is the wrong reason to leave
the guards alone.
