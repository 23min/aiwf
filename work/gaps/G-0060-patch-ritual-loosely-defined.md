---
id: G-0060
title: Patch ritual is loosely defined; no kernel-level rules for shape, scope, branch, or audit trail
status: open
---
## What's missing

"Patch" appears in the consumer-facing rituals (the optional `wf-rituals` plugin's TDD cycle / code-review / doc-lint surface), and in informal usage in this repo's history, as a unit of work that is *not* an epic and *not* a milestone — typically a small fix, a hotfix, or a focused refactor that doesn't justify the full epic-then-milestone scaffolding. The kernel says nothing about it:

- No entity kind. Patches don't appear in the closed set of six (epic, milestone, ADR, gap, decision, contract).
- No required parent. Unlike a milestone (which requires `--epic`), a patch has no structural parent.
- No FSM. There is no defined lifecycle, no terminal status, no transition rules.
- No audit-trail expectation. Commits attached to a "patch" carry no canonical trailer (no `aiwf-patch:` or equivalent), so `aiwf history` cannot project a patch's timeline the way it projects per-entity timelines.
- No branch model. A patch is whatever-shaped work on whatever-shaped branch, with no kernel-level guidance.
- No relationship to gaps, ADRs, or contracts. A patch that closes a gap has no formal way to record `closes G-NNN` in a way `aiwf check` can validate.

The result is a category of real work — small, focused, common — that aiwf has no story for. Consumers do it anyway, with whatever conventions they invent locally.

## Why it matters

The kernel's design lives or dies on its closed-set vocabulary. Six entity kinds, fixed status sets, hardcoded FSMs (CLAUDE.md design decision §1). That closure is the load-bearing thing — it's what makes every guarantee enumerable. A *seventh* shape of work that doesn't fit the closed set undermines that closure: either it gets jammed into one of the six (a "patch" pretending to be a one-AC milestone, or a gap that is really a focused refactor) and the projections drift, or it lives outside the kernel entirely and the projections are silently incomplete.

Specifically:

- **`aiwf history` cannot project patch timelines.** The verb filters on `aiwf-entity` trailers; patches have no entity id, so any commits associated with them are invisible to the timeline projection.
- **`aiwf check` has no patch-shaped invariants to enforce.** Without a defined shape, `check` cannot say anything is wrong.
- **The provenance model is silent on patches.** I2.5 introduced principal × agent × scope provenance gated through `aiwf authorize <id>`, where `<id>` is a known entity. Patches have no id; agents acting on a patch have no scope to attach.
- **The "addressed by" relation has no patch target.** A gap closed by a patch can record `addressed_by: <commit-sha>` (the existing escape hatch), but cannot record `addressed_by: P-NNN` because no such id exists. The narrative thread "this patch addressed this gap" lives only in the commit message.
- **Rituals plugin and kernel are decoupled in the wrong direction.** The rituals plugin describes patch as an engineering ritual (TDD cycle on a small fix); the kernel has no shape for it; consumers pick conventions locally; aiwf can make no guarantees.

## Resolution shape (open)

This gap captures the question, not the answer. Plausible resolutions span a wide range and the choice is genuinely open:

1. **Patches stay out of the kernel forever** — formalize this as a decision. The kernel's closed-set vocabulary is sacred; patch is purely a consumer/rituals concern. Document the boundary in CLAUDE.md so the question stops recurring. (Closes the gap as `wontfix`.)
2. **Patch becomes the seventh kind** — `P-NNN`, with its own status FSM (`open | in_progress | done | abandoned`?), no required parent, optional `closes` references. Projections, history, and audit trail extend uniformly. Largest scope.
3. **Patch maps to an authorization scope without an entity** — relax the I2.5 constraint that scopes attach to entity ids. A patch *is* a scope: a named, time-bounded unit of agent or human work, with start/end/audit but no planning-tree node. Smaller scope; reuses existing FSM machinery.
4. **Patch stays informal but gets a canonical commit-trailer** — `aiwf-patch: <slug>` with no entity backing. `aiwf history` learns to project against patch slugs. Cheapest option that still gives audit visibility, but gives up the closed-set guarantees.

The choice should land as a decision (`D-NNN`) or an ADR before any milestone work begins, because each option implies a different milestone shape downstream.

This gap is *not* `discovered_in` M-0069. It surfaced during a conversation that broadened from the M-0069 branch-model finding (G-0059) to an architectural omission this repo has been carrying for longer. Treat it as the older, deeper question of which G-0059 is one corner.
