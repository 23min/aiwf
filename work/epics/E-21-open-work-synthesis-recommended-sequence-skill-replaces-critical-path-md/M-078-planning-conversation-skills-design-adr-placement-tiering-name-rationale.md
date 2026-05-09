---
id: M-078
title: Planning-conversation skills design ADR (placement, tiering, name rationale)
status: in_progress
parent: E-21
tdd: none
acs:
    - id: AC-1
      title: ADR allocated under docs/adr/ and status proposed
      status: met
    - id: AC-2
      title: 'ADR records placement: rituals plugin, not kernel-embedded'
      status: met
    - id: AC-3
      title: ADR records pure-skill-first tiering rule
      status: met
    - id: AC-4
      title: 'ADR records name worked example: aiwfx-whiteboard with rejected alternatives'
      status: open
    - id: AC-5
      title: ADR cross-references M-074 skills ADR and CLAUDE.md principles
      status: open
---

# M-078 — Planning-conversation skills design ADR (placement, tiering, name rationale)

## Goal

Capture the design rationale that shapes the rest of E-21 as a single ADR — *where* planning-conversation skills live (rituals plugin, not kernel), *when* such skills warrant a backing kernel verb (only when usage shows the synthesis re-deriving the same data), and *what* this skill is named with its rejected alternatives. The ADR is the discoverable artefact future planners will hit when they ask the same questions about a future synthesis skill.

## Context

E-21's epic spec lists three open questions resolved during milestone planning on 2026-05-08: skill name, kernel-vs-plugin placement, and pure-skill-vs-skill+verb tiering. Each is principle-shaped — the answer applies beyond `aiwfx-whiteboard`. M-074 (under E-20) sets the precedent that skill-organisation policy belongs in an ADR, not a project-scoped D-NNN; this milestone files the complementary ADR for *placement and tiering* (M-074's covers *granularity within a topic*). Together the two ADRs define how skills get organised across the kernel/plugin boundary.

The decisions are locked at planning time. This milestone's job is recording, not deciding — the body content is largely transcription of the rationale the operator and assistant walked through. Status remains `proposed` so the ADR can be revised during M-079 implementation if the act of building the skill surfaces new constraints.

## Acceptance criteria

### AC-1 — ADR allocated under docs/adr/ and status proposed

ADR is allocated via `aiwf add adr --title "<title>"`, lives at `docs/adr/ADR-NNNN-<slug>.md`, frontmatter sets `status: proposed`. Title (refine at allocation): *"Planning-conversation skills: rituals-plugin placement; pure-skill first, kernel verb only if usage demands it"*.

### AC-2 — ADR records placement: rituals plugin, not kernel-embedded

ADR body articulates the principle "planning-conversation skills go in the rituals plugin; kernel-embedded skills are verb wrappers." Cites the existing pattern (`aiwfx-plan-epic`, `aiwfx-plan-milestones`, `aiwfx-start-milestone`, `aiwfx-wrap-epic` are all plugin-side; `aiwf-status`, `aiwf-history`, etc. are kernel-embedded verb wrappers). Notes that `aiwfx-whiteboard` is a planning conversation, not a verb wrapper, so the principle applies.

### AC-3 — ADR records pure-skill-first tiering rule

ADR body articulates the principle "ship a synthesis function as a pure skill first; promote to a skill+verb pair only when usage shows the skill re-deriving the same structured data on every invocation." Names the deferred follow-on (a `landscape`-style verb behind the skill) and documents the trigger condition for filing it (e.g., "skill repeatedly grovels through prose to extract data that should be structured"). Closes E-21's success criterion #7.

### AC-4 — ADR records name worked example: aiwfx-whiteboard with rejected alternatives

ADR body uses the `aiwfx-whiteboard` naming choice as the worked example demonstrating the placement and tiering rules in action. Records the rejected alternatives (`recommend-sequence`, `landscape`, `paths`, `focus`, `next`, `survey`, `synthesise-open-work`) with one-line rationale per rejection. The "whiteboard" metaphor's fit-rationale (ephemerality, surfacing-not-deciding, operator-at-the-board) is documented; this is the substantive content of the worked-example section.

### AC-5 — ADR cross-references M-074 skills ADR and CLAUDE.md principles

ADR body explicitly references M-074's skills-judgment ADR (the "per-verb default; topical multi-verb when concept-shaped" rule) and frames its own scope as complementary, not overlapping — M-074 covers *granularity within a topic*; this ADR covers *placement and tiering across kernel/plugin*. Cites CLAUDE.md's *"Kernel functionality must be AI-discoverable"* and *"Framework correctness must not depend on the LLM's behavior"* principles as the source authority for the placement reasoning.

## Constraints

- **No code in this milestone.** Pure ADR authorship. `tdd: none` because there is no test surface — the skill itself ships in M-079.
- **ADR scope is principle-shaped, not implementation-shaped.** Avoid stuffing this ADR with skill-body content (rubrics, output templates, Q&A flow) — that lives in M-079 in the SKILL.md body. The ADR articulates *why* and *where*; the skill articulates *what* and *how*.
- **Status remains `proposed`** through M-079. If M-079's implementation surfaces a constraint that changes the rationale, edit-body the ADR before promoting. Promotion to `accepted` happens at the E-21 wrap (in M-080) or in a follow-on milestone if there's no consensus to ratify yet.
- **No invention of unwritten rules.** The ADR records decisions made on 2026-05-08 in the milestone-planning conversation, with the rationale captured at decision time. New analysis or doctrine belongs in a separate, follow-up ADR.

## Design notes

- ADR allocation uses `aiwf add adr --title "..."` — the verb produces one commit with `aiwf-verb: add` and `aiwf-entity: ADR-NNNN` trailers. The body is then filled via `aiwf edit-body` (one further commit).
- The ADR's body sections (refine at authorship): *Context* (what question is being decided, when, why), *Options considered* (kernel-embedded vs rituals plugin; pure-skill vs skill+verb; name candidates), *Decision* (placement = rituals plugin; tiering = pure-skill-first; name = `aiwfx-whiteboard`), *Consequences* (forces all future planning-conversation skills into the plugin; future `landscape` verb is a separate kernel-side artefact when filed).
- The ADR's "worked example" subsection describes how the three rules cascade: rituals-plugin placement → `aiwfx-` prefix → name candidates evaluated against fit/clarity/PM-jargon-avoidance → `aiwfx-whiteboard` selected for ephemerality + collaborative-surface metaphor.
- Cross-reference to M-074 lives in the ADR's *Related* section; cross-reference to CLAUDE.md kernel principles lives inline in the rationale prose (with section names quoted for grep-ability).

## Surfaces touched

- `docs/adr/ADR-NNNN-*.md` (new — this milestone's primary deliverable)
- No code changes
- No CLAUDE.md changes (M-074 owns the *Skills policy* section; this ADR is filed alongside without re-editing CLAUDE.md)
- No skill files (M-079 owns those)

## Out of scope

- The actual `aiwfx-whiteboard` skill body — ships in M-079.
- A `landscape` kernel verb — deferred follow-on, possibly a future epic; this ADR only documents the trigger condition for filing it.
- Editing CLAUDE.md's *Skills policy* section — M-074's scope, and this ADR is complementary not overlapping (so no re-edit needed).
- Promotion of this ADR or M-074's ADR to `accepted` — both stay `proposed` for now; promotion is a separate decision happening at or after epic wrap.

## Dependencies

- E-20 / M-074 — the *Skills judgment ADR* this milestone's ADR cross-references. M-074 must have allocated its ADR (status `proposed` or later) so AC-5 can cite a real ADR-NNNN id rather than a placeholder. If M-074 hasn't shipped yet, M-078 must wait — confirmed at start-milestone.
- No other dependencies.

## Coverage notes

- (filled at wrap)

## References

- E-21 epic spec — open questions table; success criterion #7.
- M-074's *Skills judgment ADR* — sibling ADR on skill organisation (granularity within a topic). This ADR is its peer covering placement and tiering.
- `docs/pocv3/design/design-decisions.md` — kernel commitments; informs the placement reasoning (skills are advisory; the kernel verb surface is authoritative).
- CLAUDE.md *Engineering principles* §"Kernel functionality must be AI-discoverable" — primary authority for the placement principle.
- CLAUDE.md *Engineering principles* §"Framework's correctness must not depend on the LLM's behavior" — secondary authority; informs the pure-skill-first rule (skills are advisory; the kernel layer below them must remain authoritative).

---

## Work log

(filled during implementation)

## Decisions made during implementation

- (none — all decisions are pre-locked above)

## Validation

(pasted at wrap)

## Deferrals

- Promotion of this ADR to `accepted` is deferred to a separate decision after E-21 closure. Status remains `proposed` through wrap.

## Reviewer notes

- (filled at wrap)
