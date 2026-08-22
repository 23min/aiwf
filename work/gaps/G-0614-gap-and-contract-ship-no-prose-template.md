---
id: G-0614
title: Gap and contract ship no prose template
status: open
---
## What's missing

`internal/skills/embedded-rituals/plugins/aiwf-extensions/templates/` ships four
prose-guided entity templates — `adr.md`, `decision.md`, `epic-spec.md`,
`milestone-spec.md`. Gap and contract have none.

For those two kinds the only scaffold is `entity.BodyTemplate`, which
`aiwf template gap` prints and `aiwf add` commits. It emits the two headings
`internal/entity/required_sections.go` names for the kind and nothing under them:
no instruction on what belongs in a section, and for gap no instruction that a
body must say *where* the defect is — a file, a symbol, or an observable
behaviour.

G-0541 named this absence and listed writing the two templates as the first part
of its resolution. That part did not land. The milestone holding the scope,
M-0307, was cancelled, and the single commit that closed the gap fixed only the
citation path that pointed at the missing files. The gap is `addressed` and its
FSM has no reverse edge, so the remainder is untracked.

## Why it matters

A prose template is the only anchor a gap body has for the layer above its two
required sections, and without one that layer fragments. Measured over the 166
gap files in the active tree on 2026-08-22: 135 carry `## What's missing` and
142 `## Why it matters` — the pair the scaffold writes. Above them sit
`## Resolution shape` 28, `## Scope` 24, `## Related` 22, `## Problem` 22,
`## Options` 16, `## Direction` 14, `## Where to fix` 12, `## Fix shape` 5,
`## Resolution options` 3, `## Proposed fix shape` 3. Six of those name the same
thing — what to do about it — and `## Problem` restates the section already
required.

Nothing reports any of it. Sections beyond the required set are legal and never
flagged, and a required section absent outright is invisible to `entity-body-empty`
(G-0571), so the fragmentation is silent on every read path.

This repo absorbs the cost, because a thousand neighbouring entities carry the
convention where the template does not. A consumer tree has no neighbours. There
the materialized template is the whole of what an assistant has to go on, and for
gap and contract there is nothing to materialize.
