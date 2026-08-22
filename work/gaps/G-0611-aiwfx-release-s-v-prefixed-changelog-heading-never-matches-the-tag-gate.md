---
id: G-0611
title: aiwfx-release's v-prefixed CHANGELOG heading never matches the tag gate
status: open
---
## What's missing

`aiwfx-release`'s step 3 templates the release heading as `## [vX.Y.Z] — YYYY-MM-DD`,
with a leading `v`. The tag-push gate at `.github/workflows/changelog-check.yml`
strips the `v` from the tag name (`version="${tag#v}"`) and then greps for
`^## \[$version\]`. A heading written to the ritual's template therefore never
matches the gate that checks it.

The repo's own `CHANGELOG.md` uses the unprefixed form (`## [0.32.0] — 2026-08-03`),
so every release cut here has silently ignored the shipped template. The ritual is
the surface a consumer follows; the file is the surface a maintainer copies from.
They disagree, and only the file is exercised.

## Why it matters

The failure lands at the worst moment: after the tag has been pushed. `v*` tag
pushes are what trigger the gate, and a pushed tag is not cheaply retractable — the
remediation is a follow-up commit and a re-tag, on a ref consumers may already
have resolved through the module proxy.

An operator following `aiwfx-release` literally, in this repo or a consumer's,
hits it on their first release and has no signal until CI fails. Nothing earlier
looks: the heading is prose to every check but this one.

## Direction

Fix the template to `## [X.Y.Z] — YYYY-MM-DD`, matching both the gate and the
file. Whether the gate should also tolerate the `v` form is a separate question —
accepting both would make the two spellings equally valid and let them drift
apart again, so the narrower fix is likely right.

Worth checking at the same time whether any other shipped surface templates a
version heading, and whether `deployer.md` (which paraphrases this step) carries
the same `v`.

## References

Found while reviewing the patch that closed G-0368, which changed how the wrap
ritual writes the `[Unreleased]` entry this one is folded forward from.
