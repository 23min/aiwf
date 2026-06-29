---
id: E-0050
title: 'Gate-discipline foundation: generalize the declared-sequence gate'
status: done
---
# Gate-discipline foundation: generalize the declared-sequence gate

## Goal

Generalize the wf-patch declared-sequence gate into a general capability for any
sequence of local, reversible mutations — one gate that enumerates every action
verbatim, binds approval to exactly that list (subset approval allowed), and
aborts + re-gates on any deviation — and fix the wrap and release rituals that
currently violate the gate discipline CLAUDE.md *claims* they already follow.

## Context

A 2026-06-28 audit of the embedded skills against CLAUDE.md found that the
gate-discipline section describes a state of the world that is false: it asserts
the wraps "keep per-action gates," but `aiwfx-wrap-milestone`, `aiwfx-wrap-epic`,
and `aiwfx-release` run ungated promote / merge / branch-delete steps and bundle
two origin pushes under one approval. Filed as gap G-0295.

This epic was extracted from E-0049 (ritual lifecycle model), where G-0295 was
the "program lead" milestone that had to land before the sibling content epic
E-0048. That ordering is a real dependency, but homing it inside E-0049 forced an
interleave (E-0049 milestone 1, then pause for all of E-0048, then resume) and
left two epics `active` concurrently. The gate fix is not E-0049-specific: it is a
**shared foundation both E-0048 and E-0049 depend on** — every milestone wrap in
both epics should run under the corrected gate. Pulling it into its own epic makes
the dependency a clean DAG node instead of a mid-epic interleave.

## Scope

### In scope

- The Tier-1 decision from G-0295: generalize the declared-sequence gate to any
  local, reversible mutation sequence.
- The bright line: batch local, reversible mutations that occur at a single
  moment; exclude (a) outward / irreversible actions (push, PR-create, tag-push,
  remote-branch delete, `--force`) and (b) timing-bearing mutations (`tdd:
  required` phase promotes fire live).
- The standing rule written into CLAUDE.md's gate-discipline section and
  `.claude/aiwf-guidance.md`; CLAUDE.md's "wf-patch only" scope sentence rewritten.
- The three ritual fixes: `aiwfx-release` step 6 split into two separate push
  gates; `aiwfx-wrap-milestone` and `aiwfx-wrap-epic` ungated promote / merge /
  cleanup steps replaced with one declared-sequence gate (push excluded).
- Structural tests pinning the rule (e.g. the rituals enumerate the sequence; the
  release ritual carries two separate push gates).

### Out of scope

- The `aiwf.yaml` declared-sequence-wraps opt-in knob (G-0296, Tier 2) — stays
  deferred; lands later only if the Tier-1 standing rule proves insufficient.
- The Model 1 commit / live-phase-promote model (G-0293) — stays in E-0049.
- All skill-body content correctness and drift chokepoints (E-0048).

## Constraints

- The bright line is the load-bearing safety claim: the mechanical guarantees
  (pre-commit / pre-push hooks, `aiwf check`, CI) already catch bad end-states
  regardless of prompt count, so batching local mutations costs nothing
  mechanical; gates uniquely protect the outward / irreversible actions the hooks
  cannot reverse. Outward actions therefore keep standing gates.
- `--force` remains sovereign / human-only and never batched.
- This epic lands before E-0048 and E-0049 begin, so every subsequent milestone
  wrap in both epics runs under the corrected gate. It ships its own structural
  tests.
- Skill edits are authored in the embedded snapshot per ADR-0016.

## Success criteria

- [ ] The declared-sequence gate is documented as a general capability in
      CLAUDE.md and `.claude/aiwf-guidance.md`; the old "wf-patch only" scope
      sentence is rewritten.
- [ ] No ungated mutating action remains in `aiwfx-wrap-milestone`,
      `aiwfx-wrap-epic`, or `aiwfx-release`; the two origin pushes in the release
      ritual stand as two separate gates.
- [ ] The bright line (local/reversible batched; outward/irreversible and
      timing-bearing excluded) is stated and exercised by a structural test.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Does the declared-sequence rule live as prose only, or is a structural test feasible over ritual bodies? | no | decided in the milestone — at minimum the release-ritual two-push split is testable structurally |

## Milestones

<!-- execution order; ids allocated at plan-milestones time -->

1. Generalize the declared-sequence gate + fix wrap/release drift + ship
   structural tests (G-0295).
