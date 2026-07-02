---
id: G-0300
title: Ritual body-fill commits must route through aiwf edit-body (trailered)
status: addressed
addressed_by:
    - M-0201
---
## Problem

Ritual skills that fill an entity's body commit it without the kernel provenance
trailers, tripping the `provenance-untrailered-entity-commit` finding:

- `aiwfx-record-decision` step 7 commits an ADR / decision body with a plain
  `git commit -m "docs(adr): ..."` and no `aiwf-verb` / `aiwf-entity` /
  `aiwf-actor` trailers. This fires every time a decision is recorded. The skill
  never mentions `aiwf edit-body`.
- `aiwfx-plan-epic` (step 5) and `aiwfx-plan-milestones` (step 5) rewrite the
  entity body from a template but do not specify a trailered commit route.
  plan-milestones warns about the untrailered finding only in the `depends_on`
  context (step 6), not for the step-5 body fill; plan-epic only says "the user
  commits when ready" without saying how.

Separately, `aiwfx-record-decision` is the ADR authoring skill yet carries none
of CLAUDE.md §"Authoring an ADR" discipline — nothing warns the author against
writing gate/schedule language into an ADR body ("ratify after X", "status stays
proposed through Y").

## Decision

Route all three skills' body fills through `aiwf edit-body <id> --body-file ...`
— the canonical trailered route the `aiwf-edit-body` skill exists for — so the
body content lands with proper provenance trailers instead of a plain
`git commit`. Add a one-line ADR authoring discipline note to
`aiwfx-record-decision` pointing at CLAUDE.md §"Authoring an ADR" ("decision is
decision"; no gate/schedule language in ADR bodies).

## Scope

`aiwfx-record-decision`, `aiwfx-plan-epic`, `aiwfx-plan-milestones`. Verify at
implementation time that no pre-existing gap already covers the edit-body routing.
