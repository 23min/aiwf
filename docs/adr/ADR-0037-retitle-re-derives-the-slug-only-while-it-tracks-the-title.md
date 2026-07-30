---
id: ADR-0037
title: Retitle re-derives the slug only while it tracks the title
status: accepted
---
## Context

Two verbs write an entity's on-disk slug, and their contracts contradicted each
other.

`aiwf rename` sets the slug directly and leaves the frontmatter title alone. It
exists so an entity with a long descriptive title can live at a short readable
path. `aiwf retitle` sets the title and, per G-0108, re-derived the slug from
the new title so a title change could not leave a stale filename behind.

Re-deriving unconditionally makes `rename`'s effect last only until the next
retitle. The kernel's own planning tree shows this is not a hypothetical: 44 entities
carry a slug that differs from the one their title derives. Widening the narrow
ids embedded in those slugs reconciles 17 of them, so 27 are deliberate short
paths rather than artifacts of an id migration.

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
canonical `# <id> — <title>` heading whenever it is out of sync. The distinction
is ownership, and it turns on the canonical form rather than the heading: an
operator who wants a heading of their own writes a non-canonical one, which
retitle leaves untouched by design. Writing a *canonical* heading that disagrees
with the title claims nothing — it states the entity's own title, so retitle
correcting it is the verb doing its job. The slug has no such escape hatch, and
`aiwf rename` is the verb for claiming one, which is why the slug is preserved
and the canonical H1 is not.

## Consequences

`rename` becomes durable. A slug it sets survives a retitle for as long as it
differs from the one the title derives, so choosing a short path for an unwieldy
title is a decision that stays made. Rename an entity to precisely the slug its
title derives and it is tracking again by definition, which is also how an
operator opts back in.

Retitle's guarantee narrows honestly. It no longer promises that title and slug
never diverge; it promises they agree by default and diverge only where the
operator said so. The four surfaces that documented the old, unconditional
behavior state the new rule.

A filename that diverged for reasons other than `rename` — the narrow-id slugs
left by an id-width migration — is no longer repaired incidentally by a retitle.
`aiwf rename` is the verb for that, which keeps an explicit act behind an
explicit change. `aiwf rewidth` does not reach these: it canonicalizes the
entity's own leading id in the filename and carries the remainder through
unchanged, so a narrow id embedded in the slug text survives it.

The rule cannot distinguish a deliberate `rename` from an accidental divergence,
because both present identically on disk. It resolves that ambiguity toward
preserving what it finds, on the reasoning that silently discarding an
operator's explicit act is the worse error.
