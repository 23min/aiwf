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

The same surfaces also restate what a section *holds*. The `## Work log` entry
shape is stated three times and differently: `aiwfx-start-milestone:130` as a
heading plus `<one-line outcome> · commit <SHA> · tests <N/M>`;
`aiwfx-wrap-milestone` step 4 as "one entry per AC with the final outcome and
commit SHA"; and `templates/milestone-spec.md:112-122`, which adds an
"[o]ptional prose paragraph for non-obvious context: what changed, file:line
references, why a detour was needed".

None of the three says where anything *else* goes, and that optional paragraph
is an open invitation, so the Work log becomes the default sink for whatever an
agent judges worth saying. Reported from a consumer milestone (7 ACs,
commit-per-AC): the section reached 3,111 words against a stated shape of about
100, was 62% of the finished spec, restated content already carried by module
docstrings and contract bodies, and grew monotonically across the session
because each entry was written against the previous one rather than against the
stated shape.

The template also contradicts itself on the one routing rule that does exist.
`:119-122` has phase transitions "visible here too (red/green/refactor/done)"
and simultaneously forbids duplicating the timeline; `aiwfx-start-milestone:130`
forbids it flatly. A routing rule stated twice has already drifted the way the
timing claim did.

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

The content half takes the same owner. The template's `## Work log` comment
becomes the entry line plus the destinations no other section holds — design
reasoning to the module's own docstring, a seam's measurements to the contract
body, the phase timeline to `aiwf history`, an unpinnable property to a gap or a
recorded decision — and then points at the sections below for the rest rather
than copying their rules. Enumerating every destination would restate what
`## Deferrals`, `## Decisions made during implementation` and `## Reviewer
notes` already state a few lines down, which is this gap's own defect committed
inside its fix; naming only the homeless facts is what keeps the rule shorter
than the comment it replaces.

Routing rather than a length cap: an author will not delete a paragraph they
judge to matter without somewhere to put it, so naming the destination is what
makes the cut safe, and a ban costs once where a per-entry budget costs forever.
A ceiling is the wrong instrument for the reason the line-budget instance above
supplies — one that can be raised is raised. The checkable arm is narrow:
phase-transition words in a Work log entry contradict the rule and are
grep-shaped.

Two destinations had to be settled against surfaces that already own them rather
than invented. A property that could not be pinned, or was judged not worth
pinning, goes to a gap or a recorded decision per `wf-codebase-health` D5 and
`aiwfx-wrap-milestone` step 2 — not to `## Reviewer notes`. A rejected
alternative splits on whether it earned a decision entity: `## Decisions made
during implementation` carries it with its id, `## Reviewer notes` keeps the
ones that did not.

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
