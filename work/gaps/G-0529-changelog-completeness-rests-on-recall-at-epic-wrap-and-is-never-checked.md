---
id: G-0529
title: CHANGELOG completeness rests on recall at epic wrap and is never checked
status: open
priority: medium
discovered_in: E-0078
---
## Problem

`CHANGELOG.md`'s `[Unreleased]` section is written at exactly two moments: a
patch's own wrap, where `wf-patch` step 4 mandates an entry with no skip, and
an epic's wrap, where `aiwfx-wrap-epic` step 7 adds one entry for the whole
epic. Milestone wrap writes nothing — a milestone's delta rolls up into its
parent epic's entry by design.

An epic's entire user-visible delta therefore rests on a single act of recall,
performed once, at the end, over however many milestones and weeks the epic
ran. Nothing verifies the result.

## Why the existing checks miss it

`changelog-check.yml` fires on a pushed `v*` tag and confirms the commit's
`CHANGELOG.md` carries a matching `## [X.Y.Z]` heading. That is a check on the
release ritual, not on content: a heading above a stub passes it, and it runs
at the tag, long after the wrap where an omission happens.

No other surface looks. `aiwf check` does not read `CHANGELOG.md`, and the wrap
rituals treat the entry as prose to author rather than a claim to verify.

The failure is not hypothetical. E-0075 wrapped with an entry that omitted a
user-visible refusal; a human noticed afterwards, and the omission was tracked
and back-filled as G-0509. The entry existed and was thin — the shape a
presence check cannot catch.

## Direction

Two properties, cheapest first, both mechanical:

- **Every epic that reached `done` is cited in `CHANGELOG.md`.** Epic
  granularity is the correct unit: a gap closed inside an epic legitimately
  never appears by id, so a per-gap rule would fire on correct trees. This
  catches a missing entry, not a thin one.
- **A consumer-visible surface delta is named under `[Unreleased]` before a
  release.** The kernel already enumerates the surfaces that matter — finding
  codes, top-level verbs, `aiwf.yaml` keys, exit codes — so "a finding code
  introduced since the last release is named nowhere in `[Unreleased]`" is
  computable. This is the property that catches a thin entry, and the one that
  would have caught G-0509.

Where each fires is open. The second wants the release boundary rather than the
push, since `[Unreleased]` is legitimately behind the branch until an epic wraps.

## Not this gap

- G-0368 makes `wrap.md` the single point of authorship and has epic wrap copy
  its changelog section verbatim. That settles *where the text is written*; it
  adds no verification, and a delta missing from `wrap.md` copies through
  intact.
- G-0439 concerns cross-references inside `CHANGELOG.md` surviving a doc
  relocation, not whether an entry exists or covers what shipped.

## Provenance

Found 2026-08-03 while sequencing a release against E-0078. Auditing the 19
gaps closed between v0.30.0 and v0.31.0 showed the per-epic rollup working as
designed: 13 carried their own entry, and the 6 that did not were all E-0075's,
folded correctly into that epic's single entry. The defect is in verification,
not in the shape.
