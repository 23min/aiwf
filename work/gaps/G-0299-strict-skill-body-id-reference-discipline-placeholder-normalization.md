---
id: G-0299
title: Strict skill-body id-reference discipline + placeholder normalization
status: open
---
## Problem

Shipped skill bodies (`internal/skills/embedded/**` verb skills and
`internal/skills/embedded-rituals/**` rituals) reference **real aiwf entities** —
by id, by filesystem path, and with inline lifecycle-status assertions — and use
inconsistent / fabricated id placeholders. Two failure modes:

1. **Staleness.** A real reference rots when the entity changes status, is
   archived, or is rewidth'd. Observed: `aiwfx-whiteboard` tier examples cite
   gaps now `addressed`+archived; `aiwf-list` asserts `ADR-0004` is `proposed`
   when it is `accepted`; `aiwf-add` links a milestone via a brittle filesystem
   path (`work/epics/E-17/M-066-...md`) broken by archive + rewidth.
2. **Cross-tree leakage.** The skills ship to consumers whose entity trees
   differ. aiwf's own `G-0184` / `M-0066` / `ADR-0004` are meaningless in a
   consumer repo, so a real-id reference in a shipped skill is both stale-prone
   *and* contextually wrong.

Placeholders are also off-convention: narrow widths (`E-NN`, `M-NNN`, `D-NNN`,
`ADR-NNN`), idiosyncratic shapes (`G-XYZ`, `ADR-WXYZ`), fabricated ids
(`ADR-OPSPEC-01`), and pseudo-arithmetic (`C-NNN+1`).

## Decision (STRICT discipline + ADR doc-link carve-out)

Shipped skill bodies cite **no real entity ids, filesystem paths, or inline
lifecycle statuses.** Illustrative content uses canonical-shape placeholders
(`G-NNNN`) or shape-descriptions. Skills **may link to a design/ADR *doc*** for
"read more" (the carve-out) — they just may not cite an entity id/path/status
inline. Provenance ("why this feature exists") has durable homes already —
CLAUDE.md, the design docs, and commit trailers — and does not belong in the
consumer-facing behavioral skill body.

### Mechanical chokepoint (makes the rule real, not advisory)

A check over `internal/skills/**` that flags any **non-placeholder id-shape** in
a skill body — i.e. a digit-bearing `<prefix>-NNNN` token — while allowing
canonical placeholders (`-NNNN` literal) and id-shapes that appear inside a
`docs/.../ADR-*` markdown doc-link. Per the kernel principle "framework
correctness must not depend on LLM behavior," this is what guarantees the
discipline; the standing-rule prose is the convenient version.

### Placeholder normalization (precondition; folds in F14)

Normalize all placeholders to canonical `<prefix>-NNNN`; distinct placeholders
where two appear together (e.g. `aiwf-promote`'s supersede example); drop
pseudo-arithmetic. This must precede the check so it can allow placeholders while
flagging real ids.

## Scope

CLAUDE.md Skills policy (the standing rule); a new check in `internal/check` (or
`internal/policies`); a full sweep of `internal/skills/embedded/**` and
`internal/skills/embedded-rituals/**` removing real entity refs and normalizing
placeholders. **Land this gap first** — it rewrites every skill body for
id-hygiene; the other skill-touching gaps rebase onto it. Subsumes the
entity-reference portions of the whiteboard / aiwf-contract / aiwf-add / aiwf-list
findings.
