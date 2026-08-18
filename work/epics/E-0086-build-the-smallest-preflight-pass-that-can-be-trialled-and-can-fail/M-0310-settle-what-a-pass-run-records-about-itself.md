---
id: M-0310
title: Settle what a pass run records about itself
status: draft
parent: E-0086
tdd: none
---
## Goal

Settle what a run of the preflight pass records about itself — its cost, its
yield, and what it later turned out to have missed — so runs at successive
milestones can be compared.

## Context

This milestone is `draft` and does not start until the epic's blocking question
about a prospective per-run record resolves. Three surfaces push against the
deliverable: `growth.md` reconstructs apparatus metrics from git history at any
commit and holds that nothing must be measured in advance to stay comparable;
D-0054 records obligations rather than events, and a finished run is a completed
act; the not-in-scope list excludes an append-only event log.

A fourth problem is internal to the deliverable rather than to the corpus. The
fields that carry cost — tokens and wall time — are reported by the agent
runtime, and no command in the tree emits either. A record requiring every count
to name its producing command cannot carry them, and a record without them
cannot answer whether a seam earns its cost.

D-0066 settles where a completed pass writes itself. Whether a run's own cost
and yield belong in a second record is what the epic's question decides.

## Acceptance criteria

None allocated. The criteria depend on whether the deliverable survives the
epic's blocking question, and criteria written against a deliverable that may not
exist are the churn this epic was opened to measure.

When they are allocated they cover: the fields a run records; that cost and
yield are recorded per seam and never pooled; and one record filled from a real
run. Each is prose-shaped, so each needs the derivation M-0308 used for its
outcome count — expected value from heading shape rather than from a literal —
and each must state what it does not establish. `docs/design/oracles.md` holds
that an expectation written by the same author from the same misreading is a
mirror rather than an oracle, and both sides of a prose-to-prose test are
authored by the same work.

## Constraints

- **The fields reconcile what already exists** rather than originating a
  vocabulary. The initiative carries a dated observation of one run; E-0086's
  Scope names what instrumentation must carry; D-0067 makes the
  undispositioned-at-wrap rate required rather than optional.
- **Tests resolve entity files through the loader**, never a path literal, per
  `CLAUDE.md` §"Test design rules" and its `PolicyNoHardcodedEntityPaths`
  chokepoint. Every record these tests would read is an entity file that
  archives.
- **The mandate names its owner and what retires it.** G-0515 and D-0053 carry
  the shipped two-part form.
- **Every count names the command that produced it**, or is dropped.
- **The record is evidence, not scaffolding**, per D-0066 — which also requires
  that a pass finding nothing writes the section anyway, saying so.

## Design notes

- D-0066 settles that a completed pass writes a `## Spec measurement` section
  into the body of the entity whose claims it measured.
- D-0067 makes the undispositioned-at-wrap rate part of what instrumentation
  must carry rather than an optional extra.
- D-0054 settles that records are tiered by retrieval cost, and that a fact a
  check or a field could hold is not worth writing as prose.
- D-0038 records, of a rejected evidence flag, that check-time resolution proves
  a symbol exists rather than that it exercises the claim.
- G-0584 records what an expected value transcribed from the prose it guards
  costs: the assertion passes because someone typed the word.
- ADR-0043 settles that a section beyond an entity's required set is legal.
- ADR-0042 gates body completeness at the readiness promote, which this
  milestone reaches only if its question resolves.
- G-0530 proposes cutting three of the sections a milestone spec carries; this
  body omits them.
- G-0490 records a metrics mandate that shipped and proved unadoptable because
  its cost scaled with subjects rather than with discipline. It is the nearest
  precedent for what this record would cost.

## Spec measurement

Two runs, three sweeps each, same brief, a fresh reviewer every time, against
two successive drafts. Rows are clustered by class, because the class decides
the disposition.

| run | subject | sweeps | tokens | slowest wall | rows |
|---|---|---|---|---|---|
| 1 | first draft | 3 | 495,821 | 9.8 min | ~65 |
| 2 | rewrite | 3 | 513,234 | 10.1 min | ~70 |

Token and wall-time figures are the agent runtime's, not a command's — the
contradiction this milestone's own subject names. Row counts are of returned
table rows and are approximate; no command produced them.

The challenge ran twice, author-run, at no corpus cost, and found three findings
each time. The sweeps independently reached two of each three.

| claim | reading | measured | evidence | disposition |
|---|---|---|---|---|
| Five attributions claim more than their record carries | contradicted | true | D-0054 assigns no tier to any record; D-0038's phrase is scoped to one rejected mechanism; G-0584 is open and settles nothing; the archive bar is every thread, and one is not promoted; the initiative carries a dated observation, not a vocabulary | repaired here; tracked — G-0594 |
| Two authoring rules shipped, and the next spec written breached both | contradicted | true | the five attributions were produced after six citation-shaped references were verified by extraction; argument appeared in criteria bodies against the shipped genre line | tracked — G-0594 |
| A prospective per-run record survives the corpus | complicated | | `growth.md` on advance measurement; D-0054 on events; the event-log exclusion | recorded — blocking question on E-0086 |
| Tokens and wall time have no producing command, and the same criterion demands one | contradicted | true | the agent runtime reports both; no command in the tree emits either | recorded — the same question |
| The derivation form reaches further than the first draft claimed | contradicted | true | M-0308 used two derivations, and the outcome count from heading shape has no code side | repaired in E-0086; recorded |
| A prose Scope bullet has no determinate item count | ambiguous | | the bullet is one comma-run whose last clause reads as one item or two | recorded |
| The run record has no named home | ambiguous | | the draft named directories, never a heading | recorded |
| Cost and yield have two axes, phase and seam | ambiguous | | E-0086 names per seam in one bullet and per-phase cost and yield per seam in another | recorded |
| Obligations the draft did not name | complicated | | `PolicyNoHardcodedEntityPaths`; ADR-0042; the shipped template's section set | recorded — carried above |
| Two blocking questions deferred without an owner, one claimed settled without a term | complicated | | E-0086's Open questions | recorded — they stay the epic's |
| Milestone sections that duplicate structured data | complicated | | G-0530 names three of them | tracked — G-0530 |
| A metrics mandate that shipped and proved unadoptable | complicated | | G-0490 | tracked — G-0490 |
| A mandate with no named owner and no retirement trigger | complicated | | E-0086 requires one; G-0515 and D-0053 carry the form | recorded — carried above |
| Obligations reaching this work from live entities | complicated | | G-0587, G-0560, G-0571, D-0053, D-0056, ADR-0044 | tracked — each has an entity |
| A repair round does not converge | contradicted | true | run 2 cost more and returned more rows than run 1, against a subject rewritten to answer run 1 | recorded — this section is the record |

**What this pass missed, recorded against itself.** The two runs disagreed on
one attribution, and reading the cited record's section scope resolved it, so a
single sweep's attribution verdict is not final either. Three of the triage
session's own verification commands returned false negatives — one defeated by a
line wrap, one by a shell glob, one run against the wrong checkout — and every
one would have reported a supported claim as unsupported.
