---
id: E-21
title: 'Open-work synthesis: recommended-sequence skill (replaces critical-path.md)'
status: proposed
---

# E-21 — Open-work synthesis: recommended-sequence skill (replaces critical-path.md)

## Goal

Graduate the open-work synthesis pattern — the tiered landscape, recommended sequence, and pending-decisions Q&A flow that produced [`work/epics/critical-path.md`](../critical-path.md) — into a reproducible kernel feature. Ship a synthesis skill that any AI assistant routing through it can produce a fresh, current critical-path-style narrative on demand, with a Q&A gate for the operator to walk through pending decisions one at a time.

## Context

`aiwf status` and `aiwf render roadmap` render *state* (what entities exist, what their statuses are). Neither renders *direction* (what to do next, in what order, with which dependencies foregrounded). That synthesis lived in conversation context during E-20 planning and would have scrolled away if not captured into `critical-path.md` as a temporary holding doc.

The synthesis pattern itself is reproducible: classify open work into tiers by leverage on future work, identify dependencies and cross-references, propose a recommended sequence, identify the immediate fork ("first decision") with options, surface pending decisions for Q&A. That template is what this epic productises.

The pattern parallels existing aiwf shapes:
- `aiwf-status` — a verb produces structured data; the skill renders narrative for the user.
- `aiwfx-plan-epic` and `aiwfx-plan-milestones` — planning conversations that either produce specs (plan-epic) or decompose them (plan-milestones).
- This epic adds a planning conversation that *orders existing work* without producing a new entity.

The proximate trigger: during E-20's planning session, the operator named the gap explicitly — *"It's very difficult to keep sequence in my head when there are so many ADRs, epics, milestones, gaps."* The synthesis happened once, manually, with significant LLM judgement. Without a feature, every future session re-derives it from scratch, often inconsistently. With a feature, the synthesis is template-driven, reproducible, and Q&A-gated.

## Scope

### In scope

- A synthesis skill (working name `aiwfx-recommend-sequence`; final name decided in milestone planning) that loads tree state via existing read verbs (`aiwf status`, `aiwf list` once E-20 ships, `aiwf show`, `aiwf history`) and produces:
  - Tiered open-work landscape (gaps, ADRs, epics, milestones), classified by leverage on future work.
  - Recommended sequence ordering Tier 1 fixes before downstream work.
  - First-decision fork — the next concrete sequencing question — with options and a lean.
  - Pending-decisions list — open questions awaiting an answer, none of which block the next concrete action.
- A Q&A gate at skill exit: *"Walk through the pending decisions one at a time, or is the recommendation enough?"*. When the operator chooses Q&A, walk one decision at a time per the project's conversational convention (CLAUDE.md *Working with the user* §Q&A format).
- Skill description densely populated with natural-language phrasings the AI would emit (*"what should I work on next?"*, *"give me the landscape"*, *"where should we focus?"*, *"what's the critical path?"*, *"synthesise the open work"*).
- Skill body covers: input shape (which verbs to call and in what order), classification rubric for tier placement, output template, Q&A flow, anti-patterns ("don't replace the operator's judgement; surface and gate").
- A test fixture: running the skill on the current `poc/aiwf-v3` tree should produce something close to the existing `critical-path.md` doc. The doc serves as the regression target.
- Retirement of `work/epics/critical-path.md` once the skill ships. The epic's wrap commits the deletion.

### Out of scope

- A kernel verb backing for the synthesis. Deferred to a follow-on epic (filed once usage shows what data the skill keeps re-deriving). Initial form is pure-skill on top of existing read verbs.
- A "what blocks what" full dependency tracker. That's a kernel-data shape (uniform `blocked_by` / `blocks` fields, cross-kind blocking) — its own epic, depends on whether the synthesis skill's tier classification ends up needing structured blocking data. See *Open questions* below.
- AI-driven re-prioritisation based on user-stated constraints (*"we only have 2 days; what changes?"*). The skill produces a template-driven recommendation; constraint-aware adjustment is a future iteration.
- Incorporating external signals (calendar, deadlines, team capacity). Read-only over the entity tree; the operator brings external context.
- Supplanting `aiwf status` or `aiwf render roadmap`. Those continue to render state; this skill renders direction. Different layers, complementary outputs.

## Constraints

- **No new kernel state.** The skill is read-only synthesis over the existing entity tree. Any persistence (a STATUS-style file? a digest?) is explicitly NOT in this epic — see *Out of scope*. On-demand re-derivation is the contract.
- **Q&A gate uses one-at-a-time framing** per CLAUDE.md *Working with the user* §Q&A format: context, options with pros/cons, lean, then the operator's choice before moving on.
- **Output template is consistent across runs** given the same tree state. Tier classification and lean involve LLM judgement (acceptable); the structural shape (sections, table columns, ordering rules) does not.
- **Synthesis is honest about its limits.** The output is framed as *"recommended sequence; questions follow"* not *"this is the order."* The skill body explicitly directs the LLM toward surfacing rather than deciding — operator decides.
- **Skill-coverage policy compliance** (per E-20/M-074). The new skill goes through the policy that lands in M-074 — proper frontmatter, name matches dir, all referenced verbs resolve.
- **No verb invention** in the skill body. The skill only invokes verbs that exist on the kernel surface today; if the synthesis would benefit from a verb that doesn't exist, file a follow-up gap rather than encoding a hand-edit workaround.

## Success criteria

- [ ] A synthesis skill ships and is materialised via `aiwf init` / `aiwf update` into the consumer's AI host.
- [ ] The skill's frontmatter description includes synthesis-shaped query phrasings (per the M-074 description-quality conventions).
- [ ] Running the skill on the `poc/aiwf-v3` tree as it stood at the time of `critical-path.md`'s authorship produces output close to that doc — same tier set, same recommended sequence, same first-decision fork. Diff is permissible (LLM judgement varies); structural agreement is required.
- [ ] The Q&A gate fires after the recommendation; operator declining to Q&A exits cleanly with a one-line summary; operator opting in walks one decision at a time.
- [ ] `work/epics/critical-path.md` is deleted in the wrap commit; the `unexpected-tree-file` warning it generated is gone.
- [ ] The skill is invocable through natural language (Claude Code description-match routing) — at least three test prompts route to it: *"what should I work on next?"*, *"give me the landscape"*, *"what's blocking what?"*. The third is aspirational — depends on the skill incorporating dependency-shape signals; see *Open questions*.
- [ ] An ADR or D-NNN captures the design choice between pure-skill (this epic) and skill+verb (the deferred follow-on), with the rationale for starting pure-skill.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Final skill name (recommend-sequence, landscape, paths, focus, …) | No | Decided in milestone planning; "critical-path" rejected as misleading PM jargon. |
| Lives in kernel (`internal/skills/embedded/aiwf-...`) or in the rituals plugin (`aiwf-extensions/aiwfx-...`)? | No | Lean: rituals plugin — it's a planning ritual, not a verb wrapper. Decision recorded as ADR or D-NNN. |
| Does the skill need structured blocking data (uniform `blocked_by` field across kinds) to do tier classification well? | No | Use prose body-mention heuristics initially. If tier classification is too ad-hoc to be reproducible, file a follow-up epic for the blocking-data kernel feature. |
| Output format: pure prose narration, structured tables (like critical-path.md uses), or both? | No | Lean: structured tables for the landscape, prose for the recommendation and lean. Matches what critical-path.md does. |
| When the skill ships, what verb produces a fresh `critical-path.md`-style artefact for archival use? Or is on-demand re-derivation enough and the doc has no archival counterpart? | No | Lean: on-demand only; persisted artefact is anti-pattern (stale within hours). |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Skill output looks authoritative but is brittle to constraint changes the skill can't see. | Medium | The Q&A gate is mandatory, not optional. Output framing emphasises *"recommended sequence, questions follow"* — the conversation does the work, not the rendering. |
| Tier classification varies between runs because LLM judgement varies. | Low | Skill body documents the classification rubric explicitly so the criteria are reproducible even if the placement is fuzzy. The first-decision and pending-decisions sections matter more than perfect tier placement. |
| Skill scope creeps to swallow adjacent functions (*"should I refactor X?"*, *"is this design good?"*, *"who's blocked on what cross-team?"*). | Medium | Scope is locked: *"synthesise the open work landscape into a recommended sequence with surfaced decisions."* Every ad-hoc extension prompts the question "should this be its own skill?". |
| The synthesis pattern is wrong in ways the test fixture (critical-path.md) doesn't catch. | Medium | The fixture is one snapshot; multiple operators using the skill on real planning sessions is the actual validation. Treat this epic as v1; iterate. |

## Milestones

<!-- Bulleted list, ordered by execution sequence. Allocated and shaped by aiwfx-plan-milestones on 2026-05-08. -->

- [M-078](M-078-planning-conversation-skills-design-adr-placement-tiering-name-rationale.md) — Design ADR: rituals-plugin placement; pure-skill-first tiering; `aiwfx-whiteboard` name with rejected alternatives · `tdd: none` · depends on: — · sizing ~0.5–1d
- [M-079](M-079-aiwfx-whiteboard-skill-classification-rubric-output-template-q-a-gate.md) — Skill scaffold + body: tier-classification rubric, output template (landscape, sequence, first decision, pending), Q&A gate, anti-patterns · `tdd: advisory` · depends on: M-078 · sizing ~2–3d
- [M-080](M-080-whiteboard-skill-fixture-validation-retire-critical-path-md-close-e-21.md) — Fixture validation against `critical-path.md`; deletion of holding doc; close E-21 · `tdd: required` · depends on: M-079 · sizing ~1d

## ADRs produced (optional)

- ADR-NNNN — Synthesis skill placement (kernel vs rituals plugin); pure-skill vs skill+verb design

## Dependencies

- **Depends on E-20.** The synthesis skill calls `aiwf list` directly (not yet shipped); also expects the skills-coverage policy from E-20/M-074 to police the new skill's frontmatter. Starting E-21 before E-20 ships would force re-work of both pieces.
- **Compatible with G-071 / G-072 fixes** (see `critical-path.md`). If those fixes land before E-21, the skill operates on a cleaner baseline; if not, the skill must explicitly call out the G-071-shaped backlog as part of "expected noise" in its tier-5 output. Either is workable.
- **Compatible with the future agent-orchestration substrate** that unfreezes E-19. If the substrate generalises into a "landscape over arbitrary work axes" pattern, this skill is an instance of that pattern.

## References

- [`work/epics/critical-path.md`](../critical-path.md) — the holding doc this epic graduates. Test fixture for the skill.
- [`docs/pocv3/design/agent-orchestration.md`](../../../docs/pocv3/design/agent-orchestration.md) — the agent-orchestration substrate that may generalise this skill's pattern.
- E-20 — Add list verb. Provides one of the read verbs this skill consumes; ships skills-coverage policy that governs this skill's frontmatter.
- `aiwf-status` skill — the pattern this epic mirrors (verb produces data, skill narrates), at the planning-direction layer rather than the state-snapshot layer.
- `aiwfx-plan-epic` / `aiwfx-plan-milestones` skills — sibling planning rituals; this skill differs in that it *orders existing work* rather than producing a new entity.
- CLAUDE.md *Working with the user* §Q&A format — the convention the Q&A gate honours.
- CLAUDE.md *Engineering principles* §"Kernel functionality must be AI-discoverable" — informs the skill description's natural-language coverage requirement.
