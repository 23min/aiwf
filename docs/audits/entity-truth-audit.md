---
title: Entity truth audit — every open gap, live ADR, live decision, and normative doc measured against the kernel
status: captured
date: 2026-08-19
---

# Entity truth audit

A dated snapshot of measured divergence between what this repo's planning records
and Normative docs *claim*, and what the kernel at `e27f19eab` actually *does*.

Like [`normative-docs-drift-audit.md`](normative-docs-drift-audit.md), this is an
**inventory rather than a proposal**, and it ages by construction: every entry is
either fixed (and deleted) or still true. The `date:` is what makes it honest.
That prior audit covered the Normative doc tier only; this one covers the entity
tier as well, and re-derives the prior audit's entries rather than inheriting
them.

## Scope and method

276 subjects, complete — no sampling:

| Subject class | n |
|---|---|
| Open gaps | 160 |
| Live decisions (`accepted` + `proposed`) | 61 |
| Live ADRs (`accepted` + `proposed`) | 40 |
| Normative docs (`docs/design/*`, `architecture`, `overview`, `workflows`, `skill-author-guide`) + `CLAUDE.md` | 15 |

Archived entities are out of scope, per the forget-by-default archival convention.

Two passes:

1. **Mechanical citation resolution.** Every entity id, ADR id, source path,
   `file:line`, markdown link, `` `aiwf <verb>` `` mention, finding code, and Go
   identifier each subject cites, resolved against the working tree. ~1,700
   citations in the gap bodies alone.
2. **Semantic audit**, 49 independent reviewers over 54 clustered batches, each
   handed its subjects' pre-resolved references so its effort went to the part no
   script can do: whether the *claims* are still true.

Every reviewer worked read-only against a binary built from the tree under audit
(`go build -o … ./cmd/aiwf`), not the `aiwf` on PATH. The repo was not modified;
HEAD did not move.

**The evidence bar was the point.** Nothing was settled by reading. All 760
findings name the exact command that settles them and quote the output fragment.
203 further claims are recorded as **unverifiable** rather than asserted — mostly
end-to-end reproductions needing a mutating verb — each with the command that
would settle it. Reviewers were also asked to *clear* false positives from the
mechanical pass and say so; several did, which is why the reference classes below
are smaller than the raw mechanical counts.

## Headline

| | |
|---|---|
| Findings | **760** (135 high, all 760 command-backed) |
| Subjects with ≥1 finding | **185 / 276** (67%) |
| Subjects clean (`sound`) | **91 / 276** (33%) |
| Claims left unverifiable | 203, across 149 subjects |

**Every one of the 15 Normative docs carries claims measurement contradicts.**
Not one came back clean. They hold 56 of the 135 high-severity findings — so 5%
of the subjects carry 41% of the serious defects. A reader is entitled to treat
that tier as current truth, which makes it the worst place for this to be
concentrated and the highest-leverage place to fix.

The entity tier is healthier than the doc tier and healthier than the raw numbers
suggest: **almost every open gap is still a real gap.** Only 4 of 160 were found
already-addressed. What has rotted is the supporting scaffolding around a still-valid
subject — line numbers, counts, cited symbols, and premises that a later decision
overtook.

## Cross-cutting patterns

These are the systemic causes. Each generated findings across many unrelated
subjects, which is what makes them worth fixing at the source rather than
one record at a time.

### 1. A retired verb keeps being cited as live (133 mentions)

ADR-0039 retired `aiwf rewidth` on 2026-08-03. It is still cited in the present
tense across gaps, decisions, ADRs, and docs — including in **ADR-0029, ADR-0036,
ADR-0037, ADR-0038, ADR-0031, D-0042, G-0169, G-0229, G-0400, G-0456**, several
of which were edited *after* the retirement.

The mechanical reason is precise and worth recording: the retirement-surface
policy `internal/policies/m0290_retirement_surface_test.go` deliberately scopes
`docs/adr/` **out** of its scan. So the one check that exists for exactly this
class cannot see the tier where most of the rot landed.

Sharpest instance: **ADR-0008 now contains the tautology *"The next epic after
E-0022 is E-0023, not E-0023."*** `git log -S` shows it read *"after E-22 is
E-0023, not E-23"* until commit `f93728893` — whose subject is
`aiwf rewidth: 200 rename(s), 212 body rewrite(s)`. The ADR's own migration verb
widened its illustrative narrow ids and destroyed the contrast the sentence was
making.

### 2. ADRs reverse each other with no supersession pointer either way (3 pairs)

- **ADR-0014 §2** ("the upstream repo stays the authoring home") is reversed by
  **ADR-0016**. Neither carries `supersedes`/`superseded_by`; ADR-0014 has zero
  occurrences of "ADR-0016".
- **ADR-0014 §3** ("hooks are not part of the rituals; this decision introduces
  no new hook surface") is reversed by **ADR-0032**, which never mentions
  ADR-0014.
- **ADR-0027** ("never edits the user's live config file for any purpose") is
  broken by **ADR-0032**, accepted one day later, which cites ADR-0027
  approvingly without noting the narrowing. `aiwf update` writes the consumer's
  `aiwf.yaml` on every run with a non-empty hook registry.

`adr-supersession-mutual` exists as a check code but only polices *declared*
supersession. Reversal-in-fact has no oracle. **ADR-0007 is the model**: a
top-of-file See-also banner naming ADR-0014, saying which assumption was retired
and which layering survives.

### 3. Decisions and gaps are overtaken by later work and never revisited

The largest single class (44 `dead-premise` + 24 `already-shipped` + 8
`premise-decay`). A record is written, the world moves, nothing walks back to it:

- **D-0028** records `priority` as applying to "gap and milestone only". The
  kernel is `gap || decision` — milestone excluded, decision included. The
  reversal landed 12 days after D-0028 was accepted, inside another entity's
  body. The only `contradicted-by-code` verdict in the audit.
- **D-0003 / D-0004** both send a reader to file a gap for work that shipped
  (G-0139 addressed, M-0139 done, guards live at `internal/verb/cancel.go:86`).
  Acting on either files a duplicate.
- **G-0282**'s blocker ("the tdd toggle verb does not exist") was shipped three
  days *before* the sentence was written, and its proposed extension is exactly
  what accepted D-0048 rejects.
- **ADR-0041** narrowed **ADR-0030**, **D-0036**, **G-0416** and **G-0536** by
  inserting a `cross-branch-local-only` arm at *error* severity ahead of the
  warning-severity collision arm. None of the four records it narrowed says so.
- **21 findings** trace to the withdrawn `wf-measure-spec` skill and cancelled
  E-0085 / M-0308 — records still cite them as live.

### 4. Counts drift and nothing re-derives them (62 findings)

Exactly the failure the shipped guidance names ("keep the reasoning; derive the
facts"). Representative: G-0400's *title* says "10 of 38 aiwf verbs" — measured
16 of 40. G-0396's blast radius "roughly fifty" — measured **303**, and it was
already 204 when written. G-0254's "474 of 4355 (~11%)" — measured 694 of 9980
(~7%). ADR-0019's "five highest-leverage forces" — the shipped guidance carries
eight, and a policy test asserts all eight.

**`docs/design/growth.md` is the counter-example worth copying**: its
Measured-baseline table reproduces *to the digit* under
`scripts/growth-report.py --at <rev>`, thirteen days on. Its defects are confined
to the figures that name no command — which is the argument for the rule, made by
its own document.

### 5. Worked examples in Normative docs no longer run

`docs/workflows.md` is the worst: sections 1–3 need rebuilding, not patching. A
reviewer ran the examples in a disposable repo and found **six** preconditions the
doc omits entirely — the born-complete body gate refuses bare
`aiwf add adr|gap|decision`; every `aiwf add milestone` omits the now-required
`--tdd`; `draft → in_progress` needs ≥1 AC *with body prose*; the epic-activate
and milestone-start promotes shown back-to-back cannot run from the same branch;
`aiwf cancel <decision>` fails against the very entity the doc creates.

`docs/skill-author-guide.md`'s single worked example is unsalvageable as written
— and worse, it teaches consumers to name skills `aiwfx-*`, which
`internal/skills/skills.go:668` writes into every consumer's `.gitignore`. A
consumer following the Normative guide authors a skill that is never committed.

### 6. Records sit at `proposed` while already load-bearing

**D-0053, D-0056, D-0057** (and arguably D-0023, whose body still says "stays
`proposed`" while the entity is `accepted`). Each describes behavior that has
shipped and is cited by other accepted records. CLAUDE.md's own rule is
"Decision is decision" — either committed or not.

## What has no oracle

The audit is itself evidence for the preflight initiative's central claim: the
defects above are overwhelmingly *semantic*. They parse, they link, they pass
`aiwf check` — the tree reports **0 errors**. Specifically unpoliced:

- a claim about the codebase that measurement contradicts (the 150 `false-claim`
  findings);
- a count that has drifted (62);
- an ADR reversed in fact but not in status;
- a `file:line` citation whose line moved (`docs/design/legal-workflows-audit.md`
  pins all 49 FSM rules to lines that are off by 5–130, and names 23 functions
  under a `run<X>` convention that has never existed in `internal/check`);
- a worked example that no longer runs.

Two mechanical gaps are cheap and would have caught real findings here:

1. **Extend the retirement-surface scan to `docs/adr/`.** One scope change; would
   have caught ~10 subjects citing `rewidth`.
2. **A `file:line` citation checker.** Grep-able, and 61 `broken-reference`
   findings are in its range.

## Disposition summary

| verdict | n |
|---|---|
| `stale-claims` — subject valid, body carries contradicted claims | 173 |
| `sound` — nothing found | 91 |
| `already-addressed` — defect gone, gap never promoted | 4 |
| `unimplemented` — accepted, nothing built | 3 |
| `needs-human-decision` | 3 |
| `duplicate` | 1 |
| `contradicted-by-code` | 1 |

### Ready to act on

**Promote — defect verified gone, closing commit identified:**

- `G-0276` → addressed, `--by-commit 4718d496e` (git-stash isolation removed; AST-pinned against return)
- `G-0432` → addressed, `--by-commit f55923fe3` (filed 33 days *after* the fix landed)
- `G-0452` → addressed, `--by-commit aef85f87c` (both deliverables shipped; a policy test pins "want exactly 4 lenses")
- `G-0305` → addressed (both Direction items shipped; one divergence noted in the ledger)

**Human decision required:**

- **`ADR-0003`** is `accepted` on `main` and `rejected` on `initiative/preflight-amendments`
  — the same record, two answers depending on which ref you read. The rejection
  is a `--force` promote (the ADR FSM has no `accepted → rejected` edge).
  Reconciling `main` needs `--force` too. On `main` it is an accepted decision
  mandating a seventh entity kind against a kernel that hardcodes six.
- **`D-0028`** — accepted and inverted by the kernel.
- **`D-0042`** — subject (`rewidth`) retired; every present-tense sentence
  describes an unreachable surface.
- **`D-0053`, `D-0056`, `D-0057`** — `proposed` while load-bearing.
- **`G-0564`** — subsumed by G-0121, and rests on cancelled E-0080.
- **`docs/design/legal-workflows-audit.md`** and
  **`legal-workflows-first-principles.md`** — both are Pass-A/Pass-B artifacts
  whose header still says `in_progress` against done-and-archived milestones.
  The real question is tier, not line-editing: re-tier as historical snapshots
  (beside the existing `docs/archive/pocv3/legal-workflows-audit-r1.md`), or keep
  them Normative and pay ~50 line citations, 23 function names, 6 counts and ~10
  substantive claims. Note the trap: `TestM0122_AC5_OpenQuestionsSectionPresent`
  *requires* the stale "Open questions for Pass C" heading, so deleting it fails CI.

---

## Findings ledger

Generated from the 49 per-batch reviewer ledgers. Every finding below names
the command that settles it.



### Verdicts

| verdict | n |
|---|---|
| `contradicted-by-code` | 1 |
| `unimplemented` | 3 |
| `already-addressed` | 4 |
| `duplicate` | 1 |
| `stale-claims` | 173 |
| `needs-human-decision` | 3 |
| `sound` | 91 |

### Findings by class

760 findings · 135 high · 338 medium · 287 low · 0 carrying no command · 203 claims left unverifiable

| class | n |
|---|---|
| `false-claim` | 150 |
| `stale-count` | 62 |
| `broken-reference` | 61 |
| `dead-premise` | 44 |
| `stale-claim` | 29 |
| `already-shipped` | 24 |
| `stale-enumeration` | 20 |
| `drafting-history` | 13 |
| `imprecise-claim` | 10 |
| `premise-decay` | 8 |
| `contradicted-by-code` | 7 |
| `incomplete-enumeration` | 7 |
| `stale-measurement` | 6 |
| `contradicts-adr` | 6 |
| `stale-reference` | 6 |
| `narrow-width-ref` | 6 |
| `self-contradiction` | 6 |
| `internal-contradiction` | 5 |
| `phantom-contract` | 5 |
| `example-fails` | 5 |
| `narrow-width-id` | 5 |
| `contradicts-normative-doc` | 4 |
| `missing-cross-reference` | 4 |
| `dead-reference` | 4 |
| `stale-tense` | 4 |
| `proposed-but-settled` | 3 |
| `incomplete-scope` | 3 |
| `phantom-reference` | 3 |
| `already-removed` | 3 |
| `incomplete-claim` | 3 |
| `fabricated-id` | 3 |
| `narrowed-in-fact` | 3 |
| `unmaterialized-consequence` | 3 |
| `dead-cross-reference` | 3 |
| `scanner-false-positive` | 3 |
| `undated-measurement` | 2 |
| `stale-figure` | 2 |
| `understated-scope` | 2 |
| `overstated-precedent` | 2 |
| `spec-cell-mismatch` | 2 |
| `vague-reference` | 2 |
| `overstated-premise` | 2 |
| `phantom-flag` | 2 |
| `internal-incoherence` | 2 |
| `stale-line-reference` | 2 |
| `consequence-unmaterialized` | 2 |
| `contradicts-code` | 2 |
| `overstated-claim` | 2 |
| `stale-framing` | 2 |
| `consequence-did-not-materialize` | 2 |
| `stale-example-output` | 2 |
| `stale-context` | 2 |
| `narrow-id-width` | 2 |
| `derived-fact-duplicated` | 2 |
| `stale-status` | 2 |
| `stale-cli-invocation` | 2 |
| `phantom-verb` | 2 |
| `unimplemented` | 2 |
| `missing-supersession-pointer` | 2 |
| `superseded-clause` | 2 |
| `ambiguous-claim` | 1 |
| `tense-overclaim` | 1 |
| `broken-attribution` | 1 |
| `collateral-evidence` | 1 |
| `unlanded-followup` | 1 |
| `false-premise` | 1 |
| `wrong-prescription` | 1 |
| `example-not-locatable` | 1 |
| `inexact-quote` | 1 |
| `triage-inconsistency` | 1 |
| `understated-cost` | 1 |
| `unnamed-derivation` | 1 |
| `overreach` | 1 |
| `irreproducible-citation` | 1 |
| `status-contradiction` | 1 |
| `authoring-rule-breach` | 1 |
| `interpretive-tension` | 1 |
| `template-deviation` | 1 |
| `accepted-not-implemented` | 1 |
| `misattributed-quote` | 1 |
| `unresolved-thread` | 1 |
| `missing-context` | 1 |
| `spent-consequence` | 1 |
| `cross-entity-misattribution` | 1 |
| `dated-measurement-drift` | 1 |
| `missing-required-section` | 1 |
| `confirmed-still-real` | 1 |
| `inaccurate-enumeration` | 1 |
| `already-settled` | 1 |
| `duplicate` | 1 |
| `stale-status-header` | 1 |
| `retired-finding-code` | 1 |
| `contradicts-code-comment` | 1 |
| `authority-tier-mismatch` | 1 |
| `id-shape` | 1 |
| `missing-commitment` | 1 |
| `missing-subsystem` | 1 |
| `fabricated-example` | 1 |
| `premise-crossed` | 1 |
| `narrowed-claim` | 1 |
| `stale-proposal` | 1 |
| `narrow-width-ids` | 1 |
| `stale-symbol` | 1 |
| `phantom-identifier` | 1 |
| `prospective-tense` | 1 |
| `missing-followthrough` | 1 |
| `stale-quotation` | 1 |
| `stale-verb-name` | 1 |
| `scope-drift` | 1 |
| `provenance-in-prose` | 1 |
| `settled-open-question` | 1 |
| `phantom-package` | 1 |
| `partially-implemented-decision` | 1 |
| `false-detail` | 1 |
| `stale-precondition` | 1 |
| `unreconciled-with-decision` | 1 |
| `stale-dependency` | 1 |
| `stale-premise` | 1 |
| `misquote` | 1 |
| `stale-file-line` | 1 |
| `partial-implementation` | 1 |
| `no-mechanical-pin` | 1 |
| `weak-evidence` | 1 |
| `stale-code-snippet` | 1 |
| `misattributed-reference` | 1 |
| `stale-tracked-artifact` | 1 |
| `retired-verb-reference` | 1 |
| `unresolvable-reference` | 1 |
| `dead-forward-reference` | 1 |
| `incomplete-decision-statement` | 1 |
| `historical-context-in-present-tense` | 1 |
| `stale-cross-reference` | 1 |
| `misleading-tool-output` | 1 |
| `unrecorded-consequence` | 1 |
| `contradicts-normative-source` | 1 |
| `orphan-capture` | 1 |
| `unsupported-count` | 1 |
| `narrowed-by-later-adr` | 1 |
| `incomplete-consequence` | 1 |
| `proposed-but-shipped` | 1 |
| `ambiguous-example-id` | 1 |
| `hedged-count-drift` | 1 |
| `incomplete-direction` | 1 |
| `missing-precondition` | 1 |
| `typo` | 1 |
| `internal-inconsistency` | 1 |
| `unverified-claim` | 1 |
| `unmaterialized-trigger` | 1 |
| `stale-comment` | 1 |
| `naming-trap` | 1 |
| `stale-naming` | 1 |
| `unmaterialized-followup` | 1 |
| `missing-pointer` | 1 |
| `under-specified-fix` | 1 |
| `arithmetic` | 1 |
| `coherence` | 1 |
| `duplicated-state` | 1 |
| `false-example` | 1 |
| `claim-understates-defect` | 1 |
| `imprecise-reference` | 1 |
| `stale-anecdote` | 1 |
| `nonstandard-body-shape` | 1 |
| `unnamed-residual` | 1 |
| `contradicts-decision` | 1 |
| `under-scoped` | 1 |
| `one-directional-link` | 1 |
| `history-surface-artifact` | 1 |
| `cross-entity-status-mismatch` | 1 |
| `incomplete-inventory` | 1 |
| `phantom-finding-code` | 1 |
| `contradicts-accepted-decision` | 1 |
| `broken-internal-reference` | 1 |
| `stale-worked-example` | 1 |
| `stale-render-description` | 1 |
| `unlinked-archival-reference` | 1 |
| `contradicted-by-live-spec` | 1 |
| `stale-id-format` | 1 |
| `stale-model` | 1 |
| `cross-ref-divergence` | 1 |
| `accepted-but-unbuilt` | 1 |
| `contradicts-normative-docs` | 1 |
| `proposed-not-extant` | 1 |
| `unimplemented-clause` | 1 |
| `stale-scope-statement` | 1 |
| `confusing-rejected-alternative` | 1 |
| `scope-understated` | 1 |
| `live-instance` | 1 |
| `misattributed-convention` | 1 |
| `prediction-materialized` | 1 |
| `fabricated-id-shape` | 1 |
| `stale-comment-in-shipped-surface` | 1 |
| `contradicted-by-later-adr` | 1 |
| `broken-pointer` | 1 |
| `settled-question-still-framed-open` | 1 |
| `stale-follow-up` | 1 |
| `contradicts-own-body` | 1 |
| `premise-narrowed-elsewhere` | 1 |
| `answered-question` | 1 |
| `dead-recommendation` | 1 |
| `incomplete-premise` | 1 |
| `superseded-in-fact` | 1 |
| `unrealized-consequence` | 1 |
| `expired-window` | 1 |
| `authoring-shape` | 1 |
| `import-artifact` | 1 |
| `misquote-normative` | 1 |
| `attribution` | 1 |
| `unstated-bound` | 1 |
| `incomplete-survey` | 1 |
| `misattributed-record` | 1 |
| `over-claim` | 1 |
| `unimplemented-decision` | 1 |
| `phantom-consequence` | 1 |
| `unnamed-reference` | 1 |
| `missed-seam` | 1 |
| `reserved-namespace` | 1 |
| `example-fails-as-written` | 1 |
| `missing-verbs` | 1 |
| `incomplete-escape-hatch` | 1 |
| `design-divergence` | 1 |
| `internal-coherence` | 1 |
| `partial-pointer` | 1 |
| `phantom-tracker` | 1 |
| `consequence-not-materialized` | 1 |
| `unmet-precondition` | 1 |
| `circular-attribution` | 1 |
| `stale-followups` | 1 |
| `wrong-diagnosis` | 1 |
| `imprecise-scope` | 1 |
| `stale-line-ref` | 1 |
| `incomplete-supersession-banner` | 1 |
| `stale-code-name` | 1 |
| `already-partly-shipped` | 1 |
| `overlap` | 1 |
| `residual-thread` | 1 |
| `gap-still-live` | 1 |
| `ambiguous-reference` | 1 |
| `imprecise-attribution` | 1 |
| `unsupported-estimate` | 1 |

### High-severity findings

#### ADR-0003 · `cross-ref-divergence`

- **Claim:** ADR-0003 is accepted.
- **Measured:** True on main and origin/main; false on initiative/preflight-amendments and origin/initiative/preflight-amendments, where it is `rejected` with a 40-line rejection preamble. Two branches cut from the initiative (epic/E-0086, milestone/M-0311) carry `rejected`; two others (epic/E-0081, milestone/M-0300) carry `accepted`. The rejection needed --force because the ADR FSM has no accepted→rejected edge.
- **Command:** `for r in $(git for-each-ref --format='%(refname)' refs/heads refs/remotes); do s=$(git show "$r:docs/adr/ADR-0003-add-finding-f-nnn-as-a-seventh-entity-kind.md" 2>/dev/null | sed -n 's/^status: //p' | head -1); [ -n "$s" ] && echo "$r -> $s"; done ; git log -1 --format='%H%n%s%n%b' initiative/preflight-amendments -- docs/adr/ADR-0003-add-finding-f-nnn-as-a-seventh-entity-kind.md ; sed -n '30,35p' internal/entity/transition.go`
- **Quoted:** “status: accepted”

#### ADR-0003 · `accepted-but-unbuilt`

- **Claim:** A seventh kind with its own storage tree, FSM, allocator entry and AC-closure chokepoint.
- **Measured:** None of it exists: AllKinds() returns six; there is no KindFinding, no F-NNNN id pattern, no work/findings/ directory, and no findings-block-met or ac-has-open-findings code. The implementing epic E-0019 has been `proposed` with zero milestones since 2026-05-07 (~3 months).
- **Command:** `sed -n '24,36p' internal/entity/entity.go ; sed -n '285,292p' internal/entity/entity.go ; ls work/ ; grep -rn 'findings-block-met\|ac-has-open-findings' internal/ cmd/ ; <binary> show E-0019 ; ls work/epics/E-0019-*/`
- **Quoted:** “Add **`finding`** as a seventh entity kind. ... Stored at `work/findings/F-NNNN-<slug>.md`. ... `aiwf promote M-NNN/AC-N met` reads the finding tree, refuses with `findings-block-met` if any `open` finding has the AC in its `linked_acs`.”

#### ADR-0003 · `contradicts-normative-docs`

- **Claim:** Kernel principle #1 is amended by this ADR to seven kinds.
- **Measured:** It is not. All four Normative-tier surfaces on the same ref still say six, and the code agrees with them.
- **Command:** `grep -n '[Ss]ix entity kinds\|[Ss]even entity kinds' CLAUDE.md docs/design/design-decisions.md docs/overview.md docs/architecture.md`
- **Quoted:** “### Principle #1 amendment  > **Six entity kinds** → **Seven entity kinds** — epic, milestone, ADR, gap, decision, contract, **finding** ...”

#### ADR-0003 · `premise-decay`

- **Claim:** A corpus of 66 gaps shows the volume cost of a high-volume governance kind is acceptable.
- **Measured:** 591 gaps (399 addressed, 163 open, 29 wontfix) and 1100 entities total — roughly ninefold growth. The ADR's own projection ('at least as high-volume as gaps') therefore now means at least 591 further entities, near-doubling the corpus. The count is stale and the argument it carries inverts at the measured scale; this is the exact measurement the branch's rejection cites.
- **Command:** `<binary> list --archived --format=json | python3 -c "import sys,json,collections; d=json.load(sys.stdin); rows=d['result']; print('total entities:',len(rows)); print('by kind:',collections.Counter(r['kind'] for r in rows)); g=[r for r in rows if r['kind']=='gap']; print('gaps:',len(g),collections.Counter(r['status'] for r in g))"`
- **Quoted:** “The branch's existing 66 gaps (52 addressed) demonstrate that high-volume governance kinds are realistic; findings will be at least as high-volume as gaps once cycle-time emission turns on.”

#### ADR-0006 · `false-claim`

- **Claim:** aiwf-show is absent from the embedded skill set and its absence is tracked by an open gap.
- **Measured:** The skill exists — internal/skills/embedded/aiwf-show/SKILL.md, 8766 bytes — and G-0087 ('No aiwf-show embedded skill') was promoted to `addressed` on 2026-05-20 and archived. This is the ADR's only worked example of its fourth disposition ('deferred with a tracked follow-up'), so a reader taking the ADR as current truth is reasoning from a precedent that has since resolved.
- **Command:** `ls -la internal/skills/embedded/aiwf-show/SKILL.md ; /tmp/.../aiwf show G-0087`
- **Quoted:** “**`aiwf-show`** is deliberately absent and tracked by G-0087: `--help` covers the surface mechanically, but body-rendering branches and composite-id discovery probably warrant a skill. The right answer isn't "papered over with --help" or "shipped as a stub"; it's "deferred with a tracked follow-up."”

#### ADR-0010 · `dead-premise`

- **Claim:** Ritual fixes are authored in a separate rituals repo and mirrored via fixtures under internal/policies/testdata/.
- **Measured:** internal/policies/testdata does not exist, and the upstream 23min/ai-workflow-rituals repo is archived (ADR-0016, accepted) with the embedded snapshot at internal/skills/embedded-rituals/ as the single source of truth (ADR-0014, accepted). Same dead-authoring-path as G-0111 carries. The underlying G-0116 fix did land — aiwfx-start-epic now sequences promote/authorize on main before the branch cut — but the how-to is wrong.
- **Command:** `ls -d internal/policies/testdata ; grep -n '^status:' docs/adr/ADR-0014*.md docs/adr/ADR-0016*.md ; rg -n 'Sovereign acts land on `main` before the branch cut' internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-epic/SKILL.md`
- **Quoted:** “**G-0116 fix** — reorder `aiwfx-start-epic` steps so promote + authorize fire on main before the worktree/branch cut. `wf-patch` in the rituals repo, mirrored via the fixture pattern in `internal/policies/testdata/`.”

#### ADR-0011 · `false-claim`

- **Claim:** Every top-level Cobra verb must be referenced by a spec rule, and CI blocks a new verb that does not grow the spec.
- **Measured:** The shipped drift test accepts an allowlist entry instead. `nonLegalityVerbAllowlist` carries 31 top-level verbs (add, rename, retitle, edit-body, move, reallocate, set-area, set-priority, archive, list, show, status, history, worktree, acknowledge, contract, milestone, …) — i.e. nearly the whole verb surface — each with a one-line rationale and no spec cell. A contributor adding a verb satisfies CI with a one-line map entry.
- **Command:** `sed -n '105,150p' internal/policies/m0123_ac5_drift_test.go`
- **Quoted:** “Every top-level Cobra verb is referenced by at least one rule. ... Failure of any of these is a hard CI block. The impl cannot grow a new verb, kind, state, or workflow-legality finding without the spec growing in the same PR.”

#### ADR-0012 · `false-claim`

- **Claim:** Kernel legality codes are declared as bare string constants (`const Code... = "..."`), and the typed errors' Code() methods return those constants directly.
- **Measured:** Both are package-level `var`s of struct type `codes.Code` carrying an ID and a Class (the D-0011 typed descriptor), and Code() returns the descriptor's `.ID` field, not the descriptor. A reader following ADR-0012 to add a new legality code would declare a bare string const, which the AC-5 scanner's GenDecl arm classifies as codes.ClassStructural — so the code would silently escape the fourth arm's 'every ClassLegality code must be named by an illegal spec Rule' obligation.
- **Command:** `rg -n "CodeFSMTransitionIllegal|CodeAuthorizeKindNotAllowed" internal/entity/transition.go internal/verb/authorize.go`
- **Quoted:** “Each code value lives in exactly one named `const Code... = "..."` constant, never as a scattered string literal repeated at call sites. `FSMTransitionError.Code()` returns `CodeFSMTransitionIllegal`; `AuthorizeKindError.Code()` returns `CodeAuthorizeKindNotAllowed`.”

#### ADR-0014 · `superseded-in-fact`

- **Claim:** Ritual content is authored in the upstream 23min/ai-workflow-rituals repo and vendored into aiwf as a pinned snapshot.
- **Measured:** Reversed by ADR-0016 (status: accepted): 'Retire https://github.com/23min/ai-workflow-rituals as an authoring channel. Make internal/skills/embedded-rituals/ THE canonical authoring location for ritual content.' ADR-0014 carries no `superseded_by` and no reference to ADR-0016; ADR-0016 carries no `supersedes`. A reader arriving at ADR-0014 alone is told to author upstream, which is now an archived read-only repo.
- **Command:** `sed -n '/^## Decision/,/^## Consequences/p' docs/adr/ADR-0016-retire-ai-workflow-rituals-upstream-channel-embedded-snapshot-canonical.md ; sed -n '1,6p' docs/adr/ADR-0016*.md ; sed -n '1,5p' docs/adr/ADR-0014*.md`
- **Quoted:** “The `23min/ai-workflow-rituals` repo stays the **authoring home**, preserving the `wf-*` skills' standalone reusability and the clean coupling boundary ADR-0007 established. aiwf vendors a **pinned snapshot** of the rituals into a path inside the aiwf repo (e.g. `rituals/`), which `go:embed` then ba”

#### ADR-0015 · `consequence-unmaterialized`

- **Claim:** A consent-driven settings edit produces a trailered git commit that aiwf history can resolve. The Decision section leans on the same idea: 'the commit record sees the flag'.
- **Measured:** No commit is produced. The package that performs init/update scaffolding states it never commits, and update.go contains no commit call. Separately, at project scope the file written is .claude/settings.local.json, which is gitignored — so even a commit could not record the edit. Only the .bak file, named in the bullet above it, actually provides the revert path.
- **Command:** `grep -rn 'CommitVerbChange\|gitops.Commit' --include='*.go' internal/cli/initcmd/ internal/cli/update/ internal/initrepo/; sed -n '1,8p' internal/initrepo/initrepo.go; git check-ignore -v .claude/settings.local.json`
- **Quoted:** “- **Audit trail.** Every consent-driven settings edit lands as a trailered commit; `aiwf history` resolves the edit through its `aiwf-verb` / `aiwf-actor` trailers.”

#### ADR-0016 · `internal-contradiction`

- **Claim:** The decision is not yet ratified; ratification is pending two conditions.
- **Measured:** Frontmatter is `status: accepted`, and the promote commit landed on 2026-05-31 at 19:15Z, about an hour after the edit-body that wrote this line. The decision has been fully implemented since. The body is therefore a self-contradicting record, and it is also a direct breach of CLAUDE.md §'Authoring an ADR': 'Don't write gate language into ADR bodies (*"ratify after X"*, *"status stays proposed through Y"*). … The FSM … and `aiwf promote` are the only surfaces that constrain ADR status.' A reader who trusts the body over the frontmatter concludes the embedded snapshot is not yet canonical — the opposite of what CLAUDE.md §'Ritual content authoring' tells them.
- **Command:** `/tmp/claude-1000/-workspaces-aiwf/ec5d1c3d-5669-4772-9561-4542e8aaef62/scratchpad/audit/aiwf history ADR-0016 ; sed -n '4p;65p' docs/adr/ADR-0016-retire-ai-workflow-rituals-upstream-channel-embedded-snapshot-canonical.md`
- **Quoted:** “**Status: proposed.** Ratification waits on the implementing gap producing a credible punch list and the operator confirming the GH-repo-archive step is acceptable.”

#### ADR-0027 · `contradicted-by-code`

- **Claim:** No aiwf verb, and specifically no later init/update run, ever writes the consumer's aiwf.yaml after it is created.
- **Measured:** `aiwf update` writes the live aiwf.yaml on every run whose shipped hook registry is non-empty — which it is (ShippedHooks carries worktree-rituals-check.sh). gateAndSyncHookDecisions reads the doc, unions the hook decisions and calls doc.Write(configPath) unconditionally. This is ADR-0032 (accepted 2026-07-06, one day after ADR-0027), which puts a hooks: map inside the user's aiwf.yaml by design. Separately, `aiwf rename-area` (landed 2026-06-25, i.e. before ADR-0027 was written) edits the live file via aiwfyaml.Doc.RenameAreaMember.
- **Command:** `grep -n 'doc.Write(configPath)' internal/cli/update/hooks.go ; sed -n '165,172p' internal/cli/update/update.go ; grep -n 'RenameAreaMember' internal/verb/renamearea.go ; grep -n -A3 '^hooks:' aiwf.yaml`
- **Quoted:** “The consumer's own `aiwf.yaml` is created once, on a fresh repo, as a fully-commented scaffold — and is never rewritten by any later `init`/`update` run.  This extends ADR-0015's posture: the tool never edits the user's live config file after creation, for any purpose — not even a well-intentioned, ”

#### ADR-0029 · `dead-reference`

- **Claim:** Rewidth is an extant verb that opts out of the projection gate — repeated in the Decision ('The known exceptions — … and `Rewidth` …') and in a Consequence ('If `SetArea`, `RenameArea`, or `Rewidth` are ever extended to touch a shape-relevant field')
- **Measured:** no `Rewidth` identifier exists in internal/ or cmd/; internal/verb/rewidth.go and internal/cli/rewidth/ are deleted. ADR-0039 retired the verb via M-0290 (commit db307fdc7, 2026-08-03). ADR-0039 enumerates lapsed clauses of ADR-0008 only — nothing marks ADR-0029, and ADR-0029's last edit (78fd5c769, 2026-08-05) postdates the retirement without touching the Rewidth mentions.
- **Command:** `grep -rn 'padToCanonical\|Rewidth\|rewidth' --include='*.go' internal/ cmd/ | grep -v _test; ls internal/verb/rewidth.go internal/cli/rewidth; git log -1 --format='%h %ad %s' --date=short 78fd5c769 db307fdc7`
- **Quoted:** “`Rewidth` opts out explicitly and documents relying on `aiwf check` at the pre-push boundary as its backstop instead.”

#### ADR-0029 · `stale-enumeration`

- **Claim:** exactly three verbs skip projectionFindings, and the two surviving ones are safe because they touch no shape-relevant field
- **Measured:** the reviewed allowlist carries 14 entries: SetArea, SetPriority, RenameArea, Authorize, AcknowledgeIllegal, AcknowledgeMistag, PromoteAuditOnly, PromoteACPhaseAuditOnly, CancelAuditOnly, Archive, ContractBind, ContractUnbind, RecipeInstall, RecipeRemove. It was widened from three by commit efed1374b (2026-07-18, G-0422) — twelve days after ADR-0029 was accepted — and the ADR was never updated. SetPriority is the sharp case: it serializes the mutated entity and emits an OpWrite, i.e. it writes entity content and skips the gate, which is precisely the pattern ADR-0029's Consequences say must carry an argument.
- **Command:** `sed -n '19,38p' internal/policies/projection_findings_presence.go; sed -n '95,125p' internal/verb/setpriority.go; git log --format='%h %ad %s' --date=short -S'"setpriority.go", "SetPriority"' -- internal/policies/projection_findings_presence.go`
- **Quoted:** “The known exceptions — `SetArea`/`RenameArea` (skip the gate; safe only because they never touch a shape-relevant field) and `Rewidth` (opts out explicitly, relies on pre-push) — are accepted as-is.”

#### ADR-0030 · `contradicted-by-later-adr`

- **Claim:** Any cross-branch hit — local branch or remote-tracking ref — is non-blocking.
- **Measured:** Since ADR-0041 (also `accepted`), a hit set confined to local branch refs fires `cross-branch-local-only` at **error** severity, which blocks the pre-push hook. Only a remote-tracking hit still fires the non-blocking `cross-branch-pending`; divergent content across refs fires a third subcode, `cross-branch-collision`. A reader treating ADR-0030 as current truth would expect a reference to an id on an unpushed epic branch to push cleanly; it does not.
- **Command:** `sed -n '640,700p' internal/check/check.go ; grep -rn "cross-branch-local-only" internal/check/ | grep -v _test`
- **Quoted:** “If the id is found there — known to exist on some other local branch or remote-tracking ref — fire a distinct, non-blocking subcode, `cross-branch-pending`, instead.”

#### ADR-0033 · `contradicted-by-code`

- **Claim:** No path-changing verb leaves a stale path-link behind.
- **Measured:** `aiwf move <M-id> --epic <E-id>` emits an OpMove to a different epic directory (dest is computed from the TARGET epic's directory) and never calls RewriteLinkDestinations or planLinkRewriteWrites. Only archive.go, rename.go, retitle.go and reallocate.go do. So a path-link to a moved milestone rots exactly the way the ADR says it no longer can, and the test suite has archive_/rename_/retitle_/reallocate_linkrewrite_test.go with no move counterpart.
- **Command:** `grep -n "planLinkRewriteWrites\|RewriteLinkDestinations" -r internal/verb/ --include="*.go" | grep -v _test ; grep -n "OpMove\|filepath.Dir(target.Path)" internal/verb/move.go ; ls internal/verb/ | grep -i linkrewrite`
- **Quoted:** “Every verb that changes an entity's on-disk path rewrites the markdown link destinations in entity bodies that point at it — relative or root-relative — through one shared link-region primitive generalized from `rewidth`'s machinery.”

#### CLAUDE.md · `broken-reference`

- **Claim:** The CLI completion-drift chokepoint lives at cmd/aiwf/completion_drift_test.go.
- **Measured:** cmd/aiwf/ contains exactly one file, main.go. The drift test is TestPolicy_FlagsHaveCompletion in internal/cli/integration/completion_drift_test.go; it moved there in 9893ee16e (M-0118/AC-6). No policy pins this path in CLAUDE.md, so nothing catches the drift.
- **Command:** `ls cmd/aiwf/ && git log --oneline --diff-filter=R --follow -- internal/cli/integration/completion_drift_test.go | head -5 && head -30 internal/cli/integration/completion_drift_test.go`
- **Quoted:** “Chokepoint: `cmd/aiwf/completion_drift_test.go`.”

#### CLAUDE.md · `false-claim`

- **Claim:** The statusline is the only path by which aiwf writes .claude/settings.json, and every such write is gated by consent evaluated fresh on that invocation.
- **Measured:** ADR-0032 (status: accepted) adds a second path — materialized Claude Code hooks — whose consent is 'per-hook granularity, decided once, persisted and shared' in aiwf.yaml's `hooks:` map, and which targets the SHARED .claude/settings.json (the statusline targets .claude/settings.local.json at project scope, per ADR-0015). On a clone where `hooks.<name>.enabled: true` is already committed, `aiwf init`/`update` wires the settings entry non-interactively with no per-invocation consent at all. This repo is in exactly that state.
- **Command:** `sed -n '40,70p' docs/adr/ADR-0032-materialized-hook-consent-persisted-per-hook-aiwf-yaml-registry.md && grep -n -A 5 '^hooks:' aiwf.yaml && grep -n 'hooks' internal/config/schema.go && grep -n 'settings.local.json' docs/adr/ADR-0015-settings-json-edits-require-explicit-per-invocation-consent.md`
- **Quoted:** “aiwf does **not** edit your `.claude/settings.json` without **explicit per-invocation consent**. The narrow exception is the statusline opt-in (`aiwf init/update --statusline`)”

#### D-0003 · `already-shipped`

- **Claim:** The enforcement is not built; a reader should file a gap and a milestone under E-0033 to build it.
- **Measured:** It is built. G-0139 ("Implement cancel refusal on non-terminal children/ACs per D-0003 and D-0004") is `addressed` and archived; M-0139 is `done` with AC-1..AC-4 all `met`; the guard runs in `Cancel` at internal/verb/cancel.go:86. A reader acting on this Follow-up files a duplicate gap.
- **Command:** `<bin> show G-0139 ; <bin> show M-0139 ; sed -n 70,115p internal/verb/cancel.go`
- **Quoted:** “Impl change scope-out of M-0123. File a gap → milestone under E-0033 for: precondition in `aiwf cancel` verb body, new finding code `epic-cancel-non-terminal-children` in `internal/check/`, integration test exercising the refuse path.”

#### D-0004 · `already-shipped`

- **Claim:** The enforcement is unbuilt and will land as a gap/milestone under E-0033, adding new finding codes in internal/check/.
- **Measured:** Shipped: G-0139 is `addressed`/archived and M-0139 is `done` — but under E-0036 ('Reconcile impl to the legal-workflow spec, retiring deferred error codes'), not E-0033. The code is `verb.CodeMilestoneCancelNonTerminalACs` (codes.ClassLegality) in internal/verb/cancel_guards.go:46; `internal/check/` contains no occurrence of `milestone-cancel-non-terminal-acs`.
- **Command:** `<bin> show M-0139 ; <bin> show E-0036 ; grep -rn "milestone-cancel-non-terminal-acs" internal/check/  # (empty) ; grep -rn "CodeMilestoneCancelNonTerminalACs\s*=" internal/verb/`
- **Quoted:** “Impl change scope-out of M-0123. Likely shares the same gap/milestone under E-0033 as D-0003's Q5 enforcement (both add precondition guards in the `cancel` verb body and new finding codes in `internal/check/`).”

#### D-0006 · `contradicts-normative-doc`

- **Claim:** Scope reachability traverses exactly three edges and no governance edge.
- **Measured:** True of the code — and contradicted by two Normative-tier docs a reader is entitled to treat as current truth. provenance-model.md still defines reachability as an arbitrary chain over the full reference grammar, naming `depends_on`, `addressed_by`, `relates_to` explicitly; design-decisions.md says reachability reuses the `referenced_by` index. The decision's own Follow-up asked for precisely this doc-update ('file a doc-update for `provenance-model.md` to enumerate the edge set explicitly') and it was never made. The implementation is the three-edge model, so the docs — not the decision — are wrong.
- **Command:** `sed -n '425,460p' internal/tree/tree.go ; sed -n '186,190p' docs/design/provenance-model.md ; sed -n '172p' docs/design/design-decisions.md`
- **Quoted:** “No other edges traverse. Specifically NOT reachability edges: `depends_on` … `addressed_by` … `relates_to`, `linked_adrs`, `supersedes`, `superseded_by` (all governance-layer; not scope-membership).”

#### D-0015 · `dead-premise`

- **Claim:** Shipped ritual skill bodies may not cite .claude/templates, because a drift guard would fail.
- **Measured:** False in both halves. The named test does not exist anywhere in the Go source (ADR-0016 retired it), and five embedded ritual files now do cite `.claude/templates` and pass CI. CLAUDE.md §'Ritual content authoring' states the opposite of this bullet outright: 'the embedded ritual snapshot … is the *authoring source of truth* — you hand-edit it.' Already tracked as open G-0579, whose diagnosis matches this measurement.
- **Command:** `grep -rn "TestRituals_VendoredMatchesUpstream" internal/ cmd/ ; grep -rln "\.claude/templates" internal/skills/embedded-rituals/`
- **Quoted:** “The embedded skill bodies are **not** rewritten to point at the new path — they are a drift-checked verbatim snapshot of upstream (M-0148's `TestRituals_VendoredMatchesUpstream`), so editing them would fail the drift guard.”

#### D-0016 · `already-removed`

- **Claim:** aiwf doctor emits a de-dupe guard on marketplace-plugin / materialized-ritual overlap.
- **Measured:** The overlap check was removed wholesale by G-0194. Only the ritual-verification half survives.
- **Command:** `git log --oneline -1 -S 'loadEnabledPlugins' main ; git log -1 --format='%B' debf7f5ff ; <binary> doctor`
- **Quoted:** “`aiwf doctor` instead **verifies the materialized ritual artifacts** under `.claude/` and emits a **de-dupe guard** when an enabled marketplace plugin overlaps with materialized rituals.”

#### D-0016 · `false-claim`

- **Claim:** The loadEnabledPlugins helper still exists in the codebase.
- **Measured:** It does not exist anywhere in internal/ or cmd/.
- **Command:** `grep -rn 'loadEnabledPlugins' internal/ cmd/`
- **Quoted:** “- `loadEnabledPlugins` is retained — the de-dupe guard reuses it.”

#### D-0018 · `dead-premise`

- **Claim:** M-0161/AC-9 will sweep the stale branch-not-found spec-table entries and the policy keyword mapping
- **Measured:** M-0161/AC-9 is cancelled at phase red. The drop landed instead at M-0162/AC-1, which explicitly lists branch-cell-2 in its RETAINED set; branch-cell-2 still carries ExpectedErrorCode "branch-not-found" while its named test asserts rung-pair-illegal. All three stale artifacts D-0018 enumerates are still present.
- **Command:** `aiwf show M-0161; grep -n 'branch-not-found' internal/policies/m0158_ac2_corner_cells_test.go internal/workflows/spec/branch/rules.go internal/workflows/spec/rules.go; sed -n '62,90p' internal/policies/m0162_ac1_drop_test.go`
- **Quoted:** “**AC-9 cycle (catalog refactor):** the `branch-not-found` cell and its policy keyword-mapping are explicitly retired as part of the 9-cell drop M-0161/AC-9 enumerates.”

#### D-0023 · `false-claim`

- **Claim:** M-0162's AC-3 body carries, at spec line 232, an enumerated five-entry file list — and that list is what the Decision calls binding.
- **Measured:** Wrong on both counts, and wrong at the time of writing too. Line 232 is a References bullet reading '- `internal/cli/integration/branch_scenarios_*.go` + sibling files — the E2E surface AC-3 references' — identical at HEAD and at 3899f34, the commit that wrote D-0023's body. The actual enumeration is line 204, and it names FOUR globs, not five. The string `authorize_scenarios` occurs zero times in the M-0162 spec at HEAD and zero times at 3899f34, so the fifth entry was never in the document the decision calls binding.
- **Command:** `p=work/epics/archive/E-0030-.../M-0162-layer-4-spec-catalog-refactor-bijection-pin-registry.md; sed -n '232p' "$p"; sed -n '204p' "$p"; grep -c 'authorize_scenarios' "$p"; sp=$(git ls-tree -r --name-only 3899f34 | grep 'M-0162-layer-4'); git show "3899f34:$sp" | sed -n '232p'; git show "3899f34:$sp" | grep -c 'authorize_scenarios'`
- **Quoted:** “The M-0162/AC-3 body's enumerated file-list (spec line 232) names `branch_scenarios_*.go`, `isolation_escape_*.go`, `detached_head_*.go`, `promote_wrong_branch_*.go`, `authorize_scenarios_test.go`.”

#### D-0023 · `status-contradiction`

- **Claim:** The decision's status is proposed and will stay so until one of two named triggers fires.
- **Measured:** Frontmatter is `status: accepted`, promoted 2026-07-16 in 2287b4756. The promote commit's own reason states that NEITHER trigger fired: 'neither (a) an explicit AC-4 reviewer sign-off nor (b) a follow-up milestone wiring the 7 cells is documented'. So the body describes a two-outcome gate that was resolved by a third route, and a reader taking the body as current gets both the status and the resolution path wrong.
- **Command:** `/…/audit/aiwf show D-0023 | head -1 ; git log --format='%h %ad %n%B' --date=short --all --grep='D-0023' -i | head -12`
- **Quoted:** “Promotion: stays `proposed` until either (a) AC-4 reviewer explicitly closes the question by accepting reallocate's omission, or (b) a follow-up milestone wires the 7 cells, at which point this decision is `superseded` by the resolving milestone's implementing AC.”

#### D-0028 · `contradicted-by-code`

- **Claim:** The priority field is legal on gap and milestone, and illegal on decision.
- **Measured:** Exactly inverted for two of the six kinds. `entity.CarriesOwnPriority` returns true for gap and decision; milestone is excluded and a milestone carrying `priority:` fires `priority-not-applicable`. Schema OptionalFields confirm: KindGap and KindDecision list "priority"; KindMilestone does not.
- **Command:** `grep -rn 'priority' internal/entity/entity.go | grep -v _test ; sed -n '248,262p;588,616p' internal/entity/entity.go`
- **Quoted:** “1. **Scope** — `priority` applies to **gap and milestone only**, not epic, ADR, decision, or contract.”

#### D-0028 · `contradicted-by-code`

- **Claim:** The decision kind deliberately does not carry priority.
- **Measured:** The decision kind carries priority in the shipped kernel, and the repo's own read surfaces are built around that. The reversal was made inside G-0078's body on 2026-07-16 (commit 6c3474e), twelve days after D-0028 was accepted (6165043, 2026-07-04), and D-0028 was never updated or superseded.
- **Command:** `git show 6c3474e | grep -E '^[-+]' | grep -i 'milestone|decision' ; <audit-binary> history D-0028`
- **Quoted:** “Decision entities were considered but rejected: an open decision is typically a *blocker* you resolve because something downstream needs the answer, not a queued item you defer against competing priorities”

#### G-0022 · `phantom-contract`

- **Claim:** A trailer key `aiwf-revoked-by:` is reserved in the kernel, awaiting a future revoke verb.
- **Measured:** No such trailer key exists anywhere in the repo outside this gap body and its pre-migration archive copy. The trailer constant block in internal/gitops/trailers.go declares no `aiwf-revoked-by`; the slot that actually ends a scope is `aiwf-scope-ends` (TrailerScopeEnds, trailers.go:50). An implementer would go looking for a reservation that was never made.
- **Command:** `grep -rn "revoked-by" . | grep -v '\.git/' ; grep -n "Trailer[A-Za-z]* *=" internal/gitops/trailers.go`
- **Quoted:** “The trailer slot is reserved (`aiwf-revoked-by:`) but the verb is not implemented in I2.5.”

#### G-0070 · `false-claim`

- **Claim:** doctor prints the `recommended-plugin-not-installed` code today, so a script can grep for it as a stopgap.
- **Measured:** The whole recommended-plugins check was retired. There is no `doctor:` key in the config struct, no `RecommendedPlugins` field, and no emitter anywhere in doctor. The only source occurrences of the string are in a policy that exists to keep the retired flow from creeping back. A consumer following this gap's advice would grep for a string doctor never prints.
- **Command:** `grep -rn 'recommended_plugins\|RecommendedPlugins' --include='*.go' internal/ cmd/ ; grep -rn 'recommended-plugin-not-installed' --include='*.go' internal/ ; grep -n 'yaml:"' internal/config/config.go | sed -n '1,20p'`
- **Quoted:** “The `recommended-plugin-not-installed` finding-code string appears verbatim in the output (so a script can grep for it), but the structured `finding.data` payload doesn't exist as a queryable surface”

#### G-0070 · `dead-premise`

- **Claim:** Adding `--format=json` to doctor would close the outstanding half of M-0070/AC-3.
- **Measured:** M-0070/AC-3's contract is that each missing recommended plugin emits a warning carrying `finding.code: "recommended-plugin-not-installed"` and `finding.data: {plugin, marketplace, install_command}`. The check that produced that finding no longer exists, so a JSON envelope on doctor could not emit it. The AC is `met` on a `done`, archived milestone; nothing would be satisfied, automatically or otherwise.
- **Command:** `/tmp/.../audit/aiwf show M-0070 ; grep -n 'finding.data' work/epics/archive/E-0018-operator-side-dogfooding-completion-closes-g-062-g-064/M-0070-aiwf-doctor-warning-for-missing-recommended-plugins.md`
- **Quoted:** “When implemented, M-0070's AC-3 contract is automatically satisfied without spec changes.”

#### G-0073 · `false-claim`

- **Claim:** E-0083's and E-0084's specs currently disagree about who settles the shared finding-code question, and nothing reconciles them.
- **Measured:** E-0083's spec was edited to agree with E-0084 seventeen seconds after G-0073's own last edit. The exact text G-0073 quotes ('Settled in the rule's own milestone') was removed and replaced with 'Shared with E-0084 ... Settled jointly before either epic's first milestone lands, not inside whichever starts first'. E-0083 now also carries a risk row naming the unilateral-settlement hazard and citing G-0073. The two specs agree.
- **Command:** `git log --format='%h %ci %s' -1 f202250c1; git log --format='%h %ci %s' -1 9e9835ae1; git show 9e9835ae1 -- 'work/epics/E-0083*/epic.md'`
- **Quoted:** “E-0084's spec resolves that question "with E-0083 before either epic's first milestone lands". E-0083's spec resolves it "in the rule's own milestone". Both statements are prose, neither is wrong on its own terms, and nothing reconciles them”

#### G-0111 · `dead-premise`

- **Claim:** The wrap-epic skill body is authored in internal/policies/testdata/ and mirrored to a separate rituals repo.
- **Measured:** internal/policies/testdata does not exist. Ritual content is authored directly in the embedded snapshot at internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-epic/SKILL.md; the upstream rituals repo is archived (ADR-0016, accepted) and the embedded snapshot is the single source of truth (ADR-0014, accepted). Anyone following this resolution path would author into a nonexistent directory and look for a repo that no longer takes changes. This is the pre-resolved missing-path finding, confirmed and diagnosed.
- **Command:** `ls -d internal/policies/testdata ; ls internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/ ; grep -n '^status:' docs/adr/ADR-0014*.md docs/adr/ADR-0016*.md`
- **Quoted:** “Cross-repo coupling pattern from M-0090 / M-0096 applies — author the skill body in `internal/policies/testdata/aiwfx-wrap-epic/SKILL.md` and copy at wrap.”

#### G-0117 · `self-contradiction`

- **Claim:** Render cost scales as N entities x per-entity git scans (present tense, §What's missing).
- **Measured:** The same body's §Notes retracts this verbatim: 'M-0221 (E-0054) collapsed the per-entity git scans into one shared HEAD-history walk (`check.WalkHeadCommits`), and `internal/cli/render/resolver.go`'s `historyIndex` now serves every entity from that single in-memory pass — render cost no longer scales as *N entities × per-entity git scans*.' The retraction is correct: buildHistoryIndex(head []check.HeadCommit) at internal/cli/render/singlepass.go:51 is fed by one WalkHeadCommits pass (internal/check/head_history.go:94). A reader who stops at §What's missing acts on a false premise about where the cost is.
- **Command:** `grep -rn "historyIndex" internal/cli/render/*.go ; grep -rn "func WalkHeadCommits" internal/`
- **Quoted:** “so wall-clock cost scales as *N entities × per-entity git scans*, not as template execution. On the kernel's own tree the render is already noticeably slow, and the curve gets worse with every entity added.”

#### G-0117 · `broken-reference`

- **Claim:** PageDataResolver lives in cmd/aiwf/render_cmd.go.
- **Measured:** cmd/aiwf/ contains exactly one file, main.go. PageDataResolver is declared at internal/htmlrender/htmlrender.go:72 and the render CLI lives under internal/cli/render/. The link is doubly broken: the target does not exist, and `../../../` from work/gaps/ resolves above the repo root (work/gaps -> work -> repo root is `../../`).
- **Command:** `ls cmd/aiwf/ ; grep -rn "PageDataResolver" --include="*.go" internal/ cmd/ | grep -v _test`
- **Quoted:** “The `PageDataResolver` in cmd/aiwf/render_cmd.go → ../../../cmd/aiwf/render_cmd.go is called once per entity”

#### G-0121 · `false-claim`

- **Claim:** G-0567 is an open, unfixed cross-surface disagreement available as a real violating state to size an agreement property against.
- **Measured:** G-0567 was promoted addressed on 2026-08-10, two days after this note's own 2026-08-08 date. The fix is a single shared seam: severity.Apply composes all four aiwf.yaml severity passes (ApplyTDDStrict, ApplyAreaRequiredStrict, ApplyDocsStrict, ApplyArchiveSweepThreshold) and is called by both the full check path and the --fast path.
- **Command:** `/tmp/.../aiwf history G-0567 ; grep -rn 'severity.Apply' internal/ cmd/ --include='*.go' | grep -v _test.go ; sed -n '90,105p' internal/severity/severity.go`
- **Quoted:** “G-0567 is one such disagreement, live and unfixed.”

#### G-0160 · `false-claim`

- **Claim:** Adding (epic, active) → in_progress to entity.transitions leaves the new edge with no positive subtest.
- **Measured:** It gets one automatically. derivePromoteTargets ends in `entity.StatusStrings(entity.AllowedTransitions(rule.Kind, entity.Status(rule.FromState)))`, enumerateLegalCases emits one positiveCase per returned target, and TestM0124_PositiveDriver_LegalCells runs each as a t.Run subtest against the real binary. The driver's own doc comment names this exact cell as the multi-target example. This was equally true at the commit that introduced the driver (b6072a168, 2026-05-22 — the same day this gap was allocated), so the claim was never accurate.
- **Command:** `grep -n 'func deriveLegalTargets' -A 60 internal/policies/m0124_positive_driver_test.go; sed -n '28,60p' internal/policies/m0124_positive_driver_test.go; git show b6072a168:internal/policies/m0124_positive_driver_test.go | grep -n 'AllowedTransitions'`
- **Quoted:** “The new edge's target (`in_progress`) gets zero positive subtest coverage — the M-0124 per-cell driver's target-derivation reads from `entity.AllowedTransitions`, but the spec table's cell carries no enumeration of *which* targets it covers.”

#### G-0166 · `false-claim`

- **Claim:** no check-time cell is driven verb-then-check end-to-end
- **Measured:** the gap cell (`gap-open-promote`) is classified in ac3ForceBypass, not ac3KnownImplGaps: the driver appends `--force --reason`, the verb succeeds, `aiwf check --format=json` runs, and the envelope is asserted to carry the rule's ExpectedErrorCode bound to the promoted entity. That is the full check-time pipeline.
- **Command:** `sed -n '120,145p;180,215p' internal/policies/m0125_negative_checktime_driver_test.go`
- **Quoted:** “Today AC-3 has *zero* cells running through the full check-time pipeline.”

#### G-0168 · `already-shipped`

- **Claim:** There is no post-creation mutation verb for a milestone's tdd: field; changing it requires hand-editing frontmatter with a fictional aiwf-verb trailer.
- **Measured:** `aiwf milestone tdd <M-id> --policy <none|advisory|required>` ships. It implements exactly what the gap's re-discovery section specified: uniform-ordinary gating with no directional carve-out, optional --reason, standard `aiwf-verb: milestone-tdd` trailers, and the refuse-with-hint on a flip to `required` that would strand a met-but-phaseless AC. The verb-name fork the body leaves open ('the re-discovery proposed `aiwf milestone set-tdd`; the table above proposes `aiwf milestone tdd --policy`. Pick one shape when implementing.') resolved to the latter.
- **Command:** `/tmp/claude-1000/-workspaces-aiwf/ec5d1c3d-5669-4772-9561-4542e8aaef62/scratchpad/audit/aiwf milestone --help ; cat internal/verb/milestone_tdd.go`
- **Quoted:** “| milestone | `tdd:` | `aiwf add milestone --tdd required\|advisory\|none` | **none** |”

#### G-0169 · `dead-premise`

- **Claim:** `aiwf rewidth` is a verb of the current CLI that lacks a JSON envelope; the Notes paragraph further asserts it "gained a proper --format=json envelope (commit 29f0c8ff)".
- **Measured:** `rewidth` is not a verb at all — it was retired by M-0290 (commit db307fdc7 "feat(check): retire the width-migration verb; any narrow active id is an error"), and its absence is pinned by TestNewRootCmd_WidthMigrationVerbIsRetired. It does not appear in `aiwf --help`. The body cites it twice as extant.
- **Command:** `<binary> --help | grep -c rewidth; sed -n '160,172p' internal/cli/root_test.go; git log --oneline --all -S rewidth -- cmd/ internal/ | head -3`
- **Quoted:** “`aiwf import` (multi-entity import) and `aiwf rewidth` (id-width migration) emit their own multi-line/multi-commit output rather than routing through `FinishVerb`”

#### G-0212 · `already-shipped`

- **Claim:** The six enumerated data-loss classes are catalogued but unexamined — the gap exists to drive a future epic that will examine them.
- **Measured:** M-0243 (done, under E-0062) converted items 1, 2, 3, 5 and 6 into named stress scenarios with oracles, and M-0241 covered item 4. Four of the five reported *better* outcomes than the gap feared, and the scenario sources say so in-line. A reader taking this body as current would re-scope work that already shipped.
- **Command:** `grep -rn "G-0212" --include="*.go" --include="*.md" . | grep -v '^work/gaps/G-0212'; ls internal/stresstest/ | grep -E 'parallel_branch_reallocate|cross_worktree_edit_body_race|archive_during_active_scope|force_override_durability|concurrent_id_allocation'`
- **Quoted:** “Known classes (from history evidence + reasoning):”

#### G-0212 · `false-claim`

- **Claim:** repolock's serialization is intra-process only, leaving cross-process invocations unserialized and untested.
- **Measured:** repolock is a POSIX advisory file lock (flock(2)) on <root>/.git/aiwf.lock — an inter-process lock by construction, and its package doc says so. Cross-process invocations are exactly what it serializes, and they are now tested: ConcurrentIDAllocationScenario races real `aiwf add` subprocesses against one working copy and asserts no id is allocated twice.
- **Command:** `sed -n 1,30p internal/repolock/repolock_unix.go; sed -n 12,20p internal/stresstest/concurrent_id_allocation.go`
- **Quoted:** “The repolock (`internal/repolock/`) serializes per-repo verb invocations within a process, but cross-process invocations on the same repo via subprocess fan-out are untested in combinatorial scenarios.”

#### G-0215 · `dead-premise`

- **Claim:** A production call site passes nil for cherryPicked, parked against G-0202. The Pattern section repeats it as case 1: 'The supporting plumbing is deliberately parked at a tracked gap (e.g., `cherryPicked` at `internal/cli/check/provenance.go:67`)'.
- **Measured:** The nil is gone. provenance.go:98 computes `cherryPicked := check.WalkCherryPicks(head)` and passes the real value; the surrounding comment records the closure explicitly. G-0202 is addressed and archived. That leaves the gap's Known-instances section with no live instance at all.
- **Command:** `grep -rn 'cherryPicked' --include='*.go' internal/cli/check/ | grep -v _test; aiwf show G-0202`
- **Quoted:** “- `internal/cli/check/provenance.go:67` — `cherryPicked` parameter nil; documented as G-0202 deferred; would qualify for `// PARKED: G-0202` marker.”

#### G-0217 · `false-claim`

- **Claim:** The WRAP PENDING message fires both for an in_progress milestone (case 1) and for a done-and-merged-to-epic milestone (case 2), with identical wording. The proposed-fix table repeats this, labelling the in_progress row '(current behavior)'.
- **Measured:** The label is emitted only from renderStaleSection, reached only when `v.Stale` is true, and `v.Stale = isTerminalStatus(e.Kind, e.Status)`. isTerminalStatus admits only done/cancelled/wontfix/rejected/addressed/retired/superseded/deprecated. An `in_progress` milestone can never reach the label. (The one other way Stale is set, mergedStaleOverride, requires AheadOfTrunk == 0, which the wrap-pending arm's `v.AheadOfTrunk > 0` guard excludes.) So there is no conflation of two states — one state is mislabelled.
- **Command:** `grep -n 'Stale' internal/cli/status/worktrees.go; sed -n '820,935p' internal/cli/status/worktrees.go; awk '/^func isTerminalStatus\(/,/^}/' internal/cli/status/worktrees.go`
- **Quoted:** “The current message fires in BOTH cases with identical wording, and the phrase "WRAP PENDING" strongly suggests case 1 (operator forgot to wrap).”

#### G-0225 · `false-claim`

- **Claim:** The population lacking `aiwf-branch-sha:` is closed and historical — authorize commits emitted before M-0161/AC-6 landed. The body draws the consequence explicitly: "For an aiwf consumer who never renames ritual branches with open scopes: never triggers; AC-6 ships transparently."
- **Measured:** The population is open-ended and is produced by aiwf's own shipped rituals. The CLI populates BranchSHA only when the named branch already resolves (`if vOpts.BranchExists { vOpts.BranchSHA = branchTipSHA(...) }`, internal/cli/authorize/authorize.go:293-295), and the verb 'emits `aiwf-branch-sha:` iff non-empty' (internal/verb/authorize.go:247). The aiwfx-start-epic ritual mandates the opposite order: 'Sovereign acts on `main`; branch cut afterwards. State-announcement commits (the promote at step 6 and, if delegating, the authorize at step 7) land on `main` BEFORE the epic branch is cut at step 8.' So the trailer is structurally absent for every ritual-opened scope. Empirically, all 7 authorize commits in this repo's history lack it, including E-0034's (2026-07-21), which post-dates AC-6's landing (a744c873b, 2026-06-04) by seven weeks.
- **Command:** `git log --all --grep="aiwf-verb: authorize" --format="%H" | while read s; do if git show -s --format=%B "$s" | grep -q "aiwf-branch-sha:"; then echo "HAS $s"; else echo "NO  $s"; fi; done | sort | uniq -c   # then: git show -s --format="%h %ad %s" --date=short d21ed6828 ; git log --all --oneline --format="%h %ad %s" --date=short -S"TrailerBranchSHA" -- internal/verb/authorize.go | tail -1`
- **Quoted:** “2. The authorize commit was emitted before AC-6 (no `aiwf-branch-sha:` trailer).”

#### G-0225 · `stale-cli-invocation`

- **Claim:** `aiwf acknowledge-illegal <sha> --reason "..."` is a runnable workaround; the same form also appears at line 31 as workaround (a).
- **Measured:** There is no `acknowledge-illegal` verb. It was regrouped into the `aiwf acknowledge illegal` subverb at commit 24b25000f (M-0181/AC-5). The verb the body needs still exists and does still silence isolation-escape — only the spelling is dead — so the workaround is recoverable, but a reader copy-pasting it gets an unknown-command error.
- **Command:** `/tmp/.../audit/aiwf acknowledge-illegal ; git log --oneline --all -S"acknowledge-illegal" -- cmd/ internal/cli/ | grep regroup`
- **Quoted:** “2. **Acknowledge-illegal sweep**: for each AI commit on the renamed branch that the rule false-positives on, `aiwf acknowledge-illegal <sha> --reason "legacy scope rename"`.”

#### G-0225 · `stale-cli-invocation`

- **Claim:** An operator can end an open authorize scope with `aiwf authorize <id> end`.
- **Measured:** `aiwf authorize` accepts exactly one positional arg; there is no `end` subverb and no `--end` flag (the mode flags are `--to`, `--pause`, `--resume`). A scope ends only when some commit carries an `aiwf-scope-ends: <auth-sha>` trailer, which the kernel auto-writes on a terminal promote of the scope-entity (internal/cli/cliutil/scopes.go:30-33). So workaround 1 is not merely misspelled — there is no operator-driven end at all, and the whole 'end-and-re-authorize' path is unavailable while the scope-entity is non-terminal.
- **Command:** `/tmp/.../audit/aiwf authorize E-0001 end ; /tmp/.../audit/aiwf authorize --help`
- **Quoted:** “1. **End-and-re-authorize**: `aiwf authorize <id> end` to close the scope, then `aiwf authorize <id> --to ai/<agent> --branch <new-name>` to re-open.”

#### G-0276 · `already-shipped`

- **Claim:** The stash-based isolation and its StashStaged/StashPop helpers are still present and awaiting removal.
- **Measured:** Removed. No Stash* symbol exists in internal/gitops production source, and TestStashSymbolsDoNotExist walks every non-test .go file in the package and fails if one reappears. verb.Apply now routes through gitops.CommitVerbChange -> CommitTree, which never touches the live index or worktree.
- **Command:** `git log --oneline -S "StashStaged" -- internal/gitops/gitops.go && grep -rn "StashStaged\|StashPop" internal/ --include='*.go' && sed -n '12,24p' internal/gitops/no_stash_test.go`
- **Quoted:** “Either retires the stash and the `StashStaged` / `StashPop` pair entirely.”

#### G-0276 · `false-claim`

- **Claim:** Verb commits are built by stashing and then `git commit`-ing the whole live index; the two cited line ranges hold that code.
- **Measured:** Both cites are dead and the mechanism is gone. gitops.go:175-177 is now a map-to-slice loop inside DirtyPaths; gitops.go:113-116 is now doc-comment text inside CommitAllowEmpty explicitly stating that verb commits do NOT use this porcelain. Verb commits build a tree against a throwaway GIT_INDEX_FILE and move HEAD with update-ref.
- **Command:** `sed -n '113,116p' internal/gitops/gitops.go && sed -n '175,177p' internal/gitops/gitops.go`
- **Quoted:** “`aiwf` isolates each mutating verb's commit by stashing the user's pre-staged index (`git stash push --staged`, `internal/gitops/gitops.go:175-177`) and then committing the whole index (`Commit` → plain `git commit -m`, `internal/gitops/gitops.go:113-116`), so the commit contains exactly the verb's ”

#### G-0281 · `dead-premise`

- **Claim:** The allocator's read set is the working tree plus the single configured trunk ref, so a sibling worktree's committed-but-unpushed id and an unpushed local trunk are both invisible at allocation time.
- **Measured:** The allocator's read set is the working tree plus the configured trunk ref plus EVERY local `refs/heads/*` plus EVERY remote-tracking `refs/remotes/*`. `aiwf add` calls `entity.AllocateID(kind, t.Entities, t.AllocationIDs())`, and `AllocationIDs()` unions `TrunkIDStrings()` + `LocalRefIDs` + `RemoteRefIDs`. Since linked worktrees share `refs/heads/*`, a sibling worktree's committed id and a local unpushed `main` are both seen. The two gaps that named this defect (G-0272 sibling-worktree heads, G-0273 fetch-before-allocate) are both `addressed`; E-0052 that shipped it is `done`.
- **Command:** `grep -rn "AllocationIDs" --include="*.go" internal/ cmd/ | grep -v _test.go   &&   sed -n '227,245p' internal/tree/tree.go   &&   go test -run 'TestTree_AllocationIDs_UnionsTrunkLocalAndRemoteRefs|TestLocalRefIDs_UnionsSiblingBranchIDs' ./internal/tree/ ./internal/trunk/`
- **Quoted:** “on a feature/epic worktree (or a second session) `aiwf add gap` allocates `max(working tree ∪ origin/main)+1`, blind to sibling worktrees and to unpushed trunk”

#### G-0282 · `dead-premise`

- **Claim:** No post-create verb mutates a milestone's tdd policy, so the gated-annotation extension cannot be built.
- **Measured:** `aiwf milestone tdd <M-id> --policy <none|advisory|required>` is a registered command and shipped 2026-07-24 (3e1e350ff, M-0277/AC-1) — three days after the status section was written. Its help explicitly documents both directions.
- **Command:** `/tmp/.../audit/aiwf milestone tdd --help && git log -1 --format="%ci" 3e1e350ff`
- **Quoted:** “It is motivated by the `tdd: required → advisory` toggle, and that toggle verb does not exist: `tdd:` is currently set-once at `aiwf add --tdd` (`internal/cli/add/add.go`) with no post-create mutation verb — the gap G-0168 tracks that (open, high priority). Building the `gated` axis now would mean s”

#### G-0282 · `contradicts-decision`

- **Claim:** The weakening direction of a tdd toggle should be gated behind --reason, and a policy should assert that gate mechanically.
- **Measured:** D-0048 (status: accepted, accepted 2026-07-22) decides the exact opposite and says so as a general principle: the tdd verb ships with 'no directional or sovereign gating in either direction', and 'if scrutiny of a weaken-after-met is ever wanted it arrives as a symmetric advisory finding, never a directional verb gate'. The shipped verb matches: --reason is optional and there is no --force.
- **Command:** `sed -n '20,55p' work/decisions/D-0048-frontmatter-field-mutation-verbs-milestone-tdd-now-others-per-kind-deferred.md && git log --format="%ci %s" --follow -- work/decisions/D-0048-*.md | tail -2`
- **Quoted:** “For class-A (self-inverse) entries, allow an optional **`gated` annotation** naming the direction/condition that must carry `--reason` (and, where applicable, a human actor), and assert mechanically that the verb requires it on that path.”

#### G-0282 · `dead-premise`

- **Claim:** rewidth is a live verb whose --help omits its irreversibility, giving the proposed policy an immediate real violation to catch.
- **Measured:** rewidth was retired on 2026-08-03 (db307fdc7, M-0290/AC-1+AC-2) and is no longer a command; internal/cli/root_test.go:162 pins it as `const retired = "rewidth"`. The proposed policy therefore has no present-day violation fixture. The same claim appears a second time in the gap's 'Adjacent cleanups' section ('`rewidth` is one-way but does NOT say so in `--help` — the policy would flag it').
- **Command:** `/tmp/.../audit/aiwf rewidth --help ; git log --oneline -S"rewidth" -- internal/cli cmd/ | head -3 ; grep -n 'retired' internal/cli/root_test.go`
- **Quoted:** “has a concrete seed fixture already on hand: `rewidth` is one-way but its `--help`/Long text does not say so (unlike `archive`, which cites ADR-0004), a real present-day violation the base registry would catch immediately.”

#### G-0305 · `already-shipped`

- **Claim:** All three: aiwf is not a health-file producer; the statusline shows only aiwf findings; the statusline runs a check rather than reading a union.
- **Measured:** All three are false now. (1) `aiwf doctor --write-health` is a documented flag that writes .claude/health.aiwf.json in the fixed {generated_at, findings[{source,severity,message}]} schema with source:"aiwf", via pathutil.AtomicWriteFile into gitops.MainCheckoutRoot's .claude/ — atomic, main-checkout-resolved from a linked worktree, and gitignored at .gitignore:106 `.claude/health.*.json`. `aiwf update` refreshes it too (update.go:215). (2)+(3) The statusline globs every producer file and never runs a check.
- **Command:** `<binary> doctor --help ; cat internal/cli/doctor/health.go ; grep -n "health" internal/skills/embedded-statusline/statusline.sh ; grep -n health .gitignore`
- **Quoted:** “M-0193's render-run-check approach does not fit: aiwf is not a producer yet, the statusline surfaces only aiwf (not dotfiles) findings, and it runs a check rather than reading the union.”

#### G-0311 · `false-claim`

- **Claim:** The current workaround for a cross-cutting capability is three epics wired together by depends_on.
- **Measured:** An epic has no depends_on, and no outbound reference field at all. `depends_on` is milestone-scoped with AllowedKinds []Kind{KindMilestone} (internal/entity/entity.go:582), and entity.ForwardRefs' default arm states 'KindEpic and any future kind without outbound refs falls through to an empty list' (internal/entity/refs.go:63-66). The only verb is `aiwf milestone depends-on`; there is no `aiwf epic depends-on`. The real fallback is unwired sibling epics plus prose — worse than what the body describes, so the gap's case survives, but the sentence tells a reader a mechanism exists that does not.
- **Command:** `<binary> schema epic ; <binary> schema milestone ; <binary> milestone --help ; grep -n -A40 "func ForwardRefs" internal/entity/refs.go`
- **Quoted:** “the kernel forces it into three separate epics wired by `depends_on`, with no entity that names "the subtitle feature."”

#### G-0333 · `false-claim`

- **Claim:** No AI-discoverable channel states what --force does and does not override.
- **Measured:** Three shipped channels state it. (1) `aiwf promote --help` has carried 'coherence checks still run' since M-0115 and now reads 'coherence checks still run and the standing audit keeps reporting'. (2) Since M-0293/AC-2 (commit 527caf70d, 2026-08-05 — one day AFTER this gap's last edit-body 4a025cc, 2026-08-04) internal/check/hint.go:501 defines forceCaveatSentence, appended by HintFor to every hint offering `--force --reason`. (3) internal/skills/embedded/aiwf-promote/SKILL.md:45 and docs/design/design-decisions.md:148 state it in prose. CLAUDE.md's own AI-discoverability principle names `--help` and embedded skills as sufficient channels, so the stated principle violation no longer holds.
- **Command:** `/tmp/claude-1000/-workspaces-aiwf/ec5d1c3d-5669-4772-9561-4542e8aaef62/scratchpad/audit/aiwf promote --help | grep -n -i force; grep -n 'const forceCaveatSentence' -A 3 internal/check/hint.go; git log --format='%h %ad %s' --date=short -S 'forceCaveatSentence' -- internal/check/hint.go`
- **Quoted:** “The boundary — "`--force` overrides local FSM / preconditions, not tree-invariant error findings" — is stated in no AI-discoverable channel (CLAUDE.md, `docs/design/provenance-model.md`, or `--force --help`), yet it determines what a sovereign override can and cannot do. This violates the kernel's "”

#### G-0396 · `stale-count`

- **Claim:** If addressed_by_commit were dropped, about 50 legitimately-closed gaps would newly fire gap-addressed-has-resolver.
- **Measured:** 303. gapAddressedHasResolver fires when status==addressed and BOTH addressed_by and addressed_by_commit are empty (internal/check/check.go:975), so the exposed population is every addressed gap that is SHA-only. That is 303 today, and was already 204 on 2026-07-09 when this body landed (commit c6a9725) — the figure was never ~50. The conclusion (keep the field) survives and is strengthened; the number is off by 6x.
- **Command:** `python3 scan of work/gaps/**/*.md frontmatter for status==addressed with non-empty addressed_by_commit and empty addressed_by; plus the same scan against `git ls-tree -r c6a9725 work/gaps/``
- **Quoted:** “dropping the field would make roughly fifty legitimately-closed gaps fire the resolver warning — the exact noise this avoids.”

#### G-0396 · `false-claim`

- **Claim:** The gaps whose only resolver is a stored SHA are a bounded legacy cohort of bulk-imported entities in the G-0001..G-0055 range.
- **Measured:** 255 of the 303 SHA-only addressed gaps are numbered above G-0055, up to and including G-0593 — the newest gap in the tree, promoted at HEAD. `--by-commit` is the shipped, prescribed closure path: wf-patch's wrap gate, aiwfx-wrap-milestone step 13, and aiwfx-wrap-epic's precondition-6 remedy all instruct `aiwf promote G-NNNN addressed --by-commit <sha>`. The SHA-only population is the current default and grows with every patch, not a frozen legacy tail. A reader would conclude the field is a backward-compat concern; it is the primary resolver in active use.
- **Command:** `grep -rn "by-commit" internal/skills/embedded-rituals/ ; python3 frontmatter scan bucketing SHA-only addressed gaps by id number ; git log -1 --format=raw e27f19eab`
- **Quoted:** “That is the legacy cohort (`G-0001` through `G-0055` and peers) bulk-imported already `addressed`, which never went through a trailered `promote`, so no closure commit exists to find.”

#### G-0400 · `stale-count`

- **Claim:** The registered scenarios collectively invoke 10 distinct aiwf leaf verbs, out of 38 that exist.
- **Measured:** 16 distinct verbs are now invoked by scenario code, and 40 leaf verbs exist (39 under the body's own convention of counting `acknowledge` once). The exercised set is add, check, promote, show, reallocate, history, edit-body, retitle, rename, move, `milestone tdd`, authorize, archive, acknowledge(-illegal), cancel, list. Both numbers in the title are wrong; the title is the first thing `aiwf list` and ROADMAP.md show a reader.
- **Command:** `cd /workspaces/aiwf/internal/stresstest && ls *.go | grep -v '_test\.go$' > /tmp/nontest.txt && xargs -a /tmp/nontest.txt grep -hoP 'runAiwfJSON\([^,]+,\s*[^,]+,\s*"\K[a-z-]+' | sort -u   # plus: grep -n 'exec.Command(s.aiwfBin' *.go and grep -n runAiwfListJSON *.go   # and: <aiwf> __complete "" ; <aiwf> __complete <group> ""`
- **Quoted:** “Stress scenario catalog exercises only 10 of 38 aiwf verbs”

#### G-0400 · `phantom-verb`

- **Claim:** `rewidth` is an extant aiwf verb, wired for diagnostic logging, that no scenario exercises.
- **Measured:** `aiwf rewidth` does not exist. The width-migration verb was retired in M-0290/AC-1 (commit db307fdc7, 2026-08-03), and a test now pins the retirement across both the command tree and the registered-verbs annotation. The gap's last edit-body was 2026-07-18 (commit 362b1e096), before the retirement, so the citation was never revised.
- **Command:** `/tmp/.../audit/aiwf rewidth --help ; sed -n '160,172p' /workspaces/aiwf/internal/cli/root_test.go ; git log --format='%h %ad %s' --date=short -S'rewidth' -- internal/cli/root.go`
- **Quoted:** “15 are never exercised by any scenario: `move`, `upgrade`, `rename`, `rename-area`, `set-area`, `retitle`, `rewidth`, `archive`, `import`, `worktree-add`, `contract-bind`, `contract-unbind`, `contract-verify`, `contract-recipe-install`, `contract-recipe-remove`”

#### G-0400 · `stale-enumeration`

- **Claim:** 15 diagnostic-wired verbs are undriven by any scenario, among them `move`, `rename`, `retitle`, and `archive`.
- **Measured:** 26 verbs are diagnostic-wired today; 15 of those are exercised and 11 are not (acknowledge-mistag, contract-bind, contract-recipe-install, contract-recipe-remove, contract-unbind, contract-verify, import, rename-area, set-area, set-priority, worktree-add). `move`, `rename`, `retitle` and `archive` are all driven now — by the verb-sequence walker's operation table and the concurrent-move scenario. The body's own Notes section already says so, so the enumeration contradicts the note six paragraphs below it.
- **Command:** `grep -rn 'BeginVerbDiag\|BeginReadVerbDiag' /workspaces/aiwf/internal/cli/ --include='*.go' | grep -v _test | grep -oP 'Begin(Read)?VerbDiag\([^,]+,\s*"\K[a-z-]+' | sort -u ; sed -n '405,437p' /workspaces/aiwf/internal/stresstest/verb_sequence.go`
- **Quoted:** “Of the verbs wired for diagnostic logging (E-0061, extended by this milestone's own follow-up work), 15 are never exercised by any scenario”

#### G-0412 · `false-claim`

- **Claim:** The inaccurate text lives at archive.go:127 and in renamearea, setarea, add, cancel, promote, reallocate, rename, retitle, milestone, update and editbody.
- **Measured:** Of the twelve locations named, only internal/cli/update carries the text. archive.go:127 now reads a different ignore entirely (`prelude resolution failure is covered by the shared helper's own tests`) because the ResolveRoot call there was replaced by cliutil.ResolvePreludeEnvelope. The nine files that actually carry it are: internal/cli/cliutil/testutil/fixtures.go, internal/cli/contract/recipes.go, internal/cli/contract/verify.go, internal/cli/history/history.go, internal/cli/list/list.go, internal/cli/show/show.go, internal/cli/status/status.go, internal/cli/update/update.go, internal/cli/whoami/whoami.go. The count 'nine' happens to still hold; the set behind it is almost entirely different, so a reader who acts on the list edits the wrong files.
- **Command:** `grep -rln 'ResolveRoot only fails on missing' --include='*.go' internal/ cmd/ | sort; for f in internal/cli/archive/archive.go internal/cli/renamearea internal/cli/setarea internal/cli/add internal/cli/cancel internal/cli/promote internal/cli/reallocate internal/cli/rename internal/cli/retitle internal/cli/milestone internal/cli/editbody; do grep -rn 'ResolveRoot only fails on missing' "$f"; done; sed -n '127,132p' internal/cli/archive/archive.go`
- **Quoted:** “This wording originates at `internal/cli/archive/archive.go:127` and has since been copied verbatim into `internal/cli/renamearea`, `internal/cli/setarea`, and every M-0252/M-0253 file that ignores this branch (`add`, `cancel`, `promote`, `reallocate`, `rename`, `retitle`, `milestone`, `update`, `ed”

#### G-0432 · `already-shipped`

- **Claim:** Two distinct resolvers exist; the human print site reads ResolvedVersion and the envelope reads version.Current.
- **Measured:** `ResolvedVersion` does not exist in the live Go source at all — no declaration, no call site. Every surface routes through `version.Current()`, which itself folds the stamp in (`func Current() Info { return resolveVersion(Stamp, readBuildInfoVersion()) }`, version.go:129-131; Stamp is documented at version.go:55 as 'the single source of truth for the binary's reported version'). Human print sites: root.go:131 and root.go:234 both `cliutil.Println(version.Current().Version)`; doctor.go:169 `current := version.Current()`.
- **Command:** `grep -rn "ResolvedVersion" --include=*.go internal/ cmd/ ; grep -rn "version.Current()" internal/cli/root.go internal/cli/doctor/*.go | grep -v _test`
- **Quoted:** “`aiwf version` and `aiwf doctor` resolve the binary's version via `ResolvedVersion()` in `internal/cli/root.go`, which prefers the ldflags stamp … Every `--format=json` command's envelope … resolves the same fact via `version.Current()` (Go build info), a distinct code path.”

#### G-0432 · `already-shipped`

- **Claim:** No mechanical assertion forces the human print site and the envelope to agree.
- **Measured:** Two do, and both were added by the same fix. `PolicyVersionSingleSource` (internal/policies/version_single_source.go:54) is a CI-blocking AST policy forbidding any parallel version global outside internal/version, and `TestBinary_VersionVerb_AndEnvelopeAgree_WhenStamped` (internal/cli/integration/binary_integration_test.go:67) builds an ldflags-stamped binary and asserts `aiwf version` and the `schema --format json` envelope report the identical string. Run green at HEAD.
- **Command:** `timeout 300 go test -run 'TestBinary_VersionVerb_AndEnvelopeAgree_WhenStamped' -count=1 ./internal/cli/integration/`
- **Quoted:** “`PolicyEnvelopeVersionSource` pins the envelope's derivation; nothing pins the human-print site to match it.”

#### G-0436 · `already-shipped`

- **Claim:** docs/design/id-allocation.md still cites cmd/aiwf/admin_cmd.go.
- **Measured:** It does not. The path half of that line was repointed to internal/cli/history/history.go by commit 03127efea ("fix(docs): repoint stale cmd/aiwf/*_cmd.go source paths to internal/cli/ (G-0443)"). The only surviving drift on that line is the function names, which is G-0444's subject. This is the partly-addressed half of the gap.
- **Command:** `grep -n "admin_cmd\|runHistory\|readHistoryChain\|history.go" docs/design/id-allocation.md; git log --oneline -5 -- docs/design/id-allocation.md`
- **Quoted:** “`docs/design/id-allocation.md` cites `cmd/aiwf/admin_cmd.go` for `runHistory`/`readHistoryChain`; the file no longer exists under `cmd/aiwf/` (only `main.go` remains there).”

#### G-0452 · `already-shipped`

- **Claim:** wf-structural-sweep has three lenses and needs a fourth data-flow lens added.
- **Measured:** The shipped skill carries `## The four lenses` with `### Lens 4 — Data flow (producer→consumer)` covering all four flags the gap specifies (produced-but-unconsumed, orphan stages, duplicate derivations, data-dependency cycles), and the frontmatter description already says 'Runs four lenses'.
- **Command:** `grep -rn -i "lens" internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-structural-sweep/SKILL.md`
- **Quoted:** “Add a fourth lens to `wf-structural-sweep`, phrased **stack- and domain-agnostic** like the existing three (the skill ships into consumer repos)”

#### G-0452 · `already-shipped`

- **Claim:** The second deliverable — a referencing structural test — is outstanding.
- **Measured:** internal/policies/wf_structural_sweep_test.go pins `## The four lenses` at exactly four `###` sub-lenses including 'Data flow', plus a dedicated Lens 4 assertion. It landed in the same commit as the skill edit, and that commit names G-0452 in its subject.
- **Command:** `grep -n "Lens 4\|four lenses\|Data flow" internal/policies/wf_structural_sweep_test.go && git show --stat aef85f87c`
- **Quoted:** “A referencing structural test under `internal/policies/` covering the new lens section.”

#### G-0472 · `already-shipped`

- **Claim:** Present tense, and explicitly 'the largest thing on this page': the read-fault divergence is a live data-loss path awaiting G-0557.
- **Measured:** G-0557 is addressed and archived. All four installers now carry the identical explicit read-fault arm — ensurePreCommitHook (1454), ensureCommitMsgHook (1534) and ensurePostCommitHook (1622) each refuse on an unreadable hook with a G-0557-citing comment. The Related section's 'G-0557 — ... a live data-loss path' is false on the same grounds.
- **Command:** `/tmp/.../audit/aiwf show G-0557 --format=json ; grep -n 'Refuse instead (G-0557)' -B4 internal/initrepo/initrepo.go`
- **Quoted:** “their four members disagree about what a read fault means, three of them destroying a user's unreadable hook and reporting success. That is a wrong-output failure mode rather than a maintenance one, it is tracked as G-0557, and it is the work this family actually wants.”

#### G-0514 · `stale-measurement`

- **Claim:** M-id / E-id / C-id, M-PPPP / M-QQQQ / M-PPP / M-QQQ, M-a / M-alpha / M-007a / M-NNN/AC-X, ADR-NEW / ADR-OPSPEC are present in the shipped surfaces today and produce skill-body-id findings.
- **Measured:** Zero skill-body-id findings on the tree; the only id-shaped tokens left in the scanned surfaces are canonical placeholders (M-NNNN 143, E-NNNN 135, D-NNNN 44, G-NNNN 40, ADR-NNNN 34, C-NNNN 30, M-NNNN/AC-N 17). Every listed token was removed on 2026-08-02 by M-0288 (commits 3ca397918 'sweep shipped surfaces to canonical placeholders', a0fe2d90c 'describe rejected id shapes instead of exhibiting them'), i.e. ~2h after the gap was filed at 16:22 the same day.
- **Command:** `/tmp/.../audit/aiwf check --format=json | python3 -c "import json,sys;from collections import Counter;d=json.load(sys.stdin);print(Counter(f['code'] for f in d['findings']))"  AND  grep -rnoE '\b(E|M|G|D|C|ADR)-[A-Za-z0-9_]+(/AC-[A-Za-z0-9_]+)?\b' internal/skills/embedded internal/skills/embedded-rituals internal/skills/embedded-guidance --include='*.md' | awk -F: '{print $NF}' | sort | uniq -c  AND  git log --format='%h %ad %s' --date=short -S'ADR-NEW' -- internal/skills/`
- **Quoted:** “Measured on the shipped tree, these currently fire under that message:”

#### G-0517 · `false-premise`

- **Claim:** The narrow ids in docs/design/**, docs/overview.md and docs/architecture.md are mostly citations of real entities, so each needs individual research and a widened number — a different edit from the placeholder sweep the linted corpus got.
- **Measured:** All 71 occurrences sit in worked examples, hypothetical scenarios, or id-shape illustrations; none reads as a citation. By file: provenance-model.md 30 (all inside an authorize/promote worked example — 'Example: `aiwf authorize E-03 --to ai/claude`', a rendered `aiwf history` transcript, a scripted Human/Claude session), id-allocation.md 12 (a hypothetical collision: 'So branch A can allocate `G-035`'), overview.md 12 (a mermaid tutorial and a fictional tree `work/epics/E-01-discovery-and-ramp-up/`, plus `M-007-foo.md`), design-decisions.md 9 ('An id like `E-19`', `E-19-<slug>/`, `M-001-<slug>.md`, an `M-007`/`M-008` reallocate example), architecture.md 3 and design-lessons.md 3 (the same illustrative trio, spelled `E-19` in one and `E-001` in the other — a divergence only fiction permits), legal-workflows-first-principles.md 2 (`E-22`, `M-007` as width examples). The fiction is confirmed against the tree: overview.md's `E-01-discovery-and-ramp-up` and `M-001-map-the-system` name entities whose real slugs are different.
- **Command:** `grep -rnoE '\b(E|M|G|D|ADR|C)-[0-9]{1,3}\b' docs/overview.md docs/architecture.md docs/design/ | wc -l ; grep -rcoE '\b(E|M|G|D|ADR|C)-[0-9]{1,3}\b' docs/overview.md docs/architecture.md docs/design/*.md | grep -v ':0' ; sed -n 118,150p docs/overview.md ; sed -n 206p docs/design/provenance-model.md ; sed -n 11p docs/design/id-allocation.md ; <audit-binary> show E-0001 --format=json ; <audit-binary> show M-0001 --format=json`
- **Quoted:** “These are mostly citations of entities that were genuinely real at a narrow width, so the correct fix is widening each to the real canonical id — a different edit, made one reference at a time, with a lower payoff than cleaning the docs an assistant reads to learn the workflow.”

#### G-0530 · `false-claim`

- **Claim:** `## Work log` had a median of 0 words and was empty in half the milestones carrying it as of 2026-08-03.
- **Measured:** Re-measured at 80aee53df, the exact commit that added this gap on 2026-08-03: `## Work log` is present in 190 of 290 milestone files, median 228.5 words, empty in 1 (0.5%). At HEAD: present in 208, median 226.5, empty in 6 (3%). The claim was wrong when written, by two orders of magnitude, and it is the sentence the gap calls its sharpest case. Of the other three medians, `Surfaces touched` (21) matches exactly and `Dependencies` (15 vs. 14 claimed) is close; `References` measured 31.5 against the claimed 24.
- **Command:** `python3 scratchpad/measure_at.py 80aee53df — for each work/epics/**/M-*.md at that tree, split on ^## , strip HTML comments, word-count each named section`
- **Quoted:** “Measured over the entity tree on 2026-08-03, all four are thin: median word counts of 0, 14, 21 and 24 respectively. `## Work log` is the sharpest case — the wrap ritual mandates one entry per acceptance criterion with its outcome and commit SHA, and the section is empty in half the milestones that ”

#### G-0536 · `dead-premise`

- **Claim:** A CI `aiwf check` step cannot land green today, because ids minted on unpushed local branches are absent from the remote; the authoritative-view question is open and held by G-0536's blocker G-0556.
- **Measured:** G-0556 is `addressed` and archived; ADR-0041 is `accepted` and shipped. Post-ADR-0041 a reference resolvable only from a local branch fires `refs-resolve/cross-branch-local-only` / `body-prose-id/cross-branch-local-only` at ERROR severity and blocks at the push boundary, so a tree that reaches the remote has its references resolvable from something published — i.e. reachable by fetching remote refs, contrary to 'the ref is absent from the remote'. Measured on the tree as it stands: 0 errors, 9 warnings, none cross-branch. G-0556's own 'What has landed' section states 'giving CI the refs to follow them is G-0536's' and 'G-0536's CI position is what finally makes CI's verdict converge with the operator's, and it is the last step rather than a precondition.'
- **Command:** `/tmp/.../audit/aiwf show ADR-0041 ; /tmp/.../audit/aiwf show G-0556 ; grep -rn "cross-branch-local-only" internal/check/*.go | grep -v _test ; /tmp/.../audit/aiwf check`
- **Quoted:** “`fetch-depth` does not reach it — the ref is absent from the remote, not merely unfetched. On the tree as it stands the step reports errors on day one. Which view is authoritative is a question this gap inherits rather than answers; G-0556 holds it.”

#### G-0564 · `dead-premise`

- **Claim:** E-0080 mechanized G-0121's two composition sub-gaps, leaving this gap as the remainder.
- **Measured:** E-0080 is cancelled, not done. It was cancelled 2026-08-08 (two days after this body was written on 2026-08-06) with the reason 'Premise falsified: the walker's oracle is an absolute allowlist, not monotonic, and an invariant shape does not make an agreement property cheap. Successor direction in G-0121'. Nothing it would have mechanized shipped.
- **Command:** `/tmp/claude-1000/-workspaces-aiwf/ec5d1c3d-5669-4772-9561-4542e8aaef62/scratchpad/audit/aiwf history E-0080`
- **Quoted:** “G-0121 named four sub-gaps. Two of them — composition tests across verb chains, and tree-level post-conditions under arbitrary legal composition — are mechanized by E-0080.”

#### G-0564 · `false-claim`

- **Claim:** G-0121 is closed (or closing) by E-0080, and G-0564 is the residue.
- **Measured:** G-0121 is open, priority high, and its body has since been rewritten to carry the composition remainder itself — including both of G-0564's own threads.
- **Command:** `/tmp/claude-1000/-workspaces-aiwf/ec5d1c3d-5669-4772-9561-4542e8aaef62/scratchpad/audit/aiwf show G-0121 --format=json  ;  grep -n 'stays open for that remainder' work/gaps/G-0121*.md`
- **Quoted:** “G-0121 — the parent. E-0080 mechanizes its composition sub-gaps and closes it; this gap carries what that work deliberately leaves behind.”

#### G-0573 · `false-claim`

- **Claim:** The verb-time projection guard applies no aiwf.yaml severity escalation.
- **Measured:** projectionFindings loads the consumer's severity policy and applies it to BOTH the pre and post sides before diffing.
- **Command:** `grep -rn 'severity.Apply|severity.Load|severity.From' --include='*.go' internal/ cmd/ | grep -v _test`
- **Quoted:** “It calls `check.Run` directly, and applies none of the four `aiwf.yaml` severity passes the full `aiwf check` applies afterwards.”

#### G-0573 · `false-claim`

- **Claim:** The severity passes are composed independently at seven call sites with nothing mechanical holding them in relation.
- **Measured:** All seven now route through one seam (internal/severity.Apply), and PolicySeverityPassComposition fails CI on a pass the seam forgets, a call site that skips it, or one handed a policy literal instead of the consumer's. The live-tree assertion passes.
- **Command:** `grep -rn 'severity.Apply|severity.Load|severity.From' --include='*.go' internal/ cmd/ | grep -v _test && go test -run 'TestPolicySeverityPassComposition_LiveTree' ./internal/policies/`
- **Quoted:** “seven call sites reach `check.Run` and each decides independently which severity passes to compose — four for `check`, two for `check --fast`, zero for `status`, `show`, `render`, `doctor` and this guard. Nothing mechanical holds them in any relation to each other, so the next added pass will reach ”

#### G-0573 · `wrong-diagnosis`

- **Claim:** The escalation is invisible to verbs because the guard skips the severity passes.
- **Measured:** The guard applies the passes; the escalation is still invisible for a different reason — entityBodyEmpty reads the entity file from disk (os.ReadFile), and `aiwf add`'s projected entity has no file yet, so the rule stays silent on it and the diff has nothing to escalate. entity-body-empty is NOT in skipDuringProjection, so the exclusion is not the cause either.
- **Command:** `sed -n '128,152p' internal/check/entity_body.go && sed -n '240,272p' internal/verb/common.go`
- **Quoted:** “So a knob that escalates a finding to error severity is invisible to every verb.”

#### G-0583 · `contradicts-normative-source`

- **Claim:** The two cheap parts (measuring factual claims, challenging each criterion) run before the expensive sweep — the parts are ordered by yield.
- **Measured:** The initiative that owns this specification decides the opposite and argues against yield-ordering by name. `docs/initiatives/milestone-preflight-as-independent-review.md` §"The sweep runs before the lab": "The sweep generates questions and the lab answers them, so the sweep goes first. Ordering the parts by yield instead — cheapest and most productive first — answers a different question." CLAUDE.md's in-force scoped section makes the initiative the specification for this work. E-0086 (`active`, on branch) records the conflict as a preflight finding and puts disposing it in its Scope.
- **Command:** `sed -n '281,291p' docs/initiatives/milestone-preflight-as-independent-review.md   &&   git show epic/E-0086-build-the-smallest-preflight-pass-that-can-be-trialled-and-can-fail:work/epics/E-0086-build-the-smallest-preflight-pass-that-can-be-trialled-and-can-fail/epic.md | grep -n 'G-0583'`
- **Quoted:** “So the cheap two run first and always; the sweep reads current trunk, and its findings are hypotheses until a command settles one”

#### G-0584 · `false-claim`

- **Claim:** wf_ritual_honesty_test.go performs no section scoping; every one of its assertions is a flat body grep.
- **Measured:** The file scopes 12 times, through three helpers the gap's metric never counts: 5 `sectionUnder(` calls, 5 `headingIndexContaining(` calls (one of the 6 hits is the func declaration) and 2 `lineContaining(` calls. Exactly two of its assertions are body-scoped, and both carry an explicit written rationale (the `gitleaks detect` absence check, and a fence-truncation workaround documented in sectionUnder's own doc comment). `sectionUnder` is documented as 'Scopes an assertion to one section so a fact in an unrelated section … doesn't satisfy it.' The file's own header states the same. The gap's 'section-scoped' column equals the `extractMarkdownSection` count for every row, so the metric structurally cannot see this file's scoping — and it was already this way at the gap's filing commit (8 sectionUnder/lineContaining references at 134c33b62^).
- **Command:** `cd internal/policies && for f in wf_ritual_honesty_test.go wf_patch_changelog_test.go wf_structural_sweep_test.go wf_codebase_health_economy_test.go wf_patch_reconcile_test.go wf_rethink_wfpatch_xref_test.go; do printf "%-40s sectionUnder=%s lineContaining=%s headingIndexContaining=%s extractMarkdownSection=%s\n" "$f" "$(grep -o 'sectionUnder(' $f|wc -l)" "$(grep -o 'lineContaining(' $f|wc -l)" "$(grep -o 'headingIndexContaining(' $f|wc -l)" "$(grep -o 'extractMarkdownSection(' $f|wc -l)"; done`
- **Quoted:** “`wf_ritual_honesty_test.go` is the sharpest: sixteen substring assertions with no section scoping at all, so each proves only that a string exists somewhere in the file.”

#### G-0584 · `dead-premise`

- **Claim:** M-0308's cobra-tree derivation exists in the tree as the first worked instance a future author can copy.
- **Measured:** It was deleted. Commit 4a4019ec1 (2026-08-15, an ancestor of HEAD) withdrew both the wf-measure-spec SKILL.md and internal/policies/wf_measure_spec_test.go — the file that carried `findAllVerbs` + `backtickedAiwfMentions` derivation. The wf-rituals plugin directory no longer contains wf-measure-spec, and the parent epic E-0085 and sibling M-0309 are both `cancelled`. A reader sent to M-0308 for the exemplar finds nothing.
- **Command:** `git log --oneline --all -- internal/policies/wf_measure_spec_test.go; git merge-base --is-ancestor 4a4019ec1 HEAD && echo ancestor; ls internal/skills/embedded-rituals/plugins/wf-rituals/skills/; /…/audit/aiwf show E-0085`
- **Quoted:** “M-0308 builds that derivation for its own criteria and is the first instance of it.”

#### docs/architecture.md · `contradicts-adr`

- **Claim:** Ritual skills are authored outside this repo, in the ai-workflow-rituals repo.
- **Measured:** ADR-0016 (status: accepted) retires that upstream as an authoring channel and makes internal/skills/embedded-rituals/ THE canonical authoring location; the directory exists in-tree, and the vendoring apparatus the old model required (rituals.lock, scripts/sync-rituals.sh) is gone. CLAUDE.md §'Ritual content authoring' says the same. A contributor following this line goes looking for a repo that is archived.
- **Command:** `ls internal/skills/embedded-rituals/plugins/; ls rituals.lock scripts/sync-rituals.sh; sed -n '1,5p;33,40p' docs/adr/ADR-0016-retire-ai-workflow-rituals-upstream-channel-embedded-snapshot-canonical.md`
- **Quoted:** “A skill that wraps multiple aiwf verbs into a ritual (planning, wrap-epic, record-decision) → **not** part of aiwf core; lives in the rituals plugin (`ai-workflow-rituals` repo).”

#### docs/architecture.md · `contradicts-code`

- **Claim:** verb.Apply executes moves by shelling out to git mv.
- **Measured:** Apply's own doc comment says the opposite and the implementation is os.Rename. gitops has a git-mv wrapper but Apply does not call it.
- **Command:** `sed -n '15,40p' internal/verb/apply.go; grep -n 'os.Rename' internal/verb/apply.go`
- **Quoted:** “— runs every OpMove via `git mv`”

#### docs/design/design-decisions.md · `false-claim`

- **Claim:** `aiwf_version` is a REQUIRED aiwf.yaml field, and an aiwf.yaml consisting solely of `aiwf_version: 0.1.0` is the typical correct file.
- **Measured:** `aiwf_version` is a deprecated legacy key. It is captured into `Config.LegacyAiwfVersion`, ignored for all purposes, and actively STRIPPED from aiwf.yaml on every `aiwf update`. `aiwf doctor` no longer warns on version mismatch — it emits a deprecation note telling the operator to remove the key. This repo's own aiwf.yaml carries no such key. A reader following this table authors a file whose only line the next `aiwf update` deletes.
- **Command:** `grep -n 'LegacyAiwfVersion' -B 15 internal/config/config.go | head -30 ; grep -n 'func StripLegacyAiwfVersion' internal/config/config.go ; grep -rn 'aiwf_version' internal/cli/doctor/doctor.go ; head -5 /workspaces/aiwf/aiwf.yaml`
- **Quoted:** “| `aiwf_version` | string | yes | Engine version the repo expects (e.g., `0.1.0`). `aiwf doctor` warns on mismatch. |   ...   ```yaml aiwf_version: 0.1.0 ```  That's the entire file in normal use.”

#### docs/design/design-decisions.md · `contradicts-adr`

- **Claim:** Kernel id formats are mixed-width: 2 digits for epic, 3 for milestone/gap/decision/contract, 4 for ADR.
- **Measured:** Every kernel id kind canonicalizes to 4 digits (ADR-0008, status `accepted`). `aiwf schema` reports E-NNNN, M-NNNN, G-NNNN, D-NNNN, C-NNNN, ADR-NNNN. ADR-0008's Context section quotes this exact table as the superseded state it replaces. On-disk paths confirm it. A narrow id in the active tree is an error-severity `entity-id-narrow-width` finding.
- **Command:** `/tmp/.../audit/aiwf schema ; head -30 docs/adr/ADR-0008-canonicalize-kernel-ids-to-4-digits.md ; ls work/epics/ | head -3 ; grep -n 'CodeEntityIDNarrowWidth' internal/check/entity_id_narrow_width.go`
- **Quoted:** “| Epic | `proposed`, `active`, `done`, `cancelled` | `E-NN` | | Milestone | `draft`, `in_progress`, `done`, `cancelled` | `M-NNN` | | ADR | ... | `ADR-NNNN` | | Gap | `open`, `addressed`, `wontfix` | `G-NNN` | | Decision | ... | `D-NNN` | | Contract | ... | `C-NNN` |”

#### docs/design/design-decisions.md · `phantom-flag`

- **Claim:** `aiwf upgrade` has a `--yes` flag, and there is a confirmation prompt for it to skip.
- **Measured:** `aiwf upgrade --help` lists exactly three flags: `--check`, `--root`, `--version`. There is no `--yes`, and there never was — `git log -S'--yes' -- internal/ cmd/` returns nothing. There is also no confirmation prompt in `internal/cli/upgrade/` for such a flag to skip.
- **Command:** `/tmp/.../audit/aiwf upgrade --help ; git log --oneline -S'--yes' -- internal/ cmd/ ; grep -rn -i 'confirm|prompt' internal/cli/upgrade/*.go`
- **Quoted:** “`--check` reports the comparison without installing; `--yes` skips the confirmation prompt.”

#### docs/design/design-decisions.md · `already-removed`

- **Claim:** `aiwf.yaml` has a live `doctor.recommended_plugins` field driving a `recommended-plugin-not-installed` warning.
- **Measured:** The whole feature was removed on 2026-05-31 (commit debf7f5ff, 'chore: remove marketplace-plugin transitional machinery (G-0194)'). `Config` has no `doctor` field, `recommended_plugins`/`RecommendedPlugins` appear nowhere in `internal/config/`, and no `recommended-plugin-not-installed` finding exists. `internal/policies/m0202_devcontainer_onboarding.go` calls it 'the retired aiwf doctor warning ... that no longer exists' and treats any reappearance of the string as a policy violation. Because config.Load is non-strict at the top level, a reader who sets `doctor:` gets no error and no effect.
- **Command:** `grep -rn 'recommended_plugins|RecommendedPlugins' internal/config/*.go ; grep -rn 'yaml:"doctor' internal/ --include='*.go' ; git log --oneline -S'RecommendedPlugins' | head -3 ; sed -n 36,48p internal/policies/m0202_devcontainer_onboarding.go ; /tmp/.../audit/aiwf doctor --root /workspaces/aiwf`
- **Quoted:** “| `doctor` | mapping | no | `recommended_plugins` (list of `<name>@<marketplace>` strings, default empty) — Claude Code plugin identifiers the consumer expects to be installed for this repo's project scope. `aiwf doctor` reads `<rootDir>/.claude/settings.json`'s `enabledPlugins` map ... and emits on”

#### docs/design/design-decisions.md · `stale-enumeration`

- **Claim:** That 14-item list is the current set of commit-producing verbs, and `render --write` is one of them.
- **Measured:** Eleven commit-producing verbs are missing: `retitle`, `edit-body`, `archive`, `rename-area`, `set-area`, `set-priority`, `authorize`, `acknowledge illegal`, `acknowledge mistag`, `milestone tdd`, `milestone depends-on`. And `render --write` is wrongly included — G-0350 decoupled it into a plain write-only file operation that commits nothing, as `aiwf --help` itself now states. A reader reasoning about which operations are auditable via trailers, or which need a gate, gets both errors.
- **Command:** `grep -rn 'AcquireRepoLock' internal/cli/ --include='*.go' | grep -v _test ; sed -n 243,247p internal/cli/integration/trailer_shape_test.go ; grep -n 'render roadmap' internal/cli/root.go`
- **Quoted:** “Every mutating verb produces exactly one git commit, or no change at all. The current set: `add`, `promote`, `cancel`, `rename`, `move`, `reallocate`, `import`, `update`, `render --write`, `init`, `contract bind`, `contract unbind`, `contract recipe install`, `contract recipe remove`.”

#### docs/design/design-decisions.md · `dead-premise`

- **Claim:** The doc lives on a throwaway PoC branch, separate from `main`, which withholds the research documents and will never merge back.
- **Measured:** The doc IS on `main` (`git rev-parse --abbrev-ref HEAD` → `main`; the file's most recent commits are on main), and `docs/research/` is present in this very tree with all ten research documents. Both halves of the framing sentence are false at HEAD, as is 'the branch is not planned to merge back'. The whole doc — its 'PoC' title, its I1/I2/I2.5/I3 iteration section labels, its 'On future versions' section — is a pre-migration snapshot promoted into the Normative tier, where CLAUDE.md tells readers to treat it as current truth.
- **Command:** `git rev-parse --abbrev-ref HEAD ; ls docs/research/ ; git log --oneline --diff-filter=A -- docs/design/design-decisions.md`
- **Quoted:** “The full research arc that produced these decisions lives on `main`; this branch deliberately does not include those documents to keep the working context focused.   ...   The PoC is deliberately discardable. The branch is not planned to merge back to `main`.”

#### docs/design/design-decisions.md · `missing-commitment`

- **Claim:** By omission, that the doc's list of what the framework commits to is complete.
- **Measured:** CLAUDE.md commitment #10 — 'Uniform archive convention for terminal-status entities (ADR-0004)' — is absent from the doc entirely. ADR-0004 is `accepted`, `aiwf archive` is a shipped top-level verb, `work/epics/archive/` and `work/gaps/archive/` exist on disk, and `archive-sweep-pending` / `terminal-entity-not-archived` findings fire on this repo's own tree right now. A reader who takes this doc as the distillation of kernel commitments will not know terminal entities move at all, and the doc's own contrary statement ('Removals are not deletions ... The file stays.') reads as complete when it is not.
- **Command:** `grep -ci 'ADR-0004' docs/design/design-decisions.md ; /tmp/.../audit/aiwf show ADR-0004 ; ls work/epics/ | head -1 ; /tmp/.../audit/aiwf check --root /workspaces/aiwf --format json | head -c 600`
- **Quoted:** “(the entire document — the string `ADR-0004` and the archive convention appear nowhere; the only occurrences of `archive` are inside `../archive/pocv3/…` link paths)”

#### docs/design/design-decisions.md · `false-claim`

- **Claim:** `aiwf rename` on a bare id updates the entity's title, and does so via `git mv`.
- **Measured:** Both halves are false. (a) `verb.Rename`'s bare-id path never touches `Title` — the only `Title` occurrence in rename.go is inside a doc comment. Titles are changed by `aiwf retitle`, a separate verb. (b) No aiwf verb calls `git mv` at runtime: `gitops.Mv` is documented as test/porcelain-only and is on the forbidden-API list `internal/policies/verbs_validate_then_write.go` AST-bans from any exported `internal/verb` function; writes go through `gitops.CommitTree`'s plumbing path (read-tree → update-index → write-tree → commit-tree → update-ref). The same false `git mv` mechanism is repeated in the `reallocate` bullet ('it picks the next free id ..., `git mv`s, ...') and in the one-commit-per-verb section ('writes files (and `git mv`s) and creates the commit').
- **Command:** `grep -n 'Title' internal/verb/rename.go ; sed -n 70,80p internal/gitops/gitops.go ; grep -n 'Mv' internal/policies/verbs_validate_then_write.go ; sed -n 21,36p internal/gitops/committree.go`
- **Quoted:** “Renames preserve the id: `aiwf rename <id> <new-slug>` does `git mv` plus a title update.”

#### docs/design/design-lessons.md · `false-claim`

- **Claim:** A principle named 'immutability of done' is stated in docs/architecture.md and in the root CLAUDE.md, and is the parent rule that absorbed the reversal principle.
- **Measured:** Neither docs/architecture.md nor CLAUDE.md contains the string, case-insensitively, at all. The only normative-tree occurrence is in docs/archive/architecture.md — the Archival tier, which CLAUDE.md's Documentation hierarchy declares a frozen historical snapshot. A reader sent to look it up in a current-truth doc finds nothing.
- **Command:** `grep -rn "immutability of done" . --exclude-dir=.git --exclude-dir=.claude ; grep -in "immutab" docs/architecture.md CLAUDE.md`
- **Quoted:** “On reflection this is a tightening of the existing **"immutability of done"** principle (in `architecture.md` and the root `CLAUDE.md`), not a separate architectural rule.”

#### docs/design/design-lessons.md · `false-claim`

- **Claim:** The blockquote reproduces CLAUDE.md's verb-design checklist item, and aiwf has verbs set-status, complete, and hotfix.
- **Measured:** None of the three is a verb. CLAUDE.md's actual 'Designing a new verb' section lists entirely different examples ('another invocation of the same verb; an explicit terminal transition (`aiwf cancel`, `aiwf reallocate`); "you can't, deliberately" (`aiwf init` is one-shot); "you'd open a new entity for the inverse"'). The blockquote is a stale copy of a superseded draft of the section it purports to quote, and the two documents cite each other reciprocally, so a reader lands here from CLAUDE.md and learns three phantom verbs.
- **Command:** `for v in hotfix complete set-status; do /tmp/claude-1000/-workspaces-aiwf/ec5d1c3d-5669-4772-9561-4542e8aaef62/scratchpad/audit/aiwf $v; done ; sed -n '334,337p' CLAUDE.md`
- **Quoted:** “If the answer is "another verb of the same kind that supersedes" (e.g., another `set-status`), good. ... If the answer is "you can't, and that's deliberate — here's why" (e.g., `complete` is terminal; defects spawn a new entity via `hotfix`), good.”

#### docs/design/growth.md · `already-shipped`

- **Claim:** None of the four levers in the Levers table has shipped.
- **Measured:** The first lever row — 'a cheap-fix escape in the wrap ritual's deferral rule' — shipped on 2026-08-02 in commit 18f6b54b6, and that commit is an ancestor of growth.md's own most recent edit (b98549789, 2026-08-08). The escape is live at HEAD in the always-on guidance fragment and in both milestone rituals.
- **Command:** `git log --format='%h %ad %s' --date=short -S 'cheap-fix test' -- internal/skills/embedded-guidance/aiwf-guidance.md ; git merge-base --is-ancestor 18f6b54b6 b98549789 && echo ANCESTOR ; grep -rn 'cheap-fix' internal/skills/ | head`
- **Quoted:** “Cost measured, not estimated. None of these is currently shipped.”

#### docs/design/legal-workflows-audit.md · `stale-status-header`

- **Claim:** The doc is the in-flight working artifact of an in-progress milestone M-0121, with Pass B (M-0122) and Pass C (M-0123) still ahead of it.
- **Measured:** M-0121, M-0122 and M-0123 are all `done` and archived under E-0033, which is itself `done` and archived. Nothing about this document is in progress. A reader of a Normative-tier doc is told the surrounding work is live and that a later pass will reconcile the content — both false.
- **Command:** `/tmp/.../audit/aiwf show M-0121 --format=json  (and M-0122, M-0123, E-0033)`
- **Quoted:** “> **Methodology:** ADR-0011. **Milestone:** M-0121. **Status:** in_progress.”

#### docs/design/legal-workflows-audit.md · `dead-premise`

- **Claim:** R-AUDIT-0201/0202 (and R-RULE-141) describe a live cross-repo workflow: SKILL.md fixtures under `internal/policies/testdata/`, a drift-check test comparing them against `~/.claude/plugins/cache/ai-workflow-rituals/.../SKILL.md`, sourced from a CLAUDE.md section `§Cross-repo plugin testing`.
- **Measured:** All three legs are gone. No SKILL.md fixture exists under `internal/policies/testdata`; CLAUDE.md has no `Cross-repo plugin testing` section; the marketplace cache channel was retired by ADR-0014 (accepted) and ADR-0016 (accepted), and CLAUDE.md now says rituals are authored directly at `internal/skills/embedded-rituals/plugins/<plugin>/skills/<skill>/SKILL.md` with AC tests asserting against the embedded bytes. A milestone author following R-AUDIT-0201 would author the deliverable in a directory that does not exist and never ship it.
- **Command:** `find internal/policies/testdata -name 'SKILL.md' ; grep -n 'Cross-repo plugin testing' CLAUDE.md ; /tmp/.../audit/aiwf show ADR-0014 ; ... ADR-0016`
- **Quoted:** “When a milestone's deliverable is a `SKILL.md` in the rituals plugin repo, the canonical authoring location during the milestone is a fixture in this repo at `internal/policies/testdata/<skill-name>/SKILL.md`. AC tests assert content claims against the fixture; deployment to the rituals repo happens”

#### docs/design/legal-workflows-audit.md · `broken-reference`

- **Claim:** The doc's own Schema declares Citation is `file.go:line`. §1 pins 49 rules to specific lines in `internal/entity/transition.go` (L14, L15, L16-17, L20-23, L26-29, L32-34, L37-40, L43-47, L9-11, L64-82, L93-103, L110-120, L128-131, L137-146, L122-126, L160-164, L153-158, L193-203).
- **Measured:** Every one is wrong. The FSM table entries are off by 5 (epic `proposed` is L19 not L14; `active` is L20 not L15); the function citations are off by 25-130 lines: `ValidateTransition` is L80 (cited L64-82), `IsTerminal` is L129 (cited L93-103), `CancelTarget` is L170 (cited L110-120), `acTransitions` is L214 (cited L128), `IsLegalACTransition` is L226 (cited L137-146), `tddPhaseTransitions` is L263 (cited L160), `MilestoneCanGoDone` is L292 (cited L193-203). §10 disagrees with §1 on the same symbol: R-AUDIT-0031 cites CancelTarget at L110-120 while R-RULE-021 cites L142-191; neither is right.
- **Command:** `grep -n 'KindEpic: {|"proposed":  {"active"|var transitions|func ValidateTransition|func IsTerminal|func CancelTarget|var acTransitions|func IsLegalACTransition|var tddPhaseTransitions|func MilestoneCanGoDone' internal/entity/transition.go`
- **Quoted:** “| R-AUDIT-0001 | transition.go | L14 | Epic | `promote E-NNNN proposed → active` is legal | hard-reject |”

#### docs/design/legal-workflows-audit.md · `broken-reference`

- **Claim:** §3 cites 23 check-rule functions by the name `run<Something>` — runACsTDDAudit, runACsShape, runACsBodyCoherence, runACsTitleProse, runMilestoneDoneACs, runArchivedEntityTerminal, runTerminalEntityArchive, runArchiveSweepPending, runEntityBodyEmpty, runEntityIDNarrowWidth, runEpicActiveNoDrafts, runFrontmatterShape, runGapResolvedHasResolver, runIDsUnique, runStatusValid, runTitlesNonempty, runRefsResolve, runNoCycles, runIDPathConsistent, runUnexpectedTreeFile, runADRSupersession, runCasePaths, runProvenance. §10's Chokepoints column repeats them.
- **Measured:** There is no function whose name starts with `run` anywhere in `internal/check`. The real names carry no prefix: `acsTDDAudit`, `acsShape`, `acsBodyCoherence`, `acsTitleProse`, `milestoneDoneIncompleteACs`, `archivedEntityNotTerminal`, `terminalEntityNotArchived`, `archiveSweepPending`, `entityBodyEmpty`, `entityIDNarrowWidth`, `epicActiveNoDraftedMilestones`, `frontmatterShape`, `gapAddressedHasResolver`, `idsUnique`, `statusValid`, `titlesNonempty`, `refsResolve`, `noCycles`, `idPathConsistent`, `TreeDiscipline`, `adrSupersessionMutual`, `casePaths`, `RunProvenance`. A reader grepping any §3 citation finds nothing.
- **Command:** `grep -rn 'func run' internal/check/*.go | grep -v _test  ;  grep -rn '^func ' internal/check/acs.go internal/check/check.go internal/check/archive_rules.go`
- **Quoted:** “| R-AUDIT-0073 | check/acs.go | runACsTDDAudit | ... | R-AUDIT-0088 | check/check.go (frontmatter validators) | runFrontmatterShape | ... | R-AUDIT-0093 | check/check.go | runRefsResolve |”

#### docs/design/legal-workflows-audit.md · `false-claim`

- **Claim:** `aiwf init` hard-rejects on a repo that already carries an `aiwf.yaml`. R-RULE-101 restates it verbatim.
- **Measured:** The opposite. `aiwf init` is explicitly documented as idempotent and safe to re-run: it preserves an existing `aiwf.yaml` verbatim and refreshes only derived artifacts. There is no refusal and no hard-reject. An operator told `init` refuses would reach for a workaround that is not needed, or would wrongly conclude a re-run is destructive.
- **Command:** `/tmp/.../audit/aiwf init --help ; grep -rn 'idempot' internal/cli/initcmd/initcmd.go`
- **Quoted:** “| R-AUDIT-0125 | internal/cli/initcmd/initcmd.go | init verb | `aiwf init` | One-time setup; refuses to run on a repo that already has `aiwf.yaml` (no overwrite-existing behavior) | hard-reject |”

#### docs/design/legal-workflows-audit.md · `retired-finding-code`

- **Claim:** The kernel emits a finding code `gap-resolved-has-resolver`. R-RULE-037 repeats it.
- **Measured:** That code was renamed to `gap-addressed-has-resolver` by M-0142, a documented breaking change to the `aiwf check --format=json` `findings[].code` surface. The frontmatter field named in the statement is also wrong: the rule reads `addressed_by:` / `addressed_by_commit:`, not `resolved-by:`. A downstream reader pinning the literal from this Normative doc gets a code the kernel never emits.
- **Command:** `grep -rn 'CodeGapAddressedHasResolver' internal/check/check.go ; grep -n 'gap-resolved-has-resolver' CHANGELOG.md`
- **Quoted:** “| R-AUDIT-0089 | check/check.go | runGapResolvedHasResolver | Gap | A gap with `status: addressed` must have a `resolved-by:` frontmatter field pointing to an entity; missing fires `gap-resolved-has-resolver` | check-error |”

#### docs/design/legal-workflows-audit.md · `internal-contradiction`

- **Claim:** Promoting an epic `proposed → active` requires `--force --reason`.
- **Measured:** False, and contradicted by two other rows of the same document. R-AUDIT-0113 and R-RULE-078 both say a human reaches this transition with no flag at all, and that `--force` does not substitute. The code agrees with those rows: `requireHumanActorForSovereignAct` returns nil on any `human/` actor prefix and its error message deliberately does not offer `--force`. R-AUDIT-0050's actual subject is narrower — it forbids *automation-shaped source files* (CI, scripts, Makefiles) from containing the unforced invocation — which R-RULE-001's Note misreads into a universal flag requirement.
- **Command:** `sed -n '32,45p' internal/verb/promote_sovereign_act.go`
- **Quoted:** “| R-RULE-001 | FSM | Epic `proposed` | ... | `proposed → active` is sovereign-act (human-only, requires `--force --reason`) per R-AUDIT-0050 |”

#### docs/design/legal-workflows-audit.md · `stale-claim`

- **Claim:** The Revision-2 preamble reports a live open bug in `CancelTarget`, tracked as G-0129 and scheduled as M-0127.
- **Measured:** Three things wrong at once. (a) `CancelTarget(k Kind, currentStatus Status)` is state-aware today — it branches on `currentStatus` for ADR/Decision and Contract. (b) G-0129 is not that gap; it is `Typed finding-code constants mechanically enforced at comparison sites`, and it is addressed. (c) M-0127 is `Relocate docs/pocv3/ contents and sweep cross-references`, unrelated. The correct entities are G-0131 and M-0131 — which R-RULE-021 names correctly 74 lines later, so the document disagrees with itself about which gap this was.
- **Command:** `sed -n '170,190p' internal/entity/transition.go ; /tmp/.../audit/aiwf show G-0129 ; ... M-0127 ; ... G-0131 ; ... M-0131`
- **Quoted:** “Note: the current code's `CancelTarget` is *not* state-aware — this is a **real bug** tracked as **G-0129, scheduled as M-0127** in E-0033.”

#### docs/design/legal-workflows-audit.md · `stale-claim`

- **Claim:** R-AUDIT-0146 and R-RULE-147 both report ADR-0009 as `proposed`, i.e. a live design awaiting ratification, and carry it in §10.11 "Future / deferred".
- **Measured:** ADR-0009 is `rejected`. It is not future work; the decision went the other way. A reader planning against §10.11 would build toward a rejected design.
- **Command:** `/tmp/.../audit/aiwf show ADR-0009 --format=json`
- **Quoted:** “Driver and substrate are separated; events are trailer-only (no separate event log). **Note:** ADR-0009 is `proposed` — currently aspirational”

#### docs/design/legal-workflows-first-principles.md · `false-claim`

- **Claim:** A contract in `accepted` cannot transition to `rejected`; a violation is an error-severity finding.
- **Measured:** `accepted → rejected` IS legal for contracts, in both the FSM and the canonical spec table, and the asymmetry with ADR is deliberate per decision D-0002 ('Contract accepted->rejected is legal: operational kinds admit abrupt-stop', status accepted).
- **Command:** `sed -n '46,52p' internal/entity/transition.go; grep -n 'Q4 / D-0002' -A 8 internal/workflows/spec/rules.go; /tmp/.../audit/aiwf show D-0002 --format=json`
- **Quoted:** “| R-FP-0045 | contract FSM | `accepted → rejected` is not legal. | Mirror of R-FP-0021 — rejection is a pre-acceptance terminal. | conventional | error |”

#### docs/design/legal-workflows-first-principles.md · `false-claim`

- **Claim:** A contract may go straight from `accepted` to `retired`, skipping `deprecated`.
- **Measured:** `retired` is unreachable from `accepted`. The FSM's outgoing set for contract `accepted` is exactly {deprecated, rejected}; `retired` is reachable only from `deprecated`. No cell in spec.Rules() encodes an accepted→retired transition either.
- **Command:** `sed -n '46,52p' internal/entity/transition.go; grep -n 'KindContract' -A 12 internal/workflows/spec/rules.go`
- **Quoted:** “| R-FP-0038 | contract FSM | `accepted → retired` is legal. | A contract can be retired directly without a deprecation period (e.g., an experimental binding that didn't pan out). Less common path but the model's three-terminal structure admits it. | conventional | error |”

#### docs/design/legal-workflows-first-principles.md · `dead-premise`

- **Claim:** Fifteen enumerated questions (Q1..Q15) are still open and awaiting Pass C; the closing line reinforces it in the future tense — 'Pass C (M-0123) reconciles this against Pass A's legal-workflows-audit.md and produces the canonical Go spec table.'
- **Measured:** Pass C ran and closed. M-0123 is `done`. Six decision entities were produced (D-0002..D-0007; five accepted, D-0005 superseded), and internal/workflows/spec/rules.go cites the Q-numbers by name as resolved cells. A Normative-tier reader is entitled to treat this doc as current truth and would conclude the legality model is still undecided on 15 points.
- **Command:** `/tmp/.../audit/aiwf show M-0123 --format=json; grep -n 'Q1\b\|Q2\b\|Q3\b\|Q4 /\|Q5 /\|Q6 /\|Q8\b\|Q15 /' internal/workflows/spec/rules.go`
- **Quoted:** “## Open questions for Pass C  These are points where first-principles reasoning was ambiguous or where multiple equally-defensible derivations are possible. Pass C (M-0123) reconciles each against Pass A's catalog and produces an explicit decision entity per unresolved case.”

#### docs/design/legal-workflows-first-principles.md · `false-claim`

- **Claim:** The kernel enforces nothing about which branch a verb runs on — a contributor is told this is deliberately unpoliced.
- **Measured:** `aiwf authorize` has three branch-context preflight refusals in the kernel (branch-context-required, branch-not-found, rung-pair-illegal), and an entire layer-4 branch-choreography spec sub-package exists (internal/workflows/spec/branch/) with named cells. M-0158 ('Layer-4 branch-choreography spec cells + drift-policy extension') is done.
- **Command:** `grep -rn 'branch-context-required\|rung-pair-illegal\|branch-not-found' internal/ --include='*.go' | grep -v _test; ls internal/workflows/spec/branch/; /tmp/.../audit/aiwf show M-0158 --format=json`
- **Quoted:** “| R-FP-0172 | anti-rule | There is **no** kernel rule about which branch a verb is legal on. Branch choreography is ADR-0010's layer 4, out of E-0033's scope. | Codified in ADR-0010 and ADR-0011 §Scope. | load-bearing | n/a |”

#### docs/design/legal-workflows-first-principles.md · `false-claim`

- **Claim:** Cancelling an epic cascades termination to its non-terminal milestones.
- **Measured:** The opposite: `aiwf cancel` on an epic with non-terminal children REFUSES and lists them, per D-0003 ('Epic cancel refuses with listing when any milestone is non-terminal', accepted). Nothing cascades. An operator acting on this row would expect their in_progress milestones to be closed for them.
- **Command:** `grep -rn 'EpicCancelNonTerminalChildrenError' internal/verb/cancel*.go; grep -n 'func (e \*EpicCancelNonTerminalChildrenError) Error' -A 6 internal/verb/cancel_guards.go; /tmp/.../audit/aiwf show D-0003 --format=json`
- **Quoted:** “| R-FP-0074 | milestone × epic | `aiwf cancel` on an epic implicitly cancels (or otherwise terminalizes) its non-terminal milestones. **Marked conventional** because the exact cascade mechanism is a Pass C decision point. |”

#### docs/design/legal-workflows-first-principles.md · `false-claim`

- **Claim:** Cancelling a milestone cascades termination to its open ACs.
- **Measured:** The opposite: `aiwf cancel` on a milestone with any open AC refuses (milestone-cancel-non-terminal-acs), per D-0004 ('Milestone cancel refuses with listing when any AC is non-terminal', accepted). The spec encodes it as two Illegal cells with RejectionLayerVerbTime.
- **Command:** `grep -rn 'MilestoneCancelNonTerminalACs' internal/verb/cancel*.go; grep -n 'Q6 / D-0004' -A 12 internal/workflows/spec/rules.go; /tmp/.../audit/aiwf show D-0004 --format=json`
- **Quoted:** “| R-FP-0064 | AC × milestone | An AC's lifecycle is bounded by its parent milestone's lifecycle: cancelling the milestone implicitly cancels all of its ACs (or terminates them otherwise — the model doesn't pin the exact mechanism, but ACs cannot outlive the milestone). |”

#### docs/design/performance.md · `dead-premise`

- **Claim:** G-0340 is a live deferred lever, item 2 in the recommended bang-for-buck sequence; the doc also says 'deferred to G-0340' twice more (class 1 and the commit-graph section).
- **Measured:** G-0340 is status wontfix, archived, cancelled 2026-07-03 with a recorded reason that explicitly declines the lever on YAGNI grounds. performance.md was last committed 2026-08-04, a month after the cancellation, so the pointer was already dead at the doc's last edit.
- **Command:** `/tmp/.../audit/aiwf show G-0340 --format=json  |  git log --all --format='%H %ad%n%B---' --date=short --grep='aiwf-entity: G-0340'`
- **Quoted:** “**Path-scoped history + maintained changed-path bloom filters** (deferred, G-0340) — per-entity history ~1.3s → ~65ms”

#### docs/design/performance.md · `dead-premise`

- **Claim:** Four named gaps are live deferred levers 'carried forward'. Item 3 of the recommended sequence repeats G-0323 as the 'flat regardless of size' tier.
- **Measured:** Two of the four are wontfix and archived. G-0323 was cancelled 2026-07-03 as unsound ('Naive watermark is unsound... breaks the check-is-authoritative chokepoint'), G-0325 cancelled the same day as 'Second-order and partly redundant with the declined watermark lever'. Only G-0324 and G-0328 are still open.
- **Command:** `for g in G-0323 G-0324 G-0325 G-0328; do /tmp/.../audit/aiwf show $g --format=json | python3 -c "import sys,json;r=json.load(sys.stdin)['result'];print(r['status'],r['path'])"; done`
- **Quoted:** “Deferred levers carried forward (profiled, unstarted): **G-0323** incremental `aiwf check` via a validated watermark; **G-0324** branch hygiene; **G-0325** parallelize independent history walks + blob reads; **G-0328** golden-fixture byte-identity comparator”

#### docs/design/performance.md · `false-claim`

- **Claim:** This repo's commit-graph carries no changed-path bloom filters — restated at length in the '## commit-graph + changed-path bloom filters' section ('git's default gc writes the base graph but **not** bloom filters; you opt in explicitly'), and load-bearing for recommended-sequence item 2.
- **Measured:** Every graph file in the split commit-graph chain carries BIDX and BDAT chunks — the changed-path bloom filters are present. git maintenance is enabled (maintenance.auto true) and has been writing them.
- **Command:** `python3 -c "import struct,glob;\nfor p in sorted(glob.glob('.git/objects/info/commit-graphs/graph-*.graph')):\n b=open(p,'rb').read(4096); n=b[6]; print(p,[b[8+i*12:12+i*12].decode('latin1') for i in range(n+1)])"  ;  git config --get-regexp 'maintenance|commitGraph|gc\.'`
- **Quoted:** “| commit-graph present | yes (base chunks only — **no** changed-path bloom filters) |”

#### docs/design/performance.md · `stale-count`

- **Claim:** Baseline repo dimensions (labelled 'Recorded on the kernel repo itself, 2026-07-01') that the whole cost model is scaled against.
- **Measured:** commits 9,980 (+81%); refs 55 total / 6 local heads / 3 remotes (local branches down 8x); entity markdown under work/ 1,103 (1,147 incl. docs/adr); .git 147 MB. Only the git version (2.54.0) still matches.
- **Command:** `git rev-list --count HEAD ; git for-each-ref | wc -l ; git for-each-ref refs/heads | wc -l ; git for-each-ref refs/remotes | wc -l ; find work -name '*.md' | wc -l ; du -sh .git ; git --version`
- **Quoted:** “| commits (HEAD) | 5,510 | | refs total / local branches / remote refs | 90 / 48 / 10 | | entity markdown files | 652 | | `.git` size | 217 MB |”

#### docs/design/performance.md · `false-claim`

- **Claim:** Present-tense: the repo carries 48 local branches, most of them merged cruft, so branch hygiene is an available cheap win (recommended-sequence item 0, 'Free; shrinks allocator + check fan-out immediately').
- **Measured:** 6 local branches, all live work (main, two epic branches, two milestone branches, one initiative branch). The prune half of the lever has already been realised; only the 'oracle skips merged refs' half of G-0324 is left, and the O(refs) cost class is 8x smaller than the doc's arithmetic assumes.
- **Command:** `git for-each-ref --format='%(refname:short)' refs/heads`
- **Quoted:** “48 local branches. The allocator (`internal/trunk`) enumerates `refs/heads/*` and `refs/remotes/*` ... Many of those 48 branches are almost certainly merged ritual/epic branches. **Branch hygiene** (prune merged branches; have the oracle skip merged refs — G-0324) is a cheap partial win”

#### docs/design/provenance-model.md · `phantom-finding-code`

- **Claim:** Two named kernel finding codes are emitted when --pause/--resume find no scope in the required state.
- **Measured:** Neither string exists anywhere in the module. Both refusals are bare fmt.Errorf format strings carrying no code at all.
- **Command:** `grep -rn 'provenance-no-active-scope-to-pause\|provenance-no-paused-scope-to-resume' internal/ cmd/ ; sed -n '296,310p' internal/verb/authorize.go`
- **Quoted:** “**`--pause "<reason>"`**: pauses the *most-recently-opened active scope* for `<id>`. If none active, refuses with `provenance-no-active-scope-to-pause`. Reason required, non-empty. - **`--resume "<reason>"`**: resumes the *most-recently-paused scope* for `<id>`. If none paused, refuses with `provena”

#### docs/design/provenance-model.md · `contradicts-accepted-decision`

- **Claim:** Scope reachability traverses the full frontmatter reference graph, including depends_on, addressed_by and relates_to.
- **Measured:** D-0006 (status accepted) restricts reachability to exactly three edges and names depends_on / addressed_by / relates_to as explicitly NOT reachability edges. The implementation follows D-0006, not this doc.
- **Command:** `sed -n '180,207p' internal/verb/allow.go ; <binary> show D-0006`
- **Quoted:** “An entity E reaches scope-entity S iff there exists a chain of frontmatter references (`parent`, `depends_on`, `addressed_by`, `relates_to`, `discovered_in`, etc., as defined in the schema) from E to S. The chain is bounded by the existing kind reference grammar.”

#### docs/design/tree-discipline.md · `contradicts-adr`

- **Claim:** aiwf deliberately does not write a marker-managed block into a consumer's CLAUDE.md; the consumer's CLAUDE.md is theirs alone.
- **Measured:** ADR-0018 is accepted and decides the opposite: `aiwf init` and `aiwf update` maintain a marker-wrapped `@.claude/aiwf-guidance.md` import block in the consumer's root CLAUDE.md, default-on, opt-out only via `guidance.wire_claudemd: false`. The code implements it (`ensureGuidanceImport` in internal/initrepo/initrepo.go:884, called at :477) and this repo's own CLAUDE.md carries the block at lines 372-374. Two Normative-tier docs now disagree on a kernel commitment (CLAUDE.md commitment #5 states the shipped behavior).
- **Command:** `sed -n '1,5p' docs/adr/ADR-0018-risk-calibrated-consent-for-user-owned-file-edits.md; grep -rn "ensureGuidanceImport" internal/ --include='*.go' | grep -v _test; grep -n 'aiwf:guidance' CLAUDE.md`
- **Quoted:** “Earlier design rounds considered shipping a marker-managed block in the consumer's `CLAUDE.md` — same discipline as the marker-managed git hooks. We rejected it for two reasons: [...] **Consumer's `CLAUDE.md`** — *not* aiwf's responsibility.”

#### docs/design/tree-discipline.md · `false-claim`

- **Claim:** PathKind recognizes exactly the six listed active paths.
- **Measured:** PathKind also recognizes six archive shapes (one inserted `archive/` segment per kind, per ADR-0004), via `stripArchiveSegment`. This repo carries live archive dirs (work/{epics,decisions,gaps}/archive, docs/adr/archive) and archived entities resolve normally.
- **Command:** `sed -n '708,732p' internal/entity/entity.go; ls -d work/*/archive; /tmp/claude-1000/-workspaces-aiwf/ec5d1c3d-5669-4772-9561-4542e8aaef62/scratchpad/audit/aiwf show G-0001 --format=json`
- **Quoted:** “The loader recognizes exactly these shapes (per `entity.PathKind` → ../../internal/entity/entity.go): | epic | `work/epics/E-NN-<slug>/epic.md` | ... | ADR | `docs/adr/ADR-NNNN-<slug>.md` |”

#### docs/design/tree-discipline.md · `false-claim`

- **Claim:** Every non-entity file anywhere under work/ is reported as unexpected-tree-file.
- **Measured:** Stray tracking is scoped to four walk roots — work/epics, work/gaps, work/decisions, work/contracts. A file at the work/ root or under a non-entity sibling dir is never registered as a stray and never flagged. Measured on a fresh fixture: work/scratch.md and work/notes/n.md produced zero findings while work/gaps/notes.md produced one.
- **Command:** `aiwf check --shape-only --root <fixture> --format=json, with strays planted at work/scratch.md, work/notes/n.md and work/gaps/notes.md (fixture built in scratchpad, not in the repo); plus sed -n '280,290p' internal/tree/tree.go`
- **Quoted:** “Anything else under `work/` that is not a recognized entity file is a stray and surfaces as an `unexpected-tree-file` finding from `aiwf check`.”

#### docs/design/tree-discipline.md · `false-claim`

- **Claim:** The worked Configuration example shows allow_paths entries a consumer would need in order to exempt work/templates/*.md and work/scratch/**.
- **Measured:** Both example paths are outside the loader's walk roots, so neither can ever produce an unexpected-tree-file finding and neither needs allow-listing. Measured on a fixture with NO allow_paths configured at all: work/templates/t.md and work/scratch/deep/s.md both produced zero findings. A reader following this Normative example writes dead configuration.
- **Command:** `aiwf check --shape-only --root <fixture> --format=json, fixture carrying work/templates/t.md and work/scratch/deep/s.md and an aiwf.yaml with no tree: block`
- **Quoted:** “tree:\n  strict: true\n  allow_paths:\n    - work/templates/*.md\n    - work/scratch/**”

#### docs/overview.md · `stale-id-format`

- **Claim:** These are the id formats for five of the six kinds.
- **Measured:** ADR-0008 canonicalizes every kernel id to a uniform 4-digit emission width; the allocator zero-pads to CanonicalPad = 4, and every id in the live tree is 4-digit. Only the ADR row (`ADR-NNNN`) is still correct.
- **Command:** `grep -n 'CanonicalPad' internal/entity/allocate.go ; <binary> list --archived --format=json`
- **Quoted:** “| **Epic** | ... | `E-NN` | | **Milestone** | ... | `M-NNN` | | **Gap** | ... | `G-NNN` | | **Decision** | ... | `D-NNN` | | **Contract** | ... | `C-NNN` |”

#### docs/overview.md · `false-claim`

- **Claim:** Nothing finer-grained than a milestone is tracked by the framework.
- **Measured:** Acceptance criteria are framework-tracked sub-elements: addressed by composite id M-NNNN/AC-N, carrying their own status and TDD phase, promoted by `aiwf promote`, validated by an `aiwf check` rule. The 'no task/story kind' half is true; the 'nothing finer is tracked' half is not. The Milestone table row repeats it ('the smallest unit `aiwf` plans against').
- **Command:** `<binary> show M-0310`
- **Quoted:** “There is no `task` or `story` entity. The smallest unit `aiwf` tracks is a milestone — anything finer-grained lives in the milestone's body prose, in commits, or in the AI host's working memory, not in the framework.”

#### docs/skill-author-guide.md · `false-claim`

- **Claim:** aiwf check does not enforce the trailers; the only cost of omitting them is that `aiwf history` goes blind (skill-author-guide.md:154).
- **Measured:** `aiwf check` enforces them at error severity, and the pre-push hook therefore blocks. `provenance-untrailered-entity-commit` is constructed at check.SeverityError and wired into the `aiwf check` command path.
- **Command:** `grep -n 'CodeProvenanceUntrailedEntityCommit' internal/check/provenance.go; sed -n 580,590p internal/check/provenance.go; grep -n 'RunUntrailedAudit' internal/cli/check/provenance.go`
- **Quoted:** “Every aiwf-mutating commit must carry `aiwf-verb:`, `aiwf-entity:`, and `aiwf-actor:` trailers — `aiwf history` depends on them, and `aiwf check` does not enforce them at runtime.”

#### docs/skill-author-guide.md · `reserved-namespace`

- **Claim:** A skill author's own skill should be named `aiwfx-snapshot-status` (line 49), and the boilerplate template names skills `aiwfx-<verb>-<noun>` (line 166).
- **Measured:** `aiwfx-*` is aiwf's own materialized-ritual namespace, and `aiwf init`/`update` unconditionally writes `.claude/skills/aiwfx-*/` into the consumer's `.gitignore`. A consumer following the guide verbatim writes a skill that is silently never committed and is lost on a fresh clone; if aiwf ever ships a ritual of the same name, MaterializeWithTiers overwrites its SKILL.md.
- **Command:** `sed -n 665,672p internal/skills/skills.go; grep -n 'claude/skills' .gitignore`
- **Quoted:** “name: aiwfx-snapshot-status”

#### docs/skill-author-guide.md · `example-fails-as-written`

- **Claim:** Step 4 of the guide's single worked example creates a decision entity whose body is the scaffolded template, and the skill deliberately "does **not** write a body for the decision describing the snapshot text."
- **Measured:** Refused. `decision` is born-complete (G-0326), and `aiwf template decision` emits `## Question / ## Decision / ## Reasoning` with nothing under them, so `EmptyRequiredSections` returns all three and `requireNonEmptyBornCompleteBody` refuses. This is deeper than a missing flag: the skill's stated design — an empty body, with git log as the snapshot — is now illegal without `--force --reason`, so the example needs a new subject, not a new argument.
- **Command:** `go test -run 'TestAdd_EmptyBodyGate_BornCompleteKindsDefaultTemplateRefused' -count=1 -v ./internal/verb/`
- **Quoted:** “aiwf add decision --title "Status snapshot — 2026-05-01"”

#### docs/skill-author-guide.md · `false-claim`

- **Claim:** `aiwf render roadmap --write` produces a commit (cheat-sheet Commits column, line 28), and therefore writes the standard trailers automatically per the table's header sentence.
- **Measured:** It writes the file and never commits; the caller must stage and commit it.
- **Command:** `aiwf --help`
- **Quoted:** “| `aiwf render roadmap --write`    | Regenerate `ROADMAP.md`.                                                            | yes     |”

#### docs/skill-author-guide.md · `missing-verbs`

- **Claim:** The 15-row cheat-sheet is the set of verbs a skill may call (line 11).
- **Measured:** It omits 17+ shipped verbs/subverbs, including every one a body-touching skill needs. `edit-body`, `retitle`, `aiwf list`, `aiwf show` and `set-priority` appear ZERO times in the entire document — so Rule 1 ("if aiwf has a verb for what you want to do, call the verb … there is a verb for that") and Rule 3 (about body sections) never name the verb that edits a body. Also omitted: set-area, rename-area, archive, authorize, acknowledge illegal|mistag, milestone depends-on, milestone tdd, worktree add, render --format=html, contract verify|recipes|recipe show|recipe remove.
- **Command:** `grep -c 'edit-body' docs/skill-author-guide.md; grep -c 'aiwf list' docs/skill-author-guide.md; grep -c 'aiwf show' docs/skill-author-guide.md; grep -c 'retitle' docs/skill-author-guide.md; grep -c 'set-priority' docs/skill-author-guide.md`
- **Quoted:** “What a skill is allowed to call, what each verb does, and whether it produces a commit.”

#### docs/workflows.md · `false-claim`

- **Claim:** workflows.md:88 — epic roll-up is a deliberate non-guarantee; the only mechanism is an `aiwf check` warning on cancel.
- **Measured:** The kernel hard-refuses both `promote <E> done` and `cancel <E>` while any child milestone is non-terminal, and the standing check rule is error severity and covers `done` as well as `cancelled`. Three separate claims in this paragraph are false: (a) the transition is not allowed, (b) roll-up IS enforced, (c) the check finding is an error, not a warning, and is not the only nudge.
- **Command:** `cd <sandbox> && aiwf promote E-0001 done ; aiwf cancel E-0001 --reason test   (E-0001 active, M-0001 draft)`
- **Quoted:** “For `aiwf` an epic can transition to `done` from `active`, regardless of whether every milestone underneath is itself `done`. The framework deliberately does not enforce roll-up — it is descriptive of the team's choice, not prescriptive of when "done" is allowed. (`aiwf check` warns if you cancel an”

#### docs/workflows.md · `example-fails`

- **Claim:** workflows.md:64-65 (and :23, :126, :129, :217, :221) — the two activating promotes run back to back from whatever branch you are on; the doc never mentions branches anywhere.
- **Measured:** G-0269's branch guard refuses both. `promote <E> active` must land on the configured trunk short name (default `main`, derived from allocate.trunk `refs/remotes/origin/main`), and `promote <M> in_progress` must land on `epic/<epic-dir-name>`. They therefore cannot be run from the same branch, which is exactly what the doc shows. Worse, the doc's own step 0 is `git init -q` — on any machine where git's default branch is `master`, even the epic activation is refused.
- **Command:** `cd <sandbox on branch main> && aiwf promote E-0001 active ; aiwf promote M-0001 in_progress   [also run in a `git init -q` (master) sandbox]`
- **Quoted:** “aiwf promote E-NNNN active aiwf promote M-NNNN in_progress”

#### docs/workflows.md · `example-fails`

- **Claim:** workflows.md:55 (also :169, :216, and mermaid nodes :22, :126, :129, :131) — a milestone is created with --epic and --title alone.
- **Measured:** `--tdd <required|advisory|none>` is required at creation time for kind=milestone; the bare form exits 2 and writes nothing. Every milestone-creation example in the doc omits it.
- **Command:** `cd <sandbox> && aiwf add milestone --epic E-0001 --title "Schema migration"`
- **Quoted:** “aiwf add milestone --epic E-NNNN --title "Schema migration"”

#### docs/workflows.md · `example-fails`

- **Claim:** workflows.md:142, :150, :159 (and table rows :218, :219, :220) — ADRs, gaps, and decisions are created with --title alone and land with a scaffolded empty body.
- **Measured:** G-0326's born-complete-kind gate refuses all three at exit 2 unless --body / --body-file carries real prose (or --force --reason). Epic and milestone are exempt; adr, gap, decision, contract are not. All three of the doc's realistic-flow bash blocks fail, as do the three AI-prompt table rows that repeat them.
- **Command:** `cd <sandbox> && aiwf add adr --title "Use OAuth 2.1 with passkey support" ; aiwf add gap --title "user_sessions table has no soft-delete" --discovered-in M-0001 ; aiwf add decision --title "Support legacy basic auth for 6 months" --relates-to E-0001,M-0001`
- **Quoted:** “aiwf add adr --title "Use OAuth 2.1 with passkey support"”

#### docs/workflows.md · `false-claim`

- **Claim:** workflows.md:225 — `--write` commits ROADMAP.md.
- **Measured:** `--write` writes the file on disk and stops. HEAD does not move and ROADMAP.md is left untracked. `--help` agrees: "--write updates the file on disk only (no commit — the caller commits it)".
- **Command:** `cd <sandbox> && git rev-parse --short HEAD && aiwf render roadmap --write && git rev-parse --short HEAD && git ls-files ROADMAP.md && git status --short`
- **Quoted:** “`aiwf render roadmap` (or `aiwf render roadmap --write` if you want it committed to `ROADMAP.md`)”

#### docs/workflows.md · `missing-precondition`

- **Claim:** workflows.md:61-66 (and :217) — a milestone goes draft → in_progress directly after creation. The doc never mentions acceptance criteria at all (`grep -in 'acceptance\|add ac\|AC-' docs/workflows.md` returns nothing).
- **Measured:** Two further gates block draft → in_progress: the milestone must have at least one AC, and every AC must carry body prose under its `### AC-N` heading. A reader following the doc gets two consecutive refusals with no idea that `aiwf add ac` and an `aiwf edit-body` pass exist.
- **Command:** `cd <sandbox on epic branch> && aiwf promote M-0001 in_progress   (before and after `aiwf add ac M-0001 --title "Schema migrates cleanly"`)`
- **Quoted:** “**3. Begin work on the epic and the milestone.**”

#### docs/workflows.md · `example-fails`

- **Claim:** workflows.md:77 (and :24, :128, :132, :221) — wrapping a milestone is a single promote.
- **Measured:** Refused while any AC is still open — which is the only state reachable, since in_progress now requires at least one AC. Each AC must first be promoted to met/deferred/cancelled.
- **Command:** `cd <sandbox> && aiwf promote M-0001 done   (M-0001 in_progress, AC-1 open)`
- **Quoted:** “aiwf promote M-NNNN done”

#### docs/workflows.md · `example-fails`

- **Claim:** workflows.md:223 — a decision is cancelled with `aiwf cancel`.
- **Measured:** An `accepted` decision has no cancel target — and the doc's own §2 flow promotes D-NNNN to accepted at :161, so the row fails against the very entity the doc just built. Only a `proposed` decision cancels (to rejected); an accepted one exits solely via `promote → superseded`.
- **Command:** `cd <sandbox> && aiwf cancel D-0001   (D-0001 accepted, created and promoted exactly as workflows.md:159-161 shows)`
- **Quoted:** “| "Cancel the basic-auth decision, we changed our mind." | `aiwf cancel <that decision>` |”

### Per-subject verdicts

| subject | kind | status | verdict | findings | recommended disposition |
|---|---|---|---|---|---|
| D-0028 | decision | accepted | `contradicted-by-code` | 5 | This record is actively misleading and should not be left as accepted-and-current. Either promote D-0028 to superseded a |
| ADR-0003 | adr | accepted | `unimplemented` | 7 | Reconcile main to the verdict already ratified on initiative/preflight-amendments: promote ADR-0003 to `rejected` on mai |
| ADR-0042 | adr | accepted | `unimplemented` | 1 | Leave ADR-0042 accepted and unedited — the record is correct and the implementation is tracked by E-0083. If the gap bet |
| D-0031 | decision | accepted | `unimplemented` | 2 | Leave D-0031 accepted — it is a valid direction record and G-0368 (open, retitled to match) tracks the build. The action |
| G-0276 | gap | open | `already-addressed` | 3 | aiwf promote G-0276 addressed --by-commit 4718d496e (the commit that removed the stash). Every clause of the gap's Direc |
| G-0305 | gap | open | `already-addressed` | 3 | Promote to addressed. The one substantive divergence — the health file is generated from doctor's warnings/errors rather |
| G-0432 | gap | open | `already-addressed` | 3 | aiwf promote G-0432 addressed --by-commit f55923fe3 (verify the flag spelling against `aiwf promote --help`). The gap wa |
| G-0452 | gap | open | `already-addressed` | 3 | aiwf promote G-0452 addressed --by-commit aef85f87c. Then correct docs/initiatives/quality-signal-and-cadence.md, which  |
| G-0564 | gap | open | `duplicate` | 3 | Human call between two shapes, both of which need doing: (a) promote G-0564 wontfix as subsumed, since G-0121's live bod |
| ADR-0001 | adr | proposed | `stale-claims` | 4 | Keep `proposed`. edit-body to (a) drop or rewrite "This repo sets `trunk: poc/aiwf-v3`" — no such key or branch exists;  |
| ADR-0004 | adr | accepted | `stale-claims` | 6 | Keep status accepted — nothing here reverses the decision. Repair the two contradicted Display-surfaces bullets in place |
| ADR-0006 | adr | accepted | `stale-claims` | 4 | Keep `accepted`. edit-body three spots: (a) rewrite the `aiwf-show` Context bullet in the past tense as a *resolved* pre |
| ADR-0007 | adr | accepted | `stale-claims` | 3 | Leave accepted. Optionally refresh the placement table's ritual row to the current roster (or reword it as an illustrati |
| ADR-0008 | adr | accepted | `stale-claims` | 4 | edit-body two repairs: restore the Allocator sentence's contrast ('The next epic after E-0022 is E-0023, not E-23' — wit |
| ADR-0010 | adr | accepted | `stale-claims` | 6 | edit-body the Follow-up, References, and AI-chokepoint sections: mark the chokepoint and the aiwf status mitigation as l |
| ADR-0011 | adr | accepted | `stale-claims` | 4 | edit-body ADR-0011 to describe what shipped: name the nonLegalityVerbAllowlist escape hatch in the Drift-policy section, |
| ADR-0012 | adr | accepted | `stale-claims` | 5 | edit-body ADR-0012's §'Named code constants' and §'Scanner recognition' to record the D-0011 descriptor form (`var Code… |
| ADR-0013 | adr | accepted | `stale-claims` | 5 | edit-body: (1) re-tense or drop the 'cellcoverage carries no authorized-scope scaffolding today' sentence now that CellF |
| ADR-0014 | adr | accepted | `stale-claims` | 5 | Do not supersede the whole ADR — §1/§3/§4/§5 are load-bearing and honored. Edit-body ADR-0014 to (a) add a forward point |
| ADR-0015 | adr | accepted | `stale-claims` | 2 | Keep accepted — the decision itself is honored. Amend the Consequences section: strike or correct the Audit-trail bullet |
| ADR-0016 | adr | accepted | `stale-claims` | 3 | edit-body ADR-0016 to delete the trailing 'Status: proposed / Ratification waits on…' paragraph (CLAUDE.md §'Authoring a |
| ADR-0017 | adr | accepted | `stale-claims` | 4 | Leave `accepted`; the decision stands. File an errata edit or a gap for the two drifted claims: replace `internal/cli/ou |
| ADR-0018 | adr | accepted | `stale-claims` | 2 | Keep accepted; edit-body to (a) either add the entity-files/chokepoint-pointer rule to the sanctioned-exception list or  |
| ADR-0019 | adr | accepted | `stale-claims` | 2 | edit-body ADR-0019 to drop the closed force list in favour of a derived reference ('the highest-leverage forces, pinned  |
| ADR-0020 | adr | accepted | `stale-claims` | 5 | edit-body: drop the 'no glob-syntax validation at this layer' clause (M-0180 moved the Tier-1 gate into config.validate) |
| ADR-0022 | adr | accepted | `stale-claims` | 4 | Keep `accepted` — the kernel does exactly what this ADR decided. No body rewrite is needed for the Decision; if the ADR  |
| ADR-0025 | adr | accepted | `stale-claims` | 2 | Keep `accepted` — nothing in it is reversed. `aiwf edit-body`-equivalent doc edit to narrow the 'allocation only' wordin |
| ADR-0026 | adr | accepted | `stale-claims` | 5 | Keep `accepted`. Edit the Decision section to match what shipped: drop or soften 'aiwf validates against it on write' (t |
| ADR-0027 | adr | accepted | `stale-claims` | 2 | Two separate corrections, and the first is a governance question of the same shape as G-0579's. (1) The absolute sentenc |
| ADR-0029 | adr | accepted | `stale-claims` | 3 | edit-body: drop `Rewidth` from all three mentions, and replace the closed three-item exception list with a pointer to in |
| ADR-0030 | adr | accepted | `stale-claims` | 2 | Keep `accepted` — ADR-0041 refines rather than reverses it — but edit the Decision point 1 and the third Negative conseq |
| ADR-0031 | adr | accepted | `stale-claims` | 3 | Keep accepted; the decision stands and the code implements it. Edit the retired-verb example (`aiwf archive --apply` alo |
| ADR-0032 | adr | accepted | `stale-claims` | 3 | Correct the Consequences bullet to name the shipped registry shape (`skills.ShippedHooks []HookDef` + `HookNamesFrom`, n |
| ADR-0033 | adr | accepted | `stale-claims` | 2 | Either wire `move` through planLinkRewriteWrites (file a gap; nothing currently tracks the move case — G-0478 and G-0439 |
| ADR-0034 | adr | accepted | `stale-claims` | 3 | edit-body ADR-0034: replace "ADR-0009 era design" with the real lineage (E-0043 / M-0171; ADR-0020 and ADR-0021 are the  |
| ADR-0036 | adr | accepted | `stale-claims` | 2 | keep accepted. One narrow edit-body to drop `rewidth` from the uniform-with list (`archive` and `contract bind` both sti |
| ADR-0037 | adr | accepted | `stale-claims` | 2 | Keep status `accepted` — `internal/verb/retitle.go`'s slugTracksTitle implements the decision exactly. Per the repo conv |
| ADR-0038 | adr | accepted | `stale-claims` | 4 | edit-body: either build the companion list PolicyNoOpClaimScope is said to carry (recording internal/cli/importcmd/impor |
| ADR-0040 | adr | accepted | `stale-claims` | 1 | Narrow the one sentence to the rules that actually have a history-walking counterpart (force-non-human, force-with-on-be |
| ADR-0041 | adr | accepted | `stale-claims` | 3 | Keep `accepted`. Edit the Decision or Consequences to record the two carve-outs the implementation added and cites this  |
| ADR-0044 | adr | proposed | `stale-claims` | 2 | edit-body the Context: name CLAUDE.md's `## Derived artifacts — regenerate, don't hand-edit` as the surface that already |
| CLAUDE.md | doc | normative | `stale-claims` | 15 | edit CLAUDE.md to (a) repoint both completion-drift citations at internal/cli/integration/completion_drift_test.go (Test |
| D-0001 | decision | accepted | `stale-claims` | 2 | Leave `accepted` — the decision is still true and still load-bearing. If the body is edited, repoint the 'prose is not p |
| D-0002 | decision | accepted | `stale-claims` | 3 | Keep `accepted`. Correct the '## Spec cell' section to the cell rules.go actually attributes to this decision (`{Kind: e |
| D-0003 | decision | accepted | `stale-claims` | 5 | edit-body D-0003: replace `## Follow-up` with a one-line "implemented in M-0139 (G-0139); guard at internal/verb/cancel_ |
| D-0004 | decision | accepted | `stale-claims` | 3 | edit-body D-0004 alongside D-0003: point `## Follow-up` at M-0139 / G-0139 under E-0036, name `internal/verb/cancel_guar |
| D-0006 | decision | accepted | `stale-claims` | 4 | Keep `accepted`. The live work is on the docs, not the decision: file (or find) a gap to correct docs/design/provenance- |
| D-0007 | decision | accepted | `stale-claims` | 2 | Keep `accepted`. `aiwf edit-body D-0007` to put the Class line's 'currently undefined/permissive' into the past tense it |
| D-0008 | decision | accepted | `stale-claims` | 4 | edit-body the two places the forced-untrailered predicate is spelled out, adding the actor clause: `(sovereign-act-shape |
| D-0011 | decision | accepted | `stale-claims` | 3 | Keep D-0011 `accepted` — its core mechanism shipped and governs new code exactly as written. Edit-body two spots: drop o |
| D-0012 | decision | accepted | `stale-claims` | 2 | Keep accepted; no design change. Optionally edit-body to (a) narrow the chokepoint claim to 'non-archive internal/ sourc |
| D-0013 | decision | accepted | `stale-claims` | 2 | Keep accepted. Optional edit-body to correct the check-time code name to `fsm-history-consistent` / `illegal-transition` |
| D-0014 | decision | accepted | `stale-claims` | 5 | edit-body: replace the 'stays structural until then' / 'documented-but-unimplemented until G-0171' / deferredImplErrorCo |
| D-0015 | decision | accepted | `stale-claims` | 3 | Fold into G-0579's resolution rather than opening a second thread, but widen its scope: G-0579 explicitly scopes itself  |
| D-0016 | decision | accepted | `stale-claims` | 5 | The decision is half honored and half superseded in fact by ADR-0016 / G-0194 (commit debf7f5ff). Either edit-body a sho |
| D-0018 | decision | accepted | `stale-claims` | 4 | edit-body the Concrete sequencing section to record that M-0161/AC-9 was cancelled, that M-0162/AC-1 explicitly retained |
| D-0019 | decision | accepted | `stale-claims` | 5 | Keep accepted; edit-body to (a) fix the two ../../../ links to ../../, (b) drop or re-anchor the four line-number citati |
| D-0021 | decision | accepted | `stale-claims` | 5 | Keep accepted — the deferral verdict is unchanged and correct. Edit-body to (a) retitle the block quote as the AC-7 matr |
| D-0022 | decision | accepted | `stale-claims` | 4 | Keep `accepted` — the decision is honored and its reasoning is the durable part. edit-body two spots: (a) the Part-4 / R |
| D-0023 | decision | accepted | `stale-claims` | 6 | edit-body D-0023: (1) repoint the citation at spec line 204, which is the real enumeration, and drop `authorize_scenario |
| D-0025 | decision | accepted | `stale-claims` | 2 | Keep `accepted` — this is a durable no-relitigation record and its argument is intact. Optional light edit-body: put the |
| D-0029 | decision | accepted | `stale-claims` | 1 | Leave the decision accepted; optionally edit-body one word to drop `rewidth` from the reachability sentence (ADR-0039 re |
| D-0032 | decision | accepted | `stale-claims` | 1 | Leave accepted; the decision's substance is unaffected. edit-body one sentence in Question (and note the same sentence s |
| D-0036 | decision | accepted | `stale-claims` | 2 | edit-body D-0036 to add one paragraph (or a Consequences bullet) recording that ADR-0041 narrowed the decision's domain: |
| D-0038 | decision | accepted | `stale-claims` | 2 | Keep accepted — the conclusion survives on its other four arguments and the Consequences all landed. Edit-body the Go-to |
| D-0039 | decision | accepted | `stale-claims` | 4 | Leave D-0039 `accepted`. File (or fold into an existing doc-drift gap) an edit to docs/design/design-decisions.md §"What |
| D-0043 | decision | accepted | `stale-claims` | 2 | edit-body D-0043 to repoint the F12 citation at `docs/initiatives/archive/verb-layer-cleanup.md` (status: realized). Eve |
| D-0044 | decision | accepted | `stale-claims` | 2 | edit-body to correct the signature to `cliutil.ErrInternal(err error) error` (the Consequences paragraph is forward-look |
| D-0045 | decision | accepted | `stale-claims` | 1 | Leave `accepted`; the decision and its consequence are intact. If touched, correct the enumeration to the packages that  |
| D-0047 | decision | accepted | `stale-claims` | 3 | Keep accepted; the decision is honored. Edit-body to re-tense the three now-false present-tense claims (born-at-red seed |
| D-0051 | decision | accepted | `stale-claims` | 3 | Keep accepted; edit-body to drop the three counts (the argument they support survives unchanged) and to re-tense or remo |
| D-0053 | decision | proposed | `stale-claims` | 3 | edit-body to drop the count (reference-phrase it, e.g. 'one test per corrected fact' — the repo's own keep-the-reasoning |
| D-0058 | decision | proposed | `stale-claims` | 3 | Two separate acts, both human calls. (1) Ratify or reject — the decision describes the status quo and has sat `proposed` |
| D-0060 | decision | accepted | `stale-claims` | 3 | edit-body is not enough: `aiwf retitle D-0060` to something naming the force-predicated subset without the word 'soverei |
| D-0063 | decision | accepted | `stale-claims` | 4 | aiwf edit-body D-0063 to put the Question's G-0558 sentence in past tense ('a defect surfaced that the harness could not |
| D-0066 | decision | accepted | `stale-claims` | 3 | Keep `accepted` — E-0086 re-adopts it as settled and the decision is still the live plan. `aiwf edit-body D-0066` to (a) |
| G-0022 | gap | open | `stale-claims` | 4 | Keep the gap open. `aiwf edit-body G-0022` to (a) drop or correct the `aiwf-revoked-by:` claim in item 1 — the slot that |
| G-0060 | gap | open | `stale-claims` | 5 | keep open (the kernel question is live and undecided), but edit-body: retitle to drop the 'loosely defined' half now tha |
| G-0068 | gap | open | `stale-claims` | 3 | edit-body: correct 'Only /ac is a literal in source' — the whole composite is now skipped because `Code: CodeEntityBodyE |
| G-0070 | gap | open | `stale-claims` | 4 | Keep `open` — the JSON-envelope gap is genuine and unfixed. `aiwf edit-body G-0070` to strip the plugins framing entirel |
| G-0073 | gap | open | `stale-claims` | 6 | Keep open (the defect is real and unaddressed). edit-body to: (1) rewrite or delete the 2026-08-12 'their specs already  |
| G-0077 | gap | open | `stale-claims` | 4 | Keep open at low — the deliverable is still absent. edit-body to repoint the two PROMOTION-PLAN.md citations at docs/arc |
| G-0104 | gap | open | `stale-claims` | 3 | edit-body: re-frame the option space against today's distribution — rituals are embedded in the binary and materialized  |
| G-0110 | gap | open | `stale-claims` | 3 | Keep open but rewrite, or close as partly-moot: the practical thread ('operators need a working diff-scoped mutation run |
| G-0111 | gap | open | `stale-claims` | 3 | edit-body: (1) record thread 4 as landed skill-side (aiwfx-wrap-epic Precondition item 6, pinned by TestAiwfxWrapEpic_Pr |
| G-0117 | gap | open | `stale-claims` | 4 | Keep open for the SPA/one-page half. edit-body to rewrite §"What's missing"'s opening paragraph so the retracted perform |
| G-0121 | gap | open | `stale-claims` | 5 | Keep open — the two-branch composition axis, the falsification test, and the never-built declarative enumeration are all |
| G-0157 | gap | open | `stale-claims` | 6 | Keep open. edit-body to refresh all coordinates (worktrees.go loop at :133; helpers at :1317/:1336/:1354/:1372; show/sco |
| G-0160 | gap | open | `stale-claims` | 3 | edit-body and re-scope, do not close outright. The stated mode (a new edge in `entity.transitions` to an existing from-s |
| G-0161 | gap | open | `stale-claims` | 3 | edit-body: drop the hardcoded "18 cells" (or replace with "every Illegal cell"), mark ANTI-0010 as already covered by Te |
| G-0166 | gap | open | `stale-claims` | 4 | edit-body: (a) rename gap-resolved-has-resolver → gap-addressed-has-resolver in both places, including the quoted verb m |
| G-0168 | gap | open | `stale-claims` | 5 | edit-body: strike the `tdd:` row from the 'missing' table and the whole 'Re-discovery (2026-06-26)' section's open frami |
| G-0169 | gap | open | `stale-claims` | 4 | edit-body: strike the whole mutating-verb thread (both the "What's missing" bullet and the "Notes" paragraph) — rewidth  |
| G-0181 | gap | open | `stale-claims` | 1 | Keep open, disposition correct. edit-body to replace `(M-… / proxyLookupFailedHint)` with the commit that actually shipp |
| G-0212 | gap | open | `stale-claims` | 4 | edit-body: rewrite the 'Known classes' list to record, per item, the scenario that now exercises it (parallel-branch-rea |
| G-0215 | gap | open | `stale-claims` | 2 | Needs a human call on whether a meta-chokepoint with no current instance stays open. Either way, edit-body: the cherryPi |
| G-0217 | gap | open | `stale-claims` | 2 | Keep open, but edit-body: drop the two-states-conflated framing (the label is gated on a terminal driver status, so case |
| G-0225 | gap | open | `stale-claims` | 3 | Keep open, raise priority above `low`, and edit-body + retitle: the affected population is not "legacy" scopes but every |
| G-0229 | gap | open | `stale-claims` | 3 | Keep open (all four threads live), but edit-body to: (1) say 'Four'; (2) drop or restate the 'inner Entry decode is stri |
| G-0233 | gap | open | `stale-claims` | 1 | Keep open; the defect is live and nothing has moved on it. Optionally edit-body to drop the '~35' figure (the argument t |
| G-0234 | gap | open | `stale-claims` | 3 | Keep open at low priority. `aiwf edit-body` to drop the 'FSMTransitionError already lists the set internally' justificat |
| G-0235 | gap | open | `stale-claims` | 3 | edit-body: prune the landed items out of the 'What's missing' list rather than correcting them in Notes — the list is wh |
| G-0246 | gap | open | `stale-claims` | 3 | Keep open at medium. edit-body to correct the internal/entity/entity.go line citations (492-493 for supersedes/supersede |
| G-0249 | gap | open | `stale-claims` | 1 | Keep open at low priority; the gap it proposes is real and unbuilt. Optionally edit-body to replace 'the kernel tracks P |
| G-0254 | gap | open | `stale-claims` | 2 | Keep open. Edit-body to drop the absolute counts (this repo's own rule is keep the reasoning, derive the facts — 'mixed, |
| G-0281 | gap | open | `stale-claims` | 4 | Keep `open` — D-0037 defers it deliberately and the cross-machine residual race it targets is real. `aiwf edit-body G-02 |
| G-0282 | gap | open | `stale-claims` | 3 | edit-body G-0282: delete the `rewidth` seed-fixture claim (find a live class-C fixture or state that none exists), repla |
| G-0307 | gap | open | `stale-claims` | 2 | Keep open; edit-body to (a) replace 'and in every sibling block' with the measured state (2 of 17 top-level keys guarded |
| G-0311 | gap | open | `stale-claims` | 3 | Keep open at high priority; the primary argument is intact and stronger than the body states. edit-body to (a) fix the d |
| G-0317 | gap | open | `stale-claims` | 3 | edit-body: replace the skill_coverage_test.go example with one that holds — the matcher is a plain strings.Contains over |
| G-0324 | gap | open | `stale-claims` | 2 | edit-body: drop or date the '~46' figure and re-derive from the current ref count (5 refs walked, of which 2 are already |
| G-0333 | gap | open | `stale-claims` | 4 | edit-body: drop the 'stated in no AI-discoverable channel' claim and the AI-discoverability-violation framing (both are  |
| G-0366 | gap | open | `stale-claims` | 2 | Keep open; the defect is real and unaddressed. edit-body to drop the '(inconsistently — see the sibling gap on wf-patch' |
| G-0372 | gap | open | `stale-claims` | 3 | Keep `open`. The structural property it asks for is genuinely unbuilt. Re-measure and re-date the performance paragraph, |
| G-0375 | gap | open | `stale-claims` | 2 | Keep `open` at `high` — measurement confirms the exposure, the central chokepoint testsupport.HardenGitTestEnv still pin |
| G-0387 | gap | open | `stale-claims` | 3 | edit-body: correct the sha clause (sha now flows from every BeginVerbDiag-instrumented mutating verb, not just cancel/mo |
| G-0396 | gap | open | `stale-claims` | 4 | Keep open at low priority. edit-body to (a) correct 'roughly fifty' to the measured 303, (b) drop the 'legacy cohort' fr |
| G-0398 | gap | open | `stale-claims` | 2 | Keep open at medium. edit-body to past-tense or drop the 'Discovered indirectly' paragraph's present-tense claim (the wa |
| G-0400 | gap | open | `stale-claims` | 7 | aiwf retitle G-0400 to a count-free title (e.g. 'Stress scenario catalog leaves most aiwf verbs undriven'), then aiwf ed |
| G-0412 | gap | open | `stale-claims` | 3 | Keep open — the defect is live in nine files. edit-body to replace the whole origin-and-propagation paragraph with the m |
| G-0416 | gap | open | `stale-claims` | 3 | edit-body: restate the ambiguity as applying only to hit sets that include a remote-visible ref (ADR-0041 routes the loc |
| G-0417 | gap | open | `stale-claims` | 3 | Keep `open`. edit-body the Proposed-fix section: step 4 should name `internal/policies/m0162_ac4_bijection_test.go` (sta |
| G-0430 | gap | open | `stale-claims` | 2 | Keep open at low. edit-body to narrow the first sentence to what is actually true (upgrade delegates fetch/place to `go  |
| G-0436 | gap | open | `stale-claims` | 3 | edit-body: delete the id-allocation.md bullet (closed by G-0443) and note that the residual function-name drift on that  |
| G-0439 | gap | open | `stale-claims` | 3 | Keep open. Edit-body to widen from two movers to the four that share the scope (archive, rename, retitle, reallocate — a |
| G-0442 | gap | open | `stale-claims` | 1 | edit-body to widen the table to four rows (gap.addressed_by via --by, gap.addressed_by_commit via --by-commit, adr.super |
| G-0444 | gap | open | `stale-claims` | 2 | Keep open. On the fix, drop the `:60` line anchor (line refs into Go source rot; name the symbol only), and widen the te |
| G-0448 | gap | open | `stale-claims` | 3 | edit-body to name all four rule-selection sites (check.Run, the full CLI path, runFast, runShapeOnly) and refresh the tw |
| G-0454 | gap | open | `stale-claims` | 1 | keep open; optionally edit-body to describe the shared shape as 'strip the kind prefix, validate the tail, interpret the |
| G-0456 | gap | open | `stale-claims` | 1 | edit-body: replace 'The two envelope-arm verbs (archive, rewidth)' with 'The one envelope-arm verb (archive)' — rewidth  |
| G-0458 | gap | open | `stale-claims` | 1 | Keep open; edit-body to replace the exclusivity framing with the accurate one ('the only OPEN entry that refuses rather  |
| G-0472 | gap | open | `stale-claims` | 3 | Keep open; edit-body to retire the two paragraphs written on the G-0557 premise (rewrite in past tense: the divergence w |
| G-0473 | gap | open | `stale-claims` | 1 | Keep open and act on it — the measurements hold. edit-body only to change 'G-0470 — the live sibling concern' to reflect |
| G-0478 | gap | open | `stale-claims` | 3 | Keep open at high priority; edit-body to re-date or drop the arithmetic (79 links, 0 broken today) and replace the 'stil |
| G-0483 | gap | open | `stale-claims` | 1 | edit-body to correct the third-site sentence: only two sites pass a literal empty code (`no result returned`, `validatio |
| G-0486 | gap | open | `stale-claims` | 2 | Keep `open`. Rewrite 'What's missing' so it states the current condition (symlinks refused, so the corruption route is c |
| G-0497 | gap | open | `stale-claims` | 1 | Keep open (low). Drop or correct the '//nolint:gosec disappears' sentence: gosec is already excluded on _test.go tree-wi |
| G-0504 | gap | open | `stale-claims` | 1 | Keep open (high). edit-body the Related line to say G-0471 remains open and that E-0076, the epic that would have addres |
| G-0508 | gap | open | `stale-claims` | 2 | Keep open; `aiwf retitle G-0508` to say four near-copies so the title stops contradicting its own body. Optionally widen |
| G-0512 | gap | open | `stale-claims` | 2 | keep open; edit-body to correct the arrangement list — arrangement 2 (empty directory at a directory destination) is a s |
| G-0514 | gap | open | `stale-claims` | 4 | Keep open, but edit-body: reframe the population from 'currently fire' to 'the grammar still admits the class; the M-028 |
| G-0517 | gap | open | `stale-claims` | 2 | keep open, edit-body to replace the premise and the Resolution: the excluded paths are the same genre as the already-swe |
| G-0523 | gap | open | `stale-claims` | 1 | Keep open at high priority; nothing has shipped a second delivery channel. Edit-body the 'Fix shape' paragraph to name A |
| G-0527 | gap | open | `stale-claims` | 2 | edit-body: delete the exit-0 transcript and the two sentences around it (closed by 981115515 / G-0528), and narrow 'unme |
| G-0530 | gap | open | `stale-claims` | 5 | edit-body: drop the `## Work log` row and the 'sharpest case' sentence entirely (the section is the largest in the spec, |
| G-0535 | gap | open | `stale-claims` | 3 | Keep open (medium). edit-body: repoint Option 3's reference case from the retired sovereign policy to PolicyApplyCallers |
| G-0536 | gap | open | `stale-claims` | 4 | edit-body: (1) replace the 'from nowhere else' absolute with the accurate scope — the tree rule set does run in CI via i |
| G-0539 | gap | open | `stale-claims` | 2 | Keep open. edit-body the Resolution shape: replace the closed "three flows" enumeration with the fourth case (ritual-dri |
| G-0540 | gap | open | `stale-claims` | 2 | edit-body G-0540: repoint the duplication thread from `sovereign_test.go` (deleted) to `coherence_guard_chokepoint_test. |
| G-0545 | gap | open | `stale-claims` | 2 | Keep open, edit-body to correct two sentences: say the scope hole is unrecorded (the comment records a different one — a |
| G-0546 | gap | open | `stale-claims` | 1 | Keep open (low). edit-body to reconcile Option 2 with accepted D-0061: state which of D-0061's three structural argument |
| G-0551 | gap | open | `stale-claims` | 1 | Keep open at Option 2. Narrow one sentence to 'the audit's coherence rules return no findings for a commit carrying no a |
| G-0553 | gap | open | `stale-claims` | 2 | edit-body: drop the final two sentences of the 'A further group…' paragraph (the Fix-cell / `case-paths` `git mv` thread |
| G-0555 | gap | open | `stale-claims` | 3 | Keep open — this is the highest-value live gap in the batch. edit-body to (a) add the third helper, sharedLockHolderBina |
| G-0560 | gap | open | `stale-claims` | 3 | keep open. edit-body: drop the G-0559-gates-this sequencing (the emitter is fixed — G-0517 and this gap's tables are now |
| G-0561 | gap | open | `stale-claims` | 3 | Keep open. edit-body to repoint the `update.go:121` citation at `update.go:130` (the RefreshArtifacts arm), replace 'ver |
| G-0571 | gap | open | `stale-claims` | 3 | Keep open (E-0084 closes it and its success criteria name it). `aiwf edit-body` to replace the open-option framing with  |
| G-0572 | gap | open | `stale-claims` | 2 | Keep open at high priority; the composition defect is real and unaddressed. `aiwf edit-body` to soften the hint paragrap |
| G-0573 | gap | open | `stale-claims` | 5 | edit-body to restate the live half: the guard applies the severity policy, but entity-body-empty is read from disk so a  |
| G-0579 | gap | open | `stale-claims` | 1 | Keep open. The question it poses — correct D-0015's Consequences in place vs. supersede the decision — is the genuine op |
| G-0580 | gap | open | `stale-claims` | 1 | Keep the gap open; edit-body the 'Why it matters' second paragraph. Either restate the drift in the past tense as the mo |
| G-0581 | gap | open | `stale-claims` | 2 | edit-body to correct the scope: the unwalked shipped surface is the guidance source PLUS the wf-rituals plugin skills, t |
| G-0583 | gap | open | `stale-claims` | 3 | Keep `open` — the defect is live and nothing has shipped against it (the ritual M-0308 produced no longer exists). Eithe |
| G-0584 | gap | open | `stale-claims` | 4 | Keep open, but edit-body substantially: drop or invert the wf_ritual_honesty_test.go paragraph (it is the counter-exampl |
| G-0585 | gap | open | `stale-claims` | 1 | edit-body to repair the closing paragraph: `wf-measure-spec` no longer ships (withdrawn in 4a4019ec1; its parent epic E- |
| G-0586 | gap | open | `stale-claims` | 3 | Keep open; the design question is real. Edit-body to drop or correct the template claim: the epic template already ships |
| G-0587 | gap | open | `stale-claims` | 3 | Keep open only if the residue is judged worth a separate entity; otherwise fold into G-0548, which owns the path-citatio |
| G-0589 | gap | open | `stale-claims` | 1 | Keep open; edit-body to drop the count (say 'the tracked root markdown files' or name them) per this repo's own keep-the |
| docs/architecture.md | doc | normative | `stale-claims` | 11 | Substantial rewrite of §1 file-placement, §2 diagram, and §4 published surface. The two high-severity items (line 54 rit |
| docs/design/design-decisions.md | doc | normative | `stale-claims` | 27 | This needs a full rewrite, not patches — the framing (PoC / 'this branch' / I1-I2-I2.5-I3 iteration labels) is itself th |
| docs/design/design-lessons.md | doc | normative | `stale-claims` | 12 | Rewrite the 'On reversal' section to quote CLAUDE.md's current four acceptable answers (dropping set-status/complete/hot |
| docs/design/growth.md | doc | normative | `stale-claims` | 9 | edit-body/edit the doc: (1) delete or rewrite 'None of these is currently shipped' — the cheap-fix escape shipped in 18f |
| docs/design/id-allocation.md | doc | normative | `stale-claims` | 11 | edit the doc: drop or rewrite the 'A walk over every ref in the repo' bullet now that E-0052 landed; delete the allocate |
| docs/design/legal-workflows-audit.md | doc | normative | `stale-claims` | 31 | Two-step. (1) Immediately correct the header: M-0121/M-0122/M-0123 and E-0033 are all `done`; the doc is a completed Pas |
| docs/design/legal-workflows-first-principles.md | doc | normative | `stale-claims` | 19 | Do not silently patch cells. Either (a) re-tier the file as a historical Pass-B artifact — a status banner naming M-0123 |
| docs/design/oracles.md | doc | normative | `stale-claims` | 5 | edit-body-equivalent doc edit, four narrow corrections: (1) qualify the bolded aiwf-check sentence to 'no workflow runs  |
| docs/design/performance.md | doc | normative | `stale-claims` | 13 | Re-measure and rewrite the 'Measured baseline' table with a fresh dated row; strike or re-point the G-0340 / G-0323 / G- |
| docs/design/provenance-model.md | doc | normative | `stale-claims` | 12 | Keep the doc — it is still the design of record for a shipped subsystem — but rewrite six spots: (1) drop `provenance-no |
| docs/design/tree-discipline.md | doc | normative | `stale-claims` | 10 | edit-body-equivalent rewrite of five sections: delete or invert the 'Why no marker-managed CLAUDE.md fragment' section a |
| docs/overview.md | doc | normative | `stale-claims` | 5 | Edit-body the ID-format column to canonical 4-digit forms, rewrite the 'nothing finer than a milestone' paragraph so it  |
| docs/skill-author-guide.md | doc | normative | `stale-claims` | 11 | Rewrite, do not patch — the worked example needs a new subject (its stated design is now illegal, not just its flags), t |
| docs/workflows.md | doc | normative | `stale-claims` | 18 | Rewrite sections 1-3 against a live transcript rather than patching line by line: the flow now requires (a) --tdd at mil |
| D-0042 | decision | accepted | `needs-human-decision` | 2 | Human call between `superseded` (the FSM allows accepted → superseded; M-0290 / E-0078 is what overtook it) and leaving  |
| D-0056 | decision | proposed | `needs-human-decision` | 1 | A human should ratify (`aiwf promote D-0056 accepted`) or reject. It is A5-shaped: the record is being followed while it |
| D-0057 | decision | proposed | `needs-human-decision` | 1 | Human call: promote D-0057 accepted (the behavior is shipped, load-bearing, and documented at the seam), or state plainl |
| ADR-0021 | adr | accepted | `sound` | 4 | Leave accepted. Optionally amend the Decision's 'total' bullet to name the milestone-derivation and archive exemptions,  |
| ADR-0023 | adr | accepted | `sound` | 0 | No action. Leave accepted as written. |
| ADR-0028 | adr | accepted | `sound` | 1 | No ADR change. The outstanding work is G-0370, which is open, priority high, and states the shape of the fix; act there, |
| ADR-0035 | adr | accepted | `sound` | 0 | No action. Worth noting for whoever owns the follow-on: the ADR's own Negative consequence — that the miss-guard is now  |
| ADR-0039 | adr | accepted | `sound` | 0 | No action. If ADR-0029's Rewidth mentions are repaired, this ADR is the natural place to note that a second ADR carried  |
| ADR-0043 | adr | accepted | `sound` | 1 | Leave as is. If the Context paragraph's '217 of 564 / 27 of 64' is ever re-read as current, re-date it or drop the denom |
| D-0010 | decision | accepted | `sound` | 0 | No action. Status `accepted` is correct; D-0009 is correctly `superseded` and archived. |
| D-0017 | decision | accepted | `sound` | 2 | Leave accepted. Worth a maintainer glance: the parameter accumulation the decision predicted has happened (a fourth map  |
| D-0024 | decision | accepted | `sound` | 1 | Keep `accepted`. One small edit-body: delete the `## Status` section's first sentence ('Status starts at `proposed`; pro |
| D-0026 | decision | accepted | `sound` | 0 | Leave as-is. The decision remains a valid source of truth and its revisit trigger is still unmet. |
| D-0027 | decision | accepted | `sound` | 0 | No action. The decision's three supporting facts all hold: `area-mistag` emits at warning and is absent from `ApplyAreaR |
| D-0030 | decision | accepted | `sound` | 1 | Leave as is. One phrasing nicety at most: the opening sentence reads as though each of the three predicates honors both  |
| D-0033 | decision | accepted | `sound` | 0 | No action. |
| D-0034 | decision | accepted | `sound` | 0 | No action. The one thing worth watching is that the count 56 is a dated observation that will drift as new acknowledgmen |
| D-0035 | decision | accepted | `sound` | 1 | No action. Optionally, if a future edit touches the body, mark 'all 12 scenarios' as the count at M-0249's time (16 toda |
| D-0037 | decision | accepted | `sound` | 0 | No action. The decision is doing exactly what it was written to do: `git log --all --grep='^aiwf-verb: reallocate$'` sti |
| D-0040 | decision | accepted | `sound` | 0 | No action. Leave `accepted`. |
| D-0041 | decision | accepted | `sound` | 0 | No action. Claims verified; every consequence materialized. |
| D-0046 | decision | accepted | `sound` | 0 | Leave as accepted. No edit needed. |
| D-0048 | decision | accepted | `sound` | 0 | leave as accepted; no edit needed |
| D-0049 | decision | accepted | `sound` | 0 | Leave as is. Nothing in the body is contradicted by measurement; the one anecdotal claim (which ACs were test-only) chec |
| D-0050 | decision | accepted | `sound` | 0 | Leave as-is. The dated measurements (52 literals / 22 tests / four review rounds) are provenance-labelled observations,  |
| D-0052 | decision | accepted | `sound` | 0 | No action. Leave `accepted`. (The verification incidentally shows the no-keep-list bet paid off: `aiwf check` reports 0  |
| D-0054 | decision | accepted | `sound` | 2 | Leave `accepted`. If touched: date the two measurements (the decision's own tier argument makes it a dated observation), |
| D-0055 | decision | accepted | `sound` | 0 | Leave as accepted. The revisit trigger (a third `//go:build !testpins` file, or the negated arm growing branches) has no |
| D-0059 | decision | accepted | `sound` | 0 | No action. The Follow-ups section correctly records a still-live textual divergence rather than claiming it was fixed —  |
| D-0061 | decision | accepted | `sound` | 0 | Leave as-is. The packet's pre-resolved `unknown-go-symbol` on PolicySovereignDispatchersGuardHumanActor is a false posit |
| D-0062 | decision | accepted | `sound` | 0 | No action. Leave accepted. |
| G-0023 | gap | open | `sound` | 3 | Leave `open` at `priority: low`. If touched, an `aiwf edit-body` could add the two required gap headings and drop the im |
| G-0113 | gap | open | `sound` | 2 | Leave `open` at `priority: medium`. If an `edit-body` touches it for another reason, widen `E-19` to `E-0019`; nothing e |
| G-0178 | gap | open | `sound` | 0 | Leave open. Every claim measured true; the cited terminal entities (E-0038, M-0151, G-0177) are cited as completed histo |
| G-0253 | gap | open | `sound` | 0 | Leave open unchanged. The GOCOVERDIR flag from the mechanical pass is a false positive — it is a Go toolchain environmen |
| G-0274 | gap | open | `sound` | 0 | Leave open at medium priority. No body edit needed. Its Provenance section is correctly labeled and the terminal-status  |
| G-0289 | gap | open | `sound` | 2 | Keep open. edit-body to record that check.Run is already wired at internal/cli/doctor/doctor.go:297 (so the work is smal |
| G-0302 | gap | open | `sound` | 1 | Leave open at low priority. Optionally tighten the opening of the Problem section — 'This gap was originally framed arou |
| G-0328 | gap | open | `sound` | 0 | Leave open as-is. Optionally add a one-line note that E-0058, which named this gap as one of its deliverables, was cance |
| G-0368 | gap | open | `sound` | 2 | Keep open. Optionally soften 'Nothing links the two' to 'nothing mechanically links the two' — step 7 does advise refere |
| G-0369 | gap | open | `sound` | 0 | Keep open. Every claim measures true and the sibling-scope carve-out against G-0538 (also open) is accurate — neither su |
| G-0370 | gap | open | `sound` | 0 | Keep open. The wf-patch-sized fix it describes is still exactly the outstanding work; priority high is defensible since  |
| G-0383 | gap | open | `sound` | 0 | No action. Leave open at low priority; the stated revisit trigger (a real call site logging a path outside verb/entity/a |
| G-0385 | gap | open | `sound` | 0 | Leave open. The reproduction is a dated observation against v0.26.2 and reads correctly as one — nothing in it is presen |
| G-0414 | gap | open | `sound` | 0 | Keep open. The fix is a rename plus doc-comment and failure-message rewrite in internal/stresstest/promote_on_wrong_bran |
| G-0420 | gap | open | `sound` | 1 | Keep open (low). Worth a one-line edit-body noting G-0078 is `addressed`, since the body's 'it has no scope until G-0078 |
| G-0421 | gap | open | `sound` | 0 | Leave `open` at `priority: low`. No edit needed — this is the cleanest-stated subject in the batch, and the open design  |
| G-0433 | gap | open | `sound` | 1 | Keep open, priority low is right. Optionally drop or widen the 'only visible via --worktrees or raw frontmatter' clause  |
| G-0434 | gap | open | `sound` | 2 | Keep open. Two notes for the implementer, neither of which changes the verdict: the helper's own doc comment (provenance |
| G-0445 | gap | open | `sound` | 1 | Leave `open` at `priority: low`. Worth one line of scoping if edited: the gate is opt-in and inert unless a consumer con |
| G-0453 | gap | open | `sound` | 1 | Keep open. If it is picked up, note that the blast radius is wider than the two definition sites suggest — shortHash has |
| G-0455 | gap | open | `sound` | 0 | leave open unchanged |
| G-0459 | gap | open | `sound` | 1 | Keep open. Name G-0460 explicitly where the body says 'see the companion gap' (the allowlist entry already does, and G-0 |
| G-0460 | gap | open | `sound` | 0 | Keep open at priority high. The 'resolution shape' framing (settle the invariant first) is still the right next move; no |
| G-0461 | gap | open | `sound` | 0 | Keep open; the body needs no edit. Option 1 (roll up at ingest, one line in WalkAcknowledgedSHAEntities) is still the sm |
| G-0464 | gap | open | `sound` | 0 | Leave open and unedited; the three-line fix it proposes is still exactly what the code needs. |
| G-0471 | gap | open | `sound` | 0 | Leave open at high priority. No body edit needed. The two terminal predecessors (G-0147, G-0176) are cited as bounding-n |
| G-0477 | gap | open | `sound` | 2 | Keep open at priority low, and widen Option 1 to cover isTopLevelAiwfVersionLine (same file, same dead line) so the one- |
| G-0493 | gap | open | `sound` | 0 | Keep open. Every claim in the body measures true at HEAD; the citations to M-0283 (done) and G-0463 (addressed) are hist |
| G-0498 | gap | open | `sound` | 0 | Keep `open`. Both references resolve and the decision the gap asks for (apply the clean filter vs. ship a `-text` .gitat |
| G-0500 | gap | open | `sound` | 1 | Keep open at high. Every reference resolves and the framing is accurate; the only cosmetic nit is the worked example's u |
| G-0501 | gap | open | `sound` | 1 | Keep open at high priority. Optionally correct the one sentence about the os.Stat existence probe, which belongs to the  |
| G-0502 | gap | open | `sound` | 0 | Keep `open` at `low`. The pre-resolved `dangling-entity-ref C-0001` is a FALSE POSITIVE and needs no edit: C-0001 appear |
| G-0506 | gap | open | `sound` | 1 | Leave `open` at `priority: medium`. The Scope section's open question ('decide what records it') has two facts nearby th |
| G-0510 | gap | open | `sound` | 0 | Keep open as written; it is unusually well-measured. The one open call it flags (whether the spaced form stays accepted  |
| G-0513 | gap | open | `sound` | 0 | keep open as written; the sketch (feed the report the same union — loaded entities plus the record's entity paths — that |
| G-0516 | gap | open | `sound` | 1 | Keep open. Optionally fix the parenthetical to quote the real set (or just say 'a closed seven-phrase set'), since as wr |
| G-0518 | gap | open | `sound` | 0 | keep open as written; the proposed subcode (fires when a token resolves only after canonicalization) has no counterpart  |
| G-0519 | gap | open | `sound` | 1 | keep open as written; optionally add one clause noting the motivating instance was repaired by the same milestone's swee |
| G-0524 | gap | open | `sound` | 2 | keep open, no body change required. Optionally fix the README section attribution (§"Build", not §"Reopen in Container") |
| G-0526 | gap | open | `sound` | 1 | Keep open; it is a genuinely undecided design question with three costed options and no reflex answer. Only fix worth ma |
| G-0529 | gap | open | `sound` | 0 | Keep open. The two proposed properties are still unbuilt and the 'Not this gap' boundaries against G-0368 and G-0439 are |
| G-0533 | gap | open | `sound` | 0 | Leave open and unedited. The dated measurement block is still literally true; the only claims I could not re-derive are  |
| G-0537 | gap | open | `sound` | 0 | Leave open at low priority as written. The Sequencing section's self-assessment ('Behind any gap that names a defect. No |
| G-0538 | gap | open | `sound` | 0 | Leave open and unedited. The population is unchanged and no rule has been added since filing. |
| G-0543 | gap | open | `sound` | 0 | Keep `open` at medium. The Direction's positive control is confirmed viable: revive's `package-comments` rule is enabled |
| G-0544 | gap | open | `sound` | 0 | Leave open. Option 2 (register --principal on the four) remains the smallest change that closes the malformed-commit hol |
| G-0548 | gap | open | `sound` | 2 | Keep open. Optionally add a `Related` line to G-0587 (added 2026-08-15, after this gap), which hits the same ban from th |
| G-0549 | gap | open | `sound` | 0 | Keep `open` at medium. The first half (give the subcode its own row, filed by the severity it emits — the skill already  |
| G-0550 | gap | open | `sound` | 0 | Leave open at Option 1. The 'not reachable through any verb' scoping is also confirmed — this is about hand-crafted and  |
| G-0552 | gap | open | `sound` | 0 | Leave open at Option 3. The 84 GB / 288 GB figure is a past observation attributed to M-0291's wrap and reads correctly  |
| G-0554 | gap | open | `sound` | 1 | keep open, no body change required. Optionally sharpen the one imprecise sentence about where --statusline writes (defau |
| G-0562 | gap | open | `sound` | 0 | Leave open; it is accurately scoped and ready to act on. The write-shape fix is one line; the second question (whether i |
| G-0563 | gap | open | `sound` | 2 | Keep open. Optionally edit-body to add `authorize` (and `contract bind` / `contract unbind`) to the bare-loading list, a |
| G-0565 | gap | open | `sound` | 2 | Keep open. One low-severity wording fix if the body is touched: disambiguate 'there is no cross-branch-pending finding i |
| G-0568 | gap | open | `sound` | 0 | Leave open exactly as written; it is already scoped as an opportunistic ten-line change with an explicit close-rather-th |
| G-0569 | gap | open | `sound` | 0 | Leave open. Nothing implementing either route has landed; the phase-reset route remains the smaller one and its interact |
| G-0574 | gap | open | `sound` | 0 | Keep open. When it is picked up, note that one instance has already been point-mitigated: archive-sweep-pending is exclu |
| G-0575 | gap | open | `sound` | 2 | keep open; it is the trap itself, distinct from G-0574 (message-keyed finding identity, its root) and from G-0121 (the h |
| G-0576 | gap | open | `sound` | 1 | Keep open. Optionally date the measurement paragraph (its sibling G-0569 dates its own measurements) so a later reader k |
| G-0577 | gap | open | `sound` | 0 | Leave open unchanged. The mutation table is a dated experiment I cannot re-run read-only; nothing in it is contradicted  |
| G-0578 | gap | open | `sound` | 1 | Keep open. Consider widening the body (or the fix) to the seven sibling sites in internal/policies that carry the same s |
| G-0582 | gap | open | `sound` | 1 | Keep open; it is the right shape (a decision, not a cleanup). Worth one edit-body adding a `## Related` line naming G-05 |
| G-0588 | gap | open | `sound` | 0 | Keep open. It is a genuine kernel decision (retire the verb vs. restore the spec to normative tier), correctly filed rat |

### Claims left unverifiable

- **ADR-0001** — The unresolved mechanical references in the packet (`aiwf assign-pending`, `inbox-on-trunk`, `slug-id-on-trunk`, `unresolved-slug-ref`, `duplicate-slug-on-trunk`, `trunk-branch-missing`) are defects. · *why:* They are not defects — they are the ADR *proposing* a verb and five finding codes, which is exactly the legitimate case the brief carves out. Their absence from the CLI surface is the correct state for a `proposed` ADR that D-0037 defers. Cleared, not unverifiable, but recorded here so the mechanical flags are not re-reported downstream. · *would settle it:* `/tmp/.../aiwf help  (no assign-pending verb listed — as expected for an unbuilt proposal)`
- **ADR-0003** — Which ref's verdict the project intends to stand. · *why:* The branch rejection is signed by a human with a written, measured rationale, and main's acceptance predates it. Choosing between them is a governance decision, not a measurement. · *would settle it:* `Ask the human; then aiwf promote ADR-0003 rejected --force --reason "..." on main, or the reverse reconciliation on the branch.`
- **ADR-0004** — The first run of `aiwf archive --apply` in an existing repo is the migration, and re-runs on a clean tree are a no-op. · *why:* Running it would mutate the repository. The code path is present and shaped as described — internal/verb/archive.go returns '&Result{NoOp: true, ...}' when no candidate sweeps, and builds exactly one Plan with trailers {aiwf-verb: archive, aiwf-actor} otherwise — and a real historical sweep commit (bef74b768) matches the described body/trailer shape, but the idempotence claim itself was not exercised. · *would settle it:* `in a scratch clone: aiwf archive --apply; aiwf archive --apply (second run must report NoOp and add no commit)`
- **ADR-0006** — The aiwf-contract topical precedent and the aiwf-list/aiwf-status split precedent still hold. · *why:* Settled, not unverifiable; recorded for triage. The ADR's seven-verb contract enumeration matches the shipped CLI exactly — bind, unbind, verify, recipes, and recipe show/install/remove, no more and no less — and both aiwf-list/SKILL.md and aiwf-status/SKILL.md still ship as separate skills. internal/policies/skill_coverage.go exists and enforces frontmatter validity, top-level verb coverage, subverb coverage, and body-mention resolution as the Consequences describe. · *would settle it:* `/tmp/.../aiwf contract --help ; /tmp/.../aiwf contract recipe --help`
- **ADR-0008** — "Multiple downstream consumers have already adopted aiwf with narrow-width trees; each will need to migrate" — the population and its migration state · *why:* external repos are outside this tree · *would settle it:* `no read-only command in this repo can settle it`
- **ADR-0010** — "Periodic touchpoint: at the next epic wrap (E-0029 or beyond), audit whether the sequencing and in-branch rules held up." · *why:* E-0029 is done, but I found no wrap artefact or entity recording that this audit was performed; absence of a record is not proof it did not happen. · *would settle it:* `rg -n 'ADR-0010' work/ docs/ --glob '*wrap*' and reading E-0029's wrap artefact for a sequencing-audit section`
- **ADR-0010** — The three 'Validation' revisit triggers (bisect/revert pain from main iteration; confusion about which branch carries which planning state; AI-side ritualization friction) have not fired. · *why:* These are experience judgments about how the model has lived, not properties measurable from the tree. G-0060 ('Patch ritual is loosely defined; no kernel-level rules for shape, scope, branch, or audit trail') is still open and is adjacent to the second trigger, but I did not judge whether it constitutes the trigger firing. · *would settle it:* `n/a — human judgment; the adjacent evidence is `aiwf show G-0060`.`
- **ADR-0013** — 'Explicit fallback: if … the authorized-scope fixture scaffolding proves to exceed a single milestone's worth of work … the cellcoverage exemption is recorded explicitly (the global rule named in an allowlist with this ADR as the rationale).' · *why:* The fallback was not taken (CellFixture.AuthorizeScope landed), so there is nothing to verify. But M-0146/AC-2's exercise of the scope-gated cell lives in internal/policies/m0146_scope_machinery_test.go rather than inside the cellcoverage cell-walk itself, which is the shape the fallback describes; whether that counts as full integration or a half-fallback needing a recorded exemption is a judgment I did not make. · *would settle it:* `go test -run 'TestM0146_ScopeReachMachinery' ./internal/policies/ plus reading M-0146's AC bodies: aiwf show M-0146`
- **ADR-0015** — TTY-interactive consent prompts [y/N] before writing. · *why:* The prompt arm cannot be exercised without a real TTY; the repo's own coverage:ignore on the sibling hook path records the same limitation ('go test's stdin is never a real TTY … no pty library in this repo's dependencies'). The call site and its gating condition are verified by reading the compiled-in code path, but the interactive behavior itself is not measured here. · *would settle it:* `run `aiwf update --statusline` from an interactive terminal in a throwaway repo and observe the [y/N] prompt`
- **ADR-0016** — The GitHub repo 23min/ai-workflow-rituals is archived (read-only) with a README pointer to 23min/aiwf. · *why:* This is a claim about an external service; nothing in the working tree can settle it, and CLAUDE.md's corroborating sentence is prose, not a measurement. · *would settle it:* `gh repo view 23min/ai-workflow-rituals --json isArchived,description`
- **ADR-0017** — Decision #5's operational guarantee — every log record is emitted as exactly one Write() call, so concurrent appenders never interleave · *why:* The file handle is O_APPEND and slog's handlers write once per record, but proving no buffered writer ever splits a record needs a running multi-process experiment, which is out of scope for a read-only audit. · *would settle it:* `go test -run TestConcurrent ./internal/logger/ (internal/logger/concurrent_test.go exists and appears to target exactly this)`
- **ADR-0018** — The pre-resolved `unknown-finding-code claudemd-guidance-unwired` — cleared as a false positive. · *why:* The code exists, but as a doctor advisory string literal rather than a codespkg.Code, which the mechanical index does not see. internal/cli/doctor/guidance.go:39 constructs 'claudemd-guidance-unwired: advisory — …' and four tests in guidance_test.go assert on it. The ADR's Consequences claim ('An advisory aiwf doctor finding (claudemd-guidance-unwired) surfaces an unwired tree with the exact remediation command (aiwf update)') is accurate, including the remediation text. · *would settle it:* `already settled — grep -rn 'claudemd-guidance-unwired' internal/ → guidance.go:39 val := "claudemd-guidance-unwired: advisory — " + skills.GuidanceFile + " exists but CLAUDE.md does not import it; run `aiwf update` to wire it"`
- **ADR-0018** — "That is the only zone free of Claude Code's import-approval dialog (verified empirically on Claude Code 2.1.177; an out-of-repo import silently fails to load in a headless session)." · *why:* A claim about a third-party product's behavior at a pinned version, correctly framed as a dated empirical observation. Nothing in this repo can confirm or refute it, and the pinned version is far behind current Claude Code, so the behavior may have changed. · *would settle it:* `A headless Claude Code session with a ~/.claude/ import in CLAUDE.md, checking whether the file loads`
- **ADR-0018** — The decision's mechanism claims — verified, recorded here for completeness. · *why:* Confirmed by command: ensureGuidanceImport exists at internal/initrepo/initrepo.go:884 and is called from the init/update pipeline at :477; the opt-out is config.GuidanceConfig.WireClaudeMd (`yaml:"wire_claudemd,omitempty"`, config.go:734) documented as 'default true' in schema.go:68, and initrepo.go:887 returns ActionSkipped 'disabled via aiwf.yaml guidance.wire_claudemd'; `aiwf init --help` exposes no guidance flag (only --statusline/--wire-settings/--scope/--enable-hook/--no-prompt/--skip-hook/--allow-untagged-statusline/--actor/--root/--dry-run), so 'There is deliberately no CLI flag' holds; --wire-settings still ships as ADR-0015's instance, and ADR-0015 is still status accepted, matching 'ADR-0015 is not superseded'. · *would settle it:* `already settled — grep -rn 'ensureGuidanceImport|wire_claudemd' internal/ | grep -v _test ; /tmp/.../audit/aiwf init --help ; /tmp/.../audit/aiwf show ADR-0015 --format=json`
- **ADR-0019** — The pre-resolved `unknown-finding-code` for **embedded-rituals** is a real defect. · *why:* Cleared as a false positive: `embedded-rituals` is the on-disk directory name of the go:embed tree (internal/skills/embedded-rituals), not a check finding code. The ADR's sentence 'auto-discovered by the `embedded-rituals` tree embed' names the directory correctly. · *would settle it:* `grep -rn 'go:embed' internal/skills/*.go  (skills.go:50://go:embed embedded-rituals)`
- **ADR-0023** — A `cd` into a sibling worktree is reset back to the workspace root in a devcontainer, while a `cd` into an in-repo worktree persists. · *why:* The ADR presents this as an empirical finding about Claude Code's sandbox behavior, not about aiwf's code; reproducing it needs a live session in a devcontainer, not a repo command. · *would settle it:* `In a devcontainer session: create both placements with `aiwf worktree add`, cd into each, and observe whether the session cwd persists across turns.`
- **ADR-0028** — The deployer role agent was dispatched ~0 times across 73 sessions. · *why:* A dated observation over session transcripts that are not in the repository; the archived G-0353 is the only record. · *would settle it:* `Re-run whatever transcript census produced G-0353's figure over the session store; nothing in this repo carries the data.`
- **ADR-0029** — the ADR's own account of its authoring ('This ADR was written without noticing ADR-0022 said this already; a narrower keyword search … missed ADR-0022's phrasing') · *why:* this is drafting-history narration of the kind CLAUDE.md's §"state the conclusion, not the drafting history" rule discourages in entity bodies, but it sits in an ADR's Context explaining the relationship to a prior ADR, which is the labeled-provenance carve-out. Judgment call, not measurable — flagging it for a human rather than scoring it a breach. · *would settle it:* `human judgment against CLAUDE.md §"Writing docs, entity bodies, and code comments"`
- **ADR-0029** — the supporting claims I did verify: --shape-only runs only TreeDiscipline; verb.Apply's five pre-write refusals; SetArea/RenameArea skip the gate and mutate only Area; add/promote/edit-body/rename/reallocate/cancel all reach projectionFindings · *why:* all measured true — recorded here so the ledger shows what was checked and cleared, not just what failed · *would settle it:* `sed -n '422,446p' internal/cli/check/check.go; grep -n 'CheckForceTrailerCoherence\|checkStagedConflict\|checkUncommittedConflict' internal/verb/apply.go; grep -rn 'projectionFindings' internal/verb/*.go`
- **ADR-0030** — PACKET CORRECTION — the pre-resolved finding `unknown-finding-code: unresolved-unverified` is a false positive of the mechanical pass. · *why:* `unresolved-unverified` is a *subcode*, not a finding code, so a scan of finding-code constants misses it. It is defined in non-test source and documented in the shipped check skill; the ADR's use of it is correct. · *would settle it:* `grep -rn "unresolved-unverified" --include='*.go' internal/ | grep -v _test  →  internal/check/unverified_resolution.go:13: `const unverifiedSubcode = "unresolved-unverified"`, plus hint.go:58 and :89, and internal/skills/embedded/aiwf-check/SKILL.md:134-135`
- **ADR-0030** — VERIFIED SOUND — the escalation invariant, the read-side blob reads, and the ADR-0001 status reference all hold. · *why:* Recorded so a re-auditor does not re-derive them. Escalation: `TestCrossBranchEscalation_PendingThenUnresolved_M0259AC4` (internal/cli/cliutil/treeload_test.go:210) drives pending → branch-deleted → unresolved, untagged so it runs on every push. Read-side: `buildCrossBranchShowView` (internal/cli/show/show.go:399) reads blobs via `gitops.BlobReader` and labels the result; `internal/cli/list/list.go:38-47` carries `CrossBranchRef`/`CrossBranchCollision`/`CrossBranchRefs`. ADR-0001 is still `proposed`, as the References section says. · *would settle it:* `go test -run 'TestCrossBranchEscalation_PendingThenUnresolved_M0259AC4' ./internal/cli/cliutil/  (not run here — read-only pass used source inspection)`
- **ADR-0032** — The TTY `[y/N]` consent prompt defaults to declining, and absent a TTY init silently refuses rather than hanging. · *why:* Exercising the interactive path requires running `aiwf init` against a fresh repo with an attached TTY, which is a mutating verb and outside read-only scope. · *would settle it:* `go test -run 'TestGateHookDecisions' ./internal/cli/cliutil/ (the non-TTY arms are pinned there; the TTY-attached default would need a pty harness)`
- **ADR-0033** — "Measured directly: three of four `docs/adr` files linking into `work/` were broken by since-moved targets, a 75% rot rate" · *why:* A dated 2026-07-09 observation about a tree state that the epic this ADR authorized then repaired; it is not a claim about the current tree and re-deriving it would require checking out the pre-E-0063 tree. · *would settle it:* `git checkout 8d7379ec3^ -- docs/adr && for each markdown link into work/ test the destination's existence (in a scratch worktree, not this checkout)`
- **ADR-0035** — "All three consumers route through that one helper." · *why:* Settled: verified. `trunk.ScanCrossBranch` is called from exactly three non-test sites — `cliutil.LoadTreeWithTrunk`, `show`, and `list` — matching the three the ADR names. · *would settle it:* `grep -rn 'ScanCrossBranch' --include='*.go' internal/ cmd/ | grep -v _test.go  →  internal/cli/cliutil/treeload.go:121, internal/cli/show/show.go:412, internal/cli/list/list.go:382`
- **ADR-0035** — "any new consumer of the cross-branch collision set must read a collision result only after a local-tree miss" — still honored. · *why:* Settled: verified against the one consumer added since. ADR-0041's `cross-branch-local-only` branch and the collision branch both sit inside `if !ok {` (the local index miss) in `refsResolve`. · *would settle it:* `sed -n '630,672p' internal/check/check.go  →  `target, ok := idx[entity.Canonicalize(ref.Target)]` / `if !ok {` guarding both `crossBranch[...]` lookups`
- **ADR-0035** — "collision-stat cost drops from O(entities × refs) to O(locally-absent ids) — zero work, and sub-second wall clock, in the common all-merged state." · *why:* The structural half is verified (`absentHits` filters before `detectCollisions`, and returns nothing when every id resolves locally). The wall-clock half is a performance measurement that would need a timed before/after run on a repo of this scale — not something a read-only pass can reproduce without building the pre-M-0265 binary. · *would settle it:* `Build the binary at M-0265's parent commit and time `aiwf list --kind gap` against this tree, versus the same on HEAD. Structural half already settled by: sed -n '482,495p' internal/trunk/trunk.go → `Collisions: detectCollisions(ctx, workdir, absentHits(hits, presentLocally))``
- **ADR-0036** — A1 verdict — the code honors the decision. · *why:* Not unverifiable; confirmed on every load-bearing clause. Same-status promote NoOps above the force check and above ValidateTransition (`internal/verb/promote.go:95-96`: `if e.Status == newStatus && entity.IsAllowedStatus(e.Kind, e.Status) && !promoteWouldWrite(...) { return &Result{NoOp: true, NoOpMessage: fmt.Sprintf("%s is already %s; nothing to change", …)} }`). Cancel converges on ANY terminal, not just cancel's own (`internal/verb/cancel.go:51-52`: `if entity.IsTerminal(e.Kind, e.Status) { return &Result{NoOp: true, NoOpMessage: … "is already at terminal status %q; nothing to cancel"} }`). AC granularity mirrors both (`internal/verb/ac.go:203-204`, `:281-282`). The IsAllowedStatus conjunct implements the ADR's R1-before-R2 ordering. `terminalIllegal` still exists as the spec surface (`internal/workflows/spec/rules.go:139`). Targeted run: `go test -run 'TestPromoteAC_SameStatus_ReturnsNoOp|TestCancelAC_TerminalStatus_ReturnsNoOp|TestPromoteAC_SameStatus_NoOpEvenUnderForce' ./internal/verb/` → `ok  github.com/23min/aiwf/internal/verb  0.339s`. · *would settle it:* `go test -run 'TestPromoteAC_SameStatus|TestCancelAC_TerminalStatus' ./internal/verb/`
- **ADR-0036** — The end-to-end operator behavior (exit 0, zero commits, message text) for a same-status `aiwf promote` on a real repo. · *why:* Verifying this end-to-end requires running a mutating verb, which the read-only rule forbids. Verified instead at the verb layer (Result.NoOp / NoOpMessage above) plus the passing NoOp test suite, which constructs disposable repos. · *would settle it:* `in a throwaway repo: `aiwf promote <id> <its-current-status>; echo $?; git rev-parse HEAD` before and after`
- **ADR-0037** — The set of exactly four surfaces that previously documented the unconditional behavior. · *why:* The ADR does not name them and the pre-M-0281 text is only recoverable from history; whether each of the six current surfaces carried the old rule cannot be settled from the working tree. · *would settle it:* `git log -S 're-derives the slug' --all -- CLAUDE.md internal/ docs/ (then diff each pre-ADR-0037 revision)`
- **ADR-0039** — "The trailer value is no longer a legal one; historical commits carrying it stay readable" · *why:* I found no mechanical enumeration of legal `aiwf-verb:` trailer *values* — the trailer policies in internal/policies/ pin keys and ordering, not the value vocabulary. The second half is true by construction (aiwf history reads trailers without validating them), but 'no longer legal' is not pinned by anything I could locate. · *would settle it:* `grep -rn 'aiwf-verb' internal/policies/*.go internal/gitops/*.go and look for a closed value set`
- **ADR-0039** — "The known consumer population had migrated before this landed, and a tree that has not is out of scope by decision rather than by oversight." · *why:* a claim about repositories outside this tree · *would settle it:* `no read-only command in this repo can settle it`
- **ADR-0039** — the packet's `missing-path` internal/verb/rewidth.go and `unknown-verb` aiwf rewidth for this subject · *why:* both are the ADR *announcing the deletion* of those things — proposing/recording, not citing as extant. Correct usage, no finding. Confirmed by command: ls reports both absent, which is what the ADR says should be true. · *would settle it:* `ls internal/verb/rewidth.go internal/cli/rewidth  (both: No such file or directory, as the ADR states)`
- **ADR-0040** — 'The kernel ... enforced it in one verb of four' and 'CheckTrailerCoherence was reached from two verbs and no others' (Context, pre-fix state). · *why:* These describe the tree before E-0079 landed; they are historical Context, not current-truth assertions, and the present tree necessarily disagrees. · *would settle it:* `git show <sha-before-E-0079>:internal/verb/apply.go && git log --diff-filter=M --format='%h %ad %s' -S'CheckTrailerCoherence' -- internal/verb/`
- **ADR-0041** — The pre-resolved `unknown-finding-code` for `unresolved-unverified` is a phantom. · *why:* CLEARED — false positive, exactly the class the brief warns about. `unresolved-unverified` is a real subcode declared as an unexported constant the mechanical index does not match. · *would settle it:* `grep -rn 'unverifiedSubcode' --include='*.go' internal/check/  →  internal/check/unverified_resolution.go:13: `const unverifiedSubcode = "unresolved-unverified"`; also asserted live at internal/cli/integration/check_fast_test.go:236`
- **ADR-0041** — "Instances of this shape run at roughly one per three weeks with a window long enough to span a push." · *why:* A rate over historical incidents of a condition that leaves no distinguishing trailer. Reconstructing it would require walking every branch's ref state at each historical commit, which no read-only command reproduces. · *would settle it:* `No single command. Would need a scripted replay of `aiwf check` at each historical HEAD with the ref state of that day reconstructed — not achievable read-only.`
- **ADR-0041** — ADR-0041's Validation fixture exists as specified (mint local → push → delete remote → delete branch). · *why:* Settled, not unverifiable — recording it here for completeness: the fixture exists and drives all four stages. · *would settle it:* `grep -rn 'cross-branch-local-only' internal/cli/integration/cross_branch_publication_lifecycle_test.go  →  lines 160 and 180 assert `cross-branch-local-only` / `SeverityError` at the unpushed and unpublished stages`
- **ADR-0042** — "The knob has never been enabled: not by this repo ... and not by any consumer." · *why:* The repo half verifies (aiwf.yaml sets docs.strict and no tdd: block: `cat aiwf.yaml`); consumer repos are not reachable from this tree. · *would settle it:* `grep -n 'tdd:' -A 3 aiwf.yaml in each consumer repo`
- **ADR-0043** — a body with invented top-level headings is accepted by `aiwf add` for all six kinds and draws no finding from `aiwf check` · *why:* Running `aiwf add` is a mutating verb, excluded by the read-only rule. Settled indirectly: the only gate is requireNonEmptyBornCompleteBody -> EmptyRequiredSections, which skips absent headings, and no membership scan exists anywhere in internal/. · *would settle it:* `in a throwaway clone: aiwf add gap "x" --body-file <body with only invented `## ` headings> && aiwf check`
- **CLAUDE.md** — "The chokepoint is the AC-promote command" (§AC promotion requires mechanical evidence). · *why:* Every other use of the word 'chokepoint' in this file names a mechanical gate with a file behind it, so this reads as a claim that `aiwf promote M-NNNN/AC-N met` enforces the evidence rule. I found no evidence gate in internal/cli/promote/ and the acs-tdd-audit rule only fires for `tdd: required` milestones — but proving a negative about the promote path safely requires driving the verb, which is a mutating run I am forbidden to do. It may also be intended non-mechanically ('this is the moment the rule binds'), which would be a wording problem rather than a false claim. · *would settle it:* `In a disposable repo: `aiwf add milestone --epic <E> --tdd none --title t`, `aiwf add ac ...`, then `aiwf promote <M>/AC-1 met` and observe whether it refuses absent any test reference.`
- **CLAUDE.md** — "`make ci` runs the full CI-parity gate" and "CI runs the full gate on every push". · *why:* Verified structurally — `ci: vet lint test-cov coverage-gate-only selfcheck` covers go.yml's vet/lint/test/selfcheck jobs — but go.yml additionally runs a `-tags testpins` suite, the `vuln` (govulncheck) job, the `build` job and a linter-config rule-firing harness, which `make ci` does not. The Makefile's own comment says 'It is not literally everything CI does'. Whether 'full CI-parity' is false or merely loose is a judgment call, not a measurement. · *would settle it:* `n/a — the discrepancy is documented at Makefile:258-261; the question is only whether CLAUDE.md's wording should inherit that caveat.`
- **D-0006** — "Sources — First-principles: R-FP-0130 (legal-workflows-first-principles.md, §6c scope FSM); Audit: R-AUDIT-0186 (legal-workflows-audit-r1.md, §6 …)" · *why:* The cited research artifacts use an R-NNNN identifier scheme with no kernel entity kind behind it, and `legal-workflows-audit-r1.md` is not the filename in docs/design (which carries legal-workflows-audit.md). Whether the citations resolve to a live document was out of scope for a claims-and-disposition audit. · *would settle it:* `ls docs/design/legal-workflows-* docs/research/ && grep -rn 'R-FP-0130\|R-AUDIT-0186' docs/`
- **D-0007** — "The closed set covers every authorization use case in practice on the project today." · *why:* A claim about operator practice, not code. It could be approximated by walking every `aiwf-verb: authorize` commit and confirming each names an epic or milestone, but that measures this repo's history rather than the use-case space the sentence asserts. · *would settle it:* `git log --format='%H' --grep='aiwf-verb: authorize' | while read s; do git show -s --format='%(trailers:key=aiwf-entity,valueonly)' $s; done | sort -u`
- **D-0010** — "the kernel's own `aiwf check` ... emitted **44 errors**" under D-0009's policy · *why:* A counterfactual about a superseded code state; the errors are gone by construction now. It sits under `## Sources` as dated provenance, which is where the repo's own conventions put it. · *would settle it:* `git stash-free reconstruction: check out the M-0130/AC-2 pre-fix commit into a scratch worktree and run `aiwf check` — out of scope for a read-only audit of the live tree.`
- **D-0012** — "aiwf is pre-1.0 and no known external consumer pins this code" · *why:* Consumer repositories are outside this working tree; nothing here can enumerate downstream JSON-parsing scripts. · *would settle it:* `grep -rn 'gap-resolved-has-resolver' <each consumer repo> --exclude-dir=.git`
- **D-0013** — The `unknown-finding-code` pre-resolution for `fsm-transition-illegal` is a false positive on existence — the code does exist, as a typed `codes.Code` constant the mechanical index does not match. · *why:* Cleared by grep; recorded here because the packet flagged it. The finding above is about *where* it is emitted, not whether it exists. · *would settle it:* `grep -rn "fsm-transition-illegal" --include="*.go" internal/entity/`
- **D-0013** — A coded refusal actually exits 1 against the real binary. · *why:* Settling it end-to-end means running a mutating verb (an illegal `aiwf promote`), which the read-only rule forbids even though the refusal path writes nothing. Verified statically instead: FinishVerbOutcome returns ExitFindings when entity.Code(err) reports coded, ExitInternal for internalError, ExitUsage otherwise. · *would settle it:* `aiwf promote <terminal-entity> <illegal-status> --format=json ; echo $?   # expect 1 and error.code`
- **D-0014** — the Context's account of the pre-decision state (full-graph tree.Reaches at two sites, two production callers only) · *why:* Reaches/ReachesAny no longer exist, so the pre-state cannot be measured at HEAD; the Resolution that removed them is confirmed honored (tree.ReachesScope at internal/tree/tree.go:440 traverses exactly parent chain, composite rollup, one discovered_in hop; grep for 'func (t *Tree) Reaches\b' returns nothing). · *would settle it:* `git show <pre-M-0141-sha>:internal/tree/tree.go | grep -n 'func (t \*Tree) Reaches'`
- **D-0017** — The `unknown-go-symbol` pre-resolutions for `CommitMetadata` and `IsCherryPick` are false positives. · *why:* Cleared. Both appear only as *proposed* shapes: `IsCherryPick bool` is option 1's hypothetical field and option 2's hypothetical oracle method; `CommitMetadata` is named as the carrier to introduce *if* a second rule ever needs the signal. Neither is cited as extant, so their absence is correct rather than rot. Separately verified sound: `scope.Commit` is exactly `{SHA, Trailers}` (internal/scope/scope.go:51-54) as the body asserts, and `BranchOracle` does carry `FirstParentBranches` (internal/check/isolation_escape.go:251). · *would settle it:* `grep -rn "CommitMetadata\|IsCherryPick" --include="*.go" internal/ cmd/   # no output, as the decision intends`
- **D-0018** — the pre-AC-2 behaviour that `--branch garbage` returned branch-not-found via the future-binding carve-out · *why:* the pre-AC-2 code path is gone; only the post-AC-2 path is measurable at HEAD · *would settle it:* `git show 51462d52^:internal/verb/authorize.go and drive the fixture against a binary built from that tree`
- **D-0019** — The pre-resolved `dangling-entity-ref E-9999` finding ("A corrupt `epic/E-9999-...` does not stop the rule from policing commits on `epic/E-0030-...`") · *why:* Cleared, not a defect: E-9999 is an illustrative backticked ref name in a syntax example, which is exactly the carve-out the id-shape rule grants. A full aiwf check over the tree reports no body-prose-id finding, so the kernel agrees. · *would settle it:* `already settled — /tmp/.../audit/aiwf check --format=json → 9 findings, all terminal-entity-not-archived (8) + archive-sweep-pending (1); no body-prose-id`
- **D-0021** — The headline decision — verified, recorded for completeness. · *why:* Confirmed by command and still correct: `aiwf doctor --format=json` fails with 'unknown flag: --format' (the flag set is --check-latest / --check-rituals / --root / --self-check / --write-health / -h), and the substring the decision ships on is still emitted verbatim at doctor.go:261 as 'detached-head: advisory — no symbolic HEAD; …', with report_branches_test.go:409 still asserting on it. D-0020 is status superseded/archived, so the precedent D-0021 cites at rationale 4 is itself a superseded record — worth a glance but not a defect in D-0021, whose point is the deferral pattern rather than D-0020's content. · *would settle it:* `already settled — /tmp/.../audit/aiwf doctor --format=json → 'aiwf: unknown flag: --format' ; grep -n 'detached-head' internal/cli/doctor/doctor.go`
- **D-0021** — Whether the shipped severity vocabulary (warn/error) can satisfy the AC-7 matrix's severity: "advisory" when the future doctor-shape milestone lands. · *why:* The deferred requirement quotes severity: "advisory", but doctor.Severity ships only SeverityWarn ("warn") and SeverityError ("error"), and the detached-head line is a SeverityWarn problem whose message text merely begins 'detached-head: advisory —'. Whether the future milestone would add an advisory severity or restate the AC is a design question this audit should not answer. · *would settle it:* `The doctor-shape milestone's spec, when written; today: sed -n '1,25p' internal/cli/doctor/problems.go`
- **D-0023** — The M-0161/AC-9 body's forecast counts (paraphrased "76 total") summed only the listed surfaces. · *why:* The claim is about what an earlier milestone's author summed when producing a forecast, which no artifact records. The 76 figure is real — M-0162's own wrap section says '129 cells in catalog (~70% above the original 76-cell forecast — body line 215 hedged this as a planning estimate)' — but whether reallocate was excluded from that sum, as opposed to simply overlooked, is not recoverable. · *would settle it:* `reading M-0161/AC-9's body line 215 forecast derivation and checking whether it itemises per-surface counts that sum to 76 without a reallocate term`
- **D-0023** — The 7 reallocate scenarios remain unpinned and the follow-up cost is 7 stamps + 7 cell additions via the two named scripts. · *why:* The 'remain unpinned' half IS verified (0 Pin call sites; the file's own header comment says 'The corpus collapses to seven representative shapes'), and both scripts exist at scripts/m0162-stamp-cellid.sh and scripts/m0162-build-ac3-cells.py. What is not verifiable read-only is that running them today still produces a clean 7-cell expansion — the catalog has moved from 76 to 129 cells since. · *would settle it:* `on a scratch clone: scripts/m0162-stamp-cellid.sh over reallocate_scenarios_test.go, then scripts/m0162-build-ac3-cells.py, then `go test -tags testpins -run TestM0162_AC4_Bijection ./internal/policies/``
- **D-0024** — The shipped split architecture matches the decision's description. · *why:* Settled, not unverifiable; recorded for triage since this is the decision's whole substance. internal/policies/m0162_ac4_bijection_test.go is `package policies` with no build tag and declares TestM0162_AC4_Bijection at line 82, whose doc says it 'enforces 3 of the 4 bijection invariants … statically'. internal/cli/integration/bijection_posthook_testpins_test.go carries `//go:build testpins` and declares bijectionPostHook at line 24, wired at internal/cli/integration/setup_test.go:64 (with a no-op sibling in bijection_posthook_nontestpins_test.go). evaluateBijection exists at m0162_ac4_bijection_test.go:233 and the sabotage subtests in m0162_ac4_sabotage_testpins_test.go drive it. bijectionAllowlist() at line 146 holds exactly 14 entries. TestM0162_AC4_AllowlistClaimsResolve is at m0162_ac4_allowlist_verification_test.go:76. .github/workflows/go.yml:116 runs `go test -tags testpins … ./...` and the Makefile has a test-pins target. · *would settle it:* `awk '/func bijectionAllowlist/,/^}/' internal/policies/m0162_ac4_bijection_test.go | grep -cE '^\s+"branch-cell'   (returns 14)`
- **D-0025** — The decision's kept/rejected/adopted ledger still matches the code. · *why:* Settled, not unverifiable; recorded for triage. `filepath-join-segment-by-segment` is gone (no such policy id; internal/policies/golangci_gocritic_filepath_test.go guards the gocritic filepathJoin superset it was traded for) and M-0167/AC-2 is `met`. `no-time-now-in-core` is still the bespoke AST policy (internal/policies/no_time_now_in_core.go:104 `Policy: "no-time-now-in-core"`) and M-0167/AC-1 ('Migrate no-time-now-in-core to forbidigo, delete the bespoke policy') is `cancelled` — the reversal held. `layering-direction`, `m0137-ac3-batched-walker`, and `acks-helper-lift` all still exist, and the `grandfatherDark` ledger is still live with an anti-stale guard (TestPolicy_FiringFixtureNoStaleAllowlist). · *would settle it:* `grep -rn '"no-time-now-in-core"\|"layering-direction"\|"m0137-ac3-batched-walker"' internal/policies/ ; grep -n 'status:' work/epics/archive/E-0042-*/M-0167-*.md`
- **D-0026** — "the four consumer sites (`area-unknown`, `set-area`, `add --area`, the read-filter note) are confirmed to route through the predicate" — that the enumeration is still exhaustive. · *why:* The four named membership-decision sites all still route through entity.IsValidAreaValue (check/area_unknown.go:100, verb/setarea.go:105, cli/add/add.go:331, cliutil/area.go:44), and the same four are named in the predicate's own doc comment (internal/entity/area.go:15-16). Sites added since — config.go:327, verb/renamearea.go:67, check/area_mistag.go:66, cliutil/completion.go:186 — compare against the `entity.AreaGlobal` const rather than the predicate, but they are reserved-name rejections, completion offering, and a mistag exemption, not membership decisions, so I judged them outside the enumeration rather than a drift. Whether the decision intends them in scope is an authoring-intent question I cannot settle by command. · *would settle it:* `grep -rn "AreaGlobal|IsValidAreaValue" internal/ --include="*.go" | grep -v _test.go  (measured: only internal/entity/area.go:10 declares the literal "global"; grep -rn '\"global\"' internal/ --include="*.go" | grep -v _test.go returns that single line, confirming no parallel == "global" check)`
- **D-0027** — "`area-mistag` is a non-escalating **warning** (deliberately absent from `ApplyAreaRequiredStrict`), so a stale suppression costs a missed advisory, never a blocked push." · *why:* Settled: verified. The finding is constructed with `Severity: SeverityWarning`, and `ApplyAreaRequiredStrict`'s switch escalates six codes, none of them `CodeAreaMistag`. · *would settle it:* `sed -n '108,125p' internal/check/area_mistag.go → `Code: CodeAreaMistag, Severity: SeverityWarning`  ·  sed -n '42,54p' internal/check/area_unknown.go → `case CodeAreaUnknown, CodeAreaDeadGlob, CodeAreaOverlap, CodeAreaUnslotted, CodeAreaCoverageRootMissing, CodeAreaCoverageNoPaths:``
- **D-0027** — "The suppression is keyed purely on the entity id and is permanent." · *why:* Settled: verified. `WalkAcknowledgedMistags` returns `map[string]bool` of canonical entity ids with no other key and no expiry. · *would settle it:* `sed -n '143,165p' internal/check/area_mistag.go → `func WalkAcknowledgedMistags(head []HeadCommit) map[string]bool` / "returns the set of canonical entity ids acknowledged"; consumed at internal/cli/check/check.go:274`
- **D-0027** — "If stale-suppression friction is ever observed in practice, the scoped (entity, blessed-foreign-area) shape is the known refinement ... Deliberately not built now (YAGNI until the friction is real)." · *why:* Settled in the decision's favour: the condition has not arisen because the verb has never been used in this repo. · *would settle it:* `git log --all --grep='aiwf-verb: acknowledge-mistag' --oneline | wc -l → 0`
- **D-0037** — The batch note asserts "D-0037 records the deferral behind ADR-0003" and asks whether D-0037 still reads correctly against ADR-0003's moved status. · *why:* The premise does not hold: D-0037 does not mention ADR-0003 anywhere (`grep -n 'ADR-0003' work/decisions/D-0037*.md` → no match). D-0037 defers ADR-0001, and ADR-0001 is still `proposed`, exactly as the decision states — so D-0037 reads correctly. Establishing ADR-0003's status by command anyway: it is `accepted` at this HEAD on `main`, and `rejected` on three unmerged branches (`epic/E-0086-…`, `initiative/preflight-amendments`, `origin/initiative/preflight-amendments`). That divergence is a real finding, but it belongs to ADR-0003, not to any subject in this batch — flagging it for whichever batch owns ADR-0003, together with the fact that the epic body on those branches already writes "(ADR-0003, `rejected`)" as settled truth while trunk says `accepted`. · *would settle it:* `grep -n '^status:' docs/adr/ADR-0003-add-finding-f-nnn-as-a-seventh-entity-kind.md → `status: accepted`  ·  git show initiative/preflight-amendments:docs/adr/ADR-0003-add-finding-f-nnn-as-a-seventh-entity-kind.md | grep -m1 '^status:' → `status: rejected``
- **D-0037** — "34 reallocate events against 986 add events (~3.4%) as of 2026-07-04 ... Re-measured at decision time (2026-07-16): 35 reallocate / 1129 add (~3.1%)" · *why:* Settled forward, not backward. The historical counts cannot be re-derived without checking out the 2026-07-04 and 2026-07-16 states, and both are explicitly dated observations, which is the sanctioned form. What matters — that the trigger has not fired — is verified: reallocate is still 35 (no new events at all since the decision), add is now 1535, giving 2.3%. · *would settle it:* `git log --all --grep='^aiwf-verb: reallocate$' --oneline | wc -l → 35  ·  git log --all --grep='^aiwf-verb: add$' --oneline | wc -l → 1535`
- **D-0048** — The three deferred per-kind subverbs (`aiwf gap discovered-in`, `aiwf decision relates-to`, `aiwf contract linked-adrs`) and the rejected `aiwf relate --field` multiplexer are absent from the CLI. · *why:* Not unverifiable — confirmed absent, and confirmed to be *proposals* rather than citations. Recorded here only to close the packet's pre-resolved `unknown-verb` flags: the body says 'No code is written for these now; only the shape is fixed', and `relate` appears only as the alternative the decision rejects ('**Not** a generic `aiwf relate --field <name>` multiplexer'). Verified with `aiwf --help` (no gap/decision/relate root verbs) and `aiwf milestone --help` (only depends-on, tdd). · *would settle it:* `/tmp/.../audit/aiwf --help; /tmp/.../audit/aiwf milestone --help`
- **D-0050** — "G-0489 shipped 52 phrase-literals across 22 tests" and "successive rounds found 3, 8, 9, then 10 blocking findings". · *why:* These are dated observations about a completed review process, not standing claims about the current tree. Re-deriving 'phrase-literal' would require a definition the decision does not give, and the review-round counts lived in a transcript no longer in the repo. The decision carries a `## Provenance` section labelling them as such, which is the shape the repo's authoring rule prescribes. · *would settle it:* `git log --oneline -- internal/policies/ | grep G-0489, then counting strings.Contains assertions over ritual prose in each diff — approximate at best.`
- **D-0050** — "This governs tests written from here" — that policy tests added after 2026-07-31 honour the structure-not-phrase rule. · *why:* Roughly 20 new *_test.go files landed under internal/policies/ since the decision date; deciding whether each of their assertions is structural or phrase-content is a per-assertion judgment call across hundreds of lines, not a command. I verified the decision's own claims about its exemplar and header comment, not the whole suite's compliance. · *would settle it:* `A per-file review of `git log --since=2026-07-31 --diff-filter=A --name-only -- internal/policies/` output against the D-0050 bar; the closest mechanical proxy is the existing G-0584, which already tracks a subset of this class.`
- **D-0052** — The Reasoning section's measurement — 'roughly twice as much ordinary debris as teaching' across the three files, and 'a narrow placeholder in the shipped `description:` frontmatter' in one of them. · *why:* The measurement was taken on pre-sweep content that M-0288 has since rewritten, so the current tree cannot reproduce it. It is a dated observation supporting a decision, which is the sanctioned shape. · *would settle it:* `git show <pre-M-0288 sha>:internal/skills/embedded/aiwf-check/SKILL.md and the two ritual SKILL.md files, then classify each narrow-id token`
- **D-0053** — the entries are honest — each is verified to fail against its own pre-fix text, so none is vacuous · *why:* Establishing non-vacuity requires reverting each of the eight pinned skill-prose corrections and re-running the matching test, which is a mutating experiment. · *would settle it:* `For each pin: `git stash`-free worktree copy, revert the corrected line in the SKILL.md, then `go test -run <TestName> ./internal/policies/` and confirm red (a mutation run, e.g. `make mutate-diff`-style, outside read-only scope)`
- **D-0054** — "at a median of six commits each" · *why:* The population is undefined — commits carrying the gap's own trailer, or every repo commit between its add and its close. The two give very different medians and the body does not say which. · *would settle it:* `git log --format='%H|%at' plus a trailer scan for `aiwf-entity: G-NNNN`, once the intended population is stated`
- **D-0055** — `golangci-lint` lints the union of the declared build tags in a single pass rather than each tag configuration separately. · *why:* This is an assertion about upstream golangci-lint's behavior, not about this repo's code. The repo's own .golangci.yml states the same thing in a comment, which is a second copy of the claim rather than independent evidence; settling it means observing the linter's behavior on both tag configurations, and running golangci-lint over the module is outside the targeted read-only scope. · *would settle it:* `golangci-lint run --print-issued-lines=false ./internal/cli/integration/ 2>&1 | tail -5; then temporarily drop `testpins` from run.build-tags in a scratch copy of .golangci.yml and re-run with --config on that copy, comparing which of pin_nontestpins_test.go / pin_testpins_test.go each pass reports`
- **D-0057** — 'Nobody has asked' whether a consumer could opt in to pre-commit for verb commits. · *why:* A negative claim about external consumer reports; nothing in this repo records or refutes it. · *would settle it:* `gh issue list --search 'pre-commit hook verb commit' --repo 23min/aiwf --state all`
- **D-0062** — "the golden never moves and the reachability assertion never sees it" for a rule predicated on a sixth trailer — the residual hole the decision states rather than closes. · *why:* Demonstrating the residual means adding a rule reading a new trailer and observing the golden stay put, which is a mutation of the tree. The decision states it as a known open residual, which is the honest disposition. · *would settle it:* `add a coherenceRuleSpec entry reading an undeclared trailer on a scratch branch and run `go test -run TestCheckTrailerCoherence ./internal/verb/``
- **D-0063** — That the decision's core direction — widen the mutation space, keep the oracle invariant-shaped — remains the right one after E-0080's cancellation. · *why:* The cancel reason narrows the decision's own economics claim, and D-0063's 2026-08-08 edit already absorbed that correction; whether the residual direction should stay `accepted` or be superseded by whatever G-0121 resolves is a planning judgment, not a measurement. G-0121 (open, high) and G-0564 (open) both still cite D-0063 as the live direction, which is why I did not report it as superseded-in-fact. · *would settle it:* `n/a — human decision; the evidence is `git show 2f9c38930` against D-0063's Reasoning section`
- **G-0022** — Items 2, 3, 4 (time-bound scopes, verb-set restrictions, pattern scopes) remain unbuilt. · *why:* Verified negatively (absence of the flags), which is as strong as read-only allows; recorded here only because absence-proofs are weaker than presence-proofs. · *would settle it:* `/tmp/.../audit/aiwf authorize --help  # shows only --actor --branch --force --format --pause --pretty --reason --resume --root --to --trace; no --until/--for/--verbs/--pattern`
- **G-0022** — "Bulk-import per-entity actor attribution ... Solves the migration case where authorship is recoverable only via `git blame` on the v1 source." · *why:* The v1 source tree is not in this repo, so whether per-row authorship is in fact recoverable there cannot be measured here. The aiwf-side half is confirmed: `verb.Import` takes one `actor string` and stamps it on every plan, in both commit modes. · *would settle it:* `sed -n '419,458p' internal/verb/import.go  (confirms the aiwf-side collapse); the v1-source half needs the migration source repo`
- **G-0023** — "occasional friction is plausible: a long-running autonomous scope where every kernel-refusal-that-needs-overriding becomes a synchronous prompt to the human" — whether that friction has actually materialized since I2.5 shipped. · *why:* The gap's own revisit trigger is an observation about operating experience, not a property of the tree. No forced act inside an authorized scope is distinguishable in git from a human forcing out-of-band, because the human-only rule means the trailer is identical either way. · *would settle it:* `git log --all --format='%H %s' --grep='aiwf-force:' | wc -l, cross-read against `aiwf authorize` scope windows — and then a human judgment about whether any of those prompts was unwelcome.`
- **G-0060** — G1 verdict — the kernel-level defect still exists. · *why:* Not unverifiable; confirmed. Recorded here as the central verdict: `aiwf schema` reports exactly the six kinds and no patch; `aiwf --help` lists no patch verb; no `aiwf-patch` trailer exists anywhere in internal/. G-0366 (the cheaper adjacent roadmap-visibility gap the body names) is still open. The gap is live. · *would settle it:* `/tmp/.../audit/aiwf schema --format=json | jq '[.result.schemas[].kind]'; grep -rn 'aiwf-patch' internal/`
- **G-0068** — The seven subcodes are all still named in the aiwf-check SKILL.md (i.e. the discoverability property holds by authoring, not by policy). · *why:* Verified by reading the SKILL.md row rather than by running the policy, since running PolicyFindingCodesAreDiscoverable in isolation needs a test invocation. · *would settle it:* `go test -run 'TestPolicyFindingCodesAreDiscoverable' -v ./internal/policies/`
- **G-0073** — Measured on a scratch consumer repo, an epic hand-carrying a depends_on entry naming another epic yields `aiwf check` 0 errors and no finding on that axis. · *why:* Reproducing it end-to-end requires writing an epic with a depends_on entry, which is a mutation of a repo tree. Code inspection supports the claim — internal/entity/refs.go:33-40 emits a depends_on ForwardRef only for KindMilestone, and OptionalFields has no consumer outside `aiwf schema`'s display path (grep -rn 'OptionalFields' internal/ shows only entity.go declarations and internal/cli/schema/schema.go:116) — but no check rule was observed rejecting the key, and no rule was observed accepting it either. · *would settle it:* `in a scratch repo: aiwf init; aiwf add epic --title x; append 'depends_on: [E-0001]' to the epic frontmatter; aiwf check --format=json`
- **G-0077** — "That document — the trunk-era thesis … — is unwritten." · *why:* Absence of a document can be evidenced but the judgment of whether the current docs/working-paper.md already discharges the intent is the maintainer's. Measurably, no new file was written and the existing one has had no substantive commit since 2026-04-30; it does self-describe as a 'defended-position' thesis drawing 'on evidence from a working PoC'. · *would settle it:* `maintainer judgment; the mechanical half is `git log --oneline -- docs/working-paper.md` plus `ls docs/*.md`, both run above`
- **G-0104** — "the spike numbers (~4× faster non-race, ~2.4× with race) confirm the headroom" · *why:* A dated measurement from E-0025's spike. Reproducing it means running the full suite twice under two parallelism settings, which the brief forbids. · *would settle it:* `make test vs. a serialized run, timed — out of scope for a read-only audit`
- **G-0110** — `gremlins unleash --diff <ref>` still excludes mutants in files entirely new in the branch · *why:* Reproduction requires a branch carrying a new .go file to diff against; creating one would modify the repository, which this audit may not do. gremlins is installed (/go/bin/gremlins, reports 'version dev'; the workflow pins v0.6.0) and still exposes `-D, --diff string`, so the flag itself has not been removed. · *would settle it:* `On a scratch clone with a new .go file under internal/check: `gremlins unleash --dry-run ./internal/check` vs `gremlins unleash --dry-run --diff origin/main ./internal/check`, comparing RUNNABLE vs SKIPPED counts for that file's mutants.`
- **G-0110** — the two named commits and the M-0097 manual-review record · *why:* Verified, not unverifiable — recorded here for completeness: internal/check/epic_active_drafts.go, internal/policies/aiwf_promote_epic_active_audit.go and internal/verb/promote_sovereign_act.go all exist (the body itself annotates the rename of the third), and M-0097's `## Validation` section carries the '**Manual mutation review** (in place of `mutate-hunt` per Decisions above)' bullet. · *would settle it:* `ls internal/verb/promote_sovereign*.go; grep -n '## Validation' -A 8 work/epics/archive/E-0028-*/M-0097-*.md`
- **G-0121** — The gap's own satisfaction test — 'construct a third merge-reachable bad state of a different shape, without touching the harness, and run the harness unmodified' — has not been run. · *why:* Absence of a record is not proof it was not run. No commit, decision entity, or scenario file references such a run, but the test as specified requires a seeder who did not build the harness, and its result would live outside the tree unless someone recorded it. · *would settle it:* `git log --all --grep='falsification' --grep='third trap' -i ; aiwf list --kind decision --status accepted (scan for a recorded run)`
- **G-0121** — The verb-sequence walker's oracle is blind by construction to a check vs check --fast disagreement. · *why:* Verified structurally — the scenario's post-step assertion calls `aiwf check` alone (internal/stresstest/verb_sequence.go and checkclean.go), so no second surface is consulted. Not verified by running the walker against a tree carrying such a disagreement, since none is now reachable (G-0558 and G-0567 both closed). · *would settle it:* `go test -tags stress -run TestVerbSequence ./internal/stresstest/ against a fixture reintroducing a per-path severity divergence`
- **G-0157** — The two open slices are perf-only and can wait until worktree-count pressure makes the latency user-visible. · *why:* The gap's own resolution path says 'measure first'; timing aiwf status against a synthetic 50-worktree fixture requires creating worktrees, which is a repo mutation. · *would settle it:* `time aiwf status --worktrees on a throwaway clone seeded with 50 worktrees`
- **G-0160** — That a new edge really would produce a passing driven subtest rather than a t.Fatal in fixture setup. · *why:* Confirming it requires adding an edge to entity.transitions and running the driver, which is a mutating experiment the brief forbids. · *would settle it:* `add (epic, active) -> in_progress to entity.transitions on a scratch branch, then `go test -run 'TestM0124_PositiveDriver_LegalCells/epic-active-promote-to-in_progress' -v ./internal/policies/``
- **G-0161** — None of the twelve anti-rules has negative (assert-absence-of-enforcement) coverage today. · *why:* Absence is established by an exhaustive grep over internal/policies/ for AntiRule-referencing tests, all of which are schema/count/drift tests (m0123_ac1/ac3/ac5/ac6/ac7, m0158_scaffold). A test could in principle assert an anti-rule without naming AntiRule, so this is high-confidence rather than exhaustive. · *would settle it:* `grep -rln 'AntiRule' internal/policies/ && go test -run 'TestM0123_AC3' -v ./internal/policies/`
- **G-0166** — the two live threads are still real — (i) the AC cell is rejected at verb-time despite RejectionLayerCheckTime, (ii) the `aiwf milestone tdd` refuse-with-hint has no spec-table representation · *why:* both are confirmed by static reading of source that I did verify by command, but the AC cell's verb-time rejection is only pinned by a test I did not run (running it is safe but I limited myself to read-only inspection). Static evidence: the two spec cells still carry RejectionLayerCheckTime (internal/workflows/spec/rules.go:444 and :656); ac3KnownImplGaps still names the AC cell; `grep -rn 'milestone tdd\|MilestoneTDD\|tdd_policy' internal/workflows/spec/` returns nothing while internal/verb/milestone_tdd.go:63-78 carries the refuse-with-hint. · *would settle it:* `go test -run 'TestM0125.*CheckTime' ./internal/policies/`
- **G-0168** — "Until the verbs exist, the operator hand-edits the YAML and commits manually with a descriptive but **fictional** `aiwf-verb:` trailer" — i.e. the documented workaround is still viable. · *why:* I found no check rule or policy that validates an aiwf-verb trailer value against the real verb set (the trailer policies under internal/policies/ police source-code literals, not commit content), so the workaround appears to remain viable — but proving it would require actually committing a hand-edit, which is a mutation I must not perform. · *would settle it:* `In a throwaway clone: hand-edit a frontmatter field, `git commit -m 'x' --trailer 'aiwf-verb: edit-frontmatter' --trailer 'aiwf-entity: X' --trailer 'aiwf-actor: human/test'`, then `aiwf check` and inspect for provenance-untrailered-entity-commit / unexpected findings.`
- **G-0168** — priority: high is still the right pricing for the three remaining fields. · *why:* D-0048 explicitly defers them ('No code is written for these now') on the grounds that only tdd: showed real friction, which argues the residual is low-priority; but re-pricing an entity's priority is a human judgment, not a measurement. · *would settle it:* `n/a — human call; the evidence is D-0048 §Reasoning ¶'Deferral'.`
- **G-0169** — M-0143 wired --format/--pretty into "every mutating verb that routes through the shared cliutil.FinishVerb / DecorateAndFinish chokepoint (14 verbs)". · *why:* This is a historical statement about M-0143's scope in May 2026, not a current-state count, so today's tree cannot falsify it. For reference, 22 files under internal/cli/ call cliutil.AddFormatFlags today, and `DecorateAndFinish` no longer appears as a symbol I located in internal/cli/cliutil/apply.go (FinishVerb delegates to FinishVerbOutcome at apply.go:79). Per the repo's own 'derive the facts' rule the bare number is worth dropping either way. · *would settle it:* `git show <M-0143 merge sha>:internal/cli/integration/format_coverage_test.go and count the --format-registering leaf commands at that revision; grep -rn 'DecorateAndFinish' --include='*.go' internal/`
- **G-0169** — The three read/generate commands are the complete residual set. · *why:* I verified each named command lacks --format, but did not enumerate every Runnable leaf command in the tree to check for a fourth unwired one; TestFormatFlagUniformRollout_AC4 would catch it only if it were not exempt. · *would settle it:* `go test -run TestFormatFlagUniformRollout_AC4 ./internal/cli/integration/ after emptying formatExempt`
- **G-0178** — "SKILL.md is a cross-vendor open standard, and Codex reads the identical `name`/`description` frontmatter from `.agents/skills/`, so the Codex writer is near-verbatim (output location differs, format does not)." · *why:* An assertion about an external product's on-disk convention. Nothing in this repo can settle it, and it is the load-bearing premise for the gap's cost estimate ('near-verbatim', 'low cost'). · *would settle it:* `Consult OpenAI Codex's current documentation for its skills directory and frontmatter schema, then diff against internal/skills/embedded-rituals/plugins/*/skills/*/SKILL.md frontmatter.`
- **G-0212** — Whether the 'broad audit' the gap's title reserves ('future epic') has any remaining scope beyond the six enumerated classes. · *why:* The gap never enumerates what the audit would cover beyond its six items, so there is no claim to measure — the residue is defined only by exclusion. · *would settle it:* `A human scoping pass; mechanically, `go run ./cmd/stresstest list` enumerates what the harness covers today, but nothing states what the complete surface is.`
- **G-0215** — Cases 2 and 3 of the Pattern section (a mass-update script sweeping a production site into `, nil`; a rule author reasoning about test sites only) describe live exposure today. · *why:* Distinguishing a legitimate nil argument from a silently-degrading one requires per-call-site semantic judgment across every CallExpr in internal/cli/ and internal/check/ — which is the very analysis the gap proposes to mechanize, so there is no cheap read-only measurement of it. · *would settle it:* `prototype the proposed AST scan over internal/cli/ and internal/check/ and triage its hits by hand`
- **G-0225** — The proposed verb `aiwf scope rebind` and the `aiwf scope` verb group do not exist — the packet flags both as unknown-verb. · *why:* Cleared as a false positive rather than rot: the body is proposing these under `## Proposed fix shape` / `## Closing this gap`, not citing them as extant. Confirmed absent from the CLI surface, which is what the gap asks for. · *would settle it:* `/tmp/.../audit/aiwf --help | grep -c 'scope rebind'   # 0, as the gap intends`
- **G-0235** — "the CLAUDE.md 'derived artifacts' section plus the OpDelete-absence doc comment (commit 888d3817) ... drops the raw* shim convention, the 'shares memory' phrase, and the altitude taxonomy on YAGNI grounds" · *why:* The scoping decision lives in a commit message, which I confirmed exists and is titled as described, but whether the message states those three deferrals was not read in full. · *would settle it:* `git show --no-patch --format=%B 888d3817`
- **G-0253** — "investigate whether the toolchain exposes branch-level data" via Go 1.20+ binary coverage. · *why:* An open research question the gap poses, not an assertion about this repo. Settling it means running the toolchain against a probe binary. · *would settle it:* `go build -cover -o /tmp/probe ./cmd/aiwf && GOCOVERDIR=/tmp/cov /tmp/probe --version && go tool covdata debugdump -i=/tmp/cov`
- **G-0254** — the core defect is still live · *why:* Verified, not unverifiable — recorded for completeness: `grep -rn 'Co-Authored-By\|CoAuthored' internal/ cmd/ scripts/` returns no production hit; `grep -n 'Co-Authored-By' CLAUDE.md .claude/aiwf-guidance.md` returns nothing; docs/design/provenance-model.md discusses 'co-author' only conceptually ('the LLM is a *tool*, not a co-author') and never reaches the git trailer; both named sample commits still verify (df73aa00c carries 'Co-Authored-By: Claude Opus 4.8 (1M context)', fec1db93 does not). No D-NNNN decision resolves the question. · *would settle it:* `grep -rln 'Co-Authored' work/decisions/ docs/adr/`
- **G-0274** — Nothing in the tree already provides batch collision resolution under another name. · *why:* I confirmed `aiwf reallocate` has no batch flag and no other verb advertises collision resolution in `aiwf --help`, but I did not exhaustively read every verb's implementation for an undocumented sweep path. · *would settle it:* `grep -rn 'ids-unique' --include='*.go' internal/cli/ to confirm no verb consumes the collision findings as a work list`
- **G-0281** — The three pre-resolved reference findings are rot. · *why:* All three clear on inspection. `aiwf gaps` is PROPOSED, not cited — the body writes "`aiwf gaps import` (naming TBD)", which is a design proposal for a verb that does not exist yet, exactly the legitimate case the brief describes. `commit-tree` is the git plumbing command (`git commit-tree`), not an aiwf finding code — it is real and in production use as `gitops.CommitTree` (internal/gitops/verbcommit.go:39, the M-0186 primitive the gap's Reconciliation section builds on). `INCR` is Redis's INCR, named as an external alternative being rejected, not a claim about aiwf's Go source. · *would settle it:* `grep -rn "CommitTree" --include="*.go" internal/gitops/ | grep -v _test.go   (returns internal/gitops/verbcommit.go:39 `sha, err = CommitTree(ctx, workdir, removes, writes, subject, body, trailers)`)`
- **G-0282** — G1 — the underlying defect (no mechanical inverse-coverage chokepoint) still exists. · *why:* Verified, not unverifiable — recorded here as the positive result. No file under internal/policies/ implements an inverse/what-undoes registry; the only 'inverse' hits are an unrelated Detail string in skill_coverage.go:232 and a test name in skill_coverage_test.go:197. The 'archive cites ADR-0004' and 'set-area --clear shipped' and 'authorize has no revoke' supporting claims all measured true. · *would settle it:* `grep -rn -i 'undoes|inverse|reversing flag' internal/policies/ ; /tmp/.../audit/aiwf authorize --help (flags: --to, --pause, --resume, --force — no revoke) ; /tmp/.../audit/aiwf archive --help ("The reverse path is intentionally not implemented (ADR-0004 §\"Reversal\")")`
- **G-0282** — The `internal/check/provenance.go` (~488-490) line citation still points at the untrailered-entity audit. · *why:* Approximately true — `func RunUntrailedAudit` is declared at line 487 with its doc comment running through 486. Line-number citations drift by construction and I judged this within tolerance rather than a finding. · *would settle it:* `sed -n '480,500p' internal/check/provenance.go`
- **G-0305** — ai-dotfiles' `dotfiles-doctor --write` owns health.dotfiles.json and the schema matches on that side. · *why:* ai-dotfiles is a separate repository not present in this working tree; only aiwf's half of the contract can be measured here. · *would settle it:* `grep -n 'health.dotfiles.json' in the ai-dotfiles checkout and diff its emitted schema against internal/cli/doctor/health.go's healthFile struct`
- **G-0311** — Liminara uses umbrella epics with four peer-epic children; FlowTime is a live aiwf-v3 consumer with ~26 epics and 70+ milestones carrying single epics across 5+ surfaces. · *why:* Both are external repositories not present in this working tree; nothing here can confirm or refute their entity counts or structures. · *would settle it:* `aiwf list --kind epic --archived --format=json in each project's checkout, plus a grep for umbrella-epic parenting in Liminara's work/epics/`
- **G-0311** — A structural survey put ~40-50% of each project's active roadmap as cross-cutting work. · *why:* The survey is not in this repo and the body cites no artifact for it; the figure cannot be re-derived from any tracked file. · *would settle it:* `locating the survey artifact and re-running its classification over each project's current roadmap`
- **G-0333** — `aiwf promote <M> done --force --reason "..."` with an open AC is refused end-to-end (the runtime half of the residual-thread finding above). · *why:* Settling it requires running a mutating verb, which the read-only rule forbids. The static chain (error severity + not skipped during projection + force-independent gate) is measured and consistent, but the refusal itself was not observed. · *would settle it:* `In a disposable repo: aiwf init; aiwf add epic --title E; aiwf add milestone --epic <E> --tdd none --title M; aiwf add ac --milestone <M> --title A; aiwf promote <M> in_progress; aiwf promote <M> done --force --reason "probe"; echo $? — expect exit 1 with a milestone-done-incomplete-acs finding and no commit.`
- **G-0333** — The gap's own reported measurement that `aiwf promote <M>/AC-<N> met --force` is refused with exit 1 and state unchanged under `tdd: required`. · *why:* Same read-only bound; not re-run. · *would settle it:* `Same fixture as above with tdd: required, then: aiwf promote <M>/AC-1 met --force --reason "probe"; echo $?`
- **G-0372** — the dominant remainder is the FSM tier's ~7,000 `git cat-file --batch` blob reads (one per distinct entity-file blob observed across history) · *why:* The serial-round-trip *shape* is confirmed by source (internal/gitops/catfile.go:37-39 'Not safe for concurrent use — git's batch protocol is request / response'; BlobReader.request writes one query line and reads one response), and the pipeline's seriality is confirmed by the absence of any goroutine/errgroup in internal/check. The *count* of ~7,000 reads is an instrumented figure I cannot reproduce read-only without patching the binary, and it would in any case have grown with history. · *would settle it:* `go test -run TestFSMHistoryConsistent ./internal/check/ with a counter added to gitops.BlobReader.request, or `strace -f -e trace=write` around a check run counting writes to the cat-file subprocess's stdin`
- **G-0372** — the ~2.8s native-fs figure · *why:* This devcontainer only exposes the bind-mounted filesystem; there is no native-fs checkout of this repo available to me. · *would settle it:* `git clone /workspaces/aiwf /var/tmp/aiwf-native && time <audit-binary> check --root /var/tmp/aiwf-native`
- **G-0375** — produces 221 failures in `internal/verb` and 62 in `internal/gitops` · *why:* Those exact counts are a dated whole-package reproduction from M-0186; the brief forbids running the full suite, and both packages have grown since. The *exposure* itself I did reproduce at HEAD with two targeted runs under HOME pointed at a .gitconfig carrying `commit.gpgsign = true`: `go test -run 'TestCommitAllowEmpty$' ./internal/gitops/` fails with 'seed commit: git commit: exit status 128 / error: gpg failed to sign the data: ... gpg: signing failed: No secret key', and `go test -run 'TestCommitTree_' ./internal/gitops/` fails including the case named NoSignatureWhenGPGSignNotEnabled/unset — i.e. the commit-tree path AC-4 made gpgsign-aware inherits the leak too, exactly as the body predicts. testsupport.HardenGitTestEnv still forces only gc.auto=0 and gc.autoDetach=false (internal/testsupport/gitenv.go:39-42), so nothing central has changed. · *would settle it:* `HOME=<dir with .gitconfig setting commit.gpgsign=true> go test ./internal/verb/ ./internal/gitops/ 2>&1 | grep -c '^    --- FAIL'`
- **G-0385** — proxy.golang.org still returns divergent answers from /@v/list and /@latest in the minutes after a fresh tag lands. · *why:* Confirming it requires live HTTP calls to proxy.golang.org timed against a freshly-pushed tag — an outward network action and a tag push, neither permitted in a read-only audit. · *would settle it:* `immediately after pushing a new vX.Y.Z tag: curl https://proxy.golang.org/github.com/23min/aiwf/@v/list and curl https://proxy.golang.org/github.com/23min/aiwf/@latest in the same minute, and compare the highest version each reports`
- **G-0398** — The gap's reported measurement that `aiwf add milestone --epic <done-epic>` returns `status: findings` and leaves neither file nor commit behind. · *why:* Re-running it needs a mutating verb against a disposable repo, forbidden by the read-only rule. The static chain is measured and consistent: epic-terminal-non-terminal-children is SeverityError (internal/check/epic_terminal_children.go:71) and add.go:189 gates on `check.HasErrors(projectionFindings(t, proj))`. · *would settle it:* `In a disposable repo: aiwf init; aiwf add epic --title E; aiwf promote <E> active; aiwf promote <E> done; aiwf add milestone --epic <E> --tdd none --title M --format=json; echo $?`
- **G-0400** — Which of the still-unexercised verbs actually warrant a dedicated scenario, versus being read-only or low-risk enough to skip. · *why:* This is the scoping judgment the gap exists to prompt; no command settles it. It is what keeps the gap legitimately open despite every count in it having drifted. · *would settle it:* `n/a — a human scoping decision, not a measurement`
- **G-0414** — That the test passes today purely because of prevention plus the broadened baseline, rather than because a real misplaced landing was detected. · *why:* Distinguishing the two requires observing which branch Run took at runtime; the test asserts only result.Passed, and running it would not report the path. The static evidence is decisive enough (the guard refuses, and the source's own //coverage:ignore says promEnv.Status is never "ok"), so I report it confirmed on that basis rather than on an execution trace. · *would settle it:* `go test -run 'TestPromoteOnWrongBranchDetectionScenario_RealBinary_DetectsTheMisplacedActivation' ./internal/stresstest/ with a temporary t.Logf on promEnv.Status — requires editing the tree, so out of scope for a read-only audit`
- **G-0417** — Packet's `unknown-finding-code` flags on `branch-not-found` and `branch-cell-2`. · *why:* Both are false positives, cleared by command. `branch-not-found` is a real typed kernel code declared at internal/verb/authorize.go:53 as `codes.Code{ID: "branch-not-found", Class: codes.ClassLegality}` — the declaration shape the mechanical index does not match. `branch-cell-2` is a spec *cell id* in internal/workflows/spec/branch/rules.go:52, not a finding code at all. The packet's `missing-path` flag on branch_cell_bijection_test.go is NOT a false positive — see the first finding. · *would settle it:* `git grep -n 'branch-not-found' -- '*.go' ; grep -n 'branch-cell-2' internal/workflows/spec/branch/rules.go`
- **G-0417** — G1 — the defect still exists. · *why:* Settled, not unverifiable; recorded here for the triage read. All four cited sites are present at HEAD e27f19eab: PreflightBranchNotFoundError (internal/verb/authorize.go:109) and CodePreflightBranchNotFound (:53) exist with ZERO construction sites anywhere in the tree; internal/workflows/spec/rules.go:93 and internal/workflows/spec/branch/rules.go:56 both still carry `ExpectedErrorCode: "branch-not-found"`; internal/policies/m0158_ac2_corner_cells_test.go:89 still carries `2: "branch-not-found"`. · *would settle it:* `grep -rn 'PreflightBranchNotFoundError{' --include='*.go' .   (returns nothing — no construction site)`
- **G-0434** — the false-positive `promote-on-wrong-branch` warning fires today on M-0126-trailered commits · *why:* The defect in the helper is confirmed by reading its body (it never calls ByID) and the tree data is confirmed, but the rule's audit window is scoped to the branch's own divergence from trunk, and `aiwf check` on this clean main checkout reports no promote-on-wrong-branch finding — so the end-to-end symptom is not observable read-only at HEAD. · *would settle it:* `Add a table case to TestResolveViaPriorIDs (internal/check/provenance_test.go:1064) with an id that is BOTH live (ByID hits) and listed in another entity's prior_ids, asserting want == the live id; it fails today. The existing table has no such case, which is why the defect is unpinned.`
- **G-0439** — "link-check was red for nine consecutive runs before the paths were corrected by hand." · *why:* Settling the run count needs GitHub Actions history, which is a network call outside the read-only local surface. The claim's substance is corroborated: link-check does run on markdown pushes and PRs, docs/initiatives/ is not in .lychee.toml's exclude_path, and the four links now resolve to archive/ paths — consistent with a hand correction. · *would settle it:* `gh run list --workflow link-check.yml --limit 30 --json conclusion,headSha,createdAt`
- **G-0439** — Candidate fix 2 (exempt CHANGELOG.md from link-check) has not already been done. · *why:* Settled, and recorded here because it is the one candidate that could have quietly landed: .lychee.toml has no CHANGELOG entry in exclude_path, and its own comment names CHANGELOG as a surface link-check exists to cover ('keeps link-check focused on the project's user-facing surfaces (README, CHANGELOG, CLAUDE, ADRs)'). So the CHANGELOG thread of the gap is live, not shipped. · *would settle it:* `cat .lychee.toml`
- **G-0448** — No such orphan exists today — every rule is reachable · *why:* Proving no defined rule function is unreachable from any of the four selection sites needs a whole-program reachability pass over internal/check's exported and unexported finding-producing functions; grep cannot settle it and the repo's own tooling for this (wf-structural-sweep) is a ritual, not a command. · *would settle it:* `go run a callgraph pass (e.g. golang.org/x/tools/cmd/callgraph) rooted at check.Run + internal/cli/check's four entry points, then diff the reached set against every function in internal/check returning []Finding`
- **G-0452** — The gap's Related note that a data-flow principle 'has no rubric principle today, a possible rubric addition if the lens proves its worth' remains an open option. · *why:* It is explicitly conditional and speculative, not a deliverable; it does not block promotion and I did not treat its absence as rot. · *would settle it:* `grep -n 'data flow\|producer' internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-codebase-health/SKILL.md`
- **G-0454** — G1 verdict — the duplication still exists at exactly the three cited sites. · *why:* Not unverifiable; confirmed. `idPrefix` is consulted at three parse sites and nowhere else in the package: allocate.go:89 (known kind), canonicalize.go:45 and :103 (both `for k, prefix := range idPrefix`, discovering the kind). The gap's assumes-kind vs discovers-kind asymmetry is exactly right. · *would settle it:* `grep -rn 'idPrefix' internal/entity/*.go | grep -v _test`
- **G-0455** — G1 verdict — the four walkers still exist and are still separate. · *why:* Not unverifiable; confirmed. body.go (271 lines) declares ParseBodySections (:29), ParseBodySectionsOrdered (:91), SectionLineBounds (:155) and ParseACSections (:211), each opening its own `bytes.HasPrefix(line, []byte("## "))` scan loop — the 'three-to-four' the body hedges at is four. The gap's judgment that ParseACSections differs load-bearingly is visible in its extra `### `/`# ` arms at :227 and :236. · *would settle it:* `grep -n '^func \|HasPrefix(line' internal/entity/body.go`
- **G-0458** — measured: exit 1, no commit on same-phase input · *why:* Re-measuring needs a mutating run of `aiwf promote <id>/AC-N --phase <p>` twice against a scratch repo; the brief forbids mutating verbs. · *would settle it:* `In a throwaway repo: `aiwf init && aiwf add … && aiwf promote M-0001/AC-1 --phase red && aiwf promote M-0001/AC-1 --phase red; echo $?` — expect exit 1 and an unchanged `git rev-list --count HEAD``
- **G-0459** — Each was measured by running the real binary twice with identical arguments: all five exit 0 and land a second commit. · *why:* Re-measuring requires running five mutating verbs twice each against a scratch repo; the brief forbids mutating verbs including init. Static corroboration is strong: no duplicate guard exists in any of the three verb files, and the policy allowlist independently records each as '(measured)'. · *would settle it:* `In a throwaway repo: `aiwf init && … && n=$(git rev-list --count HEAD); aiwf acknowledge mistag G-0001 --reason x; aiwf acknowledge mistag G-0001 --reason x; git rev-list --count HEAD` — expect n+2`
- **G-0460** — 'two `authorize` invocations with identical arguments both exit 0, each lands its own commit, and `aiwf show <id> --format=json` then reports two scope records, both "state": "active", differing only by `auth_sha`.' · *why:* Reproducing it requires running the mutating `aiwf authorize` verb twice against a disposable repo, which this audit is read-only and may not do. The absence of any guard in authorizeOpen and the absence of any scope-count check rule make the outcome the only one the code permits, and ReplayScopes does emit one *scope.Scope per `aiwf-scope: opened` commit with State active, so the claim is strongly corroborated by inspection. · *would settle it:* `cd $(mktemp -d) && git init && aiwf init --actor human/t && aiwf add epic --title T --actor human/t && aiwf authorize E-0001 --to ai/claude --branch epic/E-0001-t --actor human/t && aiwf authorize E-0001 --to ai/claude --branch epic/E-0001-t --actor human/t && aiwf show E-0001 --format=json | jq '.result.scopes'`
- **G-0461** — An operator running `aiwf acknowledge illegal <sha> --for-entity M-NNNN/AC-N` sees a successful ack and the finding keeps firing · *why:* `aiwf acknowledge illegal` is a mutating verb, excluded by the read-only rule. Settled to the code level instead: the emit key, the store key and the lookup key are all read directly, and Canonicalize's composite-preserving behavior is pinned by a passing test. · *would settle it:* `in a throwaway clone with an untrailered commit touching a milestone file: aiwf acknowledge illegal <sha> --for-entity M-NNNN/AC-1 --reason x --actor human/x && aiwf check   # expect provenance-untrailered-entity-commit still firing for M-NNNN`
- **G-0471** — "Measured over one milestone-wrap session in this repo, with v0.30.0 on PATH … doctor afterwards reported seven drifted" — the three observed symptoms. · *why:* A past session's observations. They cannot be re-derived read-only from the tree, and the surrounding structural claims (which I did verify) are what carry the argument. The version anchor makes it a usable dated observation rather than a floating count. · *would settle it:* `install aiwf v0.30.0, run `aiwf update` then `aiwf doctor` against current HEAD, and count the drifted rituals — a mutating experiment, out of scope here`
- **G-0472** — `dupl` reports only the 1444/1519 pair at threshold 100 ... At threshold 60 the three later installers appear as one — `1458-1474` → `1533-1549` → `1649-1665` → back to `1458-1474`. Their common fragment is about seventeen lines. · *why:* Re-running dupl requires a golangci-lint invocation with the config's path exclusions lifted, which writes a lint cache; and the G-0557 fix inserted ~6 lines into two of the three cited fragments, so the quoted line ranges cannot still be exact even if the clustering survives. Not settled either way. · *would settle it:* `golangci-lint run --no-config --enable-only dupl --max-issues-per-linter 0 --max-same-issues 0 ./internal/initrepo/ (and again with dupl.threshold 60)`
- **G-0472** — G1 — the duplication itself still exists (the central gap verdict). · *why:* Confirmed by command, not unverifiable — recorded here only because it is the verdict the triage most needs. All four families are intact: aiwfyaml.replaceHooks(hooks.go:105) still carries '// Mirrors replaceContracts' beside an uncollapsed replaceContracts(aiwfyaml.go:726); initrepo still re-implements the predicate as strings.HasPrefix(line, "actor:") at 980 and still discards config's changed return at 995; isTopLevelActorLine's reject-actorxxx guard is still dead (the HasPrefix(trimmed, "actor:") test above it already forces the colon); recipes.go:244-276 and unbind.go:41-73 remain byte-parallel apart from the verb name; commit-msg's plain detail still carries a "$1" its migrated detail drops; and ActionRemoved still has exactly one use site (initrepo.go:1652, ensurePostCommitHook's uninstall arm) — the two facts the no-collapse verdict rests on. · *would settle it:* `already settled — grep -rn 'Mirrors replaceContracts' internal/aiwfyaml/ ; sed -n '968,1050p' internal/initrepo/initrepo.go ; sed -n '1010,1020p' internal/config/config.go ; grep -rn 'ActionRemoved' internal/ | grep -v _test`
- **G-0473** — verified by injecting a matching pair into `acs.go` and an identical pair into non-excluded `acks.go`: under the repo's own config the `acs.go` findings vanish while the `acks.go` ones fire · *why:* Re-running the injection experiment requires editing files in /workspaces/aiwf, which the audit forbids. The claim is consistent with how golangci-lint path exclusions filter by reporting path, and with the measured fact that the six live entries fire only once their path rule is lifted. · *would settle it:* `inject a >100-token clone pair into internal/check/acs.go and an identical pair into internal/check/acks.go, then run `golangci-lint run --max-issues-per-linter=0 --max-same-issues=0` under the repo's own .golangci.yml`
- **G-0477** — '17 adversarial inputs and a brute-force sweep produced none' · *why:* A dated observation about an experiment run in a prior session; the inputs are not recorded in the body or anywhere in the tree, so the count cannot be re-derived. · *would settle it:* `A fuzz target over the pair — e.g. FuzzActorPredicateAgreement asserting config.isTopLevelActorLine(l) == strings.HasPrefix(l, "actor:") for arbitrary l — would settle the equivalence mechanically instead of by remembered count.`
- **G-0478** — "On 2026-08-10 the `link-check` workflow had been red for six consecutive runs — spanning a scheduled run and two unrelated pushes" · *why:* GitHub Actions run history is not readable from the working tree · *would settle it:* `gh run list --workflow link-check.yml --limit 20`
- **G-0478** — the two core defect claims — no aiwf check rule sees docs-to-work path links, and the mover walk stops at the entity tree · *why:* settled by command, and both hold: internal/check/check.go's rule list (lines 123-173) contains no path-link rule, and planLinkRewriteWrites iterates tr.Entities only. Recorded here because 'no rule anywhere' is an absence claim a grep can only support, not prove. · *would settle it:* `grep -n 'findings = append(findings' internal/check/check.go | sed -n '1,40p'  and  sed -n '27,60p' internal/verb/linkrewrite_ops.go`
- **G-0483** — The core defect and its supporting claims. (Recorded here as verified context, not as a defect.) · *why:* Not unverifiable — listed for completeness of what was measured: FinishVerbOutcome routes uncoded non-ErrInternal errors to ExitUsage with an empty code (apply.go:128-137, doc comment line 46); ErrorEnvelope sets only Code/Message and no findings (envelope.go:52-58); checkStagedConflict returns a bare fmt.Errorf (internal/verb/apply.go:319) so the staged case falls to ExitInternal while its UncommittedConflictError twin returns ExitUsage; the exit-code contract is stated in internal/cli/root.go:348 (printHelp at :240), internal/cli/cliutil/exit.go:14-19, CLAUDE.md's CLI conventions, and docs/design/legal-workflows-first-principles.md R-FP-0162; and FinishVerb delegates to FinishVerbOutcome at apply.go:79, so 'every mutating verb funnels through' it holds transitively. G-0494 is wontfix/archived and its body is the same two-seam content G-0483 absorbed. · *would settle it:* `n/a — settled by the commands above`
- **G-0486** — The end-to-end reproduction (an epic dir with a 100755 script and a 120000 symlink, `aiwf rename`, then the exec bit surviving a fresh clone at 100644). · *why:* Reproducing it requires `aiwf init` plus a mutating verb in a throwaway repo, which this audit's read-only rule forbids. Every mechanism the gap names was instead confirmed by source read: gatherCommitOps' `os.ReadFile` (internal/verb/apply.go:240), the two hardcoded `100644,%s,%s` cacheinfo sites (internal/gitops/committree.go:112, internal/gitops/reconcile.go:41), and DivergentPaths reading the HEAD mode only to test it against `120000` (internal/gitops/divergence.go:286,294). · *would settle it:* `aiwf init in a temp repo; add an epic; place a 100755 file and a symlink in its dir; git commit; aiwf rename <E-id> <new-slug>; git ls-tree -r HEAD -- work/epics/`
- **G-0493** — `aiwf edit-body <id>` refuses with 'frontmatter changed in the working copy' / `aiwf edit-body <id> --body-file <path>` commits — on one identical working-copy state · *why:* Settling this end-to-end requires mutating an entity file and running a mutating verb, which the read-only rule forbids. The two code paths and the reorder-tolerance unit test settle it structurally, and no reachable branch between the guard and the commit plan can reverse the outcome. · *would settle it:* `In a disposable clone: reorder the YAML keys of any entity file, append a body line, then run `aiwf edit-body <id>` (expect the refusal) and `aiwf edit-body <id> --body-file <same-body>` (expect a commit).`
- **G-0497** — zero ETXTBSY across 19,200 write-then-exec cycles vs 12–17% for the bare write · *why:* A dated measurement from G-0491 requiring a fork-pressure experiment; not reproducible read-only. The commit message for the fix records the same figures, which corroborates but does not re-measure. · *would settle it:* `the fork-pressure harness described in G-0491, re-run; git show e09e4eaf0 quotes '0 ETXTBSY across 19,200 cycles at 4x fork pressure, where the unguarded write reports 17%'`
- **G-0498** — the end-to-end consequence — 'on such a repo every mutating verb is unusable', because ADR-0038's commit-side guard refuses a tree the operator has not touched · *why:* Demonstrating the refusal requires running a mutating verb in a core.autocrlf=true fixture, which the brief forbids. Every link in the chain is separately confirmed. (1) The blob divergence is real: in a scratch repo with core.autocrlf=true I committed CRLF content and got blob 422c2b7ab3b3c668038da977e4e93a5fc623169c via `git add`, while `printf 'a\r\nb\r\n' | git hash-object -w --stdin` — the exact call gitops.hashObject makes at committree.go:172 — produced c30dea8a3641ea99b125d04d599d843712292759, with `git status --porcelain` empty. (2) The verb path uses that call and only that call: committree.go:108 `blobSHA, err = hashObject(ctx, workdir, w.Content)`. (3) The comparison side hashes with --no-filters by design: divergence.go:244 `args := append([]string{"hash-object", "--no-filters", "--"}, chunk...)`, and divergence.go:76-79 names this very limit — 'a checkout smudges the working copy away from a blob git itself normalised ... The filter-awareness of the commit path is tracked separately' — which is this gap, though it does not name it. (4) No .gitattributes exists in the repo and no Go source writes one. · *would settle it:* `in a disposable consumer repo: git config core.autocrlf true, check out an entity file, then run any mutating aiwf verb and observe the ADR-0038 divergence refusal`
- **G-0500** — The measured reproduction itself — exit 0 and one id at two paths after `mv` + `aiwf edit-body --body-file`. · *why:* Reproducing it requires running a mutating verb, forbidden by the read-only rule. Statically the route is confirmed: `grep -n 'guardClaim' internal/verb/editbody.go` returns nothing, and claimguard.go's own doc comment states edit-body cannot take the guard and names this gap for the missing id-lookup. · *would settle it:* `In a disposable repo: aiwf init; aiwf add gap --title probe; git mv work/gaps/G-0001-probe.md work/gaps/G-0001-moved.md is NOT the shape — use a plain `mv` so no verb records it; then aiwf edit-body G-0001 --body-file /tmp/new.md; echo $?; git ls-tree -r HEAD work/gaps/`
- **G-0501** — the measured before/after transcript in the body (`aiwf init` exits 0, git reports `T CLAUDE.md`, a later AGENTS.md edit no longer reaches CLAUDE.md) · *why:* reproducing it requires running `aiwf init` / `aiwf update`, which the read-only brief forbids · *would settle it:* `in a disposable repo: `git init r && cd r && printf x > AGENTS.md && ln -s AGENTS.md CLAUDE.md && git add -A && git commit -m x && aiwf init && ls -l CLAUDE.md && git status --porcelain``
- **G-0502** — the measured transcript: `aiwf archive --apply` exits 0 and leaves the gitlink stranded at the old location while its siblings move, with `aiwf check` reporting 0 errors on the split tree · *why:* Reproducing it end-to-end requires running a mutating verb (`aiwf archive --apply`) against a fixture repo, which the audit brief forbids. Everything upstream of the transcript is confirmed by source: internal/gitops/divergence.go:283 `if len(fields) != 3 || fields[1] != "blob" { continue }` (grep output) with its own doc comment at :262 'a path HEAD records as anything but a blob (a directory, a submodule) is likewise absent'; internal/verb/apply.go:421 `gitops.LsTreePaths(ctx, root, "HEAD", src+"/")` feeding the carried set; and apply.go:352-356's comment making exactly the claim the gap says is falsified ('A path the record carries and the working tree lacks ... the commit strands it at the old location while its siblings move'). The directory arm's message matches verbatim: divergence.go:123 `return nil, fmt.Errorf("comparing %s: is a directory, not a file", p)`. No commit has touched internal/gitops/divergence.go since the gap was filed on 2026-08-01, and no test anywhere in internal/ mentions 160000 or gitlink. · *would settle it:* `in a disposable clone: seed a submodule gitlink under work/contracts/<id>/vendor, run `aiwf archive --apply`, then `git ls-tree -r HEAD -- work/contracts` and `aiwf check``
- **G-0506** — The reproduction in "## What's missing" — committed `tdd_phase: red`, hand-edited to `done`, then `aiwf promote M-NNNN/AC-1 --phase green` refuses with an FSM message naming `done`. · *why:* Reproducing it requires creating a disposable repo, running mutating verbs, and hand-editing frontmatter — all forbidden under this audit's read-only rule. · *would settle it:* `In a scratch repo: `aiwf add epic/milestone`, `aiwf milestone tdd <M> required`, `aiwf add ac`, `aiwf promote <M>/AC-1 --phase red`, hand-edit `tdd_phase: done` into the milestone file without committing, then `aiwf promote <M>/AC-1 --phase green` and compare the message against `git show HEAD:<path>`. The static reading strongly predicts it: PromoteACPhase reads `ac.TDDPhase` from the loaded (working-copy) tree with no HEAD comparison anywhere ahead of the FSM consult.`
- **G-0512** — The end-to-end reproduction: a candidate whose destination is occupied survives the decline and its apply fails (arrangements 1 and 3). · *why:* Reproducing it needs a disposable git repo and a mutating `aiwf archive --apply`, both barred by the read-only rule. The mechanism is confirmed by code: the carried set is built by addCarriedUnder → addFilesUnder, which returns early on every directory entry (`if d.IsDir() { return nil }`, internal/verb/apply.go:440-441), and by gitops.DivergentPaths, which reports nothing for a committed-clean file — so neither a bare destination directory nor a clean file inside one enters `carried`, and moveBlockers (internal/verb/archive.go:740-749) can only block on a member of `carried`. The failure then surfaces at internal/verb/apply.go:159-160, `fmt.Errorf("moving %s -> %s: %w", ...)`, exactly the shape the gap quotes. · *would settle it:* `in a scratch repo: mkdir -p work/gaps/archive/G-0001-x.md && <aiwf> archive --apply  (with G-0001 terminal), asserting the verb exits non-zero with `moving ... rename ...``
- **G-0513** — End-to-end: a repo with a terminal entity at HEAD whose working copy does not parse makes `aiwf archive` print the converged message. · *why:* Requires constructing a repo with a deliberately corrupted entity file and running the verb, which the read-only rule bars. All three links in the chain are confirmed by code: tree.Load returns unparseable files as LoadErrors and omits them from tr.Entities (internal/tree/tree.go:345); planArchive drops the LoadError slice (`tr, _, err := tree.Load(ctx, root)`, internal/verb/archive.go:141); maskedTerminalSkips ranges over tr.Entities only (internal/verb/archive.go:677); and the converged string is the plan==nil message at internal/verb/archive.go:59. · *would settle it:* `in a scratch repo: corrupt the frontmatter of an entity whose HEAD copy is terminal, then `<aiwf> archive --dry-run`, asserting the output names the entity instead of 'tree is converged'`
- **G-0523** — Observed: a session working in this repo carried a parent directory's CLAUDE.md as its project instructions, so the guidance import never entered context while aiwf doctor reported clean. · *why:* A one-off observation about a past harness session; nothing in the repo records which CLAUDE.md a given session resolved, and the failure mode is by the gap's own argument unobservable from inside aiwf. · *would settle it:* `No read-only command settles it. It would need a live session started with cwd inside this repo but a CLAUDE.md present in an ancestor directory, with the resolved project-instruction set inspected from the harness side.`
- **G-0524** — "A session working in this repo carried the parent-level file as its project instructions; ... three review passes over a patch ran without the repo's own conventions before the omission was caught by hand." · *why:* A past session observation. Nothing in the repo or the container records which CLAUDE.md a past harness session resolved. · *would settle it:* `no command available read-only; it would need a live session with a /workspaces/CLAUDE.md planted and the harness's resolved instruction set inspected`
- **G-0529** — Auditing the 19 gaps closed between v0.30.0 and v0.31.0 showed 13 carried their own entry and the 6 that did not were all E-0075's. · *why:* A dated one-off measurement of a historical release window; re-deriving it would require reconstructing the gap set closed between two tags and matching each against CHANGELOG prose, which is a judgment match rather than a mechanical one. · *would settle it:* `git log v0.30.0..v0.31.0 --grep='aiwf-verb: promote' --format=%s | grep -o 'G-[0-9]\{4\}' | sort -u  (then match each id against CHANGELOG.md's [0.31.0] section by hand)`
- **G-0530** — "Section count is what makes a spec read as sprawl, and it is the axis nobody has pruned." · *why:* A reader-experience judgment with no mechanical referent. The supporting half (per-unit prose length shows no trend; growth is in entity count) does check out against docs/design/growth.md:178-191. · *would settle it:* `no command; it is a design judgment`
- **G-0530** — "The same treatment would sharpen the epic and gap surfaces, neither of which has been examined this way." · *why:* A claim about what audit work has and has not been done, not about the code. · *would settle it:* `no command`
- **G-0533** — Diff-scoped at the same threshold, over a 200-commit window touching 207 Go files, the detector fires four blocks — two pairs. ... Over the most recent sixty commits: nothing. · *why:* Requires a `--new-from-rev`-style diff-scoped golangci-lint invocation over a window anchored on 2026-08-05; the window has since moved by ~13 days of commits, so re-running measures a different population rather than checking this one. Both figures sit inside the explicitly-dated `## Measured 2026-08-05` block, which the repo's own 'a measurement that is the point is a dated observation' rule sanctions. · *would settle it:* `GOLANGCI_LINT_CACHE=<tmp> golangci-lint run --config <dupl-only, no-exclusions, threshold 100> --new-from-rev=$(git rev-list -n1 --before=2026-08-05 HEAD)~200 ./...`
- **G-0535** — PolicyApplyCallersAcquireLock's pre-fix `run` name filter was case-sensitive and dropped every verb's `Run` · *why:* A statement about the policy's state before its resolution; the current source no longer contains the broken selector, and confirming it requires archaeology on the fix commit rather than a measurement of current behavior. · *would settle it:* `git log -p --follow internal/policies/apply_callers_lock.go | grep -n 'strings.HasPrefix(.*\"run\"'`
- **G-0536** — A CI `aiwf check` step would report errors on day one against the tree as it stands. · *why:* Settling it requires standing up a ref-less / shallow checkout of this repo and running the binary there, which means creating a clone — outside the read-only bound of this audit. The current-tree measurement (0 errors, no cross-branch findings) is evidence against it but is taken with this machine's full ref set. · *would settle it:* `git clone --depth 1 file:///workspaces/aiwf /tmp/ciprobe && cd /tmp/ciprobe && aiwf check   (then repeat with `git fetch origin '+refs/heads/*:refs/remotes/origin/*'` to show the published refs resolve it)`
- **G-0540** — Threads 1-3 are still live. · *why:* Verified, not unverifiable — recorded as the positive result. Thread 1: discoverability_test.go:22 still walks the AST for a func named `printHelp` and compares its file to bannerSourceRel, asserting nothing about banner text; the sibling TestDiscoverabilityHaystack_CoversBannerChannel writes its own synthetic tree so it cannot catch a real-tree const hoist either. Thread 2: apply_callers_lock_test.go:34-48 sets sawRun/sawSubverb and errors only if one of the two shapes is entirely absent, and readOnlyVerbs does exist at internal/policies/read_only.go:35. Thread 3: the `case 0:` and `default:` arms at discoverability_test.go:53-58 are present in a _test.go and driven by no test. · *would settle it:* `sed -n '22,60p' internal/policies/discoverability_test.go ; sed -n '18,50p' internal/policies/apply_callers_lock_test.go ; grep -n readOnlyVerbs internal/policies/read_only.go`
- **G-0543** — 'a fixture that fails to typecheck' also produces the dormant-rule message unchanged. · *why:* Not reproduced here (the sibling failure mode, an unresolvable --config, was reproduced and behaves exactly as the gap says). The mechanism is nonetheless determinate from source: judgeGolangciOutput derives its verdict solely from finding lines matching `^\S+\.go:\d+:\d+: `, so any run producing none and not carrying the concurrency literal takes the 'did not fire … dormant, disabled, or dropped from the enable list' arm. · *would settle it:* `cd <tmp fixture with a type error> && golangci-lint run --config <repo>/.golangci.yml --allow-parallel-runners ./... 2>&1 | grep -cE '^\S+\.go:[0-9]+:[0-9]+: '`
- **G-0544** — These four are the ONLY mutating verbs an agent can drive while leaving a commit no human is accountable for. · *why:* Verified for every verb that registers --principal (15 call sites) plus archive (which does its own inline principal coherence check) and authorize/acknowledge (human-only by their own actor validation). Not exhaustively swept over every command that could reach a commit (e.g. worktree, init, update), which would need a per-verb commit-path trace. · *would settle it:* `for v in $(aiwf --help | ...); do aiwf $v --help; done   plus an AST sweep for cliutil.FinishVerb call sites that do not also call cliutil.DecorateAndFinish`
- **G-0546** — a guard placed inside a verb would see no principal and refuse every legitimately authorized agent · *why:* A counterfactual about code that does not exist. The structural precondition is confirmed (the principal trailer is appended after the verb builds the plan), but the failure it predicts cannot be measured without writing the guard. · *would settle it:* `adding a trailer-inspecting guard inside any verb's Plan construction and running the authorize-scope integration tests, e.g. go test -run TestAuthorize ./internal/cli/integration/`
- **G-0548** — 'That patch [G-0542] added none and removed none; the condition predates it.' · *why:* Settling it means diffing the shipped-surface path-citation set across the G-0542 patch range, which needs the patch's exact base and tip shas — recoverable, but the claim is provenance rather than a live assertion about the current tree, and the current-tree count is what the gap acts on. · *would settle it:* `git log --oneline --all --grep 'G-0542' then `git diff <base>..<tip> -- internal/skills/` piped through the path regex above, comparing counts either side`
- **G-0552** — The Go build cache reached 84 GB on a 288 GB overlay and filled the filesystem to 100%. · *why:* A dated past observation from M-0291's wrap on a different host filesystem; the current mount is 932G at 90% used, so it neither confirms nor refutes the historical measurement. · *would settle it:* `du -sh $(go env GOCACHE) && df -h $(go env GOCACHE)  — run on the host where the incident occurred`
- **G-0560** — Re-measurement of the audit doc's inventory: which entries still hold at HEAD e27f19eab. · *why:* Not unverifiable; measured. STILL TRUE (17): §A workflows.md:88 non-enforced roll-up (the verb refuses — `internal/check/epic_terminal_children.go:40` + `CodeEpicPromoteNonTerminalChildren` at `internal/verb/cancel_guards.go:121`); §A tree-discipline.md 'Why no marker-managed CLAUDE.md fragment' section intact; §A skill-author-guide.md:154 'aiwf check does not enforce them at runtime' vs `internal/check/provenance.go:47 CodeProvenanceUntrailedEntityCommit`; §A render-roadmap-commits in all three places vs `--write  write ROADMAP.md to disk, no commit`; §A architecture.md:90 `git mv` vs `internal/verb/apply.go:20` 'runs every OpMove via a pure filesystem rename'; §A design-decisions.md:100 rename-updates-title; §B `aiwf upgrade --yes` (flags are --check/--root/--version only); §B provenance-model.md:239-240 phantom codes `provenance-no-active-scope-to-pause`/`-resume` (zero hits in internal/); §C narrow-width tables at overview.md:15-20, design-decisions.md:47-52, tree-discipline.md:22-27; §C design-decisions.md one-commit list still omits the same eleven verbs; §C skill-author-guide.md:150 names `priority:` as not-in-schema while gap's OptionalFields includes it; §C architecture.md:5,182 'the seven kernel commitments' vs 13 subsections / CLAUDE.md's 10; §C design-lessons.md:69 'the eight embedded skills' vs 19 dirs under internal/skills/embedded/; §C design-decisions.md omits ADR-0008/ADR-0036/ADR-0004 commitments (grep -c for ADR-0036|NoOp|CanonicalPad|4-digit = 0); §D design-lessons.md:88 'immutability of done' surviving only in docs/archive/architecture.md:446; §D design-lessons.md:90,92 citing `hotfix`/`complete`/`set-status`; §D provenance-model.md:74 pointing at a nonexistent 'Scope termination' section; §D id-allocation.md self-contradiction (independently confirmed, see that subject); §E design-decisions.md:3 and :285 two migrations out of date; §E legal-workflows-audit.md:3 'Status: in_progress'; §F ADR-0003 accepted-and-unimplemented (status: accepted, no KindFinding). NOW FALSE (1): §F's IDFormat-hardcoded-in-code entry. · *would settle it:* `the commands quoted in each finding above`
- **G-0561** — $ aiwf init against a repo carrying a mode-0000 hook prints `aiwf init: reading pre-commit hook: open …: permission denied` and exits 3. · *why:* Reproducing it requires running the mutating verb `aiwf init` against a scratch repo, which the read-only rule forbids. The code path is confirmed end-to-end by inspection — `internal/initrepo/initrepo.go:1458` returns `fmt.Errorf("reading pre-commit hook: %w", readErr)`, RefreshArtifacts/Init propagate it, and initcmd.go:118-120 maps it to cliutil.ExitInternal (=3 per internal/cli/cliutil/exit.go:19) — but the exit code itself was not observed. · *would settle it:* `mkdir /tmp/t && git -C /tmp/t init && printf '#!/bin/sh\n' > /tmp/t/.git/hooks/pre-commit && chmod 000 /tmp/t/.git/hooks/pre-commit && (cd /tmp/t && aiwf init; echo $?)`
- **G-0561** — "closing G-0557 took the number of installers reaching this message from one to four" · *why:* The 'four' half is confirmed (four `reading <hook> hook:` error arms exist). The 'one' half depends on how G-0557's own table is counted — pre-push errored outright, and post-commit errored in its regeneration-off arm, which reads as one-and-a-fraction rather than one. Not worth a finding; recorded for completeness. · *would settle it:* `grep -rn 'reading pre-push hook\|reading pre-commit hook\|reading commit-msg hook\|reading post-commit hook' internal/ | grep -v _test  (returns 4 arms at initrepo.go:1351,1458,1539,1627)`
- **G-0562** — "It fires. Observed 2026-08-06 on one `make check-fast` run" with the quoted ETXTBSY failure. · *why:* A dated observation of a load-dependent race. Reproducing it would require repeatedly running the package under concurrency, which is outside a read-only audit's remit and would not falsify the claim if it did not reproduce — the gap says as much itself. Every structural precondition for the failure is confirmed present. · *would settle it:* `go test -count=50 -race -run TestWorktreeRitualsCheckHook ./internal/policies/ under concurrent load`
- **G-0562** — All structural claims. · *why:* Verified, not unverifiable — recorded as the positive result. worktree_rituals_check_hook_test.go:30 is `os.WriteFile(path, skills.WorktreeRitualsCheckScript, 0o755)` with no WriteExecutable anywhere in the file; three tests (lines 79, 99, 130) call writeWorktreeHookScript; adoptedPackages (test_executable_write_test.go:401) holds exactly internal/stresstest and internal/contractverify; testsupport/execfile.go:60-61 takes syscall.ForkLock.RLock as the gap describes. · *would settle it:* `grep -rn 'os.WriteFile|WriteExecutable' internal/policies/worktree_rituals_check_hook_test.go ; sed -n '398,405p' internal/policies/test_executable_write_test.go ; grep -n ForkLock internal/testsupport/execfile.go`
- **G-0563** — The exact transcript (exit 1, no commit lands, entity keeps its prior status) reproduces on today's binary. · *why:* Reproducing it requires running a mutating verb (`aiwf promote`) against a scratch repo, which the read-only rule forbids. The structural chain (bare tree.Load -> check.Run emits refs-resolve/unresolved at SeverityError -> blocksWrite/HasErrors suppresses the plan -> FinishVerb maps error findings to ExitFindings=1) is fully verified by reading the wired code paths above. · *would settle it:* `In a disposable temp repo: init, commit ADR-0001 and ADR-0002, `git branch sibling`, delete ADR-0002 on the default branch, then `aiwf promote ADR-0001 superseded --superseded-by ADR-0002; echo $?``
- **G-0571** — an epic created with the full scaffold, then given a body via `aiwf edit-body --body-file` that omits `## Out of scope`, loses the section and `aiwf check` reports zero errors · *why:* reproducing requires running `aiwf add` and `aiwf edit-body`, which the read-only brief forbids; the code path is consistent with the claim (editbody.go consults no section helper, and EmptyRequiredSections skips absent headings) · *would settle it:* `in a disposable repo: `aiwf add epic --title X && aiwf edit-body E-0001 --body-file <body-without-out-of-scope> && aiwf check``
- **G-0572** — the end-to-end reproduction and the nine measured exits in the outcome table · *why:* every row requires running a mutating verb (`add`, `promote`, `milestone tdd`, `cancel`) in a disposable repo, which the read-only brief forbids. Each row is consistent with the source read above: the FSM tables explain rows 1, 2, 5 and 6; the message-keyed projection identity explains rows 3 and 4; the met-only audit filter explains rows 6 and 9; the severity switch on e.TDD explains row 8. · *would settle it:* `replay the fenced script in the gap body in a scratch repo, then `aiwf check` and each of the nine attempts in turn`
- **G-0573** — The measured repro block — `aiwf add epic` printing `ok — no findings` and committing a tree that `aiwf check` then rejects with three entity-body-empty errors under `tdd.strict: true`. · *why:* Reproducing it requires running the mutating verb `aiwf add`, which the standing read-only rule forbids. The structural reasoning above (entityBodyEmpty reads from disk; the new entity's file does not exist at projection time) says it should still reproduce, and ADR-0042 §Context reports the same behavior as measured while closing this gap — but I did not execute it. · *would settle it:* `In a disposable temp repo: aiwf init; printf 'tdd:\n  strict: true\n' >> aiwf.yaml; aiwf add epic --title 'Strict probe epic'; echo "add exit=$?"; aiwf check; echo "check exit=$?"`
- **G-0574** — The two measured refusals — `aiwf promote <m>/AC-1 --phase red` refused from a tree carrying the acs-tdd-audit finding at `(absent)`, and cancelling either open AC refused on a `done` milestone with two open criteria. · *why:* Both require running the mutating verb `aiwf promote` / `aiwf cancel` against a constructed tree, which the standing read-only rule forbids. Every ingredient the refusals depend on is confirmed by source read (message interpolation, message-in-key, neither code excluded by skipDuringProjection), but the end-to-end refusal was not executed. · *would settle it:* `In a disposable temp repo: build a milestone with tdd: required and one AC at status met with no tdd_phase, then `aiwf promote M-NNNN/AC-1 --phase red`; expect a refusal naming acs-tdd-audit`
- **G-0575** — The two-branch merge reproduction itself: `aiwf add ac` on one branch and `aiwf promote <m> done` on another merge cleanly into a done milestone with open ACs. · *why:* Reproducing it needs three branches and four mutating verbs in a scratch repo, barred by the read-only rule. Both halves are confirmed legal in isolation by code — `add ac` routes through projectionFindings (internal/verb/add.go:189) so it is refused on a done milestone single-branch, and `promote <m> done` is legal while its only AC is met — and G-0121's body independently records both traps as hand-constructed and merge-reachable. · *would settle it:* `in a scratch repo, the exact sequence in the gap's fenced block, then `<aiwf> check --format=json`, asserting a milestone-done-incomplete-acs finding at error severity`
- **G-0577** — The six-row mutation table's failure counts ("69 of 70 packages pass", "~15 failures across four packages", "the policy, plus one hardcoded pair-table case", etc.). · *why:* Each row requires editing an FSM edge in a clean clone and running the full suite — a mutation of the repo and a full-suite run, both outside my read-only remit. The package count has since drifted (75 packages today vs the 70 the table names), so the denominator at minimum no longer holds; whether the numerators do is untested. · *would settle it:* `for each mutation: `git clone . /tmp/m && cd /tmp/m && <edit one transitions[] edge> && go test ./... 2>&1 | grep -c '^FAIL'` (I ran only `go list ./... | wc -l` → 75)`
- **G-0577** — "Implemented against the kernel's own predicates and run over all six mutations above, it reports zero violations" — the vacuity demonstration for the restated rule. · *why:* Requires writing and running the restated policy against six mutated clones. · *would settle it:* `the same six-clone loop with a `for kind, s := range … if IsTerminal(kind,s) && len(AllowedTransitions(kind,s))>0` policy added`
- **G-0578** — the ETXTBSY failure is timing-dependent and was observed once during M-0306's wrap · *why:* A dated one-off observation of a flake; reproducing it requires a loaded concurrent run and is not deterministic read-only. · *would settle it:* `STRESS-style repeated runs: `for i in $(seq 50); do go test -race -parallel 8 -count=1 -run TestWorktreeRitualsCheckHook ./internal/policies/ || break; done` (mutates nothing but is a long compute run outside this audit's mandate)`
- **G-0583** — "The remaining bullets check the baseline builds and the tests pass — real, but orthogonal to whether the spec is right." · *why:* Settled but imprecise rather than false, so recorded here instead of as a finding. Step 1 has eight bullets, of which two are the build/test-green checks the gap names; the rest read the parent epic and prior milestone specs, confirm the ACs pre-exist and are filled, confirm the `tdd:` policy is intentional, and confirm the parent epic branch is checked out. The gap's substantive claim — that the one step positioned to catch a wrong specification supplies no method for doing so — holds exactly as written. · *would settle it:* `sed -n '20,40p' internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-milestone/SKILL.md  →  first bullet: "Read the milestone spec. Confirm every AC is concrete and testable. If any AC is vague, stop and ask the user to refine before starting work." and nothing following it that says how`
- **G-0583** — The defect is still live — nothing has shipped a method into the preflight. · *why:* Settled: verified live. Recording the evidence because it is the gap's central verdict. · *would settle it:* `git log --oneline -5 -- internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-milestone/SKILL.md (newest touch is 18f6b54b6, about the deferral mandate and reference-phrasing, not a preflight method)  ·  ls internal/skills/embedded-rituals/plugins/*/skills/ (no spec-measurement skill exists; M-0308's ACs are `met` against a file that is gone)`
- **G-0587** — Filed separately from the review pass it blocks because the collision exists today, independent of that work. · *why:* The 'review pass it blocks' is unnamed; the likeliest referents are the preflight-review initiative (docs/initiatives/milestone-preflight-as-independent-review.md) and G-0583, but nothing in the body identifies which, so whether the separation judgment is right cannot be checked. · *would settle it:* `aiwf history G-0587 to find the authoring commit, then git show that commit's sibling changes / the conversation-adjacent entity it was filed alongside`
- **G-0588** — "its documented worked example still validates against the current binary, including the narrow legacy ids ADR-0008 says parsers tolerate" · *why:* Settling it requires writing a manifest file from docs/archive/migration/import-format.md and running the binary against it; the audit is read-only and may write only this ledger. Corroborated but not settled: internal/manifest/manifest.go's struct tags match the doc's top-level shape (version/actor/commit.mode/commit.message/entities[].kind/id/frontmatter/body), and ADR-0008 is `accepted` with exactly the tolerance the gap cites. · *would settle it:* `extract the YAML block from docs/archive/migration/import-format.md into /tmp/m.yaml, then: <binary> import /tmp/m.yaml --dry-run --root <disposable repo>`
- **G-0588** — "Nothing consumes it. Every project that was going to be imported has been, and no legacy project remains." · *why:* A claim about the world outside this repo; no in-repo command can settle it. In-repo it is corroborated by the commit that moved the spec, and the gap itself later acknowledges the test-suite consumers, so it is not self-contradictory. · *would settle it:* `no repo command; requires the maintainer's knowledge of downstream consumers`
- **docs/design/design-decisions.md** — `aiwf update` really does strip an existing `aiwf_version:` line from a consumer's aiwf.yaml (as opposed to merely being coded to). · *why:* Settling it requires running the mutating verb `aiwf update` against a repo, which the read-only rule forbids; I verified the code path and the deprecation note instead. · *would settle it:* `In a disposable clone: printf 'aiwf_version: 0.1.0\n' >> aiwf.yaml && aiwf update && grep -c aiwf_version aiwf.yaml   # expect 0`
- **docs/design/design-decisions.md** — `aiwf render --format=html` emits exactly the page set I inferred from htmlrender.go (per-kind indexes, status.html, per-entity pages), and the stylesheet reaches the output at its full 21 KB. · *why:* Running the renderer writes into `site/`, which is a filesystem write into the repo the read-only rule forbids even though the directory is gitignored. · *would settle it:* `aiwf render --format=html --out "$(mktemp -d)" && ls "$out" && wc -c "$out/style.css"`
- **docs/design/design-decisions.md** — Setting an unknown top-level `doctor:` block in aiwf.yaml is silently ignored (rather than rejected), which is what makes the phantom config row costly. · *why:* config.Load has no top-level KnownFields/strict-key guard (the strict guards I found are scoped to the `areas:` and `docs:` blocks), but confirming the silent-ignore behavior end-to-end needs a real load against a planted file. · *would settle it:* `In a disposable clone: printf 'doctor:\n  recommended_plugins: [x@y]\n' >> aiwf.yaml && aiwf doctor   # expect no error and no plugin row`
- **docs/design/growth.md** — cold `go test ./...` is 4m47s (449s of package time, `internal/cli/integration` alone 244s) · *why:* Wall-clock measurement of the full suite; the audit brief forbids running the full suite, and the figure is machine- and cache-dependent so a re-run would not falsify it cleanly either. · *would settle it:* `go clean -testcache && time go test ./... 2>&1 | sort -k3 -rn | head`
- **docs/design/growth.md** — duplicate blocks of >=100 tokens in the test corpus number 211 — 207 of them in test files across 91 files · *why:* Requires running golangci-lint with dupl's three test-corpus exclusions removed, which means editing .golangci.yml — forbidden read-only. A scratchpad config copy would resolve exclusion paths relative to a different base and is not a faithful reproduction. · *would settle it:* `on a scratch clone: remove the three `- linters: [dupl]` exclusions (path: _test\.go, internal/cli/cliutil/testutil/, internal/cellcoverage/) from .golangci.yml, then `golangci-lint run --enable-only dupl ./... | wc -l` and count the `_test.go` share`
- **docs/design/growth.md** — the marginal-detection lever costs ~9 min per file, with 100% recall and 11% precision; the mutation-based variant reproduces the by-hand verdict 5 of 5 while flagging 39 extra tests; strict subsumption yields 0 of 5; the same run surfaced 33 undetected mutants in one file including 13 loop-filter branches · *why:* These are one-off experiment results with no command, no target file named, and no artifact committed; the mutate-hunt run is workflow_dispatch-only and hours long. · *would settle it:* `re-run mutate-hunt over the named file with --workers 1 --timeout-coefficient 15 and re-do the by-hand revert audit; the doc would first have to name which file`
- **docs/design/growth.md** — the four-shape classification of the 73 policy chokepoints (39 mandate / 24 ban / 8 uniqueness / 2 exactness) and of the 57 kernel finding codes (16 / 27 / 6 / 8), and 'Five of the sixteen kernel mandates are gated by three aiwf.yaml knobs' · *why:* Both are explicitly by-hand judgments, and the doc says so and date-stamps them (2026-08-02 and 2026-08-03). Re-deriving them is a fresh classification, not a verification. Their arithmetic is internally consistent (39+24+8+2=73 with correct percentages; 16+27+6+8=57 likewise), and the population figure 73 reproduces at the baseline revision. · *would settle it:* `an independent by-hand re-classification of all 82 current chokepoints against the same four-shape definitions`
- **docs/design/legal-workflows-audit.md** — R-AUDIT-0108 / R-RULE-094: whether `aiwf add ac` against a `done` or `cancelled` milestone is refused in practice via the validate-then-write projection, even though the verb itself carries no status guard. · *why:* Settling it requires actually running the mutating verb `aiwf add ac` against a terminal milestone in a scratch repo, which the read-only standing rule forbids. Reading `AddACBatch` settles only that the verb layer has no guard; whether `milestoneDoneIncompleteACs` / `milestoneCancelledIncompleteACs` fires on the projection and blocks the write is a runtime question. Note the `cancelled` arm is the less certain of the two. · *would settle it:* `In a disposable repo: aiwf add epic --title T; aiwf add milestone --epic E-0001 --tdd none --title M; aiwf promote M-0001 in_progress; aiwf promote M-0001 done; aiwf add ac M-0001 --title "probe" ; echo "exit=$?"`
- **docs/design/legal-workflows-audit.md** — R-AUDIT-0163 / R-RULE-042: 'the born-complete kinds refuse an empty body at creation' and the epic/milestone-warning vs born-complete-error severity split on `entity-body-empty`. · *why:* The create-time gate is visible in `aiwf add --help` (`--force` is documented as bypassing 'the born-complete-kind empty-body gate (gap/decision/adr/contract only — G-0326)'), which corroborates the claim, but confirming the *severity* split needs a tree that actually carries a present-and-empty required section on each kind and a check run over it. I did not construct one, and could not without writing files. · *would settle it:* `go test -run 'TestEntityBodyEmpty' ./internal/check/  — plus reading the severity assignment in internal/check/entity_body.go:132 against a fixture per kind`
- **docs/design/legal-workflows-audit.md** — R-RULE-149's four-subcode partition claim for `fsm-history-consistent` (illegal-transition / forced-untrailered / manual-edit / history-walk-error, disjoint per D-0008, merges skipped per D-0010). · *why:* The code exists (`internal/check/fsm_history_consistent.go`, `fsm_history_walker.go`), D-0008 and D-0010 are both `accepted`, and `gitops.BulkRevwalk` / `gitops.BlobReader` both resolve — so every named part is real. But disjointness is a property of the rule's behaviour across a constructed history, not something reading settles, and this is the single longest and most load-bearing row in the document. · *would settle it:* `go test -run 'TestFSMHistoryConsistent' ./internal/check/ -v  and inspect which subcode each fixture history yields`
- **docs/design/legal-workflows-audit.md** — Whether the ~50 remaining §4/§6/§9 per-verb and per-flag statements I did not individually exercise still hold. · *why:* I verified the rows most likely to have rotted (init, render, move, add ac, add milestone --tdd, add ac --tests, import --on-collision/--dry-run, promote --phase/--force, --principal, --root, list --archived, read-only set) and found four contradicted. That hit rate on a sample says the remainder should be treated as unaudited rather than sound; I did not have budget to drive every flag. · *would settle it:* `A row-by-row pass running `aiwf <verb> --help` for each of the 33 §4 rows and 15 §9 rows and diffing the stated flag set, defaults, and mutex pairings against the help output`
- **docs/design/legal-workflows-first-principles.md** — R-FP-0139 — 'the verb computes the projected new tree in memory, runs `aiwf check` against the projection, and only writes when the projection is clean' — and R-FP-0140/0141's rollback and pre-existing-findings-diff behavior. · *why:* Settling these by behavior needs a mutating verb run against a disposable repo, which the read-only rule forbids. Source reading alone would not settle them per the brief's rule 2. · *would settle it:* `In a throwaway clone: seed a tree with a pre-existing dangling ref, then `aiwf add gap --title x` and assert exit 0 + exactly one new commit + the pre-existing finding still present; separately introduce a projection-breaking mutation and assert `git status --porcelain` is empty after the refusal.`
- **docs/design/legal-workflows-first-principles.md** — R-FP-0144's read-only verb set (`check`, `history`, `status`, `render` without `--write`, `doctor`, `whoami`) is complete and correct as an enumeration. · *why:* The current CLI has more read verbs than the row lists (`list`, `show`, `schema`, `template`, `version`, `completion`, `contract recipes`), and the doc's own Q12 flags the set as an unresolved question — so whether the omission is an error or a deliberately partial illustration is a judgment about the doc's intent, not a measurement. · *would settle it:* `grep every top-level Cobra verb's RunE for a repolock.Acquire call and diff the non-locking set against the row's list.`
- **docs/design/legal-workflows-first-principles.md** — Whether the doc should be re-tiered rather than corrected row-by-row. · *why:* This is the disposition question the audit exists to surface; it turns on whether the project wants a preserved Pass-B artifact (per ADR-0011's three-pass methodology, where Pass B's independence from the implementation is explicitly load-bearing) or a maintained Normative catalog. Correcting rows to match the kernel would destroy the independence the doc's own banner calls load-bearing. · *would settle it:* `A human decision; mechanically, adding a drift policy that unions spec.Rules() FP-source citations against the doc's R-FP rows would make either choice enforceable.`
- **docs/design/oracles.md** — The dispatched reviewer subagent 'advises; the human gate decides', and the human gate 'blocks' at every mutation. · *why:* These are process oracles with no mechanical chokepoint; nothing in the repo can be measured to confirm or refute how they behave in practice. The doc itself elsewhere states that a guarantee depending on LLM or human recall is not a guarantee. · *would settle it:* `No read-only command settles it; the nearest evidence would be an audit of merge commits for reviewer-dispatch trailers, which the trailer vocabulary does not currently record.`
- **docs/design/oracles.md** — 'Independence is architectural, not conventional … the checker does not share a code path with the thing it judges.' · *why:* check.Run consumes tree.Load's output, so the checker does share the loader with every verb — the claim is true about the validation/loading axis split (which is a stated kernel commitment) but reads stronger than that. Whether the stronger reading is intended is an authoring-intent question, not a measurable one. · *would settle it:* `grep -rn 'func Run' internal/check/check.go together with the loader call graph shows the shared tree.Load; deciding whether that falsifies the sentence is a judgment about its intended scope.`
- **docs/design/performance.md** — `aiwf add` still spawns ~13–14 [git subprocesses], scaling O(local branches) because it runs `git ls-tree` per branch · *why:* aiwf add is a mutating verb, excluded by the read-only rule. The per-ref ls-tree mechanism is confirmed in source (internal/trunk/trunk.go:170 'Cost is one `git ls-tree` per local branch — O(local branches)'; trunk.go:218 refHits), but the subprocess count cannot be counted without running the verb — and the count would in any case now be far lower, since local branches fell from 48 to 6. · *would settle it:* `strace -f -c -e trace=clone,execve aiwf add gap 'probe' in a throwaway clone`
- **docs/design/performance.md** — The ~65ms / ~14ms path-scoped vs ~1.3s grep figures in the class-1 table, and the '~11s once' commit-graph write time · *why:* These were measured against a repo state (5,510 commits, no bloom filters) that no longer exists; the filters are now present so the 'no bloom filters' arm cannot be re-measured without deleting the commit-graph chain, which mutates the repo. · *would settle it:* `In a throwaway clone: time git log -- <path> before and after `git commit-graph write --reachable --changed-paths``
- **docs/design/performance.md** — 'Felt latency is dominated by subprocess count' as a current property of aiwf check · *why:* The measured user/system split is now ~1:1 (4.95s/4.89s) rather than the doc's ~1:2.6, which weakens but does not by itself refute the attribution. Settling it needs the syscall count the doc's own §'How to measure' prescribes. · *would settle it:* `strace -f -c -e trace=clone,execve,waitid /tmp/.../audit/aiwf check`
- **docs/design/provenance-model.md** — Example 5's auto-end: a terminal promote by an authorized agent writes aiwf-scope-ends: into the same commit and the scope becomes `ended`. · *why:* Settling it end-to-end requires running a mutating verb sequence (authorize, then a terminal promote under an ai/ actor) in a live repo, which the read-only rule forbids. The trailer key, the scope FSM's ended state and the entityview reader for aiwf-scope-ends all exist, so the pieces are present. · *would settle it:* `In a disposable clone: aiwf authorize E-XXXX --to ai/claude --reason t && aiwf promote E-XXXX done --actor ai/claude --principal human/x && git log -1 --format='%(trailers)'`
- **docs/skill-author-guide.md** — Whether `aiwf add decision --title "..." --force --reason "..."` would produce the entity the worked example intends, and what `aiwf check` then reports about it. · *why:* `aiwf add` is a mutating verb; the brief forbids running it. · *would settle it:* `In a disposable repo: `aiwf init && aiwf add decision --title "Status snapshot" --force --reason "snapshot skill" && aiwf check --verbose``
- **docs/skill-author-guide.md** — Whether a consumer-authored `.claude/skills/aiwfx-<name>/SKILL.md` is actually overwritten (rather than merely gitignored) by a future `aiwf update`. · *why:* Requires a future aiwf release that ships a ritual with the colliding name; the gitignore half is confirmed, the overwrite half is a read of MaterializeWithTiers' documented behavior, not a measurement. · *would settle it:* `In a disposable repo: create `.claude/skills/aiwfx-wrap-epic/SKILL.md` with sentinel content, run `aiwf update`, diff the file.`
- **docs/workflows.md** — workflows.md:238 — "say so once and the assistant will follow that for the session" (the AI's default behaviors at :231-236: never inventing ids, surfacing findings before pushing, treating errors as blockers). · *why:* These describe LLM behavior under the materialized skills, not kernel behavior. Nothing mechanical enforces them — by the repo's own principle that a guarantee depending on the LLM is not a guarantee — so no command can settle whether the claim holds in practice. · *would settle it:* `No single command. The closest mechanical proxy is checking that the shipped skills actually state these rules: grep -rn 'never invent' internal/skills/embedded/ internal/skills/embedded-guidance/`
- **docs/workflows.md** — workflows.md:59 — "`aiwf check` verifies that reference resolves to a real epic". · *why:* Partially verified only. The `refs-resolve` rule exists (internal/check/check.go:56, emitted at :659 and :682), but I did not construct a milestone with a dangling parent to observe the finding fire, because doing so would require hand-writing entity frontmatter, and the sandbox path for it (edit a file, then run check) was not exercised. · *would settle it:* `In a disposable repo: hand-edit a milestone's `parent:` to a nonexistent E-9999 and run `aiwf check --verbose`, expecting a refs-resolve finding.`

---

## Provenance

Audited 2026-08-18/19 against the working tree at `e27f19eab`. Method as described
under *Scope and method*. No fixes were applied in the same pass, so every
citation reflects the tree as read. Per-subject ledgers with full evidence,
including the 203 unverifiable claims and the commands that would settle them,
accompany this document.
