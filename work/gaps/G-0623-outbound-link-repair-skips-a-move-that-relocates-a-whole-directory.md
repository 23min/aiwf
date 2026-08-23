---
id: G-0623
title: Outbound link repair skips a move that relocates a whole directory
status: open
discovered_in: M-0315
---
## What's missing

ADR-0046 commits every path-changing verb to repairing the links a moved entity
carries in its own body. The shipped implementation covers moves that relocate a
single file — a flat-file entity swept into `archive/`, a milestone moved between
epics. A move that relocates a whole directory is excluded, so a dir-shaped
entity's own outbound links are not recomputed.

The excluded shape, measured by sweeping an epic into `work/epics/archive/`: the
epic body moves one level deeper, so a destination that pointed out of its own
directory needs one more `../` and does not get it. The links break exactly as
the flat-file case did before ADR-0046.

The exclusion is deliberate rather than an oversight, and removing it needs the
missing input rather than a smaller guard. When a directory relocates,
everything inside comes along — including files no verb enumerates, such as the
`wrap.md` the wrap rituals write beside an epic body. Those destinations still
name the same content afterwards, so recomputing them as though their target
stayed put is what breaks them, which is the regression
`TestRetitle_DirShapedKindKeepsLinksIntoItsOwnDirectoryResolving` now pins.

The primitive cannot tell the two apart from the paths it is given. A directory
rename and a milestone moving between epics have the same shape — the linking
file's parent changes while its basename does not — but in the first case its
siblings co-move and in the second they stay. The caller knows which; the
primitive does not.

## Why it matters

A resolution has to give the primitive the directory-level move, so a
destination resolving under the moved prefix is remapped through it rather than
treated as stationary. That is one more input to `RewriteLinkDestinationsForMove`
and a mapping arm, and it subsumes the current `dirShaped` suppression list —
which exists only because the information is absent.

Two shapes need distinguishing at that point, and a fix that treats them alike
will break one of them: a destination naming something that co-moved inside the
directory must not change, and one naming something outside it must gain or lose
the depth the move added. The pinning fixture is a body carrying both.

Worth doing when a dir-shaped entity's outbound links are observed rotting.
Until then the shipped behaviour matches what preceded ADR-0046 for this shape,
so nothing regressed — the reach is narrower than the commitment, and ADR-0046's
scope note says so.
