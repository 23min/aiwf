# Epic wrap — E-0023

**Date:** 2026-05-10
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-23-uniform-id-width
**Merge commit:** 99045d4

## Milestones delivered

- M-0081 — Canonical 4-digit IDs in parser, renderer, and allocator (merged f2e1377)
- M-0082 — Implement `aiwf rewidth` verb and apply to this repo's tree (merged 0b2d6bd)
- M-0083 — Drift check, normative-doc amendments, and skill content refresh (merged f917e70)

A discovered-during-wrap patch — `fix/rewidth-preflight-checks` — also rode in on the epic branch (merged 2b48d62), hardening M-0082's `aiwf rewidth` verb with a default-on preflight before the destructive `--apply` step.

## Summary

The kernel's id-width policy is now uniformly 4 digits across all six entity kinds (epic, milestone, ADR, gap, decision, contract — and forward-compatible for finding F-NNNN per ADR-0003). Parsers tolerate narrower legacy widths on input so existing trees, branches, and commit trailers continue to validate without history rewrite; renderers and allocators always emit canonical width. Downstream consumers carrying narrow-legacy trees migrate via `aiwf rewidth --apply` (one commit, idempotent, archive entries preserved per forget-by-default; default-on preflight gates `aiwf check` errors before the apply commits).

This repo migrated to canonical width as part of M-0082's wrap (200 file renames + 212 body rewrites in commit `f937288`); M-0083's drift-check rule `entity-id-narrow-width` is now silent on the post-rewidth tree. The CLAUDE.md "What aiwf commits to" §2 collapses to a single uniform rule; ADR-0003 §"Id and storage" reflects F-NNNN with a cross-reference to ADR-0008.

## ADRs ratified

- ADR-0008 — Canonicalize kernel IDs to 4 digits; parsers tolerate narrower legacy widths on input. _Promoted from `proposed` to `accepted` at wrap; the entire epic implements this policy._

## Decisions captured

None as standalone D-NNNN entries. Three mid-implementation decisions are recorded inline in milestone-spec Reviewer notes:

- **`Canonicalize` is below-grammar-floor-tolerant** (M-0081) — narrow-width inputs below the per-kind grammar floor (`E-1`, `M-22` for the M-NNN floor) pass through verbatim rather than being padded. The grammar's per-kind floor defines the input space; `Canonicalize` tolerates legacy widths but does not invent well-formed ids from non-conforming input.
- **Width-tolerance fallback in `design-doc-anchors-valid` policy** (M-0082) — when a path-form reference doesn't resolve at its authored width, the policy retries the canonical-width form. Same theme as M-0081's parser tolerance for entity ids; lets docs/pocv3/ references survive the migration window until M-0083's narrative sweep canonicalizes the prose.
- **ADR exempt from `entity-id-narrow-width` mixed-state classification** (M-0083) — ADR's grammar (`ADR-\d{4,}`) was always 4-digit canonical; including it in the rule's classification would taint pre-migration trees as "mixed."

## Follow-ups carried forward

- G-0093 — _Mixed kernel ID widths can't survive PoC graduation_ — closed by this epic; the gap should resolve at wrap (status flipped from `open` to `resolved`).

No new gaps were opened during the epic. The discovered preflight surface gap was addressed inline via the `fix/rewidth-preflight-checks` patch rather than deferred.

Two epic-spec out-of-scope items remain on file as the canonical follow-ons:

- **G-0091** — preventive check rule for path-form refs (mentioned in the epic spec as out-of-scope; survives independently).
- **§07 TDD architecture proposal Slice 2 (F-NNNN as 7th kind)** — explicitly called out in the epic spec as a downstream consumer; the F kind is now born canonical because of this epic's allocator change.

## Handoff

The kernel's canonical-width policy is mechanically locked end-to-end:

- `entity.CanonicalPad = 4` is the single source of truth for id width across allocator, renderer, and verb-side helpers.
- `entity.Canonicalize(id)` and `entity.IDGrepAlternation(id)` give consumers width-tolerant lookup and history-grep helpers.
- `aiwf rewidth` is the on-demand migration verb; default-on preflight (`aiwf check` error gate + layout-shape warnings) protects downstream consumers from surprising commits.
- `entity-id-narrow-width` check rule fires only on mixed-state active trees — silent on uniform-narrow (consumer hasn't migrated) and uniform-canonical (consumer has migrated cleanly).
- ADR-0003 / CLAUDE.md / `internal/policies/testdata/aiwfx-whiteboard/SKILL.md` reflect canonical width; the rituals plugin's 5 enumerated files match (rituals SHA `808ad70bb368c7d687a207cc7b749e0b11529323`).

What's left open and intentional:

- Body-prose ids inside inline backtick spans and code fences are preserved verbatim per the rewidth verb's spec — these are literal id text in documentation, not real path-form references.
- Archive entries (`<kind>/archive/...`) remain at their birth-width per ADR-0004 forget-by-default.
- Old commit trailers in pre-migration commits are not rewritten; they continue to match canonical-id queries via parser tolerance.

## Doc findings

Scoped wf-doc-lint over the epic's change-set:

**Broken code references:**

- `docs/adr/ADR-0008-...md` — three references to `internal/verb/import.go::canonicalPadFor` (lines 8, 38, 117). The function was deleted in M-0081's AC-1 (consolidated into `entity.CanonicalPad`). Deliberately preserved as historical pre-migration commentary in an ADR body — flagged in M-0081's and M-0083's milestone-spec Reviewer notes. The reviewer can decide whether to add a post-migration footnote pointing at `entity.CanonicalPad`; the kernel mechanically validates that all *current* references resolve, which they do.

**Removed-feature docs:** none.

**Orphan files:** none introduced.

**Documentation TODOs:** none new in the epic's design/plans tree.

**Mechanical chokepoint status:**

- `TestPolicy_DesignDocAnchors` — green (post-rewidth design-doc references resolve via the width-tolerance fallback added in M-0082 prep).
- `TestPolicy_DocTreeNarrowIDsCanonicalized` — green (M-0083's doc-tree sweep + 16-entry allowlist).
- `TestPolicy_ThisRepoTreeIsClean` — green (M-0081 AC-6 chokepoint; no id-width-shaped findings).
- `TestPolicy_ThisRepoDriftCheckClean` — green (M-0083 AC-5 chokepoint; the new drift-check rule is silent on this repo's uniform-canonical post-rewidth tree).
- `TestAiwfxWhiteboard_AC8_MaterialisationDriftCheck` — green (kernel fixture matches the active rituals-plugin install at SHA `808ad70bb368c7d687a207cc7b749e0b11529323`).
