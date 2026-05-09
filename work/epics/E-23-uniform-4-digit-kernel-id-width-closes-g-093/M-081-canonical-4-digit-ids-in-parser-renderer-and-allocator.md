---
id: M-081
title: Canonical 4-digit IDs in parser, renderer, and allocator
status: draft
parent: E-23
tdd: required
---
## Goal

Make the kernel's id width policy uniform: parser accepts both narrow and canonical widths on input; renderer canonicalizes every display surface to 4-digit form; allocator emits 4-digit form for every kind. Hardcoded narrow-width ids in the test suite update to canonical form. After M-A ships, every consumer's existing tree (still on narrow-width filenames) continues to validate; new allocations are canonical; display is uniform.

The change is pure-additive at the parser layer — acceptance widens, no existing valid input becomes invalid. Old trees, branches, skills with hardcoded narrow widths keep working indefinitely.

## Context

ADR-0008 names this milestone as the load-bearing kernel change before the migration verb (M-B) and the drift check (M-C) can ship. The migration verb depends on parser tolerance to read existing trees; the drift check depends on the allocator emitting canonical ids so that "mixed-state tree" is a meaningful signal.

Today's policy is encoded in `internal/verb/import.go::canonicalPadFor`, which returns 2 for epic, 4 for ADR, and 3 for everything else. This milestone unifies it: every kind returns 4. The renderer and the allocator both consume `canonicalPadFor`; consolidating the policy into a single source of truth is implicit in the change.

## Acceptance criteria

(ACs allocated separately via `aiwf add ac` after milestone creation; bodies seeded at allocation time.)

## Constraints

- **Pure-additive parser change.** The parser's accept set widens; no existing valid input becomes invalid. AC-2 asserts both-widths-equivalence; AC-6 asserts this repo's existing tree continues to validate.
- **No git history rewrite.** Old commit trailers (`aiwf-entity: E-22`) keep matching via parser tolerance; the kernel never re-emits narrow-width ids in new commits. AC-4 covers this.
- **Single source of truth for pad width.** The allocator and renderer both consume `canonicalPadFor`; if M-A introduces a duplicate (a new pad-width helper alongside the existing one), the next milestone's TDD pass catches the drift. AC-1 covers this.
- **Composite ids included.** `M-NN/AC-N` and `M-NNNN/AC-N` parse equivalently; the existing composite-id parser is one of the audited sites. AC-2 covers composite cases.
- **TDD: required.** Each AC drives a red→green→refactor cycle. AC-5 (test-fixture sweep) is content edit (no new logic) but its assertion is mechanical: a structural grep that returns matches only inside an allowlist.

## Design notes

### Pre-implementation: id-parsing call site audit

The TDD pass starts with a grep-derived enumeration of every id-parsing call site in the kernel. Initial list (committed to this section as the work begins; updated if new sites surface during implementation):

- `internal/entity/` — `ParseID`, `Split` (composite-id parser for `M-NN/AC-N`), frontmatter forward-ref validators for `parent`, `depends_on`, `linked_acs`, `linked_entities`, `discovered_in`, `linked_adr`, `superseded_by`, `waived_by`.
- `internal/gitops/` — `ParseTrailers` (extracts ids from `aiwf-entity:` trailer values).
- `internal/check/` — `refsResolve` and any rule that parses ids from frontmatter or body content.
- `internal/render/` — anchor/link generators for HTML output; id-bearing fields in any rendered envelope.
- `internal/verb/` — id-handling in verbs that take ids as positional args (`promote`, `cancel`, `show`, `history`, `reallocate`, `authorize`, `edit-body`, `retitle`, `rename`).
- `cmd/aiwf/` — Cobra completion functions emitting/accepting entity ids.

Each call site is exercised by a table-driven test in M-A's TDD pass with both narrow and canonical inputs, asserting equivalent resolution.

### Canonical pad width unification

`canonicalPadFor(kind)` currently lives in `internal/verb/import.go` (with a duplicate formatting site in `internal/entity/AllocateID`). This milestone consolidates to a single source — preferably `internal/entity/` since pad width is an entity-level property, not a verb-specific one. The verb-side function becomes a thin re-export or is deleted; verbs consume the entity-side helper directly.

If consolidation is more disruptive than the milestone's scope wants, both sites update consistently to return 4 for every kind, with a short follow-up gap to consolidate later. The duplication is currently silent but it's the kind of footgun that bites M-C's drift check — easier to fix here.

### Renderer canonicalization scope

The renderer reads on-disk filenames (which may be narrow) but rewrites display output to canonical width. Two cases:

- **Id appears in structural output** (`aiwf show`'s frontmatter render, JSON envelope, HTML anchor): pass the id through a `canonicalize(id)` helper that left-pads to canonical width. Helper lives in `internal/entity/`.
- **Id appears in body prose** (markdown content, e.g., `aiwf show`'s rendered body): the body content is what the user wrote on disk; the renderer doesn't rewrite prose. Body-content canonicalization happens in M-B's `aiwf rewidth` verb, not at render time.

The distinction matters: M-A canonicalizes *structural* surfaces only. Body content stays as authored until M-B rewrites it.

## Surfaces touched

- `internal/entity/` — `AllocateID`, `ParseID`, composite-id parser (`Split`), frontmatter validators, new `canonicalize(id)` helper.
- `internal/gitops/` — `ParseTrailers` (trailer-value tolerance).
- `internal/check/` — `refsResolve` and any rule with id-parsing.
- `internal/render/` — display-surface canonicalization.
- `internal/verb/` — verbs that take id arguments (most consume entity helpers; few or no direct changes expected).
- `cmd/aiwf/` — Cobra completion functions.
- `internal/**/*_test.go`, `internal/**/testdata/`, `cmd/aiwf/*_test.go` — fixture sweep to canonical width.
- `internal/policies/` — possibly a new policy test asserting the both-widths-equivalent invariant if the drift-prevention shape fits cleanly.

## Out of scope

- The migration verb `aiwf rewidth` — that's M-B.
- The drift-check rule `entity-id-narrow-width` — that's M-C.
- File renames in this repo's `work/` and `docs/adr/` trees — those happen when `aiwf rewidth --apply` runs in M-B's wrap.
- ADR-0003 amendment (F-NNN → F-NNNN) — that's M-C.
- CLAUDE.md commitment #2 update — that's M-C.
- Embedded skill content refresh — that's M-C.
- Rituals plugin coordination — that's M-C.
- Doc-tree narrow-id sweep (`docs/`, `README.md`, `CHANGELOG.md`) — those refresh in M-C's doc updates.
- Consolidating `canonicalPadFor` if it requires substantial restructuring beyond a thin re-export. If consolidation is non-trivial, both sites update consistently with a follow-up gap.
