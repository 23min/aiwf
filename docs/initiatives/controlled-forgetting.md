---
title: Controlled forgetting — current sources by default, history by deliberate recall
status: captured
date: 2026-08-15
---

# Controlled forgetting — current sources by default, history by deliberate recall

## Classifier note

This is an initiative document. `initiative` is not yet an official aiwf entity
kind
([G-0311](../../work/gaps/G-0311-no-cross-cutting-initiative-tier-above-epic-for-multi-component-features.md)),
so it lives under `docs/initiatives/` in the forward-looking tier: a captured
idea awaiting evidence and promotion to tracked entities.

This initiative is the smallest experimental slice of
[`current-context-as-a-first-class-projection.md`](current-context-as-a-first-class-projection.md).
It does not supersede that larger design or commit to building it. It tests the
premise that gives the larger design value: whether making obsolete and
irrelevant history absent from the default reading set improves work without
hiding useful evidence.

The first trial runs inside
[`milestone-preflight-as-independent-review.md`](milestone-preflight-as-independent-review.md),
but this remains a separate initiative. Preflight owns review and measurement;
controlled forgetting owns what material is offered by default and how a reader
deliberately widens into history. If the method proves useful, ordinary planning
and implementation sessions can use it without adopting the rest of preflight.

## Initiative statement

aiwf keeps history because provenance, reversibility, and prior failures matter.
The same history becomes harmful when every old plan, superseded commitment, and
archival document is ambient context for current work.

Controlled forgetting means:

> **Current sources are offered by default. Historical material remains
> retrievable, but enters only through an explicit, labeled recall step.**

Nothing is deleted. Nothing becomes inaccessible. The change is to the default
retrieval path: history stops being an undifferentiated extension of current
truth.

This initiative addresses two mechanically different questions:

1. **Is this source current or historical?** Entity kind, lifecycle status,
   archive location, and documentation role can answer this.
2. **Is this current source relevant to this subject?** A human or LLM still
   selects this, and records why.

The first can be made reliable. The second cannot be inferred safely without an
applicability model that this initiative deliberately does not introduce.

## Existing substrate

The experiment starts from behavior aiwf already has:

- `aiwf list` shows nonterminal entities by default and requires `--archived` to
  widen into terminal entities;
- entity statuses distinguish accepted commitments from proposed, superseded,
  rejected, retired, cancelled, and completed records;
- deprecated contracts remain operationally in force during their sunset;
- the repository documentation hierarchy distinguishes normative current truth
  from forward-looking, exploratory, and archival material; and
- epic and milestone templates already carry `## References`, though those
  sections are broad author-provided hints rather than complete applicability
  declarations.

The experiment composes these existing signals. It adds no persisted state.

## The smallest experiment

The experiment is a read-only retrieval protocol inside milestone preflight.
It has two lanes that never lose their labels.

### Lane 1 — current sources

The default source index contains metadata only:

- accepted ADRs;
- accepted decisions;
- accepted or deprecated contracts, with `deprecated` visible; and
- documents in the repository's normative current-truth tier.

Each entry carries only the information needed for selection:

```text
role | id or path | title | lifecycle status | headings where applicable
```

The reviewer scans the index, records each selected source and the reason for
selection, then reads only those bodies. Eligibility is not applicability: an
accepted ADR may be current without applying to the milestone.

The subject's bounded starting material remains available regardless of index
selection:

- the milestone specification;
- its parent epic;
- declared dependencies; and
- sources the subject directly cites.

Direct citations are candidate context, not proof that a source is current,
applicable, correct, or complete.

### Lane 2 — historical hazards

History is not part of the default reading set. Preflight may deliberately open
a historical-hazard lane because finding prior failed approaches is one of that
review's purposes.

The lane begins with metadata rather than bodies: entity title, kind, lifecycle
status, and archive state. A historical body is opened only with a recorded
reason, such as:

- the subject borrowed a premise from it;
- it is a predecessor or superseded design;
- its title suggests an analogous cancelled approach; or
- a current source explicitly points to it for rationale.

Anything recovered through this lane is labeled **historical evidence**. It may
explain a failure, reveal a precedent, or raise a question. It cannot establish
a current obligation merely because it was once accepted or implemented.

This preserves the useful part of memory without letting history masquerade as
present truth.

## First host: milestone preflight

Controlled forgetting is neither a prerequisite to preflight nor a feature to
build after preflight. Its first form is an experiment **within** milestone
preflight:

1. Pin the subject and corpus as preflight requires.
2. Build the current-source metadata index from existing repository signals.
3. Select current candidates and record reasons before reading their bodies.
4. Compare the selection with the subject's `## References`; where practical,
   freeze independent selection before revealing that section.
5. Read the union, retaining whether each source was independently selected,
   author-referenced, or both.
6. Open the historical-hazard lane deliberately, select by title/status, and
   record a reason before reading any historical body.
7. Pass substantive findings and empirical questions to preflight's own report.
8. Record the retrieval cost and yield separately for the two lanes.

Perfect blinding is not a guarantee in this experiment. References also occur
inside acceptance criteria, design notes, and dependencies. The trial records
what was visible rather than manufacturing a redacted subject or claiming an
independence property it did not achieve.

Preflight remains responsible for contradiction findings, empirical
measurement, experiments, safety, and disposition. Controlled forgetting
produces a bounded, role-labeled input; it produces no pass or clearance.

## What the trial records

For each consecutive milestone trial:

- current-source index entries and input size;
- current sources selected and bodies actually read;
- selection reasons;
- independently selected but unreferenced sources;
- author-referenced but independently unselected sources;
- stale or historically classified author references;
- whether the historical lane was opened;
- historical titles inspected, bodies opened, and reasons;
- consequential findings unique to each lane;
- sources read that produced no useful evidence;
- input tokens and wall time for indexing, current reading, and historical
  recall separately; and
- relevant context later found during implementation that the preflight missed.

The initial trial should use consecutive milestones rather than hand-picked
difficult work. Otherwise the result measures the selector used to choose the
trial as much as it measures controlled forgetting.

## What would count as success

The experiment supports a durable current-source capability if it shows that:

- readers inspect materially fewer source bodies than an undifferentiated
  corpus sweep;
- current obligations remain discoverable;
- historical recall still finds consequential precedents or prior failures;
- lifecycle/status labels prevent historical evidence from being treated as a
  current commitment; and
- the retrieval cost is proportionate to the specification changes, avoided
  rework, or disqualifying findings it produces.

A finding-free run is not evidence of success by itself. Success is about
bounded retrieval with measured coverage and later escape detection, not a
claim that the selected context was complete.

## Promotion ladder

The initiative adds machinery only when the preceding step demonstrates the
need.

### Step 0 — ritual-only trial

Use existing statuses, archive behavior, repository documentation tiers, and
ordinary read commands. Generate indexes ephemerally inside preflight. Add no
kernel command, configuration, frontmatter, or migration.

### Step 1 — portable current-source view

Promote a read-only kernel projection only if the trial is valuable but manual
index construction is duplicated, inconsistent, too repository-specific, or
too easy to omit.

The smallest candidate is:

- one machine-readable normative-document marker, likely `normative: true`;
- lifecycle-derived commitment eligibility; and
- a deterministic metadata-only current-source listing.

The current documentation hierarchy then explains the marker rather than
remaining a second exhaustive registry. Historical entity retrieval continues
through existing explicit archive/list/show surfaces unless evidence shows a
separate index is necessary.

### Step 2 — durable subject context

Per-subject declarations are considered only after repeated evidence that
correctly discovered context is subsequently lost or allowed to go stale. The
relevant failure classes are:

- a source is identified once but lost across later sessions or handoffs;
- work keeps relying on a superseded, retired, archived, or moved source because
  no reverse binding exists; or
- `## References` cannot carry the necessary distinction without ambiguity or
  duplicate sources of truth.

Only this evidence can justify `commitments:`, `current_docs:`, inheritance,
`set-context`, lifecycle gates, and brownfield onboarding. A high-yield one-time
preflight justifies discovery, not persistence.

## Explicit non-goals

The experimental initiative does not add:

- entity frontmatter fields;
- subject-to-source bindings or inheritance;
- a mutating verb;
- configuration or repository onboarding;
- required-at-lifecycle-seam checks;
- a taxonomy of concerns, topics, capabilities, areas, or applicability;
- automatic relevance inference;
- section-level applicability;
- document ids or document lifecycle verbs;
- generated context bundles or semantic summaries;
- automatic mutation from a preflight finding;
- filesystem access controls that prevent an assistant from opening history; or
- a claim that the selected source set is complete.

AIWF cannot stop an assistant with repository access from searching every file.
The initiative instead makes the bounded route explicit, cheap, and reviewable,
and makes any widening into history visible.

## Risks and controls

| Risk | Control |
|---|---|
| A forgotten historical record contains the best warning | Preflight deliberately opens the metadata-only historical-hazard lane and records later escapes. |
| A current source is accepted or normative but wrong | Source role means intended standing, not truth; preflight still reviews and measures claims. |
| Titles or headings are uninformative | Record selection misses and fall back to broader reading without calling the bounded pass complete. |
| `## References` anchors the reviewer | Freeze independent selection first where practical and record visibility honestly. |
| The historical lane becomes ambient history again | Keep it a separately named step and require a reason before opening each body. |
| Ritual discipline is unreliable | That is evidence for the Step 1 read-only projection, not for durable bindings. |
| A portable normative marker creates a second registry | Replace the exhaustive registry rather than maintaining both as canonical sources. |

## Relationship to the larger initiatives

This initiative owns the current-versus-historical retrieval boundary and the
evidence needed to promote it.

The independent-preflight initiative owns the reviewer, corpus and subject
pinning, claim and finding reports, lab, experiments, safety, and human
disposition. It is the first trial host, not a downstream consumer of an
unbuilt kernel feature.

The current-context initiative remains the design-space record for durable
subject bindings and mechanical projection. It is neither a prerequisite nor a
committed sequel. Its features are promoted individually only when this
initiative's evidence reaches the corresponding step in the promotion ladder.

The dependencies therefore point one way during the experiment:

```text
existing lifecycle and documentation signals
                    |
                    v
       controlled-forgetting protocol
                    |
                    v
      milestone-preflight trial and metrics
```

No preflight verdict mutates retrieval state, and no successful retrieval run
attests that preflight passed.

## Open questions

- How many consecutive milestone trials are enough to distinguish a useful
  pattern from one unusually dense subject?
- Does feeding the full historical title/status index produce enough unique
  findings to justify its token cost, or should historical retrieval begin from
  explicit search terms?
- How often does `## References` omit consequential sources, retain stale ones,
  or duplicate citations elsewhere in the subject?
- Which current-document classification can a consumer repository provide
  during a ritual-only trial when it has no documentation hierarchy?
- Does ordinary implementation benefit from the same protocol after preflight,
  or does the preflight report already carry the needed bounded context forward?

## Promotion status

Nothing is promoted. The initial deliverable is evidence from a ritual-level
trial, not a kernel feature. No gap is filed for later steps until the preceding
step demonstrates the named failure.

## Provenance

Captured from the design discussion about additive history, LLM memory, and the
need to forget without deleting. The larger current-context projection was
reduced under YAGNI pressure to isolate the smallest falsifiable premise:
whether current-by-default retrieval plus deliberate historical recall improves
context quality before any durable bindings exist.
