---
title: Entity links by id, not path — retire the rewrite-on-move machinery
status: captured
date: 2026-08-24
---

# Entity links by id, not path — retire the rewrite-on-move machinery

## Classifier note

This is an initiative capture, not a decision. It records a design direction
surfaced while measuring M-0316, together with the precedent that makes it a
known-shape problem rather than an open one. Nothing here is ratified; promotion
to an epic or a gap is the next step, and the migration question in §6 is the
one that decides whether it is worth taking.

It is deliberately not an ADR: ADR-0033 already ratified the opposite direction
(path-links are first-class and repaired on move), and superseding a live ADR is
an act that needs its own deliberation with a working alternative in hand.

## 1. The convenience and its cost

aiwf entity ids are stable by construction. `G-0478` names the same entity
whether its file sits in `work/gaps/` or `work/gaps/archive/`, and the loader
resolves ids across both trees. An id reference survives every rename, every
archive sweep, every reallocation, at zero maintenance cost.

Prose in this repo does not use ids as links. It uses **paths** — 
`[G-0478](../../work/gaps/G-0478-entity-moves-leave-stale-path-links-in-docs-reported-only-after-the-push.md)` —
because a path is clickable on GitHub and in an editor, and a bare id is not.

That single convenience is the origin of an entire subsystem. Measured
2026-08-24 at `419a0890a`:

- **509 lines** of production code across `internal/verb/linkregion.go`,
  `linkrewrite.go` and `pathrewrite.go`.
- **2,275 lines** of link-specific tests beside them.
- **Five verbs** wired through it: `archive`, `rename`, `retitle`, `reallocate`,
  `move`.
- One accepted ADR (ADR-0033), one extension ADR (ADR-0046), one epic (E-0088)
  with four milestones, and two open gaps (G-0478, G-0439) covering the half the
  verbs deliberately do not reach.

Most of the 509 lines are not path arithmetic. They are **masking**: deciding
which spans of a document may be edited at all. Fenced code blocks, inline code
spans, and the boundaries of a link destination all have to be recognized,
because a document about links contains text that looks exactly like links and
must survive byte-identical. A naive rewriter is about twenty lines and corrupts
code samples silently.

## 2. What other systems do

Three families, and every tool in this space sits in one of them.

**A — the link names a thing, not a place.** Sphinx `:ref:`, Hugo's `relref`,
DITA keyrefs, Antora xrefs, Obsidian's `[[Note Name]]`, Logseq and Roam block
refs `((uuid))`. A resolver turns the reference into a location at render time.
Moves cost nothing, because no source text ever named a location.

**B — leave a forwarding address.** HTTP 301, MediaWiki's redirect-on-page-move,
Jekyll's `redirect_from`. The vacated path keeps working. aiwf has already
rejected this: ADR-0033's fourth bullet preserves ADR-0004's move-based archive,
and tombstones sit on the "deliberately not in scope" list.

**C — rewrite the links when the file moves.** VS Code updates markdown links
when a file is dragged; IntelliJ does the same; Obsidian offers "automatically
update internal links". This is what aiwf built.

The instructive case is Obsidian, because it is nearest to this repo's shape. Its
default link is a **name**, not a path, so moving a note between folders costs
nothing — the folder was never part of the reference. It only rewrites on
*rename*, because there the name genuinely changed. Logseq and Roam remove even
that: a block ref is a UUID, immune to rename as well.

The pattern across family A is the same one aiwf already committed to at the
kernel level and then declined to use in prose: **identity decoupled from
location**.

## 3. What aiwf already has that family A requires

The expensive component of family A is the resolver — an index mapping a symbolic
reference to a current location. Sphinx builds one; Obsidian builds a vault
index; Antora builds a content catalog.

aiwf has one. `tree.Load` resolves every id across active and archive trees, and
`aiwf check` already validates that entity cross-references resolve. The piece
that makes family A cheap is present and is not used for links.

## 4. Candidate shape A — the renderer resolves

Prose carries a symbolic reference (`[[G-0478]]`, or a form the renderer knows).
`aiwf render` resolves it to a real destination in `site/`. `aiwf check` reports
one that resolves to nothing, which is close to a rule it already runs.

Rewriting on move disappears entirely: `linkregion.go`, `linkrewrite.go`, the
outbound path added by ADR-0046, and the routing in five verbs all become
unnecessary.

The cost is the convenience that started this: raw markdown on GitHub shows
literal text where a link used to be. For a repo whose planning tree is read on
GitHub as often as through a renderer, that is a real loss and probably
disqualifying on its own.

## 5. Candidate shape B — reference-style links with a generated definition block

CommonMark has native indirection, and GitHub renders it:

```markdown
See [G-0478][] and [ADR-0033][] for the specification.

[G-0478]: ../../work/gaps/G-0478-entity-moves-leave-stale-path-links-in-docs-reported-only-after-the-push.md
[ADR-0033]: ../../docs/adr/ADR-0033-entity-path-links-are-first-class-and-rewritten-on-move.md
```

Both render as ordinary clickable links. The prose carries only ids; **every path
in the document sits in one block, at the bottom, in a fixed syntactic form**
(`[label]: destination`, at line start).

What that changes:

- A mover **regenerates the block** from the id-to-path map the loader already
  computes. It never scans prose again.
- **The masking machinery becomes unnecessary** — most of the 509 lines. Fences,
  inline code spans and destination boundaries stop mattering, because nothing
  edits prose. A derived block is replaced wholesale.
- It is single-source-of-truth applied properly: the id is the fact, the path is
  derived, and the block is a cache whose invalidation rule is "any move".
- It is the pattern this repo already uses for `ROADMAP.md` and `STATUS.md` —
  derived, regenerated, never hand-maintained.

Honest costs. A one-time migration must convert every existing inline link, which
is itself a move-shaped operation the current primitive could perform. Authors
must write the reference form, which is less familiar — enforceable by making an
inline path-link into `work/` an `aiwf check` finding. And a link to a
**non-entity** file still carries a raw path, so the machinery shrinks rather
than vanishing.

## 6. Open questions

| Question | Why it decides the shape |
|---|---|
| Is the migration sweep affordable across the whole tree, including `docs/archive/`? | The archival tier is forget-by-default, so it may be left on raw paths — which means both formats coexist and the rewriter cannot be deleted, only narrowed. |
| Does a generated definition block survive human editing in practice? | If authors hand-edit the block, it stops being derived and becomes a second source of truth — the exact failure mode it was adopted to avoid. |
| Does GitHub's renderer handle the collapsed form `[G-0478][]` consistently in every surface that matters (file view, PR diff, blame)? | Unverified. Worth measuring before committing, not assuming. |
| Do G-0478 and G-0439 close under this, or persist? | Both concern links from `docs/` into `work/`. A generated block reaches those files only if `docs/` is in scope for the generator, which ADR-0033's second bullet currently forbids for verbs. |

## 7. Relationship to E-0088

E-0088 is not superseded by this and was not wasted work. It fixed two real
defects that corrupt documents in the format the repo has today — `move`
rewrote no links at all, and archiving broke a moved file's own outbound links.
Those needed fixing regardless of what format comes next.

It also makes shape B *cheaper*, not more expensive: the id-to-path resolution,
the move planning, and the `FileOp` machinery are precisely what a
definition-block generator would reuse.

What this initiative argues against is **growing** the rewrite-on-move machinery
further. The next improvement in this area is more likely to delete code than to
add it.

## 8. Provenance

Surfaced 2026-08-24 during M-0316, from a question about whether the link
subsystem's size was proportionate to the operation it performs. G-0478's own
resolution shape already gestured at the same direction — *"whether path links
into `work/` should be discouraged in favour of bare ids, which the loader
already resolves and which survive every move. That would shrink the surface
rather than police it."* This document gives that sentence the precedent survey
and the two candidate shapes it lacked.
