---
id: E-0086
title: Build the smallest preflight pass that can be trialled and can fail
status: proposed
---
## Goal

Build the smallest version of the preflight pass that can be trialled and can
fail: the parts run, the findings land somewhere a human triages, and
consecutive runs produce a number that says whether to keep it.

## Context

`docs/initiatives/milestone-preflight-as-independent-review.md` is the
specification. Its threads, seams, corpus, prohibitions, stop rule and base
rates live there and are not restated here; a claim here the initiative does
not carry is a defect in one of the two.

Already decided, and not re-opened by this epic: findings are not a seventh
entity kind (ADR-0003, `rejected`); the ledger is a `## Spec measurement` body
section (D-0066); disposition happens at a gate rather than a check rule
(D-0067); the sweep precedes the lab, and G-0583's contrary claim is disposed in
Scope. `M-0308/AC-2` and `AC-3` are `met` on a `done` milestone.

What this epic adds is a bar the specification does not set. The initiative's
stop rule says when a *pass* ends — every row is measured, pinned, recorded, or
tracked. Nothing says when the *method* should be abandoned, and that is what
instrumenting the parts separately makes judgeable.

## Scope

- **The reviewer brief**, as a `SKILL.md`. It carries every load-bearing part
  the initiative lists.
- **`wf-preflight-sweep`** — the reading method: what to sweep, how to select
  from an index, what may be concluded.
- **`wf-preflight-lab`** — runs commands, owns the execution safety boundary,
  and is the only place a claim is settled true. Ships the experiment's shape —
  a disposable tree, a kill criterion stated before building, and works /
  does-not / inconclusive — without a trigger for when one fires.
- **The sweep-to-lab handoff**, at the seams where the lab is dispatched: every
  unknown row, carrying the command the sweep would have run, and every
  contradicted row.
- **The criterion challenge**, riding whichever artifact the seam already
  invokes. It borrows `wf-vacuity`'s probe vocabulary, not that skill's
  `## Clean` verdict.
- **The ledger**, with a disposition column, and the seam instruction that
  writes it into `## Spec measurement`.
- **A row in `docs/design/oracles.md`'s inventory.**
- **The wrap-gate enumeration**, in `aiwfx-wrap-milestone` and
  `aiwfx-wrap-epic`: every undispositioned row listed, with rows total and
  settled stated alongside.
- **Seam instructions at epic drafting and milestone preflight**, each naming
  what it dispatches and carrying the stated escape.
- **The `reviewer` role card**, which emits an approval verdict the
  no-clearance rule removes.
- **Instrumentation, per seam and never pooled** — per-phase cost, what each
  phase found, what each finding changed, rows undispositioned at wrap, what
  the pass missed and surfaced later, and the records each pass creates, priced
  by the tier each lives in (D-0054).
- **Each corpus tier named by its own shape**, never by a repository path:
  commitments by kind and status, entities by the status-carrying index
  including archived ones, documents by property.
- **The initiative's Threads table and promotion status**, updated for every
  thread this epic promotes.
- **Disposing the stale premises carried by the gaps listed here** — G-0582
  (promotes to `addressed` by commit), G-0560 (a body edit to its ADR-0003
  bullet), G-0583 (promotes to `addressed` once the claims the initiative
  contradicts are answered).

## Out of scope

- **The patch-start seam.**
- **A trigger for when an experiment fires.**
- **A check rule over the ledger** — D-0067.
- **The clearance sites in shipped skills** — G-0585, wholly.
- **The compaction handoff** — G-0586.
- **`aiwf show` rendering a terminal reason** — G-0590.
- **The forward-tense reference check** — G-0591.
- **Any kernel change.** No finding code, no config field, no schema entry.

## Constraints

- **The initiative is the specification**, and every milestone under this epic
  runs the pass on itself before it starts, dispatched to a reviewer that is
  not its author.
- **The trial must be able to fail, and every mandate names what retires it.**
  Every milestone shipping a part of the pass also ships the measurement that
  would show that part not earning its cost.
- **Consecutive subjects at each seam**, not chosen ones.
- **The brief ships as an artifact with a mandated shape**, so what it asks for
  does not vary with who dispatches it. G-0263 names the author-written brief as
  the exposure and mandating the shape as the lever.
- **Every `SKILL.md` edit lands with its referencing structural test**, and
  those tests assert shape, not wording (D-0050).
- **No new vocabulary without a defect it fixes.** The terms are the
  initiative's: the reading states *contradicted*, *ambiguous*, *complicated*,
  *unknown*; the measured values *true*, *false*, *unable to measure safely*;
  the dispositions *pinned*, *recorded*, *tracked*, *superseded*.
- **No surface this epic ships offers a verdict reading could earn.**
- **Shipped surfaces carry no rationale or development history.**
- **Every count this epic or its milestones writes down names the command that
  produced it, or is dropped.**

## Success criteria

Observable at epic close. Milestone ACs carry the mechanical bar.

- [ ] A milestone can be started by dispatching a reviewer that is not its
      author, and the ledger it returns is readable afterwards by someone who
      was not in that session.
- [ ] A wrap gate lists every undispositioned row.
- [ ] A claim is settled true only where a command, its expected result, its
      observed output and its environment sit together.
- [ ] The reviewer's source selection is recorded with a reason per source, and
      the ones the subject cites nowhere are named as candidate omissions.
- [ ] At every seam where the lab is dispatched, it receives every unknown row
      and every contradicted row.
- [ ] Both seams name what they dispatch, and skipping the pass requires a
      stated reason at a gate.
- [ ] Consecutive trial runs record per-phase cost and yield per seam.
- [ ] No shipped surface this epic touches names a non-consumer repository path
      as its corpus. G-0548 owns the tree-wide half.

## Threads this epic promotes

| thread | this epic |
|---|---|
| T1 independent reviewer | promoted |
| T2 epic context | promoted |
| T3 audit scope | promoted — commitments, documents, entities |
| T4 code comments | not promoted — the initiative carries no code tier |
| T5 the lab | promoted |
| T6 experiments | partial — the shape ships; the trigger does not |
| T7 stated escape | partial — both seams carry it |
| T8 delivered as a skill | promoted — the brief, `wf-preflight-sweep`, `wf-preflight-lab` |
| T9 examples | promoted, split across sweep and lab |

## Open questions

| Question | Blocking? | Owner |
|---|---|---|
| What counts as something the pass missed, and who records it? | yes | The trial's only measure of what the pass did not catch. Needs its own term or none. |
| Can a prose deliverable carry an acceptance criterion that can fail? | yes | G-0584 names M-0308's derivation as the first instance of a form that can, and `findAllVerbs` survives in `internal/policies/`. The first milestone reuses that form or states why it does not reach this deliverable. |
| Does `oracles.md` admit a pass that offers no verdict? | yes | That document requires a verdict rather than a vibe; this pass offers none by constraint. Decides whether the inventory row is legal. |
| Does a prospective per-run record survive the rules against it? | yes | `growth.md` reconstructs apparatus metrics from git history at any commit and states nothing has to be measured in advance to stay comparable; D-0054 records obligations rather than events, and a finished run is a completed act; the not-in-scope list excludes an append-only event log. Decides whether the instrumentation Scope bullet is buildable as written. |
| Does the initiative owe T3 a code tier? | no | A defect in the initiative, unresolved between the two documents. |
| Does the patch-start seam get an entity, or stay a deferral with no destination? | no | Needs a destination or an explicit decline. |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Prose deliverables resist the AC-evidence bar | high | Named as a blocking question. |
| Nothing makes an assistant run a prose pass | med | ADR-0019's advisory precedent; D-0067 names the measurement that would earn a check rule. |
| The trial measures operator patience rather than the method | high | Consecutive subjects, per-seam instrumentation, and a defined measure of what the pass missed. |
| The pass acquires vocabulary faster than evidence | med | The no-new-vocabulary constraint. |
| The corpus the sweep reads is itself drifted | med | G-0560 for the normative tier. |
| The wrap gate depends on an assistant enumerating honestly | med | The trial records the undispositioned rate. |

## Milestones

To be proposed after the preflight pass runs on this draft.

## References

- `docs/initiatives/milestone-preflight-as-independent-review.md` — the specification
- ADR-0043, D-0066 — the ledger's surface
- ADR-0019 — the advisory `wf-*` genus
- ADR-0007, ADR-0006 — skill placement and coverage
- ADR-0028 — role-agent dispatch routing
- ADR-0044, G-0587, G-0589 — the corpus described by shape
- ADR-0024 — the rejected reference-skill shape
- ADR-0004, ADR-0003 — archive reversal; the seventh kind
- D-0067, D-0054, D-0053, D-0056, D-0050, D-0038 — disposition, what is worth
  recording and where, what retires a mandate, what earns a gap, test shape,
  AC evidence
- `docs/design/oracles.md`, `docs/design/growth.md`
- `CLAUDE.md` §"AC promotion requires mechanical evidence", §"Skills policy"
- G-0271, G-0317, G-0530, G-0548, G-0560, G-0571, G-0580, G-0582, G-0583,
  G-0584, G-0585, G-0586, G-0590, G-0591, G-0592
- M-0308, M-0309, E-0085 — historical evidence; none binds

## Spec measurement

Four sweeps, same brief, fresh reviewer each, against successive revisions of
this draft. Reading states are the sweeps'; dispositions are this session's. The
measured column is blank throughout — the lab is not dispatched at this seam, so
a row here ends recorded or tracked. A command in an evidence cell grounds a
disposition; it does not settle a claim, which needs the command, its expected
result, its output and its environment together.

| claim | reading | measured | evidence | disposition |
|---|---|---|---|---|
| G-0591 names a deferral with no destination as worse than one to declined work | contradicted |  | `grep -n` over G-0591 — it distinguishes delivered from declined | repaired; recorded |
| D-0052 settles shape-descriptions for repository paths | contradicted |  | D-0052's Decision is the `skill-body-id` keep-list and id shapes | repaired; recorded |
| The lab receives only factually-contradicted rows | contradicted |  | the initiative sends every contradicted row | repaired; recorded |
| ADR-0007 carries a "no meaning outside aiwf" placement test | contradicted |  | `grep -n discriminator` — ADR-0007 asks whether a skill surfaces a kernel capability | repaired; recorded |
| G-0580 records that the backstop misses templates | contradicted |  | G-0580 names role-agent cards and verb skills under `internal/skills/embedded/` | repaired; recorded |
| D-0056 grounds declining the patch-seam deferral | contradicted |  | D-0056 excludes defects from its declines | repaired; recorded |
| `oracles.md`'s inventory requires a declared failure asymmetry | contradicted |  | `oracles.md:176` — the columns are oracle, class, judges, fires at, on failure | repaired; recorded |
| A non-vacuous form for a prose AC does not yet exist | contradicted |  | G-0584 names M-0308's derivation as the first instance; `findAllVerbs` survives in `internal/policies/` | repaired; recorded |
| `--by-commit 052a398` is the commit that rejected ADR-0003 | unknown |  | `git show 052a398` → `aiwf promote ADR-0003 accepted -> rejected`, one file | recorded |
| `--by-commit` and `--by` are drifting spellings of one flag | unknown |  | `aiwf promote --help` — two distinct flags | recorded |
| The ledger's home is `## Spec measurement` | unknown | | D-0066 | recorded |
| A template section would scaffold the ledger | contradicted | | D-0066 places it as evidence, not scaffolding | repaired in the initiative; recorded |
| Epic-drafting rows reach a wrap gate | complicated | | the initiative gives epic drafting no wrap | repaired in the initiative; recorded |
| The criterion challenge sits inside the sweep's brief | contradicted | | the initiative gives it no dispatch of its own | repaired; recorded |
| The backstop's reach is evidence that a `SKILL.md` deliverable is pinned | complicated | | G-0317 | tracked — G-0317 |
| A shipped prose rule has no seam to enforce it | complicated | | G-0526 | tracked — G-0526 |
| Role-agent dispatch is absent from the always-on guidance | complicated | | G-0370 | tracked — G-0370 |
| The patch seam has no defined shape | complicated | | G-0060 | tracked — G-0060 |
| The ledger's write path passes a body-section membership scan | complicated | | E-0084 | tracked — E-0084 |
| Root and derived documents are unclassified for a corpus tier | complicated | | G-0589 | tracked — G-0589 |
| The path clause of the shipped-surface rule has no check | complicated | | G-0548 | tracked — G-0548 |
| A bare `docs/` path in a shipped skill trips no check | unknown | | G-0587 names the experiment as unrun | tracked — G-0587 |
| An epic wrap must close or correct the epic spec's own gap claims | complicated | | G-0515 | recorded — a milestone obligation |
| `skill-author-guide.md` rule 5 requires a skill to run `aiwf check` before returning success | complicated | | the sweep is forbidden to measure | recorded — unresolved between the two documents |
| `oracles.md` requires a verdict; this pass offers none | ambiguous | | `oracles.md:33` | recorded — blocking open question above |
| Independence is only as strong as the brief, and the author writes it | complicated | | G-0263 | recorded — see below |
| One dispatch exhausts the corpus | contradicted | | four sweeps each selected sources the others did not | repaired in the initiative; recorded |
| The specification sets no bar for ending | contradicted | | the initiative's stop rule ends a *pass*; the epic's bar is for the *method* | repaired — the two subjects are now distinguished; recorded |
| Asserting the sweep-before-lab ordering as decided conflicts with G-0583 being open | contradicted | | the epic disposes G-0583's contrary claim in Scope | repaired — the link is now stated; recorded |

Every row is unsettled as to truth. They stay open until milestone preflight,
where a lab is dispatched.

**What this pass missed, recorded against itself.** Later sweeps reached sources
the earlier ones did not, and each repair round introduced a fresh contradicted
row. The reviewer that found them was reading a brief this draft's own author
wrote — the exposure G-0263 names.
