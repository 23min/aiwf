---
id: M-0354
title: Gate everything the binary embeds and clear what it finds
status: draft
parent: E-0096
tdd: advisory
acs:
    - id: AC-1
      title: The gate scans every file the binary embeds
      status: open
    - id: AC-2
      title: The gate reports aiwf ids, source paths and design-doc citations
      status: open
    - id: AC-3
      title: The shipped tree carries no gate finding
      status: open
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

### AC-1 — The gate scans every file the binary embeds

The policy scans exactly the files `go list` reports as embedded for the packages
under `internal/` and `cmd/`. **Pass criterion**: in a fixture module, a file
embedded from a newly named directory is scanned with no change to the policy, and a
file only a test embeds is not. **Edge cases**: a glob directive, a single-file
directive, a directory directive. **Code references**: the new policy and its test
under `internal/policies/`.

### AC-2 — The gate reports aiwf ids, source paths and design-doc citations

In a scanned file the policy reports a real entity id at any width, a path whose
first two segments exist in this tree, a path under `docs/` other than `docs/adr/`,
and a citation of a named CLAUDE.md section. **Pass criterion**: each of those shapes
yields one finding naming the file and line; the canonical `<prefix>-NNNN`
placeholder, an example path absent from the tree such as `internal/billing`,
`docs/adr/`, and a mention of CLAUDE.md without a section yield none. **Edge cases**:
a path inside a markdown link destination, a path inside backticks, an id inside a
CSS comment. **Code references**: the new policy under `internal/policies/`, reusing
the classifier in `internal/check/skill_body_id.go` and the path rule in
`internal/policies/cli_text_internal_ids.go`.

### AC-3 — The shipped tree carries no gate finding

The policy reports nothing on this repository's tree. **Pass criterion**: its test
passes on the tree, with every leak measured at `fde32ce8a` removed by rewording:
a source path becomes the behaviour or command it stood for, a design-document path
is dropped or replaced by the consumer-facing command, the stylesheet comments lose
their milestone ids, and the self-references name the consumer's repository or
nothing. **Edge cases**: the `aiwf-check` skill rows that describe `skill-body-id`
by this repository's tree paths are reworded, not deleted, while the rule still
exists. **Code references**: the files listed in E-0096 *Context*.

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
