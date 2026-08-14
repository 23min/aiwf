---
title: Milestone preflight as an independent review — prose audit, lab, and experiments that can disqualify
status: captured
date: 2026-08-14
---

# Milestone preflight as an independent review — prose audit, lab, and experiments that can disqualify

## Classifier note

This is an initiative document. `initiative` is not yet an official aiwf entity
kind ([G-0311](../../work/gaps/G-0311-no-cross-cutting-initiative-tier-above-epic-for-multi-component-features.md)),
so it lives under `docs/initiatives/` in the forward-looking tier: a captured
idea awaiting promotion to tracked entities. It moves to
`docs/initiatives/archive/` with `status: realized` once every thread below has
been promoted — promotion, not completion.

## The specification

Quoted verbatim from the originating conversation. It is reproduced without
correction because a paraphrase is what this document exists to replace.

> I want a reviewer to review the next miletone, also in context of the epic,
> and I want it to do an audit, or a review, or whatever you may call it, of all
> the areas that the milestone ill touch, but specifically broader than that,
> any doc in docs/ and any entity (ADR, gap, decision, closed epic/milestone)
> that has prose that can confuse this milestone, or that is in conflict with
> what it says. Ie, before we build anything I want to make sure that all prose
> that is related to this milestone somehow is not contradictory and that the
> milestone is thereby well specified. I am thinking that this is something I
> want to do in the future with difficult milesstones, and that we perhaps need
> a skill for this. That's a separate discussion. But things have not gone well
> recently with the latest epics where we basically had to recall, cancell,
> rewrite, a lot of specs, prose and code. So can you do this? Do you have a
> good approach for this? So in general, the "preflight" could be a *lot* more
> involved, ie, prose scan, including code comments and perhaps even a lab that
> proves what is currently true and possibly even small experiments that can
> give direction, either complement a milestone or disqualify it, or validate
> it.

## Threads

Eight distinct asks, and their capture status as of this document's date.

| # | Thread | Captured by |
|---|---|---|
| T1 | An **independent reviewer** performs the pass, not the milestone's author | nothing |
| T2 | The pass reads the milestone **in the context of its epic** | nothing |
| T3 | The prose audit spans **all of `docs/` and any entity** — ADR, gap, decision, closed epic or milestone — that could confuse or conflict | partially, narrowly |
| T4 | **Code comments** are in scope | `wf-measure-spec` |
| T5 | A **lab that proves what is currently true** | `wf-measure-spec` step 1 |
| T6 | **Small experiments** that complement, validate, or **disqualify** a milestone | nothing |
| T7 | Applied selectively, to **difficult milestones** | nothing |
| T8 | Delivered as a **skill** | `wf-measure-spec` |

### T1 — an independent reviewer

The pass is a review, performed by someone other than the milestone's author.
This is the same independence property `wf-review-code` rests on: an author
re-reading their own specification is the failure mode the pass exists to close,
and cannot be substituted for it.

It is also what makes T3 affordable. A breadth that is prohibitive for an author
mid-implementation is one pass for a dispatched reviewer, and the cost objection
that narrowed T3 was reasoning about the wrong reader.

### T2 — epic context

A milestone can be internally consistent and still contradict its epic's scope,
constraints, or a sibling milestone. Nothing currently reads the epic during
preflight.

### T3 — the prose audit's real scope

The specification names the scope directly: any doc in `docs/`, and any entity
that has prose which could confuse or conflict. The selector is *relatedness to
this milestone*, and the test is *could this mislead the builder*.

`wf-measure-spec` implements a narrower selector — prose that names the code the
work will touch — chosen because a name search terminates and a concept search
returns a large fraction of the corpus. Measured on this repo at the date above:
`docs/` holds 144 markdown files, 53 of them normative (`docs/adr` and
`docs/design`, about 100,000 words), against roughly 1,100 entity files. The
normative core is a readable corpus for one reviewer pass; the entity set is not,
and needs a relatedness filter that does not yet exist.

A replay of five recorded drift cases found the narrow selector reaches prose
that names something the change touches and misses two classes: prose that
drifted in an earlier change, which names symbols the current change does not
touch, and defects of shape rather than wording, which name no symbol at all.

### T5 — the lab

Prove what is currently true by running commands rather than by reasoning. This
is the one thread that shipped substantially intact, and it is the best-evidenced
part: in the motivating episode it caught every defect on its own.

### T6 — experiments that can disqualify

The most distinctive ask, and the one with no capture at all. Validating a
specification's claims and testing whether the milestone is worth building are
different activities. A pass that only checks claims will confirm a
well-specified bad idea; the highest-value outcome available here is a milestone
cancelled before implementation, and nothing built so far can produce it.

### T7 — reserved for difficult milestones

The pass is expensive by design and is not meant to run on everything. What makes
a milestone "difficult" is undefined and worth defining, since it is the trigger.

## What exists today, and how it falls short

[G-0583](../../work/gaps/G-0583-the-milestone-preflight-asks-for-judgment-with-no-method.md)
captured this specification as "the preflight asks for judgment with no method"
and recast it as three parts — measure, challenge, sweep. T1, T2, T6 and T7 were
not carried across. [E-0085](../../work/epics/E-0085-measure-the-spec-before-the-code-record-what-the-measurement-changed/epic.md)
and its first milestone were planned from that gap rather than from this text,
which was not recorded anywhere until this document.

The result is sound within its scope: a ritual that measures claims, challenges
criteria, and sweeps prose naming the code, terminating in a recorded outcome.
It is a fragment of what was asked for, and the missing threads include the two
that carry the most weight — the reviewer, and the experiments.

## Open questions

| Question | Blocking a promotion? | Resolution path |
|---|---|---|
| What filter selects the entities related to a milestone, given roughly 1,100 of them? | T3 | Candidates that do not depend on prose search: reverse references, directory adjacency, the epic's own reference list. Reverse references cover entities only, not docs or templates. |
| What makes a milestone "difficult" enough to trigger the pass? | T7 | Likely operator judgment at first; a mechanical trigger needs evidence from runs. |
| What shape does an experiment take, and what does a disqualifying result do to the milestone? | T6 | Unexplored. The disqualifying outcome implies a verb — a milestone cancelled at preflight — and that path already exists. |
| Does the reviewer pass subsume `wf-measure-spec`, or call it? | T1 | The lab is a step the reviewer performs; the shipped ritual may become its first section rather than a peer. |

## Provenance

Stated in conversation before [G-0583](../../work/gaps/G-0583-the-milestone-preflight-asks-for-judgment-with-no-method.md)
was filed; recorded here on this document's date, after a comparison against the
shipped ritual found the gap had carried across a fragment. Superseding nothing —
this is the first record of the specification itself.
