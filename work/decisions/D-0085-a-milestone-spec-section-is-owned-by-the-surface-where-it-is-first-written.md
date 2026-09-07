---
id: D-0085
title: A milestone-spec section is owned by the surface where it is first written
status: proposed
---
## Question

Which surface owns a milestone-spec section's rule — when that section is
filled, and what it holds?

## Decision

Each section has exactly one owner: the surface where the section is **first**
written.

- `aiwfx-start-milestone` owns `## Closes`, `## Work log`, `## Decisions made
  during implementation` and `## Deferrals`.
- `aiwfx-wrap-milestone` owns `## Release note`, `## Validation` and
  `## Reviewer notes`.
- The milestone-spec template owns the sections an author fills while writing the
  spec, and for the rest carries the heading and names the owning ritual.

Every other surface names the sections it touches and states no rule about them.
An agent card lists what it fills or reads; a ritual that confirms a section it
does not own routes to the owner instead of restating.

"First" is load-bearing, because four of the seven straddle: the wrap ritual also
writes `## Closes` when its sweep finds an unlisted gap, `## Work log` when it
records the merge SHA, and `## Decisions made during implementation` and
`## Deferrals` when a review finding lands in one of them. Written-in-both is the
common case, not the exception, so an ownership rule that keyed on "where it is
written" would return two answers for the majority of sections. First-write
decides all seven.

`## Validation` resolves as pasted at wrap. Both rituals already drove that —
nothing in the implementation ritual asks for it, and the wrap fills it before
dispatching the review — but the contrary instruction was live rather than
merely stale: an implementing agent reads `builder.md`, which listed the section
as one it maintains in flight. That instruction is overruled here, not
reconciled.

A section added or retired moves its rule to the surface where it is first
written under the new arrangement.

## Reasoning

Ownership follows the reader. For every section written after the spec is
authored, the reader of its rule is an agent inside a ritual, and the template is
not in front of them: the spec author has the template open while scaffolding,
and the implementer does not re-open it mid-milestone. That is the whole argument
for the split, and it is a claim about who is reading, not about what survives on
disk.

Comment survival does not discriminate here and is not the reason. The scaffolded
comments for the sections the template *keeps* survive into finished specs no
better than the ones it sheds. A template comment reaching almost no finished
spec is a fact about all of them, so it cannot be what separates the sections the
template owns from the sections it does not.

The alternative of making the template own every section states each rule in one
file but puts it where the rule does not bind, and depends on an agent opening a
template mid-milestone to learn an entry's shape. Splitting instead by rule class
— the template owning what a section holds, a ritual owning when it is filled —
has a real claim: the reader can apply that axis unaided, where this one requires
knowing which ritual writes the section. It is rejected because a section's two
rules are read together and changed together, so splitting them guarantees two
visits for every edit, whereas first-write yields one owner that a reader
reaches by asking a question the ritual sequence already answers.

The cost is that ownership is now spread across three surfaces rather than
concentrated in one, so "where is this stated" has three possible answers. That
is accepted because the answer is derivable rather than remembered.

The pointer form is the one established for the entity templates, which replaced
a restated field vocabulary with the command that prints it. This case differs in
that no command derives the answer, so a pointer names a prose surface.

This rule is a mandate, so it needs a mechanical owner rather than a habit. The
assignment carries one. It is written three times — in the template and in each
ritual — so a reader inside any one of them is self-sufficient, which is the
whole argument above; three copies drift unless something compares them, and a
policy does. It reports a section below the template's map with no owner, one
the map claims that the template no longer carries, one claimed by both rituals,
an owner naming a ritual that ships no skill, a ritual copy that assigns a
section differently from the template's or omits it, a map line naming a ritual
without assigning anything, and a wrap-owned section named in the builder agent
card. Every side comes from the documents, so a rewording passes and a
disagreement reports — the relationship shape D-0070 leaves available over a
shipped surface.

Reading one copy would have been weaker than it looks. A single map is internally
consistent whatever it says, so moving a section between owners in it would pass
in silence; measured before the ritual copies were compared, exactly that went
unreported. What makes a reassignment visible is a second surface still saying
the old thing.

What no check reaches is whether a rule's *prose* is stated twice — a ritual
re-specifying a section it does not own reads as ordinary instruction, and
pinning it would pin a wording. That half stays a review obligation, and it is
the half where a second statement drifts, so it is worth naming rather than
treating the policy as complete coverage.

What would retire the mandate is making the map derivable from one
machine-readable source rather than written into prose. The kernel's
required-section table is not that source today: for a milestone it carries
`Goal` and `Acceptance criteria`, and none of the sections at issue here.
