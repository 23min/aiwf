---
id: G-0338
title: Compress CLAUDE.md to a primed rules-only digest under the size limit
status: open
---
## What's missing

The root `CLAUDE.md` is ~93k chars — 2.3x Claude Code's 40k limit. One section
(`## Go conventions`, ~55k) is 59% of the file; it interleaves the actual rules
with rationale, "why this rule exists" war-stories, `History:` footnotes, dense
inline entity-genealogy citations, and a ~40-row "What's enforced and where"
table. None of that primes the assistant — the rules do; the rest dilutes
attention and blows the budget.

CLAUDE.md is a priming document: it loads every turn, so it must be a tight,
scannable directive set, not a narrative. Verbose is worse than terse here — a
rule buried in a 200-word story primes less effectively than the bare directive.

## Why it matters

Over the 40k limit the file risks truncation (silently dropping the tail — the
`@`-imports live at the very bottom); and truncated or not, the bulk dilutes the
priming attention that keeps behaviour on-rails. Because Go work here is near
constant, the Go rules must stay primed — so the fix is compression in place,
not extraction to on-demand docs, which would lean on the assistant remembering
to read them (the LLM-dependency the kernel principle forbids).

## Proposed fix shape

Rewrite CLAUDE.md as a rules-only digest:

- Keep every load-bearing directive and every don't/gotcha, condensed to the
  imperative. Distinguish a nuanced *rule* (removing it changes what to do —
  keep) from *rationale* (only explains why — cut).
- Cut rationale, "why this rule exists" war-stories, and `History:` footnotes
  outright — not relocated to a rationale doc. The why stays recoverable in the
  referenced gaps/ADRs, the design docs, and git history.
- Cut the "What's enforced and where" table; replace with a one-line pointer
  (chokepoints live in `internal/policies` + `internal/check` + the git hooks;
  run `make ci` / `make coverage-gate`).
- Minimise inline entity references; keep an id only where it is a genuine
  where-to-find-more pointer, not provenance.
- Keep tight pointers to the deep docs (design-decisions, provenance-model, ADRs).

## Constraints

- Some `internal/policies` tests pin specific CLAUDE.md content (the devcontainer
  section, the test-running sections, the gate-discipline bullet's phrasing).
  Compression must preserve those anchors or update the tests in the same change.
- git history preserves the current CLAUDE.md, so nothing is truly lost.
- Refines the M-0211 dividing principle: repo-development guidance stays primed
  in CLAUDE.md, but as tight directives — rationale, lookup tables, and history
  do not.

## Verification

- CLAUDE.md is under the 40k limit (or, if the limit only warns, materially
  smaller and rules-dense).
- No `internal/policies` test regresses: the pinned CLAUDE.md anchors still
  resolve, or the tests are updated in the same change.
