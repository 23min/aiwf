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
| T1 | An **independent reviewer** performs the pass, not the milestone's author | dropped — E-0086, D-0069 |
| T2 | The pass reads the milestone **in the context of its epic** | dropped — E-0086, D-0069 |
| T3 | The audit spans **`docs/`, every ADR and decision, and the entities related to the work** | dropped — E-0086, D-0069 |
| T4 | **Code comments** are in scope | never promoted — no attempt carried a code tier |
| T5 | A **lab that measures and proves what is currently true — or false** | promoted, reduced — the lab *rule* only; E-0086 |
| T6 | **Small experiments** that complement, validate, or **disqualify** a milestone | dropped — E-0086, D-0069 |
| T7 | ~~Applied selectively, to difficult milestones~~ — **revised**; it runs by default, escapable by a stated reason at a gate | dropped — E-0086, D-0069 |
| T8 | Delivered as a **skill** | dropped — E-0086, D-0069 |
| T9 | **Examples are audited, not only prose** | dropped — E-0086, D-0069 |

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

- **Every ADR and decision**, unrestricted — and *unrestricted* is load-bearing.
  The accepted ones are the project's commitments, and contradicting one
  contradicts something the project agreed to. The rest are not commitments and
  are in scope for a different reason: a rejected record is an agreement *not*
  to do something, so a specification proposing it is re-opening a settled
  question; a superseded one says what changed; a proposed one is a live
  collision. Measured at `4f8995df0`, the non-accepted records are 16 of 111,
  and their titles are 126 words of the index's 927 — a corpus a
  commitment-eligibility filter would remove for 126 words of scanning.
  (Per-file `status:` and `title:` from the first frontmatter block, over
  `docs/adr/`, `work/decisions/` and both archives.)
- **The normative documents**, not `docs/` flat. A project that tiers its
  documentation by authority has already written the selector: a contradiction
  only means something if the document carrying it is current truth. An
  exploratory synthesis disagreeing with a specification is not a finding, and an
  archival snapshot disagreeing is the snapshot doing its job — sweeping either
  produces findings at a rate set by document genre rather than by specification
  defects. A project with no such tiering has to establish one before this tier
  is usable, which is a real precondition rather than a detail.
- **The entities related to the work** — gaps, epics and milestones, closed ones
  included. Those the work references are in scope by definition; the rest are
  reached by scanning titles. **The index carries each entity's status**, because
  status is to an entity what the authority tier is to a document: it says what
  kind of thing the reader is holding.

  This matters most where it is least obvious. A cancelled entity's body is the
  proposal frozen at the moment it died — still reading as a live plan, with one
  word of frontmatter the only sign otherwise — so it is never citable as a
  commitment. Its cancellation *reason*, which lives in the verb's commit rather
  than the body, is a finding about reality attributed to a named human, and is
  often the better material of the two. Both are in scope; they are not the same
  kind of claim. A terminal status is not a reliability verdict either: a
  milestone can be `done` with criteria asserting properties of a file that no
  longer exists.

  The rule generalises past cancellation, and it is the existing archive
  commitment rather than a new one. Nothing non-current — a superseded decision,
  a rejected ADR, an archived epic — establishes a current obligation by being
  useful, because reversal is absent: re-adoption means a new entity referencing
  the archived one, per
  [ADR-0004](../adr/ADR-0004-uniform-archive-convention-for-terminal-status-entities.md).
  Such a record can raise a finding or answer a question. It cannot bind the
  work.

No tier requires reading the corpus. The document and commitment tiers are
measured under *Corpus and cost*; the entity tier is measured here, at the
snapshot named there:

| index | entries | words | built by |
|---|---|---|---|
| every entity, archived included, carrying status | 1,092 | 13,985 | `aiwf list --archived` |
| the same population, titles only | 1,092 | 10,709 | `title:` from the first frontmatter block of every entity file |
| active entities only, carrying status | 266 | 3,507 | `aiwf list --kind <k>` over the six kinds |

The status-carrying form is the one this tier requires and the one *Corpus and
cost* prices; a title-only grep is 23% cheaper and cannot tell a reader what
kind of document they are holding. Restrict the grep to the first frontmatter
block or it counts the illustrative frontmatter inside a fenced example as an
entity.

So the entity tier needs no filter beyond the scan itself — a reader takes the
whole index, selects on subject, and reads the selection. Archived entities stay
in scope: three quarters of the corpus is archived, and the specification names
closed epics and milestones explicitly.

One run supports this rather than an argument. Sweeping a draft epic, a reviewer
selected five entities from the title scan alone, beyond those the draft
referenced; four carried findings that changed the draft, and the sharpest was
archived — it recorded a prior agent making the identical mistake the draft was
making. That is the answer to "no filter reaches the ones that matter": the scan
is the filter, as it is for the commitments.

The limit is worth stating with the result. It is one observation, on a subject
unusually dense in entities because its subject was the tooling itself. A
milestone with a narrower subject should expect a lower yield from this tier, and
that would not falsify the method.

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

The most distinctive ask. Validating a specification's claims and testing whether
the milestone is worth building are different activities. A pass that only checks
claims will confirm a well-specified bad idea; the highest-value outcome
available here is a milestone cancelled before implementation, and nothing built
so far can produce it.

An experiment is not a fourth mechanism. It is the third settler for a row type
the ledger already has. A **feasibility claim** — this approach works, this is
simpler, this is cheaper — is a claim the lab cannot reach, because its subject
does not exist yet. The routing table's third row is exactly that case, and the
experiment is what fills its measured column.

- **Trigger — unresolved.** The intended test was a *load-bearing feasibility
  claim*: the specification asserts the approach works, and the milestone is not
  worth building if that is false. Applied to this project's four live epic
  specifications it decided one and could not decide three, because the second
  half is a counterfactual about value that no specification contains a sentence
  stating. It is the estimate this document refuses elsewhere, wearing a content
  test's clothes. No replacement is proposed here. Until one exists an experiment
  is run when a human asks for one, and the trigger stays an open question rather
  than a rule nobody can apply.
- **Shape — the smallest thing that would show the approach works, discarded
  afterwards.** It is built in a disposable tree per the prohibitions, never in
  the tree under review, and it is not a seed for the implementation. What
  survives is the ledger row: what was built, what it did, the command that
  produced it, and the **kill criterion** stated before building — without which
  the probe cannot fail, and a probe that cannot fail settles nothing.
- **Outcome — works, does not, or inconclusive.** *Does not* is a recommendation
  to cancel the milestone or re-specify it. The pass never acts on it. Cancelling
  is a mutation behind a human gate, and *inconclusive* is a real third answer —
  a probe that failed to settle the question is not evidence the approach works.

### T7 — a stated escape, not a judgment

The specification originally reserved the pass for difficult milestones, then
asked for it on every milestone and patch. Neither is right, and the measurements
say why.

The argument for running it everywhere was that gating costs more than it saves —
*"not worth designing to save an index scan"*. That priced the wrong artifact.
The 122,000-token observation swept the active commitments tier alone, whose
index at that ref was about 1,600 tokens; the index this specification now
requires is roughly eighteen times larger. And the index is not the cost: the
pass that follows it is, at 122,000 tokens against a subject whose median is
around 2,200.

Against that, the base rate. Of 37 cancelled milestones, 36 never reached
`in_progress` and **27 were cancelled in a same-day batch confined to a single
parent epic** — six such batches, plus two more spanning two epics each. These
are epic-level re-plans, not per-milestone specification defects. One epic in
the project's life had implemented code withdrawn. The dominant rework is the
gap-and-fix loop, which already closes at a median of one day.

> Deduplicate `git log --all --grep='aiwf-verb: cancel'` by its `aiwf-entity:`
> trailer to 37 milestones, group by commit date, and resolve each milestone to
> its parent epic directory. Grouping by epic alone, ignoring date, gives 30 —
> a different claim, and not the one this paragraph makes.

So the pass runs by default and is **escapable by a stated reason at a gate** —
the form `wf-patch` already uses for its own review step: *"no independent
review: &lt;reason&gt;"*, stated where a human can veto it rather than skipped
silently. Two escapes are mechanical rather than judgments, and both are cases
this specification currently has no answer for: **a subject with no
specification** (an untracked patch, roughly one in five here), and **a change
with no claim about the present** to put to the lab.

This is not the original T7. That asked the operator to judge a milestone
difficult in advance, which is the estimate the specification elsewhere refuses.
This asks only whether the pass has a subject and something to check.

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

### A specification holds four kinds of content, and each needs a different check

A specification is mostly proposal — intent, design, fit with what exists. Only a
slice of it is claims about the world as it currently stands, and one slice is
not about the world at all but about whether the specification's own criteria can
be satisfied honestly. The parts of the pass are not several ways of checking one
thing; they check different content and return different verdicts.

| content | check | verdict |
|---|---|---|
| a claim about the present | the lab — run a command | true, false, or unable to measure safely |
| fit with what is already committed | the sweep — read the commitments | a finding, or nothing found |
| whether a criterion can be met vacuously | the challenge — read the criterion against itself | a finding, or nothing found |
| whether the proposal works | an experiment — build the smallest version | works, does not, or inconclusive |

Routing a question to the wrong one produces a confident wrong answer, which is
worse than no answer. Asking the sweep to settle a fact is the specific mistake
the blind trial below records.

Only the lab and the experiment have a positive arm, and each earns it by running
something. Neither reading check does: their verdict is a finding or the absence
of one, never a pass. That asymmetry is the subject of the next section, and it
is what keeps *nothing found* from being read as *consistent*.

**The challenge is the cheapest of the four and the last to be written down.** It
takes each acceptance criterion and asks two questions of it: what
consumer-visible failure does this prevent, and could a builder satisfy its
letter while leaving a consumer no better off? It needs no corpus, no command,
and no index — only the specification itself — so it costs a fraction of the
sweep and can run wherever the criteria are. It has recorded yield: it is what
found the one M-0307 criterion that could not be written without asserting that
the guidance says what the test says it says. It is `wf-vacuity`'s move applied
to a specification rather than a test suite, and it borrows that skill's probe
vocabulary rather than minting a second one.

### Reading raises a doubt; it never clears one

The sweep's output carries four states: **contradicted**, **ambiguous**,
**complicated**, and **unknown — needs measurement**. There is deliberately no
*verified* state. A claim nothing settled is a visible blank, not a pass.

This is structural rather than a matter of discipline. When a prose reader is
free to report that a claim looks sound, it will, and the reader downstream stops
looking — ending up more confident than before the review, and pointed at the
wrong thing. Removing the state removes the failure.

The rule constrains the sweep's output, not what reading can establish. Reading
settles plenty: that a decision record imposes an obligation, that two documents
disagree, that a sentence carries a second reading. Each of those lands as a
finding — *complicated*, *contradicted*, *ambiguous* — and findings are the
sweep's whole product. What reading may not produce is the *absence* of a
finding turned into a verdict on the claim. So there is no reclassification
escape either: a reviewer cannot decide a question is normative rather than
factual and clear it on that basis. That reclassification is precisely what the
blind trial recorded, and the four states leave nowhere for it to land.

A claim a command settles true is a real outcome and it does have a home, but not
here. It is written by the lab, into a column of its own, and never by the reader
— see *The report*.

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

**Settled**, and the ordering above is what stands. Three surfaces in the tree
say otherwise, and their status decides how each is handled. `M-0308/AC-2` and
`AC-3` are `met` on a **`done`** milestone, so they are historical evidence and
establish no current obligation; their mechanical evidence is gone besides, since
both were met against `wf-measure-spec`, absent from the tree. G-0583 is
**`open`**, so its claim that "the cheap two run first and always" is live and
needs a disposition rather than an argument.

The strongest case for the other order was measured before it was rejected.
Classifying the eleven cancelled epics' `--reason` bodies —
`git log --all --grep='aiwf-verb: cancel' --format='%H%n%B'`, filtered to epic
ids — puts five within the lab's reach against two within the sweep's, which
argues for running the lab first on yield. It loses to what the blind trial
recorded: the lab cannot produce a false clearance, and the sweep can, so the
sweep's position ahead of it is what stops a reader treating an unmeasured claim
as settled.

### Timing: preflight, with the commitments sweep available earlier

Measurement runs just in time. An epic planned and then deferred leaves any
measured fact stale by the time work starts, which rules out settling facts at
planning time.

Prose ages differently from code. A contradiction between a spec and a standing
decision is still a contradiction weeks later, while call sites and counts move.
So the commitments sweep can usefully run at epic drafting — asking *what has
already been decided in this area* — while the lab stays at milestone preflight.

### Three seams

- **Epic drafting** — the sweep alone. Theory building: what is already settled here. The lab is not dispatched, which is why the two are separate artifacts. This is where the sweep has its measured catches: cancellations cluster as epic-level re-plans rather than per-milestone defects. Its rows are dispositioned in the drafting session, before the epic is committed — there is no implementation yet for a finding to be repaired by or made moot by, and no wrap in reach. An *unknown* row here ends **recorded** or **tracked** rather than measured, since the lab is not dispatched at this seam.
- **Milestone preflight** — the challenge, then the sweep, then the lab. Two dispatches, the ledger between them; the challenge needs neither, since its only input is the criteria. An experiment runs where a human asks for one, pending T6.
- **Patch start** — the challenge and the lab. The sweep runs when the patch has a tracked subject whose premises it inherited; a patch with no entity has no specification to check, which is the escape T7 names rather than a judgment about size.

### Corpus and cost

Reading the corpus whole does not scale; reading its **index** does. Scan the
index, select on subject, read only the selection. Measured on this repo at
`d5dd687ea`, each row naming the command that produced it:

| tier | index | in full | index built by |
|---|---|---|---|
| commitments — every ADR and decision | 1,037 | 90,328 | `title:` from the first frontmatter block, over `docs/adr/`, `work/decisions/`, and both archives |
| normative documents, minus the ADRs already counted above | 1,420 | 58,359 | each path plus its `^#{1,3} ` headings, over the normative tier's 14 remaining files |
| hand-authored root documents | 491 | 17,290 | the same, over the 4 tracked root markdown files that are neither derived nor append-only |
| entities, archived included | 13,985 | 932,830 | `aiwf list --archived`, which carries the status the entity tier requires |
| **total** | **16,933** | **1,098,807** | |

Sixty-five times cheaper. The three document tiers total 2,948 words and fit in
a single read; the entity tier is nearly five times their combined size and
dominates what the index costs, so a project weighing this method should price
that tier first.

**These figures are a dated observation and go stale**, which is why the
snapshot is named beside them rather than left to be inferred: a measurement
without the tree it was taken against is a number nobody can re-derive. Anyone
quoting them re-runs the commands. No priced tier includes `docs/initiatives/`,
so they are independent of this document's own state.

The two tiers select at different granularities, and the difference is worth
keeping. A commitment carries one title stating a conclusion, so the index
selects a **document**. A design document carries roughly twenty headings stating
subjects, so its index selects a **section** — the reader opens one section of a
sixty-thousand-word tier rather than the tier. That is the better half of the
method, and it is available wherever documents carry structured headings.

Two exclusions are load-bearing rather than economies. **Derived documents** — a
generated roadmap, a status page — are excluded because a contradiction inside
one is a stale render rather than a finding about the specification. **Archival
and exploratory documents** are excluded on the authority argument in T3.

What grows with the corpus is the index, at roughly ten words an entry, so a
project with ten times the decisions still has a scannable index. The method
depends on titles and headings being informative; a project whose decisions are
titled "Decision 12" gets nothing from the index and must read everything.

One full sweep, measured: 99 titles scanned, 19 selected, 16 read, 11 findings —
122,000 tokens and about six minutes in one subagent.

**One sweep is not an exhaustive pass.** Four sweeps run against successive
revisions of one epic draft — same brief, a fresh reviewer each time — each
selected sources the others did not, and cutting the subject by a third did not
thin the result. The fourth reached `docs/skill-author-guide.md`, whose rule
that a skill run `aiwf check` before returning success collides with this pass's
own no-measure prohibition; the first three never opened it. So a seam is
planned around a pass that samples the corpus rather than one that exhausts it,
and a second dispatch on the same subject is productive rather than
confirmatory. What terminates the work is the stop rule — every row
dispositioned — not a reader who stops finding things.

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

The sweep and the lab are **two artifacts, dispatched separately**, and the
ledger is the handoff between them. The lab receives every row a command could
settle — the *unknown* rows, which already carry the command the sweep would have
run, **and the *contradicted* rows**, whose own definition is that one side is
wrong and a command decides which. A contradicted row that never reaches the lab
is a doubt nothing discharges, which is the treadmill this pass exists to avoid. That makes the phase boundary structural
rather than a matter of the reader's discipline: the sweeper cannot fill the
measured column, because it is not the agent that runs anything.

Two forces decide it, and neither is a preference. The seams need it: the
commitments sweep runs at epic drafting where the lab does not, so a single
artifact would have to be invoked partially — the shape that reliably goes wrong.
And the two phases need different views of the repository: the sweep reads at a
pinned ref with the working tree off limits, while the lab must measure current
state, because a fact measured early is stale by the time work starts. One agent
holding both would eventually cite the current tree as corpus, which is the
failure the pin exists to prevent.

- **A reviewer brief.** The instructions the dispatched sweep reviewer receives.
  This is the load-bearing artifact, not a wrapper around one: independence is
  what makes the audit's breadth affordable, and the brief is where independence
  is established or lost. Drafted below.
- **A ritual the sweep reviewer follows**, holding the reading method: what to
  sweep, how to select from an index, what may be concluded.
- **The challenge**, run against the criteria alone. It needs no corpus, no
  command and no dispatch of its own, so it can ride with whichever artifact the
  seam already invokes — which is why it is the cheapest part and the one to run
  first.
- **A lab**, dispatched with the sweep's unknown and contradicted rows. It owns the execution
  safety boundary and the experiment, which is lab-class work — it builds, runs,
  and returns works, does not, or inconclusive — rather than a third artifact.
- **Seam instructions** at the three entry points, each naming the artifacts it
  needs rather than restating them.
- **The prompt that puts the ledger in the entity, and the `reviewer` role
  card** — both Class D in the appendix. D-0066 places the ledger in a
  `## Spec measurement` body section written once a pass has run, which is
  evidence rather than scaffolding; a template section every entity carries
  empty is the shape that decision excludes. So the missing artifact is a seam
  instruction that writes the ledger there, not a template. The one role card
  named *reviewer* emits an approval verdict, the state the no-clearance rule
  removes. An implementer who builds only the five artifacts above ships a pass
  whose output reaches no entity and whose dispatched agent is briefed to
  approve.

### The report

A ledger, one row per factual claim the specification makes. It has two halves,
written by different parts of the pass and never by the same one:

| claim | reading | measured | evidence | disposition |
|---|---|---|---|---|
| what the spec asserts | contradicted / ambiguous / complicated / unknown | true / false / unable to measure safely, or blank | the command, its expected result, its output and its environment — or blank | pinned / recorded / tracked / superseded, and the repair if one was made — or blank |

The **reading** column is the sweep's, and carries the four states above. It has
no *verified* value, so a claim nothing settled stays visible as *unknown*.

The **measured** column is the lab's, and is the only place a claim is ever
settled true. Filling it takes four things in the evidence cell, not one: the
**command**, the **expected result** that would make the claim true, the
**observed output**, and the **environment** it ran in — which binary, built from
which tree. A command with no stated expectation settles nothing, because any
output can be read as confirming; and an unstated environment is how a pass
measures older code and reports it as current. That quartet is what makes the
guarantee mechanical rather than a matter of the reader's discipline. **No
command and no oracle, no pass.**

The **disposition** column is the session's, filled during triage at wrap rather
than by the pass, and it is what the wrap gate enumerates.

A blank is the point of the format. A blank reading or measured cell is the only
way an unchecked claim stays visible; a row blank in both is a claim nobody
reached; and a blank disposition at wrap is a finding nobody decided about, which
is exactly what the gate exists to put in front of a human.

Findings that are not claims — a commitment the spec fails to discharge, an
example that no longer works — carry a reading state and leave the measured
column blank unless a command reaches them. A *complicated* row names the
specific obligation.

### The prohibitions

- **Reading may not clear a question.** The sweep's output is a finding or the
  absence of one; no reading may assert that a claim is sound. A claim is settled
  true only in the measured column, only by a command, and only with that
  command's output recorded beside it.
- **The sweep may not measure while it reads.** A reader that starts running
  commands mid-audit conflates the two verdicts and produces a confident wrong
  answer. It records what it would run instead. This is a phase rule, not a
  position on measurement: the lab that follows the sweep exists precisely to run
  what the sweep wrote down, and it is what settles anything.
- **The pass writes nothing outside a disposable tree.** In the repository under
  review it runs only commands that leave no trace. Anything that writes — a
  mutating verb, a scaffold, an experiment's build — runs against a throwaway
  repository or a worktree of its own, and is discarded. Outward actions are
  never available to it: no push, no remote delete, no deploy, no network beyond
  a fetch. A claim that cannot be settled inside those bounds is recorded *unable
  to measure safely*, which is a blank rather than a pass. A pass measuring the
  tool's own behavior builds the binary from the tree under review rather than
  trusting the one on the path, or it measures older code and reports it as
  current.
- **The pass may not edit code, and may not act on its own verdict.** It changes
  the specification, or it returns a disqualifying result for a human to act on.
  Cancelling a milestone is a mutation and stays where every mutation is — behind
  a human gate.
- **It terminates.** The sweep searches names rather than concepts and follows a
  borrowed claim one hop. Both bounds exist so a pass finishes.
- **Every row reaches a disposition, and a repair is not one.** A contradicted
  row names a `file:line` in the corpus, and ends **pinned** (a check that fails
  without the fix), **recorded** (a decision entity, or a declined line in the
  ledger itself), **tracked** (a gap), or **superseded** by another row in the
  same pass. These are the three dispositions the project's code-health force
  already names, plus the one a multi-row pass needs. Repairing the sentence is a
  fact the row records alongside its disposition, never instead of one — a defect
  merely fixed is the silent correction that force forbids, and it is how this
  corpus reached the state the pass keeps finding.

  Three consequences, each load-bearing:

  **Pinning is usually unavailable, and that is expected.** A structural test
  over prose asserts shape, not that a paragraph still says a particular thing.
  So most prose findings end *recorded* or *tracked*, and only findings that are
  code- or shape-shaped can be pinned.

  **The disposition may be an existing entity.** A recurring class is tracked
  once and individual repairs cite it; a fresh entity per instance is what the
  project's own rule on what earns a gap declines. This is what makes the force
  affordable for prose findings rather than a gap mill.

  **Declining is the cheap form of recorded**, costing a line in the ledger
  rather than a decision record — again the force's own wording, not an
  exception to it.

  The pass performs none of this. It returns rows; the session that dispatched it
  triages them — at wrap for the seams that reach one, where the findings have had
  the life of the work to be repaired or made moot, and at epic drafting before the
  epic is committed, where they have not; and the human gates the result. What makes
  the triage happen is that gate enumerating every undispositioned row — see D-0067, which
  records why it is a gate rather than a rule, what mechanizing it would have
  cost, and the measurement that would reverse it.

### The stop rule

A pass ends when every row is **measured, pinned, recorded, or tracked** — not
when it runs out of findings, and never on a reading that found nothing. A row a
command settled needs nothing further; every other row carries one of the three
dispositions, and a row merely repaired carries none of them.

This is deliberately the rule the project already applies to code review, which
converges when a fresh reviewer *"finds no defect that is not already pinned,
recorded, or tracked"*, explicitly **not** on "no findings ever". Terminating on
disposition rather than on absence is what lets a pass finish without anyone
declaring a claim sound: a blank with a named owner is a finish state, and a
blank with no owner is not. It supplies the one thing the no-clearance rule
otherwise removes — a condition under which work may begin — without
reintroducing a verdict that reading could earn.

### The reviewer brief

The blind trial's brief produced eleven findings and pushed every measurable
question to an open-questions list rather than guessing at it. Its load-bearing
parts, which any implementation keeps:

- **Pin two snapshots, and name them separately.** The **corpus** is read at a
  stated ref with the working tree off limits — without that the reviewer reads
  material postdating the work and calls it context. The **subject** — the
  specification under review, and the epic it sits in — is named on its own,
  because it is routinely uncommitted: the shipped guidance has an author edit an
  entity in the working tree so a human can read the diff before it commits, so
  the recommended authoring flow produces exactly the subject a corpus pin cannot
  address. One pin for both makes the reviewer either read a stale subject or
  treat the working tree as fair game for the corpus too. Name the subject, name
  the ref, and say the subject is the only thing read outside it.
- **Select from titles before reading, and record the selection with its reason.**
  Making the choice auditable is what separates a sweep from a search for
  confirmation.
- **Forbid measurement explicitly**, and require that anything the reviewer wants
  to measure is recorded as a question with the command it would run.
- **Ask for contradiction, complication, and silent assumption** — three distinct
  questions, not one request for "problems".
- **Check each attribution against its source.** Where the subject says a record
  settles, requires or names something, open that record and confirm the exact
  sentence carries the exact claim, and report each as supported — with the
  sentence — or unsupported. A claim citing a record for more than it carries
  reads as plausible, and survives every reading that does not open the record.
  The code-review lens carries the same step for a diff; a specification is
  where the claims concentrate, and nothing else reaches them.
- **Withhold the outcome.** A reviewer told what went wrong finds what it was
  told. This is what the blind trial establishes and what a sighted run cannot.
- **Report the sources the reviewer selected that the subject cites nowhere.**
  These are the candidate omissions, and they are what the selection record is
  for. Withholding the subject's own reference list to make the comparison
  two-sided does not work and should not be attempted: a specification that
  argues from its sources names them throughout its prose, so the reverse
  direction — what the author cited that the reviewer would have skipped — is
  empty by construction. Measured on one full run, every reference row was
  already in the reviewer's independent selection, and the sixteen sources the
  reviewer selected that the subject never mentioned were the entire yield.
- **Instruct it to say so plainly when it finds nothing**, rather than
  manufacturing findings — and, per the prohibitions, without converting that
  into a clearance.

### Promotion status

**Inventory as of 2026-08-19.** One thread is promoted in reduced form, seven
are dropped, one was never promoted.

- **Promoted, reduced** — T5, as the lab *rule* and nothing else: a claim is
  settled true only where a command, its expected result, its observed output
  and its environment sit together. It is a ban on what may be written down,
  not the dispatched lab this document specifies. E-0086 carries it as its whole
  scope.
- **Dropped** — T1, T2, T3, T6, T7, T8, T9, and with them the sweep, the
  criterion challenge, the mandated ledger, the wrap-gate enumeration, the
  per-seam instrumentation and the seam instructions. The reason is one
  property of the method: a generative review over prose has an unbounded output
  space, so its finding count measures reviewer effort rather than artifact
  quality, successive rounds are not comparable, and zero is unreachable by
  construction. D-0069 records it with the measurements and the commands that
  reproduce them.
- **Never promoted** — T4. Neither attempt carried a code tier, so nothing ever
  described what a code-comment audit would read.

Two implementation attempts, both terminal.

[E-0085](../../work/epics/archive/E-0085-measure-the-spec-before-the-code-record-what-the-measurement-changed/epic.md)
was the first, and it is cancelled. It was planned from
[G-0583](../../work/gaps/G-0583-the-milestone-preflight-asks-for-judgment-with-no-method.md)
— a partial capture of this specification, filed before the specification was
recorded — and the ritual it shipped was withdrawn for contradicting the
document it was built from: it let a reader clear a claim, and it left the
author in the reviewer's seat. Its second milestone was cancelled before
starting. Its first is `done`, with criteria asserting properties of a file that
no longer exists.

What survives the cancellation is this document, plus:

- [D-0066](../../work/decisions/D-0066-record-a-spec-measurement-as-a-body-section-not-a-trailer.md)
  — a completed pass records itself as a body section written with `aiwf edit-body`,
  because a trailer has no slot and an empty body write produces no commit.
- [G-0584](../../work/gaps/G-0584-ritual-skill-tests-assert-section-scoped-literals-that-cannot-fail.md)
  — the ritual policy tests match literals transcribed from the prose they guard.
- [G-0585](../../work/gaps/G-0585-rituals-clear-a-question-by-reading-it.md)
  — every site the table below lists grants a verdict on the basis of reading.
  G-0585 records that enumeration as a floor rather than a census, since the same
  clearance phrase recurs across surfaces.
- [G-0586](../../work/gaps/G-0586-the-handoff-block-carries-settled-findings-and-drops-open-ones.md)
  — the compaction handoff forwards settled conclusions and has no slot for an open question.
- [G-0587](../../work/gaps/G-0587-a-shipped-skill-cannot-name-the-docs-corpus-a-review-must-read.md)
  — a shipped ritual is barred from naming the corpus this pass must read.

[E-0086](../../work/epics/archive/E-0086-reduce-the-preflight-pass-to-the-lab-rule/epic.md)
was the second. It planned seven milestones and built one — the reviewer brief,
on a branch that was not merged. It ran this document's method four times
against its own epic draft and recorded the result in its `## Spec measurement`
section; that record is what closed it. The epic is closed re-scoped to the lab
rule, and D-0069 carries the rejection, the counts, and the commands that
reproduce them.

What survives it is D-0069, and that `## Spec measurement` section as the
record of the four sweeps.

## Open questions

| Question | Blocking a promotion? | Resolution path |
|---|---|---|
| What triggers an experiment? | T6 | The intended test — a load-bearing feasibility claim — decided one of this project's four live epic specifications and could not decide three. Its second half is a counterfactual about value that no specification states. Needs the author, not a measurement; everything else in *What ships* is buildable without it. |
| Do the title index and the no-clearance rule hold in a project that is not this one? | all | Both were measured here only. The index depends on titles being informative, which is a property of this repo's conventions rather than of the method. The document tier additionally depends on the project ranking its documentation by authority, which most do not. |
| Does reverse-reference walking reach beyond entities? | T3 | It walks entities only, not documents or templates, so a document contradicting a specification is reached by the index scan or not at all. Stated as a limit of the method rather than a question awaiting an answer. |

## Appendix — where the existing skills fight this specification

An independent audit read 24 of 48 shipped skills, agent cards and templates
against the rules above, selected from titles and frontmatter descriptions.
Findings are grouped by class, because the class decides the disposition: one
fix pattern across many sites is one piece of work, not many.

### Class A — a clearance earned by reading

The largest class and the one the specification exists to remove. Legitimate
where a command earns the verdict; forbidden where reading does. Several skills
mix both under one word.

| site | quoted | note |
|---|---|---|
| `wf-review-code:127` | "something you checked and found sound; not a defect, since you **verified** it" | an output-template line whose literal text is a clearance |
| `wf-codebase-health:775` | "Strong / Weak / Missing… **A Strong that survives a real refutation attempt is actually strong**" | the refutation is itself a read |
| — | — | *(Withdrawn. The rule at `wf-codebase-health:409`, and its mirror in the always-on guidance, terminate on **disposition** — "finds no defect **that is not already pinned, recorded, or tracked**… Not 'no findings ever'". That is not a clearance, and it is the stop rule this specification adopts under* The prohibitions*.)* |
| `wf-vacuity:68–70` | "`## Clean`" / "assertions / properties **confirmed** to constrain behaviour — a real bug makes them fail" | probe 1 is a command and legitimate; probe 2 is reading and lands in the same section |
| `wf-tdd-cycle:79` → `aiwfx-start-milestone:152` | "this audit is **agent-performed** — not a tool invocation" → declared as "branch-coverage audit **clean**" | a reading walk produces a clearance a downstream reader consumes as settled |
| `wf-doc-lint:164` | "**No findings** — `<N>` docs checked" | checks 5–6 are commands; checks 2–3 are self-described heuristics; one summary line cannot distinguish them |
| `aiwfx-wrap-milestone:65` | "If the report is **clean**, note 'doc-lint: clean'" | inherits the ambiguity above into the wrap record |
| `wf-structural-sweep:51` | "a **scorecard** (Strong / Weak / Missing per principle)" | lens 3 is "the reasoner's lens"; the scorecard cannot express *unknown* |
| `wf-rethink:123` | "**A rethink that confirms the current design is a successful audit**" | a fit-checker returning a verdict on whether the proposal works |

The composing precedents already in the corpus are worth copying rather than
re-deriving: `wf-property-test:88` — *"Say 'checked over generated inputs,' not
'verified.'"* — and `wf-vacuity:92`, *"A clean report is 'no weakness found,'
not 'tests verified correct.'"*

### Class B — the handoff drops the unsettled half

`aiwfx-handoff` produces the block pasted into a context compaction. Its rule at
`:31` forwards *"a finding not to re-open"*, instructing the next session to
treat a question as closed. There is no counterpart line for a question still
open, and the ten-line cap makes the omission structural rather than incidental.
The pointer it substitutes for detail surfaces no ledger, because no entity
template carries one.

### Class C — the seams do not run the pass

Each of these is the seat this specification claims, currently occupied by a
reading pass the implementer runs on their own work.

| site | quoted |
|---|---|
| `aiwfx-start-milestone:20` | "**Confirm every AC is concrete and testable.** If any AC is vague, stop" |
| `aiwfx-start-epic:30` | "**Confirm** the Goal, Scope, Out of scope and Constraints… are concrete prose" |
| `aiwfx-plan-epic:25` | reads `ROADMAP.md`, `work/epics/`, `work/gaps/` — never the ADRs and decisions |
| `wf-patch:36` | "state the user-observable goal **in your own words**" — no sweep, lab or ledger before step 6 |

### Class D — nothing prompts an author to record a ledger

`templates/milestone-spec.md` carries no section for one, and
`templates/epic-spec.md`'s Open questions table — the nearest existing structure
— has no evidence column and admits only questions, never claims. The section is
legal on every kind (ADR-0043), but D-0066 places it as evidence written once a
pass has run rather than as scaffolding every entity carries, so the prompt
belongs in the seam instruction and not in a template: a claim nobody measured is
invisible by default rather than impossible to record.

`agents/reviewer.md:18` compounds it: the one role card named *reviewer* emits
"**approve**, request-changes, or questions" and carries no ledger, no four
states, no ref-pinning and no forbid-measurement clause.

### Class E — a shipped skill cannot name the corpus it must read

`aiwfx-record-decision:148–154` bans a shipped ritual from embedding a path
under `docs/`: *"It does not embed a markdown link to a decision record or
design doc under `docs/`… do not point at a repo file the reader does not
have."* The reviewer
brief must open `docs/adr/` and `docs/design/`. Both rules are current, both are
right in their own terms, and they collide. The brief has to express its corpus
as a shape rather than a path, or it is not shippable.

### Class F — read and no conflict found

`aiwfx-wrap-epic`, `aiwfx-whiteboard`, and `aiwfx-plan-milestones` steps 6–10
were read without a conflict being found. That is a statement about the reading,
not a property of those files. Twenty-four further files — the verb skills,
`aiwfx-release`, `deployer.md`, `wf-property-test` and two templates — were
covered by vocabulary greps only and not read.

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
