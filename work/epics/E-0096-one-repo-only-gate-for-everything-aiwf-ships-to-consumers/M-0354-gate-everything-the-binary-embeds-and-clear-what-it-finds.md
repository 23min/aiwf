---
id: M-0354
title: Gate everything the binary embeds and clear what it finds
status: draft
parent: E-0096
tdd: advisory
---
## Goal

One policy test shows that nothing the `aiwf` binary embeds names this repository's
entity ids, source paths, design documents or CLAUDE.md sections, and the embedded
tree comes out clean.

## Closes

- (none)

## Context

Measured at `fde32ce8a` (E-0096 *Context*): the embedded files carry 14 lines citing
this repository's source paths across six files, three design-document paths, ten
lines of milestone ids in `internal/htmlrender/embedded/style.css`, and four
self-references. `skill-body-id` reads only the skill, ritual and guidance markdown
and only for ids; nothing reads paths, the stylesheet or the recipes.
`go list -f '{{.EmbedFiles}}'` names every embedded file: 76 across
`internal/htmlrender`, `internal/recipe` and `internal/skills` on that commit.

## Acceptance criteria

## Constraints

- The scanned set comes from `go list`'s embedded files, not from a list in the
  policy.
- Reuse rather than copy: the id classifier `skill-body-id` uses, and the CLI-text
  policy's rule that a path is this repository's only when its first two segments
  exist in the tree.
- The policy lives in `internal/policies`; nothing is added to `internal/check`.
- A `docs/` path is reported wherever it appears, link destinations included. No
  shipped file links into `docs/` today.
- The development-history clause is not mechanized. The self-references are
  reworded by hand.

## Out of scope

- Deleting `skill-body-id` and `skill-body-claude-md-section` from `aiwf check`,
  and the operator text `internal/check` prints: the next milestone.
- CLAUDE.md's sentence allowing a shipped link to a design document: it is brought
  in line once E-0092 no longer holds CLAUDE.md.

## Dependencies

- None.
