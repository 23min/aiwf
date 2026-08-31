---
id: G-0636
title: Milestone-spec section rules are restated across five surfaces with no owner
status: open
---
## What's missing

When each milestone-spec section gets populated — continuously during
implementation, or once at wrap — is stated independently in five shipped
surfaces. None of them owns the fact, and two of them already disagree.

`agents/builder.md:16` lists `## Validation` among "the milestone spec's
in-flight sections", maintained during implementation, and `:42` repeats it.
`templates/milestone-spec.md`'s own comment on that section says *"Pasted at
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

The same surfaces also restate what a section *holds*. The `## Work log` entry
shape is stated three times and differently: `aiwfx-start-milestone:130` as a
heading plus `<one-line outcome> · commit <SHA> · tests <N/M>`;
`aiwfx-wrap-milestone` step 4 as "one entry per AC with the final outcome and
commit SHA"; and `templates/milestone-spec.md`'s `## Work log` comment, which
states the same first line and then routes everything else away — design
reasoning to the code, anything else to its own section.

That last one is the settled case. The two surfaces above it still state the
shape independently, so a change to it has three places to reach. Reported from
a consumer milestone (7 ACs,
commit-per-AC): the section reached 3,111 words against a stated shape of about
100, was 62% of the finished spec, restated content already carried by module
docstrings and contract bodies, and grew monotonically across the session
because each entry was written against the previous one rather than against the
stated shape.

The template used to contradict itself on the one routing rule that does exist,
carrying the phase timeline as something visible in the Work log while also
forbidding its duplication. That is a state a reader can still encounter in a
spec scaffolded before the correction: the template now routes the timeline to
`aiwf history` alone, and `aiwfx-start-milestone` forbids duplicating it flatly.
A routing rule stated twice had already drifted the way the timing claim did,
which is the pattern this gap is about rather than an outstanding instance.

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

The ratchet behind the Work log's growth is not specific to one agent or one
milestone: this repo's always-on guidance fragment carries a line budget whose
constant has only ever been raised, each time against the ceiling the previous
raise set. Two unrelated surfaces, one mechanism.

## Resolution shape

**Name one owner and point the rest at it** — the same move `db843cead` made
for vocabulary, not a hand reconciliation of five copies.

Which surface should own a given rule is decided by where that rule binds, and
the template owns less than it appears to. Its per-section comments are consumed
when a spec is scaffolded and do not survive into the file: measured 2026-08-30,
1 of 325 milestone specs still carries the `## Work log` comment, and recent
specs carry none at all. So the template can own a rule read while a spec is
being authored, but not one that binds during implementation — that rule has to
live in the ritual the implementing agent is following. The rituals and agent
cards would then say which sections they touch, and stop restating rules whose
owner is elsewhere.

Deciding that ownership settles the open contradiction as a side effect:
whichever way `## Validation` resolves, it resolves once. That decision is the
substance of this gap and is why it is not a mechanical edit — it is why the
G-0635 patch routed here rather than reconciling the copies in passing, having
already stated this fact wrongly and had it corrected at review.

The content half is settled for `## Work log` and open for the rest. That
section's purpose is now stated in the template and in `aiwfx-start-milestone`:
the index from each AC to the commit that implemented it, which is the one fact
no other record holds. The two invitations that grew it — the template's optional
prose paragraph and the ritual's "audit trail of mid-flight context" — are gone.
What remains open here is every other section's content rule, and whether the
section survives at all: retiring it is tracked in G-0530, and depends on
`aiwf history` being able to see an implementation commit.

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
