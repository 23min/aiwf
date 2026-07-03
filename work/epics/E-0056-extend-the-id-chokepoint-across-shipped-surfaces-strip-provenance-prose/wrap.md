# Epic wrap — E-0056

**Date:** 2026-07-03
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0056-extend-the-id-chokepoint-across-shipped-surfaces-strip-provenance-prose
**Merge commit:** a54000ce

## Milestones delivered

- M-0227 — Extend the id chokepoint to all shipped surfaces; clean id leaks (merged 6ea3db6e)
- M-0228 — Strip shipped-prose history/rationale; broaden the authoring principle (merged 6632c82c)
- M-0229 — Drop dead doc-links; encode reference discipline in record-decision (merged 9ee3ac33)

## Summary

E-0056 closed the coverage gaps in the `skill-body-id` chokepoint and swept the
prose that violated the consumer-scoping rule, so every surface aiwf materializes
into a consumer's `.claude/` — verb and ritual skills (bodies *and* `description:`
frontmatter), role-agent cards, entity templates, the always-on guidance fragment,
and the statusline comments — reads as imperative, consumer-scoped instruction
with no aiwf-internal ids, development history, rationale, or dead references.
M-0227 extended the check from `SKILL.md` bodies to the full shipped-surface set
(frontmatter, non-`SKILL.md` files, `embedded-guidance/`, `embedded-statusline/`)
and cleaned the id leaks it then caught. M-0228 stripped the residual
history/rationale prose and broadened the authoring principle in `CLAUDE.md`.
M-0229 removed the dead `docs/`/`internal/` doc-links across every shipped skill,
added a universal link guard (external-URL-or-anchor only), and encoded the
reference discipline in `aiwfx-record-decision`. The epic seed G-0348 and the
dead-links gap G-0315 are resolved by this work (promoted to `addressed`).

## ADRs ratified

- none — the doctrine landed as `CLAUDE.md` guidance updates (the broadened
  authoring principle) and skill guidance (the reference-discipline rule), not
  new architectural decisions.

## Decisions captured

- None as standalone D-NNN. Mid-flight choices were lightweight operator scope
  calls recorded in each milestone's `## Decisions made during implementation`
  section: M-0229's scope-widening to all shipped skills and the review-time
  prose fold-in; M-0228's statusline-bullet drop and the Nygard-blockquote trim.

## Follow-ups carried forward

- G-0346 — Wrap rituals merge onto mainline without reconciling a diverged trunk
  first (open). The reconcile-first practice it prescribes was applied by hand in
  this wrap; shipping it into the `aiwfx-wrap-epic` / `aiwfx-wrap-milestone` /
  `wf-patch` bodies is the durable form.

## Handoff

The id chokepoint now covers every shipped surface, so the leak class cannot
recur — the check is inert in a consumer repo (it scans only the authoring-source
trees under `internal/skills/`). G-0224 (stale `branch-not-found` reference in the
start rituals) is deliberately out of scope, owned by lifecycle epic E-0049. The
next hygiene item is shipping the reconcile-first wrap practice (G-0346).
