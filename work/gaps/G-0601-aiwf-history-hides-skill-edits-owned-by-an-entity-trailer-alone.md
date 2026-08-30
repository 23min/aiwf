---
id: G-0601
title: aiwf history hides skill edits owned by an entity trailer alone
status: open
discovered_in: M-0312
---
## What's missing

`aiwf history <id>` selects commits by grepping for an `aiwf-entity:` trailer naming the
id, then discards any whose `aiwf-verb:` and `aiwf-actor:` trailers are both empty
(`internal/entityview/historyevent.go`). The discard exists to drop a false positive:
`--grep` matches a wrapped prose line beginning `aiwf-entity: <id>` where git's trailer
parser finds no genuine trailer. Proxying that test through verb-or-actor also discards
commits whose entity trailer is genuine and whose only fault is carrying nothing else.

M-0312 made a shipped-surface edit provable by requiring `aiwf-entity` alone, so such a
commit satisfies the `skill-edit-provenance-backstop` gate while producing no row in the
history projection for the entity it names.

Measured on the milestone's own implementation commit: it carries `aiwf-entity: M-0312`,
the gate reads it, and `aiwf history M-0312` does not list it.

Re-measured 2026-08-30 over 10,709 commits: 8,344 carry an `aiwf-entity:` trailer and 44
carry it alone, every one of them invisible to `aiwf history`. A commit carrying entity
plus actor and no verb does render, so the rule is verb-or-actor rather than verb.

A live instance outside the skill-edit case: `2c79510fa` implemented G-0650 and carries
`aiwf-entity: G-0650` because the backstop requires it. `aiwf history G-0650` lists the
add, two body edits and the promote — the paperwork — and not the commit that did the
work.

The same blindness is why a milestone's `## Work log` is the only index from an
acceptance criterion to the commit that implemented it. An AC implementation commit could
carry `aiwf-entity: M-NNNN/AC-N`, and composite ids are already valid in that trailer,
but the projection would discard it. Retiring that section (G-0530) depends on this.

## Why it matters

Auditability is the whole point of provenance, and `aiwf history <id>` is the audit
surface an operator reaches for. A rule that makes an edit gate-visible but
history-invisible records the fact somewhere only the gate looks, which is most of the
way back to the state G-0220 recorded — the edit reaches consumers and the surfaces a
human consults say nothing.

## Options

The obvious repair is not available. Requiring `aiwf-verb` on these commits is exactly
what D-0071 rejected: no aiwf verb commits source, the closed set `trailer-verb-unknown`
enforces carries no value meaning "I edited a shipped surface", and minting one
reintroduces the fabricated-trailer defect G-0150 closed.

What remains is a choice about the projection rather than the trailer set — whether
`aiwf history` should render entity-trailered commits that carry no verb, and if so how
to label a row with no verb to name. That is a design decision with its own consequences
for every other consumer of the trailer, and it is why this is filed rather than fixed in
place.

The shape the fix takes: the query already extracts eleven trailer keys through git's own
parser, so extracting `aiwf-entity` the same way and testing that instead of
verb-or-actor separates a genuine trailer from a prose match by the mechanism the false
positive is about, rather than by a proxy. Changing the filter changes existing output —
the 44 commits above begin appearing in histories that omit them today, which is the
defect being fixed rather than a regression, though tests pinning history output move
with it.
