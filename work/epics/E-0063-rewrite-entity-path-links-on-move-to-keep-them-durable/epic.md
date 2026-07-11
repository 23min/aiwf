---
id: E-0063
title: Rewrite entity path-links on move to keep them durable
status: active
---

# E-0063 — Rewrite entity path-links on move to keep them durable

## Goal

Make markdown links between entity files survive the file-moving verbs
(`archive`, `rename`, `retitle`, `reallocate`), so an author can cite an entity
with a clickable path-link and trust it stays correct — instead of watching it
rot silently the next time its target moves.

## Context

aiwf offers two ways to reference an entity. A **bare id** (`G-0045`, or a
frontmatter reference field) is id-addressed: the loader's `Tree.ByID` resolves
it across the active tree and `archive/` by construction, so it survives any
move. A **markdown path-link** (`[the loader gap](../work/gaps/G-0045-slug.md)`)
is path-addressed: it bakes a static relative filesystem path into prose, and
nothing revisits it when the target moves.

`aiwf archive` sweeps a terminal entity into its per-kind `archive/`
subdirectory as a pure `git mv` (it emits only `OpMove`, never `OpWrite`), so it
never rewrites any other file's references. `rename` and `retitle` likewise
change an entity's on-disk slug via a pure move and rewrite nothing. Bare-id
references survive all of these for free; path-links rot. The result is measured,
not hypothetical: of the four `docs/adr/*.md` files that link into `work/`,
three were broken by since-archived / since-rewidth'd targets — a 75% rot rate
in the most actively-maintained corner of `docs/`.

The gap that surfaced this, G-0392, first proposed *banning* path-links — a new
`aiwf check` rule steering authors to bare ids. That trades stability for
navigability (a bare id is not clickable on GitHub or in an editor) and taxes a
legitimate authoring convenience. This epic takes the opposite stance: the rot
is caused by moves not updating links, not by path-links being wrong, so the fix
is to **update them on move** and keep path-links first-class.

The precedent already exists and is built the safe way. `rewidth` rewrites
markdown link *destinations* today — `linkPathPattern` plus the
`splitLinkPathRegions` / `rewriteOutsideChunk` region-splitter in
`internal/verb/rewidth.go` operate only on `](…)` destination tokens, exclude
code fences / inline code / URLs, leave `/archive/` and external paths alone,
and are pure and idempotent. It is limited to root-relative `(work/…)` links and
the id-width transform, so it does not cover the relative `../…/work/…` links
that actually rotted, nor the directory-prefix (archive) or slug (rename /
retitle) changes — but the machinery to build on is proven. `reallocate` rewrites
references too, via an id-token substring replace (`idPattern.ReplaceAll`) that
lands the right path only incidentally (the slug is unchanged) and is not
link-region-scoped.

The complementary detection layer already shipped: `wf-doc-lint`'s
markdown-link-integrity check (from G-0390) reports broken links for the surface
this epic deliberately does not auto-rewrite (non-entity `docs/*.md`, `README`)
and as a backstop for raw-`git mv` bypass.

## Scope

### In scope

- A shared, pure, idempotent link-destination rewrite primitive under
  `internal/verb`, generalized from `rewidth`'s machinery to handle **relative**
  destinations (`](../../work/…)`, any depth) and recompute paths against the
  linking file's own directory.
- Wiring the primitive into `archive` (insert the `/archive/` segment; handle a
  multi-entity sweep and epic-subtree dir-rename), `rename`, and `retitle` (swap
  the slug portion), each moving from a rewrite-free move to one that also emits
  the necessary body writes.
- Unifying `reallocate`'s path-link rewriting onto the shared primitive so it is
  link-region-scoped and precise, while keeping its bare-id prose rewrite.
- An ADR recording the decision that entity path-links are first-class and are
  rewritten on move.

### Out of scope

- **A new pre-push `aiwf check` rule / any ban on path-links.** Enforcement lives
  at move-time; the shipped `wf-doc-lint` advisory covers the residual.
- **Auto-rewriting non-entity narrative** (`README`, `CONTRIBUTING`, non-entity
  `docs/*.md`): a verb commit must not reach outside the entity set it owns.
  These stay on advisory detection.
- **Cross-repo / external links**, and links broken by a raw `git mv` that
  bypasses the verbs (advisory backstop only).
- **Reopening ADR-0004** — archive still physically moves; no redirect stubs or
  tombstones.

## Constraints

- **The invariant:** any verb that changes an entity's on-disk path rewrites the
  markdown link destinations in entity bodies that point at it — relative or
  root-relative — through the one shared primitive; prose, code spans, URLs, and
  external paths are left untouched.
- **Entity bodies only.** The primitive rewrites files the loader owns; the
  non-entity surface is detection-only.
- **No hot-path cost.** All rewriting runs inside the (occasional) move verbs, so
  the pre-push chokepoint is untouched — a hard requirement, not a nice-to-have.
- **Pure and idempotent** rewrite core, mirroring `rewidth`'s existing guarantee.
- `tdd: required` across the epic — this is invariant-bearing path/serialization
  logic.

## Success criteria

- [ ] A markdown link between two entity bodies remains correct after the target
      is archived, renamed, or retitled — no manual fix, verified against a
      fixture reproducing the real ADR rot.
- [ ] The reframed decision is recorded: every ADR listed in the *ADRs produced*
      table below is merged.
- [ ] `aiwf check` pre-push wall-clock is unchanged by this epic (the fix adds no
      pre-push rule and no new history walk).
- [ ] The `wf-doc-lint` advisory remains the sole coverage for non-entity
      narrative links, unchanged.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Should `reallocate`'s unification (the optional milestone) ship in this epic or defer, given it fixes precision not rot? | no | Decide at the milestone boundary after the primitive lands; drop it from scope if the epic tightens. |
| Does `archive`'s larger commit (now touching linking bodies) create merge-surface concerns worth a knob? | no | Assess empirically during the archive milestone; `reallocate` already accepts the same trade. |

## Risks (optional)

| Risk | Impact | Mitigation |
|---|---|---|
| Relative-path arithmetic edge cases (`.`/`..` normalization, slash discipline) | med | Property tests over generated tree layouts + move sets; reuse `filepath.ToSlash` conventions. |
| Multi-move (sweep, epic-subtree rename) recomputes against a stale layout | med | Compute all destinations against the final post-move layout, not incrementally; dedicated test. |
| Wider `archive` / `rename` commits raise merge conflicts | low | Precedented by `reallocate`; surface via normal review. |

## Milestones

- `M-0245` — Shared link-destination rewrite primitive: lift `rewidth`'s
  region-splitter, generalize to relative destinations, pure + idempotent, unit-
  and property-tested · depends on: —
- `M-0246` — Wire `archive`: `/archive/`-insertion transform, body writes, sweep
  and epic-subtree multi-move · depends on: `M-0245`
- `M-0247` — Wire `rename` + `retitle`: slug-swap transform, body writes,
  composite-AC no-op path · depends on: `M-0245`
- `M-0248` — Unify `reallocate` onto the primitive for link-region precision
  (optional; refinement, not a rot-fix) · depends on: `M-0245`

## ADRs produced (optional)

- `ADR-0033` — Entity path-links are first-class and rewritten on move

## References

- G-0392 — the gap this epic addresses (originally proposed a ban; superseded by
  this reframe)
- G-0390 — shipped `wf-doc-lint` markdown-link-integrity, the advisory backstop
- ADR-0004 — uniform archive convention (preserved: archive still moves)
- `internal/verb/rewidth.go` — the link-region rewrite precedent to generalize
- `internal/verb/reallocate.go` — the id-token reference rewrite to unify
- `internal/verb/archive.go` — the pure-`OpMove` sweep to extend
