---
id: G-0060
title: Patch ritual is loosely defined; no kernel-level rules for shape, scope, branch, or audit trail
status: open
priority: high
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

## Investigation (2026-07-22): option 2 sharpened — a decision-weight patch kind

A closer look at option 2 ("patch becomes the seventh kind"), grounded against the actual `decision`/`gap` schemas and the `wf-patch` skill, narrows it from "a new kind, shape TBD" to a specific, minimal proposal — and surfaces one already-filed gap that competes with part of it.

**The many-to-many concern is smaller than it first looks.** A patch closing more than one gap is *not* structurally blocked today: `aiwf promote G-NNNN addressed --by-commit <sha>` (`internal/verb/promote.go:374`) can run once per gap against the same merge SHA, and each gap records its own `addressed_by_commit` independently. Only the branch name and the statusline HUD (`patch/G-NNNN-<slug>`, one gap id) are 1:1 — cosmetic, not a data-model gap.

**What's actually missing is a queryable record, and `wf-patch` already half-answers it in prose.** Step 4 of the skill requires a `CHANGELOG.md` entry specifically because "a patch has no parent to roll up into: its own wrap is the only chance the change is ever recorded." That's the same problem this investigation is about, already patched with unindexed prose instead of a structured entity — no `aiwf list --kind patch`, `aiwf show P-NNN`, or `aiwf history P-NNN`.

**A minimal schema, modeled on `decision`'s actual weight** (`internal/entity/entity.go:603-613`), not `epic`'s:

```go
KindPatch: {
    Kind:            KindPatch,
    IDFormat:        "P-NNN",
    AllowedStatuses: []string{"open", "done", "abandoned"},
    RequiredFields:  commonRequired,
}
```

FSM `open → done | abandoned` (mirrors gap's `open → addressed | wontfix` shape — a patch is a unit of work in flight, not a proposal awaiting ratification the way `decision`/`ADR` are).

**No new reference field is needed on either side.** `gap.addressed_by` is already kind-unrestricted (`internal/entity/entity.go:597`, "accepts any kind — empty `AllowedKinds`"), and `promote.go` layers no separate kind restriction on `--by`. `aiwf promote G-0113 addressed --by P-0042` would work with zero schema changes to `gap`, retiring the raw-SHA `addressed_by_commit` escape hatch in favor of a real reference. `aiwf show` already computes "Referenced by (N)" backlinks generically for every kind, so `aiwf show P-0042` would list every gap it closed for free — no `closes:` field on patch required.

**Cost is closer to `gap`'s implementation weight than `epic`'s.** Against this repo's own "What's enforced and where" list: allocator + id-format regex, the per-kind schema/FSM tables above, skill-coverage policy, the completion-drift test, and the `internal/policies/` pinning suite are real new surface; archive-sweep (ADR-0004) and `aiwf history` (generic per-kind dispatch) are likely free by construction.

**The unresolved cost is at the ritual level, not the schema level.** `wf-patch`'s own text says "Does not touch planning state, milestones, or roadmaps. Patches are off-roadmap by design" — a real `patch` kind contradicts that sentence and needs two new insertion points (`aiwf add patch` at branch creation, `aiwf promote P-NNN done --by-commit <sha>` folded into tracker closure). It also forces a filing-cardinality question this kernel's other six kinds don't have: does *every* patch get a `P-NNN` (reintroducing the ceremony `wf-patch` exists to skip for the "just a typo" case), or only patches worth a structured record (a new "sometimes-filed" shape, unlike every existing kind)?

**Related, and cheaper: G-0366** (ROADMAP.md renderer is epic-only; patch-closed gaps are invisible) independently surfaces the same visibility problem for `ROADMAP.md` specifically, and proposes a generated "recent patches" section sourced from `addressed_by`/`addressed_by_commit` — no new kind. Worth doing regardless of how this gap resolves; it may also reduce how much a `patch` kind is still worth its cost once shipped.
