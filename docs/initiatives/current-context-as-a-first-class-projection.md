---
title: Current context as a first-class projection over additive history
status: captured
date: 2026-08-15
---

# Current context as a first-class projection over additive history

## Classifier note

This is an initiative document. `initiative` is not yet an official aiwf entity
kind
([G-0311](../../work/gaps/G-0311-no-cross-cutting-initiative-tier-above-epic-for-multi-component-features.md)),
so it lives under `docs/initiatives/` in the forward-looking tier: a captured
design awaiting promotion to tracked entities.

This is not an ADR. It records the design reached so far, including its explicit
non-goals and unresolved questions, but does not ratify the feature or commit to
an implementation sequence. In particular, its relationship to
[`milestone-preflight-as-independent-review.md`](milestone-preflight-as-independent-review.md)
remains open while the two initiatives are iterated in tandem.

This is also not a semantic drift checker. It projects which sources are meant
to bind current work; it cannot prove that those sources are mutually
consistent or still match the code. The existing
[`normative-docs-drift-audit.md`](normative-docs-drift-audit.md) demonstrates
that a document can be normative and wrong at the same time.

## The problem

aiwf preserves history deliberately. Entities are created rather than deleted;
terminal ones are archived; accepted commitments can later be superseded,
deprecated, retired, or rejected. Git preserves every prior form. This gives the
project provenance and reversibility, but it also makes the repository a poor
answer to a simpler question:

> What is meant to bind this work now?

A human can often reconstruct the answer by reading statuses, paths, links,
commit history, and surrounding prose. An LLM can do the same, but only by
making substantive selection judgments that are easy to miss: whether an old
decision still applies, whether an archived plan is merely evidence, whether a
design document is current truth, and whether a related source was omitted.
The larger the history becomes, the more likely a plausible but obsolete
statement is mistaken for a current fact.

The failure is not that history is additive. The failure is asking history to
serve simultaneously as audit trail, memory, and current state. Those are
different views over the same repository.

The initiative is to make **current context** a first-class, mechanically
projected view. History remains additive and queryable, while ordinary work
starts from a bounded set of declared commitments and normative documents.
Forgetting becomes a property of the default projection, not deletion of the
record.

## Desired outcome

For an epic, milestone, or tracked patch subject, aiwf can answer:

- which accepted commitments and normative documents bind it;
- which sources were declared by the subject and which were inherited;
- whether every referenced source is still eligible;
- where a stale reference must be rebound explicitly; and
- which repository-wide sources were available when a human or independent
  reviewer checks whether the declaration is complete.

The answer is deterministic metadata, not an LLM-generated summary and not a
copy of the sources' bodies. Substantive interpretation remains in the source
documents, the code, and independent review.

## Terminology

The vocabulary deliberately avoids overloading *authority*, because aiwf
already uses authorization for user delegation and scope.

- **Authorization** — who may perform work, for which scope, through the
  existing `aiwf authorize` model.
- **Normative document** — a document maintained as current truth for the
  project.
- **Commitment** — an accepted ADR or decision, or an accepted/deprecated
  contract that remains in force.
- **Current context** — the effective commitments and normative documents for
  one work subject, derived from its direct declarations plus inheritance.
- **Context sources** — the repository-wide inventory of sources eligible to
  be declared as current context.

Current context is not an authorization mechanism, and a normative document
does not authorize an actor.

## Design

### 1. Two eligible source classes

#### Commitments

Eligibility follows the lifecycle already encoded by each entity kind:

| Kind | Eligible statuses | Ineligible statuses |
|---|---|---|
| ADR | `accepted` | `proposed`, `superseded`, `rejected` |
| Decision | `accepted` | `proposed`, `superseded`, `rejected` |
| Contract | `accepted`, `deprecated` | `proposed`, `retired`, `rejected` |

A deprecated contract remains eligible because the existing contract lifecycle
defines it as in force but discouraged during a graceful sunset. Its status is
shown prominently in projections. Retirement or rejection ends eligibility.

Eligibility only places a source in the repository inventory. It never makes
that source apply globally or automatically to a subject.

#### Normative documents

An ordinary Git-tracked Markdown document opts in with one canonical
frontmatter marker:

```yaml
normative: true
```

The marker applies to the whole document. Headings are discovery metadata, not
a section-level applicability language. Entity files and documents under the
standard archival trees are excluded even if they carry the marker. Host
guidance such as `CLAUDE.md` is normally left unmarked; a repository may mark a
Markdown document outside `docs/` when it genuinely represents project truth.

This replaces a hard-coded path registry as the machine-readable contract.
Human guidance can explain the documentation model, but correctness does not
depend on an LLM remembering that explanation.

Normative documents are rewritten in place as current truth. Git owns their
revision history. A document that needs to preserve an old narrative is
archived and ceases to be an eligible source.

### 2. Explicit subject declarations

Work subjects carry two frontmatter fields:

```yaml
commitments:
  - ADR-0001
  - D-0001
current_docs:
  - docs/architecture.md
```

The distinction between absent and empty is load-bearing:

- absent means legacy or not yet reviewed;
- `commitments: []` means reviewed and no commitment applies;
- `current_docs: []` means reviewed and no normative document applies.

The fields remain optional while an epic or milestone is being drafted. They
become required at existing lifecycle seams rather than through a new event:

- epic activation;
- milestone start;
- tracked patch start/integration, with the gap as the patch subject.

Untracked patches receive the preflight procedure but create no durable
current-context entity or event. If repeated use later proves that omission
harmful, it can earn a separate design then.

### 3. Additive inheritance

An epic declares its broad context. A milestone inherits that context and may
declare only milestone-specific additions. A patch for a tracked gap uses the
gap's declaration as its durable context.

Inheritance is additive:

- a child cannot subtract an inherited source;
- a child does not repeat an inherited source;
- the effective projection deduplicates sources and records every declaration
  origin; and
- if a later parent edit makes an existing child declaration redundant,
  validation reports a nonblocking cleanup finding rather than changing the
  child silently.

This keeps each fact at one owner. A milestone's effective context may contain
ten sources while its own frontmatter contributes only two.

### 4. One write path

Context declarations are changed through a dedicated verb:

```text
aiwf set-context <subject> \
  --commitment <id> ... \
  --current-doc <path> ...
```

The verb replaces the subject's complete **direct declaration**, not its
effective inherited context. It requires exactly one choice for each category:
one or more values, or the explicit empty form `--no-commitments` /
`--no-current-docs`.

The mutation is atomic and follows the existing one-mutation/one-commit
contract. It validates syntax and source eligibility; it cannot prove that the
human selected every semantically relevant source. Its output separates direct
from inherited context so `--no-commitments` cannot be mistaken for erasing an
epic's commitments from a milestone.

Ordinary normative documents continue to be edited with normal repository
tools. aiwf owns their marker and reference validation, not their content or a
new document lifecycle command set.

### 5. Projection and discovery surfaces

`aiwf show <subject>` always includes a concise `current_context` projection:

- source reference;
- current lifecycle status where applicable;
- direct or inherited origin; and
- stale/ineligible labeling.

It does not inline source bodies or generate a semantic summary.

`aiwf list --context-sources` returns the repository-wide eligible inventory in
two sections. Commitment rows carry kind, id, title, and status. Normative
document rows carry path, title, and H1-H3 headings so a reviewer can select
without reading the entire corpus.

There is no implicit global-source set. The inventory supports discovery; it is
not an applicability rule.

### 6. Lifecycle changes require explicit rebinding

When a referenced ADR or decision is superseded, a contract is retired or
rejected, or a normative document is archived, the old reference becomes
ineligible. aiwf does not silently substitute a successor.

Existing structured successor links may be displayed as assistance. Where no
such link exists, the projection says that no replacement is declared. The
operator explicitly chooses whether to bind the successor, choose another
source, or declare none.

An archived source never becomes current again in place. Re-adoption creates a
new live commitment or normative document, preserving the meaning of archive as
forgotten by default.

### 7. Document paths are identities

The path of a normative document is its unique identity. aiwf does not add a
document-id namespace or `move-doc` verb.

An unreferenced document can be moved with Git. A referenced move is an explicit
identity migration:

1. create the new live marked document while the old one remains;
2. rebind each referring subject with `aiwf set-context`;
3. confirm that the old path has zero referrers; and
4. archive the old document.

Validation names every referrer of a missing, unmarked, or archived target. A
specialized move verb is deferred unless this rare procedure produces repeated,
material friction.

## Validation behavior

The feature is enabled per repository:

```yaml
current_context:
  required: true
```

Once enabled, existing lifecycle gates enforce declarations. `aiwf check`
grades missing or stale declarations by the subject's operational relevance:

| Subject state | Finding |
|---|---|
| active epic | error |
| `in_progress` milestone | error |
| open gap currently targeted by a tracked patch | error |
| proposed/draft/dormant nonterminal subject | warning |
| terminal or archived subject | no current-context finding |

Terminal and archived entity frontmatter is historical evidence and remains
untouched. `aiwf show` may render it with its status, but ordinary validation
does not ask the operator to repair history.

Only mechanically decidable properties are enforced:

- required declarations are present at the lifecycle seam;
- ids and paths resolve;
- commitment statuses are eligible;
- normative targets carry `normative: true` and are not archival;
- child declarations do not duplicate inherited sources; and
- stored references are syntactically canonical.

Completeness and semantic applicability remain judgments. The framework makes
those judgments visible and reviewable rather than pretending to automate them.

## Brownfield adoption

Adoption does not rewrite repository history.

1. Establish an explicit adoption cutoff/baseline.
2. Mark the documents that are live normative truth now.
3. Declare context for the execution frontier only: active epics,
   `in_progress` milestones, and gaps targeted by current tracked patches.
4. Enable `current_context.required` after that frontier is coherent.
5. Require dormant legacy work to acquire context when it next moves toward an
   execution seam. A separate exhaustive audit of dormant work is optional.

The exact persisted shape of the cutoff remains an implementation-design
question. Its required semantics are already bounded: legacy absence must be
distinguishable from an explicit empty declaration, and adoption must not flood
a brownfield repository with findings for history nobody is executing.

Fresh repositories created by `aiwf init` enable the policy immediately. An
upgrade preserves the existing `aiwf.yaml` and treats a missing block as
compatibility/unconfigured, not failure. `aiwf update` reports the feature and
the size of the frontier to review; help, examples, and the shipped skill guide
the work; `aiwf doctor` keeps the unconfigured status visible. Ordinary
`aiwf check` does not nag an upgraded repository before the policy is enabled.

The migration performs no automatic semantic edits. A tool can count subjects,
enumerate eligible sources, and validate chosen references, but it cannot infer
which prose should bind a project.

## Relationship to independent preflight

The relationship to
[`milestone-preflight-as-independent-review.md`](milestone-preflight-as-independent-review.md)
is deliberately not fixed yet. The initiatives should be tested against each
other before deciding whether one is sequenced first, one consumes the other,
or a smaller shared seam is sufficient.

If preflight consumes current context, the current design requires an
anti-anchoring sequence:

1. Give the independent reviewer the subject and repository-wide context-source
   index, but withhold the author's current-context declarations.
2. The reviewer selects candidate sources and records a reason for each choice.
3. Freeze that selection.
4. Reveal the declared and inherited current context.
5. Read the union of independently selected and declared sources.
6. Report selected-but-unbound sources as possible missing edges and
   bound-but-unselected sources as items to inspect, never as automatic edits.

The full historical entity title/status index remains a separate discovery
surface. Archived plans can reveal prior mistakes, but they are evidence rather
than commitments and cannot re-enter current context implicitly.

`aiwf set-context` and preflight remain separate operations. A completed report
does not automatically mutate bindings, and a context mutation does not attest
that a review occurred. The preflight claim ledger remains a ledger of factual
claims and findings, not an applicability registry.

## YAGNI boundaries

The first implementation, if promoted, does **not** include:

- a concern, topic, capability, or applicability taxonomy;
- automatic relevance inference from paths, symbols, embeddings, or prose;
- automatic inclusion based on `area` (areas are uncommon and project-specific);
- section-level applicability metadata;
- global sources silently applied to every subject;
- stable ids or lifecycle verbs for ordinary documents;
- a document-move command;
- source-body copies, semantic summaries, or a generated context bundle;
- rewriting archived entities or historical commits;
- a new context event or append-only event log;
- successor inference beyond links already stored by an entity kind;
- preflight attestations coupled to context mutations; or
- a future gap for any omitted mechanism before repeated friction demonstrates
  the need.

These exclusions are part of the design, not a list of unfinished features.

## Performance constraints

The projection must remain cheaper than the reasoning it is intended to focus.

- Build the eligible source inventory once per command, not once per subject.
- Resolve direct references and inheritance from working-tree state; do not add
  a full-history revwalk to `show`, `list`, lifecycle verbs, or ordinary checks.
- Index metadata only. Do not read or emit full source bodies when listing or
  projecting context.
- Keep ordering deterministic so renders and tests are reproducible.
- Measure on repositories larger than this one before adding a cache. A cache
  is not justified until the uncached implementation exceeds an agreed budget.
- Treat the independent review's token and wall-clock cost separately from
  kernel command latency. The reviewer may read selected prose; the kernel
  should only identify and validate it.

The existing
[`check-performance-incremental-revwalk-cache.md`](check-performance-incremental-revwalk-cache.md)
initiative concerns full-history checks. Current context should not acquire that
cost class unless evidence forces it there.

## What this can and cannot solve

Current context reduces accidental reliance on obsolete or irrelevant history.
It does so by making selection explicit, bounded, inherited, and mechanically
validated.

It cannot guarantee that:

- a selected commitment is substantively correct;
- two eligible sources do not contradict each other;
- a normative document still matches the code;
- the author selected every relevant source; or
- an LLM interprets nuanced prose correctly.

Those limits are why independent review, executable measurement, and normal doc
maintenance still matter. The feature narrows where judgment occurs and exposes
its inputs; it does not remove judgment.

## Questions to settle through tandem iteration

| Question | Why it remains open |
|---|---|
| Is the full projection needed, or would a smaller source inventory plus preflight discipline address most observed failures? | The design is coherent, but the implementation surface is broad; this is the central YAGNI challenge. |
| Does preflight consume current context, follow it in sequence, or share only the source-index interface? | Either initiative can be evaluated independently, and choosing the relationship now would turn an untested assumption into architecture. |
| What latency and index-size budgets make the feature acceptable? | The metadata operations should be cheap, but no prototype has measured them across differently sized repositories. |
| How is the brownfield adoption cutoff represented durably? | The semantics are fixed; the smallest schema and migration mechanism are not. |
| Does blind candidate selection still add enough value once good declarations exist? | It protects against author omission and anchoring, but its marginal yield needs measurement. |
| Do warning-severity findings for dormant work remain useful rather than becoming ambient noise? | The lifecycle distinction is principled; brownfield trials must test its operational effect. |

## Promotion status

Nothing in this initiative has been promoted. No gap is filed for an aspirational
extension, and no ADR is accepted on the strength of this document. Promotion
should begin only after the initiative and independent-preflight design have
been reviewed against each other and the YAGNI/performance questions above have
been pressure-tested.

## Provenance

Captured from the 2026-08-15 design conversation about additive history,
forgetting, normative documents, explicit relevance, and independent preflight.
The conversation began from the risk that an LLM presented with all repository
history can mistake obsolete prose for current fact. It converged on a bounded
projection rather than deletion, taxonomy, or automatic semantic selection.
