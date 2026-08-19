---
id: M-0310
title: Settle what a pass run records about itself
status: draft
parent: E-0086
tdd: none
acs:
    - id: AC-1
      title: Every recorded miss names the line that would have shown it
      status: cancelled
    - id: AC-2
      title: Every recorded addition names the tier it lives in
      status: cancelled
---
## Goal

Settle what a run of the preflight pass records about itself — what it missed,
and what it added to the corpus — so runs at successive milestones can be
compared.

## Context

E-0086 records two things per run and derives the rest. D-0068 defines the
first: a specification defect, found between the pass and the milestone's wrap,
that a source in the pass's own corpus contradicted, named by the line that
would have shown it. The second is the records a pass creates, priced by the
tier each lives in.

Everything else a run produces — rows returned, rows by disposition, what each
finding changed, rows undispositioned at wrap — is recoverable from the
committed ledger and git history, so D-0054 puts it outside prose. What a pass
costs to run is an audit question and is not recorded here.

This milestone sits sixth in the epic's sequence. Both fields are recoverable
after the fact, so nothing has to be recorded in advance for consecutive runs to
stay comparable.

## Acceptance criteria

Both criteria pin a shape an accepted record already makes necessary, per the
epic's constraint on tests over prose.

### AC-1 — Every recorded miss names the line that would have shown it

D-0068 admits a miss only with the corpus line that contradicted the
specification. A test asserts every entry in the miss slot carries a
`file:line`, so an entry recording a claim nobody can check fails.

The shape is testable because D-0068 requires it, not because a phrase was
chosen to be matched.

### AC-2 — Every recorded addition names the tier it lives in

D-0054 names three tiers a record can live in. A test derives the tier names
from D-0054 and asserts every recorded addition names one of them, so an
addition carrying no tier, or an invented one, fails.

Deriving from D-0054 rather than from this milestone's own prose is what keeps
the expected side independent of the side under test.

## Constraints

- **Two recorded fields; everything else is derived.**
- **Tests resolve entity files through the loader**, never a path literal —
  `PolicyNoHardcodedEntityPaths`. Every record these tests read archives.
- **The mandate names its owner and what retires it.** G-0515 and D-0053 carry
  the shipped two-part form.
- **Every count names the command that produced it**, or is dropped.
- **The record is evidence, not scaffolding**, per D-0066 — which also requires
  that a pass finding nothing writes the section anyway, saying so.

## Design notes

- D-0068 settles what counts as a miss and who records it.
- D-0054 names the three tiers a record can live in, and holds that a fact
  another owner already carries is not worth prose.
- D-0066 settles that a completed pass writes a `## Spec measurement` section
  into the body of the entity whose claims it measured.
- D-0067 makes the undispositioned-at-wrap count part of what the trial carries;
  the ledger committed at the wrap commit is that count.

## Dependencies

- D-0068, D-0066, D-0067, D-0054 — accepted.
- The milestones ahead of this one, which produce the runs it records.

## References

- `docs/initiatives/milestone-preflight-as-independent-review.md` — the specification
- D-0068, D-0066, D-0067, D-0054, D-0050 — the miss, where a pass records
  itself, the required count, what is worth recording and where, test shape
- G-0490 — a metrics mandate that shipped and proved unadoptable
- G-0530 — milestone sections that duplicate structured data

## Spec measurement

Two runs, three sweeps each, same brief, a fresh reviewer every time, against
two successive drafts. Rows are clustered by class, because the class decides
the disposition.

| run | subject | sweeps | tokens | slowest wall | rows |
|---|---|---|---|---|---|
| 1 | first draft | 3 | 495,821 | 9.8 min | ~65 |
| 2 | rewrite | 3 | 513,234 | 10.1 min | ~70 |

Token and wall-time figures are the agent runtime's, not a command's. They are
a dated observation of these two runs, not a field the record carries: what a
pass costs to run is an audit question and is out of scope here. Row counts are of returned
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
