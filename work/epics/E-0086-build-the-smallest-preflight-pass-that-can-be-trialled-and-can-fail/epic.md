---
id: E-0086
title: Build the smallest preflight pass that can be trialled and can fail
status: done
---
## Goal

Keep the one rule the preflight work produced evidence for, and record why the
rest was dropped.

The epic originally set out to build the smallest version of the milestone
preflight pass that could be trialled and could fail. D-0069 records the
rejection of that method, with the measurements and the commands that reproduce
them. What survives is the lab rule.

## Context

`docs/initiatives/milestone-preflight-as-independent-review.md` is the
specification. Its threads, seams, corpus, prohibitions, stop rule and base
rates live there and are not restated here.

D-0069 rejects the dispatched generative reading pass. Its argument, in one
line: a generative review over prose has an unbounded output space, so the
finding count measures reviewer effort rather than artifact quality, successive
rounds are not comparable, and zero is unreachable by construction. The epic was
built to produce a number saying whether the method works, and that number
cannot mean what it was meant to mean.

The `## Spec measurement` section below is this epic's own run of the method on
itself, kept unchanged as the evidence behind that rejection.

Already decided, and not re-opened here: findings are not a seventh entity kind
(ADR-0003, `rejected`); the ledger is a `## Spec measurement` body section
(D-0066). `M-0308/AC-2` and `AC-3` are `met` on a `done` milestone.

## Scope

One thing.

- **The lab rule, recorded as a ban.** A claim is settled true only where a
  command, its expected result, its observed output and its environment sit
  together. Nothing else settles a claim: not a reading, not a reviewer's
  confidence, not a citation to a record that says so.

  It is a ban, not a mandate. It costs once — at the moment someone writes a
  claim down — and it names no procedure anyone must run, no artifact anyone
  must produce, and no seam anyone must wire. That is why it survives a re-scope
  that dropped everything with a per-subject cost.

## Out of scope

Everything else the epic planned. D-0069 carries the reasoning and the
measurements; this is the list.

- **The dispatched generative reading pass**, and the reviewer brief that
  carries it.
- **The sweep**, and the sweep-to-lab handoff.
- **The criterion challenge.**
- **The ledger's disposition column as a mandated artifact.**
- **The wrap-gate enumeration**, and the seam instructions in
  `aiwfx-wrap-milestone` and `aiwfx-wrap-epic`.
- **The per-seam instrumentation** — what a pass missed and what it added.
- **The `docs/design/oracles.md` row** claiming a model-judged advisory class.
- **Any kernel change.** No finding code, no config field, no schema entry.

Of these, only the reviewer brief reached an artifact; the rest were planned and
never built. D-0069 names the command that shows it.

## Constraints

- **Every count this epic's body writes down names the command that produced it,
  or is dropped.**
- **A test over prose pins a property some rule makes necessary, never words an
  author chose** (D-0050).
- **No new vocabulary.**

## Success criteria

Observable at epic close.

- [ ] The lab rule is recorded where a later reader can cite it, stated as a ban
      on what may be written down rather than a procedure anyone must run.
- [ ] Each dropped part is named, and the reason for dropping it is recorded
      with the commands that reproduce every measurement it rests on — so the
      drop can be re-examined rather than only re-argued.
- [ ] Every count in this epic's body names the command that produced it.
- [ ] Every milestone of this epic is terminal, and no artifact carrying the
      dropped reading method is left in the tree.

## Threads this epic promotes

| thread | this epic |
|---|---|
| T1 independent reviewer | dropped |
| T2 epic context | dropped |
| T3 audit scope | dropped |
| T4 code comments | not promoted — the initiative carries no code tier |
| T5 the lab | promoted, as the lab rule only — the ban, not the dispatched procedure |
| T6 experiments | dropped |
| T7 stated escape | dropped — nothing left to escape |
| T8 delivered as a skill | dropped — a ban does not need a skill |
| T9 examples | dropped |

## Milestones

None. The one thing kept is a ban, which is delivered by recording it (D-0069)
rather than by building anything.

- M-0310 — cancelled. Its subject is what a pass run records about itself:
  instrumentation for a pass that no longer runs.
- M-0311 — the reviewer brief. Disposition recorded in Open questions.

## Open questions

| Question | Blocking? | Owner |
|---|---|---|
| Does `wf-preflight-brief` retire, or get rewritten around the lab rule? | epic close | The brief is the reading half. It ships only on an unmerged branch, so retiring it costs no deletion. |
| Does the initiative owe T3 a code tier? | no | A defect in the initiative, unresolved between the two documents. |
| Does the patch-start seam get an entity, or stay a deferral with no destination? | no | Needs a destination or an explicit decline. |

## References

- `docs/initiatives/milestone-preflight-as-independent-review.md` — the
  specification, unchanged by this epic except for its Threads inventory
- D-0069 — the method's rejection, with the measurements and the commands that
  reproduce them
- D-0066 — the ledger's surface, kept: this epic's own `## Spec measurement`
  section below is written in the form it settles
- D-0067, D-0068 — decided for the dropped pass; each remains correct about what
  it decided and neither now has a pass to govern
- D-0050 — test shape over prose
- ADR-0003 — the seventh kind, rejected
- E-0085 — the one prior implementation attempt, cancelled

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
| A bare `docs/` path in a shipped skill trips no check | unknown | true | `aiwf check`, from a binary built from this worktree, over a tree where `agents/planner.md` carries bare `docs/adr/` and `work/decisions/`: expected 0 errors if unenforced, observed 0 errors. G-0548 carries the same conclusion in prose | measured — G-0587's unrun experiment is run; what counts as a path citation stays with G-0548 |
| An epic wrap must close or correct the epic spec's own gap claims | complicated | | G-0515 | recorded — a milestone obligation |
| `skill-author-guide.md` rule 5 requires a skill to run `aiwf check` before returning success | complicated | | the sweep is forbidden to measure | recorded — unresolved between the two documents |
| `oracles.md` requires a verdict; this pass offers none | ambiguous | | `oracles.md:33` | recorded — Scope takes the inventory's model-judged, advising class |
| Independence is only as strong as the brief, and the author writes it | complicated | | G-0263 | recorded — see below |
| One dispatch exhausts the corpus | contradicted | | four sweeps each selected sources the others did not | repaired in the initiative; recorded |
| The specification sets no bar for ending | contradicted | | the initiative's stop rule ends a *pass*; the epic's bar is for the *method* | repaired — the two subjects are now distinguished; recorded |
| Asserting the sweep-before-lab ordering as decided conflicts with G-0583 being open | contradicted | | the epic disposes G-0583's contrary claim in Scope | repaired — the link is now stated; recorded |

Every row is unsettled as to truth. No preflight pass will settle them: D-0069
rejects the method. The rows stay as they are — a dated record of what four
sweeps returned, not a live queue. Dispositions naming a repair to this epic's
Scope are void where the re-scope removed the text they cite; the reading and
evidence cells are the sweeps' own and are unchanged.

**What this pass missed, recorded against itself.** Later sweeps reached sources
the earlier ones did not, and each repair round introduced a fresh contradicted
row. The reviewer that found them was reading a brief this draft's own author
wrote — the exposure G-0263 names.
