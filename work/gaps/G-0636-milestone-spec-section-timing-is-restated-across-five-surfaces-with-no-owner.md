---
id: G-0636
title: Milestone-spec section timing is restated across five surfaces with no owner
status: open
---
## What's missing

When each milestone-spec section gets populated — continuously during
implementation, or once at wrap — is stated independently in five shipped
surfaces. None of them owns the fact, and two of them already disagree.

`agents/builder.md:16` lists `## Validation` among "the milestone spec's
in-flight sections", maintained during implementation, and `:42` repeats it.
`templates/milestone-spec.md:127` says of that same section: *"Pasted at
wrap."* Both materialize into consumer repos through `aiwf init` /
`aiwf update`. Nothing reconciles them, and no check notices.

Measured 2026-08-25, the surfaces carrying a timing claim about these sections
(paths relative to `internal/skills/embedded-rituals/plugins/aiwf-extensions/`):

- `agents/builder.md` — names three sections as in-flight, twice
- `agents/reviewer.md` — which sections a reviewer is given to read
- `skills/aiwfx-start-milestone/SKILL.md` — appends to `## Work log` and
  `## Deferrals` during implementation
- `skills/aiwfx-wrap-milestone/SKILL.md` — brings them up to date and finalizes
  the rest at wrap
- `templates/milestone-spec.md` — per-section comments stating when each is
  filled

The direction that would fix this is already established. `db843cead`
(*"point templates at the vocabulary instead of restating it"*) and `7ad6dffdc`
(*"tell a spec author which record holds the reasoning"*) both replace a
restated fact with a pointer to the surface that owns it. That sweep has not
reached section-population timing.

## Why it matters

The disagreement is not decorative: it changes what an agent does. A builder
reading `builder.md` maintains `## Validation` as it works; one reading the
template writes it once at wrap. Both are following shipped instruction, and
the milestone spec ends up filled differently depending on which surface the
agent happened to load.

The cost compounds for anyone editing this area. A change to the wrap ordering
has to locate and update every copy, and a copy missed reads as authoritative
because there is no marker saying it is derived. Three consecutive review
rounds on a single ritual patch found false timing claims, each one written by
an author who had checked *a* surface — just not the one that contradicted it.
With no owner, checking is not a defined operation.

## Resolution shape

**Name one owner and point the rest at it** — the same move `db843cead` made
for vocabulary, not a hand reconciliation of five copies.

`templates/milestone-spec.md` is the natural owner: it is the artefact whose
sections these are, it already carries a per-section comment for each, and a
spec author has it open while filling them. The rituals and agent cards would
then say which sections they touch, and stop saying when those sections are
otherwise filled.

Deciding that ownership settles the open contradiction as a side effect:
whichever way `## Validation` resolves, it resolves once. That decision is the
substance of this gap and is why it is not a mechanical edit — it is why the
G-0635 patch routed here rather than reconciling the copies in passing, having
already stated this fact wrongly and had it corrected at review.

Worth checking during the fix, not assumed now: whether a check could hold the
result. A rule that no surface outside the owner states population timing is
grep-shaped, but the phrasing varies ("in-flight", "at wrap", "continuously"),
so it may only be enforceable as a review habit. Judge that once the owner
exists.

## Where to fix

- `internal/skills/embedded-rituals/plugins/aiwf-extensions/templates/milestone-spec.md`
  — the proposed owner; per-section comments become the single statement.
- `internal/skills/embedded-rituals/plugins/aiwf-extensions/agents/builder.md`
  — carries the contradiction at `:16` and `:42`.
- `internal/skills/embedded-rituals/plugins/aiwf-extensions/agents/reviewer.md`
  and the two milestone rituals — restate timing that would become derived.

Every one of these ships to consumers, so each edit must ride a commit naming
this gap for `skill-edit-provenance-backstop`.
