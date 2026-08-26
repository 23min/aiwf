---
id: G-0643
title: aiwfx-whiteboard calls an operator ordering file uncommitted; this one is not
status: open
priority: low
---
## What's missing

`aiwfx-whiteboard`'s SKILL.md classifies an operator-authored ordering file by a
property this repo's own file does not have:

> An ordering the consumer authors is a third case: neither committed nor
> regenerated, so it is reconciled rather than overwritten.

Measured 2026-08-26: `git ls-files TODO.md` returns it and `git check-ignore -v
TODO.md` exits 1. It is committed, and deliberately so — `6328f48cb` removed the
`/TODO.md` entry from `.gitignore` with the reason *"An ordering that is
hand-maintained but gitignored cannot be reviewed or reverted. Tracking it puts
every change to it in the diff, which matters most when an assistant is the one
proposing the change."*

The conclusion the sentence draws is right and is unaffected: such a file is
reconciled rather than overwritten whether or not it is tracked. Only the
property it names as the reason fails, and it fails in the repository that ships
the skill.

## Why it matters

Being tracked is what makes an assistant's proposed edit reviewable before it
lands — the reason `6328f48cb` gives. A reader who takes the skill's premise and
gitignores their ordering file loses exactly the property the commit was after,
having followed shipped advice to do it.

The two halves also disagree on the same axis as `wf-doc-lint`'s neighbouring
clause, which says such a file is *"often neither generated nor gitignored"*.
Both skills materialize into the same consumer repo from one `aiwf update`, so a
reader meets both and has to adjudicate.

## Resolution shape

Drop the committed/uncommitted claim rather than invert it: the sentence needs
only *neither generated nor regenerated* to reach its conclusion, and tracking is
the consumer's choice either way. `6328f48cb`'s reasoning is the argument for
recommending tracking if the skill wants to say anything at all.

The file sits under `internal/skills/embedded-rituals/`, so the edit rides a
commit carrying an `aiwf-entity:` trailer per `skill-edit-provenance-backstop`.
