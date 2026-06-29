---
id: G-0313
title: Consumer operating guidance drifts into CLAUDE.md, not the shippable guidance
status: open
---
## Problem

Consumer-facing *operating* guidance — rules for driving aiwf the tool in any
repo — keeps accreting in this repo's root `CLAUDE.md`, which never ships to
consumers. The shippable home is the embedded guidance source
(`internal/skills/embedded-guidance/aiwf-guidance.md`), which `aiwf init` /
`aiwf update` materializes into a consumer's gitignored `.claude/aiwf-guidance.md`
and wires into their root `CLAUDE.md` via a marker-managed `@`-import
(ADR-0018, E-0040). A rule that belongs there but lands in this repo's `CLAUDE.md`
instead is invisible to every consumer and forks from the single source of truth.

Three surfaces are in play:

- This repo's `CLAUDE.md` — about *developing aiwf itself*; correctly never ships.
- `internal/skills/embedded-guidance/aiwf-guidance.md` — the tracked authoring
  source; ships to consumers.
- `.claude/aiwf-guidance.md` (this repo) — the gitignored *materialized output*;
  this repo dogfoods, consuming the same copy a consumer gets via the `@`-import.
  Editing it directly is futile — the next `aiwf update` clobbers it.

## The dividing principle (audience, not importance)

- "How to OPERATE aiwf in any repo" → embedded guidance source → ships.
  (gate-per-mutation, reallocate-not-`git mv`, AC mechanical-evidence,
  one-decision-at-a-time, never-suggest-pause, the body-prose-id discipline.)
- "How to DEVELOP aiwf itself" → this repo's `CLAUDE.md` → correctly not shipped.
  (Go conventions, the test-parallelism discipline, `make ci` cadence, the
  release process, the chokepoint table, ritual-authoring locations.)

Because this repo dogfoods and imports the materialized guidance, an operating
rule placed in the embedded source is followed here *and* shipped — one source,
no fork. Placed directly in `CLAUDE.md`, it forks: aiwf's own repo has it, every
consumer is blind to it.

## Why it matters

Per the kernel principle "framework correctness must not depend on LLM
behavior," a discipline with no chokepoint drifts. Today nothing catches
"consumer-facing operating rule written into `CLAUDE.md` but never mirrored into
the shippable guidance." This rhymes with E-0048's thesis (shippable-content
integrity + drift chokepoints) — the embedded guidance is a shippable artifact
just like the skills.

## Proposed fix shape

1. **Audit + migrate.** Classify the current `CLAUDE.md` content by audience;
   extract the consumer-facing operating rules into the embedded guidance source,
   leaving `CLAUDE.md` to defer to the guidance for the general statement and keep
   only the repo-development specialization. Hybrid sections (e.g. gate
   discipline: the general rule ships, the `make ci` / `wf-patch` specifics stay)
   are split, not moved wholesale.
2. **Keep the always-on fragment tight.** It loads every turn, so it carries only
   high-leverage operating rules; lower-frequency detail routes to the verb /
   ritual skills (on-demand) or the design docs (referenced via the doc-link
   carve-out). The four-tier layering: always-on guidance -> on-demand skills ->
   reference docs -> (not shipped) `CLAUDE.md`.
3. **Chokepoint (the hard part).** A fully mechanical "this rule belongs in
   guidance" test is infeasible — tool-op vs dev-process is a judgment call. A
   lighter chokepoint is conceivable (e.g. flag named operating-rule anchors that
   appear in `CLAUDE.md` but not the embedded guidance, or assert a curated set of
   operating anchors is present in the embedded source), but its precise shape is
   a design question for the milestone. At minimum, an authoring rule in
   `CLAUDE.md` naming the dividing principle, with discipline as the interim
   chokepoint until something mechanical lands.

## Scope / sequencing

Candidate milestone in E-0048 (shippable-content integrity). Sequence after the
foundation milestones (M-0195 / M-0196) so any skill-body touches rebase onto the
swept bodies and inherit the edit-then-structural-test backstop. The migration is
trunk planning plus embedded-source edits; the chokepoint, if mechanized, is a
check or policy test sized at milestone-planning time.
