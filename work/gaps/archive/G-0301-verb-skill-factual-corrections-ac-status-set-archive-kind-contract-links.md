---
id: G-0301
title: 'Verb-skill factual corrections: AC status set, archive kind, contract, links'
status: addressed
addressed_by:
    - M-0198
---
## Problem

Verified factual errors in the `aiwf-*` verb skills (the entity-reference and
id-placeholder portions of these skills are handled by the strict skill-body
id-hygiene gap and the placeholder normalization; the finding-code documentation
drift is owned by G-0283; this gap is the residual factual content):

- **`aiwf-check`** lists the acceptance-criterion status set as
  `{open, met, cancelled}` ("three statuses"). The kernel set is
  `{open, met, deferred, cancelled}` (`internal/entity/entity.go`); `deferred` is
  a live terminal AC state the wrap rituals use, and `aiwf-promote` lists all
  four. (The separate problem that the skill omits emitted finding codes such as
  `milestone-done-incomplete-acs` is owned by G-0283, which also adds the missing
  documented-superset-of-emitted chokepoint.)
- **`aiwf-archive`** says the volume offender is "typically gaps or **findings**."
  `findings` is not an entity kind; `--kind` accepts only
  `epic, contract, gap, decision, adr` (`internal/cli/archive/archive.go`).
- **`aiwf-contract`** points contributors at `tools/internal/recipe/embedded/`
  for upstreaming a recipe; the real path is `internal/recipe/embedded/` (no
  `tools/` prefix). Its cancel description states a contract moves from
  `proposed`/`accepted` to `rejected`, omitting the `deprecated -> retired` cancel
  case the kernel implements.
- **`aiwf-authorize`** links the provenance-model doc with relative depth
  `../../docs/...`, which resolves to `internal/skills/docs/...` (does not exist);
  the correct depth is `../../../../docs/...`.
- **`aiwf-add`** has a self-contradictory example ("a typo, `M-007` for `M-007`"
  — both ids identical) and cites design docs with fragile pinned line numbers
  (`...:22`, `...:139`).

## Decision

- `aiwf-check`: AC status set -> `{open, met, deferred, cancelled}` ("four
  statuses"). (Finding-code documentation -> G-0283.)
- `aiwf-archive`: drop "or findings".
- `aiwf-contract`: recipe path -> `internal/recipe/embedded/`; add the
  `deprecated -> retired` cancel case.
- `aiwf-authorize`: fix the doc-link depth to `../../../../docs/...` (a doc-link,
  allowed under the ADR doc-link carve-out).
- `aiwf-add`: make the two example ids distinct; cite doc section names rather
  than pinned `:line` anchors.

## Scope

`aiwf-check`, `aiwf-archive`, `aiwf-contract`, `aiwf-authorize`, `aiwf-add`. The
entity-reference + placeholder bits of these skills are out of scope here (handled
by the id-hygiene gap); the aiwf-check finding-code documentation is out of scope
here (G-0283). Rebase onto the id-hygiene gap.
