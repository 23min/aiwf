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
| `## Work log` | `aiwf history` — retired, see below |
| `## Dependencies` | the `depends_on:` frontmatter field |
| `## Surfaces touched` | the milestone's own diff |
| `## References` | inline links |

Three of the four are thin: measured over the entity tree on 2026-08-03, median
word counts of 14, 21 and 24 for `## Dependencies`, `## Surfaces touched` and
`## References`.

`## Work log` was the opposite of thin, which is why it went first. Counted with
the per-AC subsections the template prescribed, it had a median of 226.5 words
and was among the spec's largest sections; the 0-word median that opened this
gap came from a method excluding those subsections. Re-measured 2026-08-30: 175
populated Work logs, 122 carrying prose beyond the stated one-line entry, median
283 words against a stated shape of about fifteen.

Its owner covers more than it did. `aiwf history` holds the phase timeline, the
promotes, and — where the `aiwf-tests` trailer is written — the test counts. It
now also holds the link from an AC to the commit that implemented it, wherever
that commit carries the criterion's id: the projection selects on the entity
trailer alone, and the per-AC commit instruction writes the composite id into
it. That link was the one fact this section uniquely held.

The template assigns the rest by name — a trade-off or a rejected approach to
`## Reviewer notes`, a decision to `## Decisions made during implementation`,
design reasoning to the code it explains — and states that anything else has its
own section. So the retirement no longer has to rehome the link.

The per-AC test counts were the one item in the prescribed entry line with no
other owner, and the retirement resolved them by dropping rather than rehoming:
nothing derived them and nothing read them back. Where a project wants them, the
`aiwf-tests` trailer `aiwf promote --phase <p> --tests` writes is the route, and
`tdd.require_test_metrics` makes it required. No ritual instructs it.

`## References` has the weakest claim of the remaining three, and it is worth
stating so the row is not read as equivalent to its neighbours. The other two
name an owner outside the body — a frontmatter field, a diff. This one names
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

That destination is reached. `## Work log` is gone from the template, both
milestone rituals, `aiwfx-wrap-epic`, the builder and reviewer agent cards,
`wf-tdd-cycle`, and the `aiwf-check` and `aiwf-show` verb skills. The
`embedded-no-work-log-section` policy holds it retired: measured before it
landed, an exact revert of the removal tripped nothing in the repo, and three
single-line edits each restored the convention silently. Milestone specs already
carrying a Work log keep it — no check reads the section either way.

## Why it matters

Section count is what makes a spec read as sprawl, and it is the axis nobody has
pruned. Per-unit prose length shows no trend across this repo's history — the
growth is in entity count — so the sections a spec must carry are the part of
the per-entity cost that is actually within reach.

Each of the four is also the duplication D-0054 bans: a fact with an owner,
copied into prose that nothing re-derives. They predate that decision, which is
why they are still shipped.

## Resolution shape

Cut each from `milestone-spec.md` and from every ritual and agent card naming
it. These are template and ritual edits; no kernel semantics change and no ADR.

`## Work log` is done, and it is the only one of the four whose replacement was
designed. What remains is the other three, each blocked on a question this gap
does not answer. `## Dependencies` duplicates `depends_on:`, but the frontmatter
field is not rendered anywhere a spec reader looks. `## Surfaces touched`
duplicates the diff, which is reachable only while the branch is. `## References`
has the weakest claim of the three and the least available owner — see below.
Retiring any of them wants its own measurement of what a reader loses.

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
