---
id: ADR-0037
title: Retitle re-derives the slug only while it tracks the title
status: proposed
---
## Context

Two verbs write an entity's on-disk slug, and their contracts contradicted each
other.

`aiwf rename` sets the slug directly and leaves the frontmatter title alone. It
exists so an entity with a long descriptive title can live at a short readable
path. `aiwf retitle` sets the title and, per G-0108, re-derived the slug from
the new title so a title change could not leave a stale filename behind.

Re-deriving unconditionally makes `rename`'s effect last only until the next
retitle. The kernel's own planning tree shows this is not a hypothetical: of 900
entities, 44 carry a slug that differs from the one their title derives, and
roughly two thirds of those are deliberate short paths rather than width
artifacts left by an id migration.

The conflict surfaced while extending same-state NoOp convergence (M-0281,
ADR-0036). Adding a filename conjunct to retitle's convergence guard turned
`aiwf retitle <id> "<current title>"` — previously a refusal, then a NoOp — into
a command that silently rewrote a deliberately chosen slug. That made the
underlying contract conflict visible rather than creating it.

## Decision

Retitle re-derives the slug only while the slug still tracks the title. A slug
that does not track the title was chosen deliberately with `aiwf rename`, and
retitle preserves it — on a title change as well as on a same-title re-run.

"Tracks the title" is derived, never stored: the slug tracks the title when
rebuilding the path from the entity's current title reproduces the path on disk.
No frontmatter field records whether a slug was customized, and no history walk
is consulted.

The body H1 is unaffected by this decision. Retitle continues to rewrite a
canonical `# <id> — <title>` heading whenever it is out of sync, because no verb
lets an operator set that heading independently of the title. The distinction is
ownership: retitle owns the H1 outright, and owns the slug only as the default
it applies while nothing else has claimed it.

## Consequences

`rename` becomes durable. A slug it sets survives every subsequent retitle, so
choosing a short path for an unwieldy title is a decision that stays made.

Retitle's guarantee narrows honestly. It no longer promises that title and slug
never diverge; it promises they agree by default and diverge only where the
operator said so. The four surfaces that documented the old, unconditional
behavior state the new rule.

A filename that diverged for reasons other than `rename` — the narrow-id slugs
left by an id-width migration — is no longer repaired incidentally by a retitle.
`aiwf rename` and `aiwf rewidth` are the verbs for that, which keeps an explicit
act behind an explicit change.

The rule cannot distinguish a deliberate `rename` from an accidental divergence,
because both present identically on disk. It resolves that ambiguity toward
preserving what it finds, on the reasoning that silently discarding an
operator's explicit act is the worse error.
