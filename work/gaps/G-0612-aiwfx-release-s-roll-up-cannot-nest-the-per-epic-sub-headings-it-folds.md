---
id: G-0612
title: aiwfx-release's roll-up cannot nest the per-epic sub-headings it folds
status: open
---
## What's missing

`aiwfx-release` step 3 builds a version section whose template groups entries as
plain `### Added` / `### Changed` / `### Fixed` / `### Removed` headings with
bullets beneath, and then instructs: *"If the project keeps an `[Unreleased]`
section at the top of CHANGELOG, move its contents into the new release section."*

But `[Unreleased]` does not hold bullets. Under the per-epic accumulator shape,
it holds `###` sub-headings of its own — `### Changed — E-NNNN: <summary>`,
each with paragraphs beneath. Those are the same heading level as the release
template's category groups, so there is no nesting that puts one inside the
other. The instruction describes a move that the two shapes do not admit.

## Why it matters

The release ritual's one job at this step is to relocate accumulated entries, and
its guidance stops exactly where the real decision starts. The operator is left
to invent the reconciliation at release time — flatten the per-epic headings into
bullets, promote them to `##`, or drop the template's category grouping — and
whichever they pick becomes an undocumented precedent the next release either
follows or silently diverges from.

This is not hypothetical drift: it is why the repo's own released sections do not
match the shipped template.

## Direction

Settle which shape a released version section has, then make the two rituals
agree. The likely answer is that `aiwfx-release`'s template is the stale one —
the per-epic `### <Category> — <id>: <summary>` entry is what both wrap and
`wf-patch` produce, and folding forward should preserve it rather than
re-grouping. That would reduce step 3 to renaming the `[Unreleased]` heading and
opening a fresh empty one, which is what the repo's release process in `CLAUDE.md`
already describes.

If so, the template goes and the step gets shorter — check `deployer.md`, which
paraphrases the same grouping.

## References

Adjacent to the category-set question filed alongside this one: the release
template offers `Removed`, which the wrap ritual's set does not. Both were found
while reviewing the patch that closed G-0368.
