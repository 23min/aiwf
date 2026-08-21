---
id: G-0601
title: aiwf history hides skill edits owned by an entity trailer alone
status: open
discovered_in: M-0312
---
## What's missing

`aiwf history <id>` renders a row only for a commit carrying `aiwf-verb`. M-0312 made a
shipped-surface edit provable by requiring `aiwf-entity` alone, so such a commit
satisfies the `skill-edit-provenance-backstop` gate while producing no row in the history
projection for the entity it names.

Measured on the milestone's own implementation commit: it carries `aiwf-entity: M-0312`,
the gate reads it, and `aiwf history M-0312` does not list it. Every row that projection
does render carries a verb.

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
