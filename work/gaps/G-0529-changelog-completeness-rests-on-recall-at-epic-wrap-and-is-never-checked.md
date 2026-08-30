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

The failure is not hypothetical, and it has now happened twice. E-0075 wrapped
with an entry that omitted a user-visible refusal; a human noticed afterwards,
and the omission was tracked and back-filled as G-0509. The entry existed and
was thin — the shape a presence check cannot catch.

Cutting v0.34.0 on 2026-08-30 surfaced three shipped changes that reached the
release with no entry at all: G-0647's `## Closes` section and the three rituals
that read it, the always-on guidance's "a test pins a rule, not an input" rule,
and G-0628's new `aiwf-promote` skill section. Every mechanical gate passed; a
human asking whether skills and guidance had been updated is what caught them.
The release spanned 235 commits, which is the case where recall is weakest and
complete notes matter most.

The two routes differ, and neither is guarded. G-0647 ran `wf-patch`, whose step
4 makes the entry mandatory — "every patch adds one, even if it's a single line
stating nothing user-facing changed" — and the branch's single implementation
commit touched four files, none of them `CHANGELOG.md`. The ritual asked and
nothing checked, so the entry was never written rather than lost in a merge.
G-0628 never entered a ritual at all: it landed as a direct `docs(skills)`
commit on the trunk, so no surface asked. A check scoped to the patch route
would catch the first and miss the second.

Two of the three rode in on `docs(` commits, and that prefix is why they read as
harmless. It means "no user-visible change" in most repos and the opposite here:
aiwf ships its guidance and rituals as product, embedded under
`internal/skills/`, so a `docs(guidance)` or `docs(skills)` commit touching that
tree changes what every consumer receives on upgrade. The commit convention and
the shipped surface disagree, and nothing reconciles them.

The consequence is unrepairable. Versions are immutable by this project's own
rule, so a change shipped undocumented in vX.Y.Z stays undocumented there; a
later release can only describe it as history. A consumer diagnosing behaviour
that changed under them finds no entry, and the absence is indistinguishable
from the change never having happened.

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

That second property as stated would not have caught v0.34.0's three. Guidance
and ritual text introduce no finding code, no verb, no config key and no exit
code, yet they ship to every consumer on upgrade. The enumerated surfaces have
to include the embedded trees under `internal/skills/` for the property to reach
the case that actually failed — which also makes the `docs(` prefix a signal
worth reading rather than a reason to skip.

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
