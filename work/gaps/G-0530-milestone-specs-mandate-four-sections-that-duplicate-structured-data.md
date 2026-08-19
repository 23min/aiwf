---
id: G-0530
title: Milestone specs mandate four sections that duplicate structured data
status: open
priority: medium
---
## What's missing

The milestone spec template ships fifteen top-level sections, and four of them
carry content something else already holds:

| section | what already holds it |
|---|---|
| `## Work log` | `aiwf history` and the commit trailers it reads |
| `## Dependencies` | the `depends_on:` frontmatter field |
| `## Surfaces touched` | the milestone's own diff |
| `## References` | inline links |

Measured over the entity tree on 2026-08-03, all four are thin: median word
counts of 0, 14, 21 and 24 respectively. `## Work log` is the sharpest case —
the wrap ritual mandates one entry per acceptance criterion with its outcome and
commit SHA, and the section is empty in half the milestones that carry it.

`## References` has the weakest claim of the four, and it is worth stating so the
row is not read as equivalent to its neighbours. The other three each name an
owner outside the body — a git log, a frontmatter field, a diff. This one names
prose elsewhere in the same file. `relates_to` would be a structured owner, but
it is not available here: `internal/entity/entity.go` declares it under
`KindDecision` alone, and of the 47 files in the tree carrying the field, every
one is a decision and none is a milestone. Whether prose-elsewhere is enough to
retire the section is the open part of this row; the other three do not depend on
the answer.

## Why it matters

Section count is what makes a spec read as sprawl, and it is the axis nobody has
pruned. Per-unit prose length shows no trend across this repo's history — the
growth is in entity count — so the sections a spec must carry are the part of
the per-entity cost that is actually within reach.

Each of the four is also the duplication D-0054 bans: a fact with an owner,
copied into prose that nothing re-derives. They predate that decision, which is
why they are still shipped.

## Resolution shape

Cut the four from `milestone-spec.md` and from the wrap ritual's step 4. Both are
template and ritual edits; no kernel semantics change and no ADR.

`## Reviewer notes` is the largest section by word count and is deliberately not
on the list: it carries the declined-finding record that keeps a fresh reviewer
from re-raising a settled question, which is the leak D-0054 narrowed rather
than widened. `## Decisions made during implementation` and `## Validation` are
likewise held: their content has no other owner.

## Method limits

The measurement pools every milestone in the tree against whatever template
generation it was written under, and the template has changed more than once.
A section can therefore read as thin because it was dropped or renamed midway,
not because authors decline to fill it. The four above were confirmed present in
the current template, so the finding holds for milestones written today — but
the medians themselves are not per-generation and should not be quoted as if
they were.

Settling this wants a per-generation audit: bucket entities by the template
revision in force when they were created, and re-measure within each bucket. The
same treatment would sharpen the epic and gap surfaces, neither of which has
been examined this way. Gap is the largest population and has no template at all,
so its structure is convention carried in guidance and skills rather than a file
that can be edited.

The classification of `entity-body-empty` in the shipped-surface table of
`docs/design/growth.md` records it as a mandate. It is a ban: it fires only on a
present-and-empty section, so omitting a heading satisfies it. The table needs
that correction, and the mechanism it obscures is worth stating in its place —
a template that seeds headings and a ban that forces them filled compose into a
mandate, while neither is one alone.
