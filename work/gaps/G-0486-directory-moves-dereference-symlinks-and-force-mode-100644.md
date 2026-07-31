---
id: G-0486
title: Directory moves dereference symlinks and force mode 100644
status: open
priority: medium
discovered_in: M-0282
---
## What's missing

A directory-moving verb rewrites every file it carries as a regular file at mode
`100644`, and reads content in a way that dereferences symlinks. Both happen on a
completely clean tree, and neither is reported.

Measured. An epic directory containing a `100755` script and a `120000` symlink,
committed and clean, then `aiwf rename` on the epic:

- the symlink became a regular `100644` file whose blob is the *link target's*
  content, so the tree now carries a second file with `id: E-0001` in it;
- the executable became `100644`.

Both survive a fresh clone, so this is the record and not a working-tree artifact.
`aiwf check` reports **0 errors** — only an `unexpected-tree-file` warning about
the extra file, which is a different observation entirely.

Two mechanisms, both in the commit path:

- `gatherCommitOps` reads each carried file with `os.ReadFile`, which follows
  symlinks. Its `os.Lstat` call is used only to decide whether a move destination
  is a directory, not to classify the entries inside it.
- `CommitTree` and `ReconcilePaths` each build their `update-index --cacheinfo`
  argument with a hardcoded `100644,%s,%s`.

## Why it matters

This is content no verb computed, committed under that verb's trailer — the same
class as the laundering E-0075 addresses, but with **zero HEAD divergence**, so
the precondition that epic is building cannot see it. A guard keyed on a dirty
set is looking at the wrong axis for this one.

The blast radius is bounded by how often a non-markdown file lives inside an
entity directory, which today means contract directories carrying schemas,
fixtures or validator scripts. That is not exotic — a validator is an executable
by construction.

There is a second-order consequence worth stating because it interacts with
E-0075. After such a commit the affected paths read as modified forever, because
the working tree still holds a symlink and an executable while the record holds
neither. Under the precondition E-0075 proposes, every later verb touching that
directory would refuse, and no aiwf verb could clear it. Symmetrically, a bare
`chmod +x` on a tracked file is a real diff to git, so the same guard would
refuse a change that cannot launder any content at all.

## Scope

The commit construction path: `gatherCommitOps`' content read, and the two
`cacheinfo` call sites that force a mode.

Out of scope: the laundering E-0075 covers, which is about divergence rather than
representation; and any change to which files a directory move carries.

## Resolution options

1. **Preserve mode and object type.** `Lstat` each entry, record its mode, and
   pass it through to `cacheinfo`; hash a symlink's target as a `120000` blob
   rather than reading through it. Correct, and it makes the round-trip lossless.
   Cost: the write path grows a mode concept it does not have today.
2. **Refuse to carry anything that is not a regular file at `100644`.** Narrower,
   and arguably right for a tree whose shape is already closed — an entity
   directory is not a general-purpose folder. Costs a refusal on trees that work
   today, and needs a story for contract fixtures.
3. **Carry only recognized entity files and leave the rest alone.** Ties this to
   the `unexpected-tree-file` rule already warning about such files. Changes what
   a directory move commits, which is a wider change than it looks.

Option 1 is the lean: it is the only one that does not change which files a move
carries, and the defect is a representation bug rather than a policy question.
Whichever lands, the permanent-dirty interaction above should be checked against
E-0075's guard before that guard ships.
