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
| `## Work log` | partly `aiwf history` — see below |
| `## Dependencies` | the `depends_on:` frontmatter field |
| `## Surfaces touched` | the milestone's own diff |
| `## References` | inline links |

Three of the four are thin: measured over the entity tree on 2026-08-03, median
word counts of 14, 21 and 24 for `## Dependencies`, `## Surfaces touched` and
`## References`.

`## Work log` is the opposite, and its row needs the correction the gap-truth
audit recorded and nobody applied. Counted with the per-AC subsections the
template prescribes, it had a median of 226.5 words and was the spec's
third-largest section; the 0-word median came from a method that excluded the
subsections. Re-measured 2026-08-30: 175 populated Work logs, 122 of them
carrying prose beyond the stated one-line entry, median 283 words against a
stated shape of about fifteen. It is the largest section, not the thinnest.

Its owner is also only half right. `aiwf history` holds the phase timeline, the
promotes and — where the `aiwf-tests` trailer is written — the test counts. It
does **not** hold the link from an AC to the commit that implemented it. That was
the one fact the section uniquely held, and it no longer is: `aiwf history`
selects on the entity trailer alone, and the per-AC commit instruction writes the
criterion's id into it, so the link is answerable without the spec. What remains
unheld elsewhere is the narrative — what a detour cost, why an approach was
abandoned — which is what a retirement has to decide about.

`## References` has the weakest claim of the four, and it is worth stating so the
row is not read as equivalent to its neighbours. The other three each name an
owner outside the body — a git log, a frontmatter field, a diff. This one names
prose elsewhere in the same file. `relates_to` would be a structured owner, but
it is not available here: `internal/entity/entity.go` declares it under
`KindDecision` alone, and of the 47 files in the tree carrying the field, every
one is a decision and none is a milestone. Whether prose-elsewhere is enough to
retire the section is the open part of this row; the other three do not depend on
the answer.

## What replaces it

`## Work log` is not deleted into a hole. What a human opening an archived
milestone wants is not a per-AC log but what the milestone delivered, and that
is also the one input the epic wrap's `## Changelog entry` has never had — it is
written from milestone titles and merge SHAs, with no milestone-level source.

So the section is replaced by a short `## Release note`: a few sentences of
user-visible delta, bounded by its consumer's format, with a named downstream
reader the Work log never had. The per-AC mechanics move to `aiwf history`, where
they are derived and cannot drift. The milestone then keeps two prose sections
written after the work — `## Release note` and `## Reviewer notes` — each bounded
and each with a reader.

The section's purpose is now stated in the template and in
`aiwfx-start-milestone`, and the two texts that invited unbounded prose are gone.
That is the interim state, not the destination.

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
