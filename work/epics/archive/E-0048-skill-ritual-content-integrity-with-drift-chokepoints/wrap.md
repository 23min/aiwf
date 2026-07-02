# Epic wrap — E-0048

**Date:** 2026-07-02
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0048-skill-ritual-content-integrity-with-drift-chokepoints
**Merge commit:** 83d891cd

## Milestones delivered

- M-0195 — strict skill-body id-reference discipline + full body sweep + placeholder normalization (G-0299)
- M-0196 — skill-edit→structural-test backstop policy (G-0220)
- M-0197 — finding-code docs + documented-superset chokepoint (G-0283)
- M-0198 — verb-skill factual corrections (G-0301)
- M-0199 — wf-tdd-cycle / wf-review-code honesty + wf-doc-lint reframe + audit/vacuity-before-`met` reorder (G-0297, G-0294, G-0309)
- M-0200 — skill descriptions + whiteboard + prose polish (G-0298)
- M-0201 — planning-ritual body-fill routing + plan-milestones Next-step (G-0300, G-0248, G-0331)
- M-0202 — devcontainer onboarding banner + drift chokepoint (G-0279)
- M-0210 — drift chokepoint for the trailered-commit block in the wrap rituals (reframed from ADR-0024; the reference-skill extraction was rejected)
- M-0211 — migrate consumer-operating guidance from CLAUDE.md to the shippable embedded-guidance source + drift chokepoint (G-0313)

## Summary

E-0048 made every shipped skill / ritual / guidance body accurate and
self-contained, and added mechanical chokepoints so a future edit can't silently
reintroduce drift. It landed the three planned chokepoints — the skill-body
id-reference check (pre-push `aiwf check`), the finding-code documented-superset
policy, and the skill-edit→structural-test backstop — plus two that emerged
mid-flight: the trailered-commit drift chokepoint (M-0210) and the guidance
operating-anchor chokepoint (M-0211). Honest scope shift: M-0210 was reframed
from ADR-0024's proposed `wf-commit-trailers` reference-skill extraction to a
chokepoint that keeps the block inline and polices it — ADR-0024 was rejected.
No shipped skill body cites a real entity id in prose; the id-reference check is
the mechanical guarantee.

## ADRs ratified

- None. ADR-0024 (shared ritual content lives as a referenced reference skill)
  was **rejected** during M-0210 — the reframe to a drift chokepoint superseded
  the extraction; rationale is recorded in the ADR's rejection reason and the
  M-0210 spec.

## Decisions captured

- M-0210 chokepoint-over-extraction (ADR-0024 rejected) — in ADR-0024's rejection
  reason + the M-0210 spec.
- M-0211 "audience, not importance" dividing principle (consumer-operating
  guidance ships via the embedded source + on-demand skills; repo-development
  guidance stays in `CLAUDE.md`) — in the new `CLAUDE.md` §"Consumer-operating
  guidance vs repo-development guidance" authoring rule + the M-0211 spec.
- Skill-body-id check location (`internal/check`, pre-push) — resolved in the
  epic's Open-questions during M-0195.
- No new ADR/D-NNN was warranted at wrap: every decision above is durably
  captured in an existing home (ADR-0024, the milestone specs, the `CLAUDE.md`
  authoring rule).

## Follow-ups carried forward

- G-0219 (`aiwfx-wrap-milestone` SKILL.md asymmetric — missing wrap-milestone
  trailer step) — related to M-0210's drift class but was not a declared E-0048
  source gap; left open for separate assessment of whether M-0210's chokepoint
  fully addresses it.

## Doc findings

Doc-lint / cross-reference coherence was covered by the scoped epic-level review
(folded into this wrap per the operator's request). Verdict: APPROVE for merge,
no correctness defect. It found two documentation-honesty defects, both fixed
before the mainline merge: (1) three of the five new chokepoints were missing
from `CLAUDE.md`'s "What's enforced and where" registry — rows added; (2)
`epic.md`'s scope still described M-0210 as the ADR-0024 extraction — reconciled.
Residual (non-blocking, accepted): a few real entity ids remain inside code
fences in verb skills (`aiwf-show`, `aiwf-acknowledge`, `aiwf-edit-body`) — the
skill-body-id check's documented prose-scope exempts fenced code, so these are
authoring discipline, not check-catchable.

## Handoff

E-0048 is closed. The sibling epics E-0050 (gate-discipline model) and E-0049
(commit/TDD model) carry the deferred subsets. The five chokepoints are now the
standing mechanical guarantee that shipped skill / ritual / guidance content
stays correct over time.
