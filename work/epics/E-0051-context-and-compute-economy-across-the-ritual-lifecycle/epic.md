---
id: E-0051
title: Context and compute economy across the ritual lifecycle
status: proposed
---
# Context and compute economy across the ritual lifecycle

## Goal

The ritual lifecycle uses the right context and the right compute at each
boundary: read-heavy, judgment-light steps run on a cheap model, and the
session topology keeps build context where it accumulates while OS-enforcing
worktree isolation. The operator gets the doctrine *emitted by the rituals at the
right moment*, not buried in a doc.

## Context

A 2026-06-28/29 workflow review (during E-0050) surfaced two efficiency axes the
rituals never addressed:

1. **Compute economy.** Several ritual steps are read-heavy and judgment-light —
   `aiwfx-wrap-epic`'s ADR harvest (walk every epic commit + diff), the doc-lint
   sweeps, the start-ritual preflights, even the independent two-lens review.
   Today they run inline on the main (expensive) model. Nothing tells a ritual to
   dispatch the scan to a cheap-model subagent that returns a shortlist for the
   main model to adjudicate.

2. **Context / session economy.** Claude keys its conversation store by launch
   directory, so there is no clean cross-directory resume. The durable context in
   aiwf is the planning tree (specs, ACs, decisions), not the chat. That makes a
   session topology possible — a home session on `main` for planning + epic
   activation + epic wrap, and one continuous build session *anchored in the epic
   worktree* for all of that epic's milestones. Anchoring the build session makes
   worktree isolation **OS-enforced** (the process cwd is the worktree, so a stray
   command cannot leak to `main`) rather than discipline-dependent — the operator
   layer's version of "correctness must not depend on memory" (closes the
   discipline-dependent side of G-0099). The build context that is expensive to
   reconstruct (milestone 1's gotchas at milestone 3) stays in the continuous
   build session; only the planning chat is shed, and that is captured in entities
   by design.

Both axes share one theme — spend context and compute where they pay off, shed
them where they don't — so they are one epic, not two.

## Scope

### In scope

- **Pillar A — cheap-model delegation for scan-steps.** A doctrine (in guidance)
  plus per-step wiring: `aiwfx-wrap-epic` ADR harvest and doc-lint, start-ritual
  preflights, and the wrap independent-review step dispatch to a cheap-model
  subagent that returns a shortlist / report; the main model adjudicates. Name the
  model tier per step (Haiku for pure scan; Sonnet for review judgment).
- **Pillar B — session-topology doctrine + ritual hand-off emission.** Document
  the home-session / anchored-build-session-per-epic / home-session-wrap topology.
  Wire the advisory hand-offs into the rituals: `aiwfx-start-epic` emits
  "open a build session in `<worktree dir>` — prompt: `<re-hydration prompt>`";
  `aiwfx-wrap-milestone` emits, on the epic's last milestone, "return to the home
  session to wrap the epic." The hand-offs are emitted by the ritual at the right
  moment (AI-discoverable), not left as operator lore.
- The standing rule that planning reasoning worth keeping is recorded as an ADR /
  `D-NNNN` / rich spec body during planning, so a fresh build session re-hydrates
  fully — making any manual handoff belt-and-suspenders, not the mechanism.

### Out of scope

- The gate model itself (E-0050) and the commit/TDD model (E-0049).
- Skill-content correctness and drift chokepoints (E-0048).
- Programmatic multi-agent orchestration of the build itself (this epic is about
  *delegating bounded scan steps* and *human session topology*, not auto-driving
  milestones).

## Constraints

- Advisory, not load-bearing: a cheap-model shortlist is always adjudicated by the
  main model; a session hand-off is a printed recommendation, never an automated
  directory switch. Correctness never depends on the delegation or the topology.
- OS-enforced isolation is the point of pillar B — the doctrine must not reduce to
  "remember to `cd`"; the anchored session *is* the mechanism.
- Subagent dispatch reuses the existing `Agent` channel and the parent-side
  worktree-isolation precondition (CLAUDE.md §"Subagent worktree isolation") for
  any subagent that writes.

## Success criteria

- [ ] At least the named read-heavy ritual steps (ADR harvest, doc-lint sweeps,
      preflights) document a cheap-model subagent dispatch with the model tier
      named, and the doctrine is stated in guidance.
- [ ] `aiwfx-start-epic` and `aiwfx-wrap-milestone` emit the session hand-off
      advisories at the right moment, pinned by a structural test.
- [ ] The session-topology doctrine is documented where consumers see it
      (guidance fragment), not only in CLAUDE.md.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Which cheap-model tier per step (Haiku vs Sonnet)? | no | decided per-step at milestone planning |
| Does the hand-off prompt live in the ritual body or a generated fragment? | no | decided in the pillar-B milestone |

## Milestones

<!-- execution order; ids allocated at plan-milestones time -->

1. Pillar A — cheap-model delegation doctrine + wire the scan-steps.
2. Pillar B — session-topology doctrine + ritual hand-off emission (+ structural test).
