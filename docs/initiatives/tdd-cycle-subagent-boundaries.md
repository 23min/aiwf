---
title: TDD-cycle subagent boundaries — bounding cycle lifetime to close the LLM-discipline gap
status: captured
date: 2026-07-16
---

# TDD-cycle subagent boundaries — bounding cycle lifetime to close the LLM-discipline gap

## Classifier note

This is an initiative document. `initiative` is not yet an official aiwf
entity kind ([G-0311](../../work/gaps/G-0311-no-cross-cutting-initiative-tier-above-epic-for-multi-component-features.md)),
so this file lives under `docs/initiatives/` as an umbrella capture,
following the precedent of
[`id-lifecycle.md`](id-lifecycle.md) and
[`agent-agnostic-execution-topology.md`](agent-agnostic-execution-topology.md).

This is not an ADR: it does not ratify a decision. The one design it
reconciles that reached ADR status ([ADR-0009](../adr/ADR-0009-orchestration-substrate-vs-driver-split.md))
was rejected, not superseded — a fresh ADR is a possible later output of
this initiative, not a precondition for it.

This is not a plan: it intentionally avoids committing to epics,
milestones, or sequencing. Its job is to hold the shape of the problem
still long enough that a right-sized plan can be drafted from a coherent
center, rather than from the oversized design ADR-0009 attempted.

## Initiative statement

Several separate artifacts answer overlapping parts of one question: *how
does aiwf make TDD-cycle discipline hold structurally, instead of depending
on an LLM remembering the rules across a long conversation?*

1. **[G-0252](../../work/gaps/G-0252-wf-tdd-cycle-red-first-ordering-unguarded-for-consumer-tdd-required-ac-cycles.md)**
   (`status: open`) — `wf-tdd-cycle` asks for a failing test before the
   implementation, but nothing mechanically confirms the test *preceded*
   the code. Root cause: the skill is advisory text a long conversation can
   drift through.
2. **[ADR-0009](../adr/ADR-0009-orchestration-substrate-vs-driver-split.md)**
   (`status: rejected`, 2026-07-16) — proposed bounding a cycle's lifetime
   to a fresh subagent invocation as the structural fix, bundled with a
   substrate/driver naming split, a ~17-key `aiwf-cycle-*` trailer schema,
   and a parallel multi-AC execution model. Rejected as oversized relative
   to E-0019 (still unstarted): its Decision 3 (isolation-escape) had
   already shipped independently via M-0106, using the kernel's existing
   provenance trailers rather than the new schema this ADR proposed;
   Decisions 1-2 remained speculative infrastructure with no consumer.
3. **[E-0019](../../work/epics/E-0019-parallel-tdd-subagents-with-finding-gated-ac-closure/epic.md)**
   (`status: proposed`, unstarted) — the epic that would have implemented
   ADR-0009's design, including the finding-gated AC closure model. Its
   design doc, [`parallel-tdd-subagents.md`](../pocv3/design/parallel-tdd-subagents.md),
   depends on the `finding` entity kind, which is not implemented (only six
   kinds exist in the kernel today, despite ADR-0003/ADR-0004 being
   `accepted` as ratified *designs*).
4. **[D-0017](../../work/decisions/D-0017-isolation-escape-cherrypicked-param-shape.md)**
   (`status: proposed`) — a tactical param-shape decision for
   isolation-escape's cherry-pick detection, already implemented in M-0106.
   Orthogonal to this initiative's scope; tracked separately in the same
   decision-review pass that produced this document.
5. **[G-0099](../../work/gaps/G-0099-worktree-isolation-parent-side-precondition.md)**
   (`status: open`) — its resolution shipped via M-0106 ("closes G-0099" per
   the milestone's own title), but the gap was never promoted to
   `addressed`. Bookkeeping drift, not an open design question.

## Prior art already shipped

**[M-0106](../../work/epics/archive/E-0030-branch-model-chokepoint-branch-flag-sequencing-isolation-escape-finding/M-0106-kernel-finding-isolation-escape-closes-g-0099.md)**
(`status: done`, 13/13 ACs met) shipped the `isolation-escape` `aiwf check`
rule: a subagent's commits are verified reachable from its declared
worktree branch, using the kernel's existing `aiwf-scope` / `aiwf-branch` /
`aiwf-branch-sha` provenance trailers. This gives worktree-isolation
enforcement for free, independent of whatever cycle-orchestration model
this initiative eventually recommends — it does not need to be re-designed
or re-scoped by follow-on work here.

## Sequential-subagent-per-AC — considered, deferred

Bounding a single AC's TDD cycle to one fresh subagent invocation, run
**sequentially** (one AC at a time, in its own isolated worktree, merged and
triaged before the next dispatch) was the initial narrower alternative to
ADR-0009's rejected parallel model. It was weighed on 2026-07-16 and
**deferred**: the added ceremony (worktree create/dispatch/merge/triage per
AC, cross-AC context loss between fresh subagent invocations, a heavier
rollback unit than backing out a few conversation turns) was judged to
outweigh the benefit for the common case of small, coupled ACs. It is not
itself a mechanical guarantee — a fresh subagent still has to get one AC's
ordering right in a single pass, it just has less surface to drift on. The
lower-cost alternative below closes the same principle directly instead.

Worth re-litigating later if a milestone with several large, genuinely
independent ACs makes the ceremony cost worth paying — the worktree-isolation
primitive (`aiwf worktree add`, M-0106's `isolation-escape` check) is already
in place either way and needs no rework if this direction is revisited.

## Scoped target — mechanical checks over workflow redesign

Rather than redesign how cycles execute, target the specific unenforced
claims directly, each a real kernel chokepoint gap rather than a workflow
change:

1. **[G-0252](../../work/gaps/G-0252-wf-tdd-cycle-red-first-ordering-unguarded-for-consumer-tdd-required-ac-cycles.md)**
   — red-first ordering unguarded for `tdd: required` AC cycles.
2. **[G-0140](../../work/gaps/G-0140-implement-evidence-flag-on-aiwf-promote-ac-met-per-d-0005.md)**
   — implement the `--evidence` flag per
   [D-0005](../../work/decisions/D-0005-ac-mechanical-evidence-promote-time-evidence-flag-binds-claim-to-test-symbol.md)
   (`status: accepted`, promoted 2026-07-16 — the design was already being
   treated as decided in practice; the flag itself is unbuilt). Arguably the
   higher-leverage of the two: CLAUDE.md already states AC-met requires
   mechanical evidence for *every* milestone, not just `tdd: required` ones,
   and today that's reviewer-discipline only.
3. **[G-0286](../../work/gaps/G-0286-acs-shape-tdd-phase-over-demands-a-phase-on-every-ac-under-tdd-required.md)**
   — `acs-shape/tdd-phase` over-demands a phase on every AC under
   `tdd: required`, reddening the tree on an `advisory → required` upgrade.
   Friction cleanup, not a discipline hole, but cheap and adjacent.
4. **[G-0334](../../work/gaps/G-0334-milestone-can-start-and-finish-with-zero-acceptance-criteria-no-guard.md)**
   — a milestone can start and finish with zero ACs, no guard. Adjacent axis
   (milestone-level completeness rather than per-AC ordering/evidence) but
   the same "advisory ritual, no kernel backstop" shape.

**Deferred, re-litigate once 1-4 land:**

- **[G-0253](../../work/gaps/G-0253-branch-coverage-audit-is-statement-scoped-not-per-arm-branch-coverage.md)**
  — branch-coverage-audit is statement-scoped, not per-arm. Real gap
  (a defensive branch's untested arm can still read as "covered"), but the
  fix needs AST-level arm enumeration or new toolchain support — a
  meaningfully bigger lift than 1-4. Revisit once the others are built and
  its relative priority can be judged against whatever friction actually
  showed up.

## Open design questions

- **What happens to E-0019?** Drafted against ADR-0009's now-rejected
  parallel model. Needs an explicit call — re-scope to the narrower
  sequential design, or close it and file a fresh, smaller epic — not
  settled by this document.
- **Where do findings live** when a subagent surfaces one, given `finding`
  isn't shipped? Candidates: human-triaged inline text (simplest, matches
  the sequential model's checkpoint cadence); a lightweight body-section
  convention on the AC itself; or deferring until `finding` ships. No
  producer exists yet to force the choice.
- **Does the scoped-target list need an ADR at all?** G-0252/G-0140/G-0286/
  G-0334 are each ordinary kernel-check/verb work with an existing gap and
  (for G-0140) an accepted decision behind them — no new architectural
  commitment, so no ADR is needed to build them. Only resurfaces if the
  sequential-subagent direction above is ever revisited.

## Housekeeping surfaced while writing this

- `aiwf promote G-0099 addressed --by-commit <M-0106's wrap commit>` — the
  gap's resolution already shipped; only the status field is stale. Still
  open as of this writing.
- ~~`aiwf promote D-0005 accepted`~~ — done, 2026-07-16. Was `proposed` while
  already being treated as decided in practice (G-0140's body cites it as
  "committed in M-0123 phase 1"; the spec cells already encode it).

## Risks and boundaries

**Risk: re-accumulating ADR-0009's scope creep.** The sequential-subagent
step should stay the whole first step. Resist folding parallelism,
findings-gating, or the cycle-trailer schema back in until the sequential
version is dogfooded and its own friction (if any) is measured — the same
discipline [D-0037](../../work/decisions/D-0037-defer-adr-0001-g-0281-and-emb-pending-a-measured-id-collision-trigger.md)
applied to the id-lifecycle cluster.

**Risk: E-0019 sitting orphaned.** A `proposed` epic whose originating ADR
is `rejected` is a small tree-hygiene gap (nothing in `aiwf check` flags
epic-to-ADR provenance drift). Left open by this document; worth a decision
in its own right before this initiative's narrower direction is built.

## Desired future property

A TDD cycle for one acceptance criterion cannot drift off-discipline over a
long conversation, structurally — reachable via direct mechanical checks on
the actual unenforced claims (ordering, evidence) rather than a workflow
redesign. The scoped-target list is the current path to that property; the
sequential-subagent direction remains available as a later option if
dogfooding the checks alone still leaves friction on large, independent-AC
milestones.
