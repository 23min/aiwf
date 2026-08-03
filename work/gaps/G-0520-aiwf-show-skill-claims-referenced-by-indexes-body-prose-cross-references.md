---
id: G-0520
title: aiwf-show skill claims referenced_by indexes body-prose cross-references
status: open
priority: medium
---
## What's missing

The `aiwf-show` skill's "Output shape" section describes the **Referenced by**
block as:

> every other entity citing this one (typically `parent:` links from milestones
> to their epic; `depends_on:` links; cross-references in body prose).

The final clause is false. The reverse-reference index is built exclusively from
frontmatter: `tree.buildReverseRefs` (`internal/tree/tree.go:346`) iterates
`entity.ForwardRefs` (`internal/entity/refs.go:28`), which reads only the typed
per-kind reference fields — `parent`, `depends_on`, `supersedes`,
`superseded_by`, `discovered_in`, `addressed_by`, `relates_to`, `linked_adrs`.
An id mentioned in body prose contributes nothing to `referenced_by` on any
surface: not the `aiwf show` text block, not the JSON envelope's
`referenced_by`, and not the render's "Linked entities" list
(`internal/cli/render/resolver.go:592`, which unions `ForwardRefs` with
`tree.ReferencedBy`).

The two mechanisms are genuinely easy to conflate, because body-prose ids *are*
validated — the `body-prose-id/*` check family resolves them and flags dangling
or malformed ones. Validated is not indexed.

## Why it matters

That sentence is the one place an agent looks to learn what the reverse edge
covers, and it is wrong in the two directions that matter:

1. **It over-promises prose.** An agent that records a cross-reference only in
   body prose expects the target to surface it under "Referenced by". Nothing
   will.

2. **It omits the derived contract.** For the structured fields the reverse
   edge is *computed at tree load* — so it must never be hand-authored. The
   skill never says this, and a reader who takes the prose clause at face value
   concludes the opposite: that back-references are prose the author maintains.

The second failure has been observed in practice. A downstream consumer created
a decision carrying `--relates-to <two sibling decision ids>`, then reasoned
that because `relates_to` has no post-creation mutation verb, the two targets
"can't gain a back-reference by verb" — and proposed a follow-up
`aiwf edit-body` on each target to write a prose back-reference, per a local
convention that prose-only edges be bidirectional. Both targets already carry
the edge, derived, on all three surfaces.

The cost of acting on that reading is a hand-maintained second copy of a fact
the kernel computes, plus one extra `edit-body` commit per relation edge ever
created — the single-source-of-truth violation the kernel exists to prevent,
reached by reading our own documentation correctly. The local convention is not
at fault: a `relates_to` edge is structured, not prose-only, so the convention
never applied. The skill is what makes that boundary invisible.

## Fix shape

Rewrite the "Referenced by" line in the `aiwf-show` skill to:

1. name the frontmatter reference fields as the index's actual and only input;
2. state that body-prose ids are check-validated but not indexed, so a prose-only
   mention creates no edge;
3. state the operative consequence — reverse edges for those fields are derived,
   so they are read here and never authored as prose back-references.

`.claude/skills/` is gitignored and materialized by `aiwf init` / `aiwf update`,
so `internal/skills/embedded/aiwf-show/SKILL.md` is the only file to edit; the
working copy regenerates.

## Out of scope

The sibling discoverability question is a separate change and is not addressed
here: the root help banner describes `--relates-to` as "related entities
(decision)" and says nothing about the edge being creation-time-only (G-0168,
D-0048) or about its inverse being derived. That silence spans every relation
flag symmetrically — `--depends-on`, `--discovered-in`, `--relates-to`,
`--linked-adr`, plus the promote-time resolver flags — so annotating one flag
alone would be lopsided. It wants its own gap and its own placement decision.

## Related

- **G-0168** / **D-0048** — decision `relates_to` has no post-creation mutation
  verb. D-0048 settled the verb shape and deferred the code until real friction
  appeared; the downstream report in Provenance is that friction, on
  `relates_to` specifically.
- **G-0504** — a different axis, not a parent. That gap is about a *materialized
  copy* going stale against its embedded source, and verb skills are the one
  family it says `aiwf doctor` already byte-checks. Here the embedded source is
  itself wrong about the kernel, and a byte-perfect copy of a wrong source still
  reports healthy.

## How this class is caught

No chokepoint compares a shipped skill's prose against the kernel behavior it
describes. `internal/policies/skill_edit_structural_test_backstop.go` requires a
referencing structural test for every `SKILL.md` edit, but scopes itself to
`internal/skills/embedded-rituals/**`; the verb-skill tree is excluded by
construction. What catches this class today is
`internal/policies/verb_skill_factual_test.go` — per-fact pins, each added when a
drift is found by hand, section-scoped via `sectionUnder`. This gap's fix lands
one there. Whether the general chokepoint is worth building is a separate
question this gap does not settle.

## Provenance

Downstream consumer report, 2026-08-03. The false clause was confirmed by
reading `internal/tree/tree.go:346` and `internal/entity/refs.go:28` against the
skill text, and the render arm by `internal/cli/render/resolver.go:592`.
