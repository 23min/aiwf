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
> the areas/surfaces that the milestone ill touch, but specifically broader than that,
> any doc in docs/ and any entity (ADR or decision and referenced gap(s) or closed epic(s)/milestone(s))
> that has prose or examples that can confuse this milestone, or that is in conflict with
> what it says. Ie, before we build anything I want to make sure that all resulting findings
> that related to this milestone somehow are not contradictory or ambiguous and that the
> milestone is thereby well specified. I am thinking that this is something I
> want to do in the future with all milesstones and patches, and that we perhaps need
> a skill for this. The feeling is that things have not gone well since the project has gone from basically greenfield to brownfield with a lot of history, and 
> recently with the latest epics where we basically had to recall, cancell,
> rewrite, a lot of specs, prose and code. So can you do this? Do you have a
> good approach for this? So in general, the "preflight" could be a *lot* more
> involved, ie, prose scan, including code comments and perhaps even a lab that
> measures and proves what is currently true (or false) and possibly even small experiments that can
> give direction, either complement a milestone or disqualify it, or validate
> it.

## Threads

The asks, and their promotion status as of this document's date. Numbers are
stable identifiers, not an ordering; a withdrawn thread keeps its number rather
than being renumbered, so references stay valid.

| # | Thread | Promoted to |
|---|---|---|
| T1 | An **independent reviewer** performs the pass, not the milestone's author | — |
| T2 | The pass reads the milestone **in the context of its epic** | — |
| T3 | The audit spans **`docs/`, every ADR and decision, and the entities related to the work** | E-0085, partially |
| T4 | **Code comments** are in scope | E-0085 |
| T5 | A **lab that measures and proves what is currently true — or false** | E-0085 |
| T6 | **Small experiments** that complement, validate, or **disqualify** a milestone | — |
| T7 | ~~Applied selectively, to difficult milestones~~ — **withdrawn**; it runs on every milestone and patch | n/a |
| T8 | Delivered as a **skill** | E-0085 |
| T9 | **Examples are audited, not only prose** | — |

One requirement does not come from the specification. It emerged from the trials
and governs the rest: **reading may never clear a question.** It is recorded
under *What the trials established*.

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

### T3 — the audit's scope

The specification names three tiers, and the selector differs by tier:

- **Every ADR and decision**, unrestricted. These are where commitments live, and
  a specification that contradicts one is contradicting something the project
  agreed to.
- **`docs/`**, which holds 144 markdown files, 53 of them normative (`docs/adr`
  and `docs/design`, about 100,000 words).
- **The entities related to the work** — gaps, epics and milestones, closed ones
  included. Those the work references are in scope by definition; the rest are
  reached by scanning titles.

No tier requires reading the corpus. Measured on this repo at the date above:

| index | entries | words |
|---|---|---|
| ADRs and decisions | 100 | 1,050 |
| every entity, archived included | 1,041 | 11,270 |
| active entities only | 231 | 2,620 |

Scanning every entity title in the project costs about 11,000 words, against
78,000 to read the commitments alone. So the entity tier needs no filter beyond
the scan itself — a reader takes the whole index, selects on subject, and reads
the selection. Archived entities stay in scope: four fifths of the corpus is
archived, and the specification names closed epics and milestones explicitly.

The failure mode to watch is selecting too *few*, not too many. Reading twenty
candidates drawn from a thousand titles is cheap; missing the one that
contradicts the spec is the whole cost of the pass.

A narrower selector — prose that names the code the work will touch — terminates
cheaply but reaches less. A replay of five recorded drift cases found it reaches
prose naming something the change touches, and misses two classes: prose that
drifted in an earlier change, which names symbols the current change does not
touch, and defects of shape rather than wording, which name no symbol at all.

### T5 — the lab

Prove what is currently true by running commands rather than by reasoning. It is
the best-evidenced part of the specification: in the motivating episode, measuring
the spec's claims caught every defect on its own.

### T6 — experiments that can disqualify

The most distinctive ask, and the one with no capture at all. Validating a
specification's claims and testing whether the milestone is worth building are
different activities. A pass that only checks claims will confirm a
well-specified bad idea; the highest-value outcome available here is a milestone
cancelled before implementation, and nothing built so far can produce it.

### T7 — withdrawn

The specification originally reserved the pass for difficult milestones. It now
asks for it on every milestone and patch, so there is no trigger to define and
nothing to build here.

The measured cost supports the revision: one full sweep is about six minutes and
122,000 tokens in a single subagent. Gating was worth designing while the sweep
meant reading a corpus; it is not worth designing to save an index scan.

### T9 — examples, not only prose

An example that no longer works misleads exactly as a wrong sentence does, and it
is the shape that failed in the motivating episode: what was stale there was not
a claim in a paragraph but a path inside an instruction.

Examples differ from prose in a way that makes them worth separating. **A prose
claim can usually only be doubted; an example can often be settled.** A command
either resolves against the command tree or it does not. A sample body either
carries the sections its kind requires or it does not. A flag either exists or it
does not. So a good part of this thread belongs to the lab rather than the sweep,
and returns true or false rather than a suspicion — the one place the audit gets
cheaper and more reliable at the same time.

In scope: fenced command blocks, sample documents and structures, illustrative
snippets and placeholder values, wherever they appear in the material the audit
already reads.

## What the trials established

### A specification holds three kinds of content, and each needs a different check

A specification is mostly proposal — intent, design, fit with what exists. Only a
slice of it is claims about the world as it currently stands. The three parts of
the pass are not three ways of checking one thing; they check different content
and return different verdicts.

| content | check | verdict |
|---|---|---|
| a claim about the present | the lab — run a command | true or false |
| fit with what is already committed | the sweep — read the commitments | consistent, or not |
| whether the proposal works | an experiment — build the smallest version | works, or does not |

Routing a question to the wrong one produces a confident wrong answer, which is
worse than no answer. Asking the sweep to settle a fact is the specific mistake
the blind trial below records.

### Reading raises a doubt; it never clears one

The sweep's output carries four states: **contradicted**, **ambiguous**,
**complicated**, and **unknown — needs measurement**. There is deliberately no
*verified* state. A claim nothing settled is a visible blank, not a pass.

This is structural rather than a matter of discipline. When a prose reader is
free to report that a claim looks sound, it will, and the reader downstream stops
looking — ending up more confident than before the review, and pointed at the
wrong thing. Removing the state removes the failure.

The four are distinct, and the distinction is what makes each actionable:

- **Contradicted** — the text says the opposite of what the spec says. One of
  them is wrong, and a command decides which.
- **Ambiguous** — the text addresses the point and can be read more than one way.
  Nothing is false; a builder reading it the other way builds the other thing.
  The fix is to disambiguate the text, not to measure anything.
- **Complicated** — an existing obligation applies to the work and the spec does
  not discharge it. Nothing false; something missing, which a builder discovers by
  colliding with the rule mid-implementation. To stay falsifiable a complication
  names the specific obligation, never a document that seems related.
- **Unknown** — nothing read settles the claim, and a command could. This is the
  state that replaces "looks sound".

### The pass produces a ledger, not a findings list

Every factual claim the specification makes, with the command that settled it or
a blank beside it. A findings list lets "the review was clean" stand in for "six
claims checked, two unchecked"; a ledger cannot express the first.

### The sweep runs before the lab

The sweep generates questions and the lab answers them, so the sweep goes first.

Ordering the parts by yield instead — cheapest and most productive first — answers
a different question: whether the sweep is worth paying for at all. That question
is settled by its measured cost rather than by its position, and at an index scan
plus a handful of documents the answer is yes on every pass. Once the sweep is
running regardless, the only ordering that matters is the one that puts questions
before the thing that answers them.

### Timing: preflight, with the commitments sweep available earlier

Measurement runs just in time. An epic planned and then deferred leaves any
measured fact stale by the time work starts, which rules out settling facts at
planning time.

Prose ages differently from code. A contradiction between a spec and a standing
decision is still a contradiction weeks later, while call sites and counts move.
So the commitments sweep can usefully run at epic drafting — asking *what has
already been decided in this area* — while the lab stays at milestone preflight.

### Three seams

- **Epic drafting** — sweep the commitments. Theory building: what is already settled here.
- **Milestone preflight** — sweep, then lab, then experiments where a doubt survives. Runs in subagents.
- **Patch start** — the same, scaled to the change. A patch closing a tracked item has inherited premises by construction.

### Corpus and cost

The commitments corpus is the accepted ADRs and decisions. Reading it whole does
not scale; reading its **titles** does. Measured on this repo:

| | entries | words |
|---|---|---|
| title index | 100 | 1,050 |
| the same documents in full | 100 | 77,879 |

Scan every title, select on subject, read only the selected. What grows with the
corpus is the index, at roughly ten words an entry, so a project with ten times
the decisions still has a scannable index. The method depends on titles being
informative; a project whose decisions are titled "Decision 12" gets nothing from
the index and must read everything.

One full sweep, measured: 99 titles scanned, 19 selected, 16 read, 11 findings —
122,000 tokens and about six minutes in one subagent.

### The blind trial

The commitments sweep was run against M-0307, a milestone cancelled at preflight,
by a reviewer given the spec and the title index with the outcome withheld and
every read pinned to the ref preceding cancellation. It was allowed to read and
forbidden to measure.

Against the three defects that cancelled M-0307:

| defect | result |
|---|---|
| the premise was wrong — the path resolved for no kind, not two | **reported sound** |
| three shipped surfaces carried the citation, not one | correctly raised as needing measurement |
| one criterion could not be written non-vacuously | **found**, from the decision that bans phrase pins |

It also returned eight findings the original preflight did not record, the
sharpest being that the proposed fix routes every kind through a scaffold a
sibling milestone had already established the born-complete kinds cannot use —
the same defect with its polarity reversed.

The first row is the load-bearing result. The decision naming the templates says
four exist and names them, which agrees with "two of six" unless you try it. What
breaks it — that the two closest-named files open with their own frontmatter, so
the verb refuses them — is written nowhere. No reader reaches it, and the reviewer
did not merely miss it but marked the claim sound. That is the evidence for the
no-clearance rule above, and for the routing table: a fact was put to the
fit-checker.

### The premise, measured

Across four months: 110 acceptance criteria reworded or abandoned after being
written, and 36 milestones cancelled out of 261 created. The churn is chronic
rather than a recent regression, though the most recent month runs above the
others.

Where the recorded failures came from is consistent across the three that are
documented: each took a premise from another entity rather than from looking.
M-0307's came from a gap whose own title was wrong; E-0085 was planned from a
partial capture of this specification; M-0306 followed from a sibling milestone
rather than from the surfaces themselves. That is not a trigger — the pass runs
everywhere — but it says which claims in a specification deserve the first
command.

These count churn, not wrongness — a cancelled criterion may have been wrong, or
the scope may have changed. The numbers are consistent with the specification's
premise and are not proof of it.

## What ships

The threads above say what is wanted and the trials say how it behaves. This
section says what an implementer builds, because a method with no named artifacts
is one where every decision falls to whoever writes it that day.

### The artifacts

- **A reviewer brief.** The instructions a dispatched reviewer receives. This is
  the load-bearing artifact, not a wrapper around one: independence is what makes
  the audit's breadth affordable, and the brief is where independence is
  established or lost. Drafted below.
- **A ritual the reviewer follows**, holding the method: what to sweep, how to
  select from a title index, when to measure, what may be concluded.
- **Seam instructions** at the three entry points, each naming the pass rather
  than restating it.

Whether the reviewer performs the lab directly or invokes a separate ritual for
it is open, and is the one structural question left in the artifact set.

### The report

A ledger, one row per factual claim the specification makes:

| claim | state | evidence |
|---|---|---|
| what the spec asserts | contradicted / ambiguous / complicated / unknown | the command that settled it, or blank |

There is no *verified* state. A blank in the evidence column is the point of the
format: it is the only way an unchecked claim stays visible.

Findings that are not claims — a commitment the spec fails to discharge, an
example that no longer works — carry the same four states, and a *complicated*
row names the specific obligation.

### The prohibitions

- **The pass may not clear a question.** Reading raises doubts; only a command
  settles one. No output may assert that a claim is sound.
- **The sweep may not measure while it reads.** A reader that starts running
  commands mid-audit conflates the two verdicts and produces a confident wrong
  answer. It records what it would run instead.
- **The pass may not edit code.** It changes the specification, or it returns a
  disqualifying result.
- **It terminates.** The sweep searches names rather than concepts and follows a
  borrowed claim one hop. Both bounds exist so a pass finishes.

### The reviewer brief

The blind trial's brief produced eleven findings and pushed every measurable
question to an open-questions list rather than guessing at it. Its load-bearing
parts, which any implementation keeps:

- **Pin the corpus.** Every document is read at a stated ref, and the working
  tree is off limits. Without this the reviewer reads material that postdates the
  work and calls it context.
- **Select from titles before reading, and record the selection with its reason.**
  Making the choice auditable is what separates a sweep from a search for
  confirmation.
- **Forbid measurement explicitly**, and require that anything the reviewer wants
  to measure is recorded as a question with the command it would run.
- **Ask for contradiction, complication, and silent assumption** — three distinct
  questions, not one request for "problems".
- **Withhold the outcome.** A reviewer told what went wrong finds what it was
  told. This is what the blind trial establishes and what a sighted run cannot.
- **Instruct it to say so plainly when it finds nothing**, rather than
  manufacturing findings — and, per the prohibitions, without converting that
  into a clearance.

### Promotion status

T4, T5 and T8 are partially delivered by
[E-0085](../../work/epics/E-0085-measure-the-spec-before-the-code-record-what-the-measurement-changed/epic.md),
which was planned from
[G-0583](../../work/gaps/G-0583-the-milestone-preflight-asks-for-judgment-with-no-method.md)
— a partial capture of this specification, filed before it was recorded. T1, T2,
T6 and T7 are unpromoted.

## Open questions

| Question | Blocking a promotion? | Resolution path |
|---|---|---|
| What shape does an experiment take, what triggers one, and what does a disqualifying result do to the milestone? | T6 | **Unanswered, and it needs the author of this specification, not a measurement.** The thread with the most value and the least precedent. The disqualifying outcome implies cancelling a milestone at preflight, a path the kernel already supports. Everything else in *What ships* can be built without it; this cannot be inferred from the trials. |
| Does the reviewer perform the lab directly, or invoke a separate ritual for it? | T1 | The one structural question left in the artifact set. Bears on whether the seams name one artifact or two. |
| Does the entity half of T3 survive at all? | T3 | The commitments corpus — ADRs and decisions — is settled and scannable. Whether gaps and closed milestones also need reading is unanswered: roughly 1,100 entities have no filter that reaches the ones that matter, and reverse references walk entities only, not docs or templates. The honest fallback is to state the reduction rather than carry an ambition with no mechanism. |
| Do the title index and the no-clearance rule hold in a project that is not this one? | all | Both were measured here only. The index depends on titles being informative, which is a property of this repo's conventions rather than of the method. |

## Provenance

Stated in conversation before [G-0583](../../work/gaps/G-0583-the-milestone-preflight-asks-for-judgment-with-no-method.md)
was filed; recorded here on this document's date, after a comparison against the
shipped ritual found the gap had carried across a fragment. Superseding nothing —
this is the first record of the specification itself.

*What the trials established* was added the following day, after the commitments
sweep was run blind against a cancelled milestone and the numbers in it were
measured against this repo. Everything in that section is an observation from
those runs rather than a position argued in advance; where a trial contradicted
an earlier position, the trial's result is what stands here.
