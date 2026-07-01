---
id: M-0198
title: Verb-skill factual corrections (status set, kind, paths, links)
status: in_progress
parent: E-0048
depends_on:
    - M-0197
tdd: advisory
acs:
    - id: AC-1
      title: aiwf-check documents the four-status AC set, kernel-derived
      status: met
    - id: AC-2
      title: aiwf-archive drops findings as an archivable kind
      status: met
    - id: AC-3
      title: aiwf-contract cites the real recipe path and complete cancel FSM
      status: met
    - id: AC-4
      title: aiwf-authorize provenance-model doc-link resolves
      status: met
    - id: AC-5
      title: aiwf-add example is self-consistent and cites sections not lines
      status: met
---
## Goal

Five `aiwf-*` verb skills carry verified factual errors that mislead an AI
assistant or operator reading them for authoritative guidance:

- `aiwf-check` understates the acceptance-criterion status set as three
  (`{open, met, cancelled}`); the kernel set is four —
  `{open, met, deferred, cancelled}` — and `deferred` is a live terminal AC
  state the wrap rituals use.
- `aiwf-archive` lists "findings" as an archivable kind; `findings` is not an
  entity kind, and `--kind` accepts only `epic, contract, gap, decision, adr`.
- `aiwf-contract` points contributors at a nonexistent `tools/`-prefixed recipe
  path, and its cancel description omits the `deprecated → retired` cancel case
  the kernel implements.
- `aiwf-authorize` links the provenance-model doc with a relative depth that
  resolves to a nonexistent path.
- `aiwf-add` has a self-contradictory example (identical ids) and cites design
  docs by fragile pinned line numbers.

This milestone corrects all five and pins each with a structural test under
`internal/policies/` — **source-derived** where the skill enumerates a
kernel-defined set (AC statuses, contract cancel targets), so the enumeration
becomes a permanent guard that cannot silently drift from the kernel again,
rather than a one-time fix. The edits are to `embedded/` verb skills, so the
ritual `skill-edit → structural-test` backstop does not apply; `skill-body-id`
(G-0299) does, and the doc-links stay masked/allowed under its carve-out.

Out of scope (owned elsewhere): the finding-code documentation drift
(`G-0283`, delivered by M-0197), and the entity-reference / id-placeholder
hygiene of these skills (the strict skill-body id gap).

Source: G-0301. Parent epic E-0048.

## Acceptance criteria

### AC-1 — aiwf-check documents the four-status AC set, kernel-derived

The `aiwf-check` skill's `acs-shape/status` row names the acceptance-criterion
status set as `{open, met, deferred, cancelled}` (four statuses), matching the
kernel's `entity.AllowedACStatuses()`.

Test: a structural test derives the set from `entity.AllowedACStatuses()` and
asserts the skill's `acs-shape/status` row — scoped to that table row, not a
flat body grep — names exactly those four statuses (and no longer says "three").
Source-derived, so the row cannot drift from the kernel set.

### AC-2 — aiwf-archive drops findings as an archivable kind

The `aiwf-archive` skill no longer presents "findings" as an archivable kind;
its kind vocabulary is consistent with the verb's `--kind` accepted set
(`epic, contract, gap, decision, adr`).

Test: assert the skill body contains no "findings"-as-kind reference (the
specific "gaps or findings" error is gone), and that the kinds it does name are
a subset of the `--kind` set derived from `internal/cli/archive/`.

### AC-3 — aiwf-contract cites the real recipe path and complete cancel FSM

The `aiwf-contract` skill references the upstream-recipe path
`internal/recipe/embedded/` (not the nonexistent `tools/internal/recipe/embedded/`),
and its cancel description documents the `deprecated → retired` cancel case
alongside `proposed`/`accepted → rejected`.

Test: assert the skill references `internal/recipe/embedded/`, that this path
exists on disk, and that no `tools/`-prefixed recipe path remains; assert the
cancel documentation covers every `CancelTarget(KindContract, …)` target
(source-derived — `proposed`/`accepted → rejected` and `deprecated → retired`).

### AC-4 — aiwf-authorize provenance-model doc-link resolves

The `aiwf-authorize` skill's provenance-model markdown link uses depth
`../../../../docs/pocv3/design/provenance-model.md`, which resolves to an
existing file from the skill's source location.

Test: extract the link's relative destination, join it to the skill file's
directory, and assert the resolved path exists — catching broken relative depth
mechanically rather than by eye.

### AC-5 — aiwf-add example is self-consistent and cites sections not lines

The `aiwf-add` skill's "typo" example uses two distinct ids (not the same id
twice), and its design-doc citations name sections rather than pinned `:NN`
line anchors.

Test: assert the typo example names two different ids; assert the skill body
contains no `docs/….md:NN` pinned-line citation (fragile anchors that rot as the
docs change).
