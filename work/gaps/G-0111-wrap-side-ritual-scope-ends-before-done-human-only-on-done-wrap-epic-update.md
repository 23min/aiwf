---
id: G-0111
title: 'Wrap-side ritual: scope ends before done, human-only on done, wrap-epic update'
status: open
discovered_in: M-0096
---
## What's missing

E-0028 shipped the start-epic ritual proper (`aiwfx-start-epic` skill plus kernel chokepoints `epic-active-no-drafted-milestones` and the M-0095 sovereign-act refusal on `aiwf promote E-NN active`). Four wrap-side concerns were deliberately deferred from E-0028's scope or surfaced later through wrap experience:

1. **Scope-end-before-`done` behavior change.** Today `aiwf promote E-NN done` auto-ends any `aiwf authorize E-NN --to ai/<id>` scope as a side effect. The sub-decision in G-0063 (the original gap E-0028 closes) argued scope-end should happen *before* `done`, not as part of it — so the authorize scope is a deliberate close, not a status-flip artifact. Requires a verb-behavior change plus an ADR capturing the timing decision.
2. **Human-only enforcement on `aiwf promote E-NN done`.** Pairs with the start-side rule M-0095 ships. `done` is also a sovereign moment (declaring completion); a non-human actor flipping an epic to `done` should require `--force --reason` the same way activation does. Adding the `done` edge is a one-line change in `internal/entity/sovereign.go`'s `sovereignActShapes` closed set — the verb gate `requireHumanActorForSovereignAct` (in `internal/verb/promote_sovereign_act.go`) and the static CI/script audit (in `internal/policies/aiwf_promote_epic_active_audit.go`) consult that list and fire on new entries automatically per M-0130's consolidation.
3. **`aiwfx-wrap-epic` update for the new wrap-timing.** The rituals-plugin skill needs to coordinate with the verb-side timing change — running `aiwf authorize E-NN --end` *before* `aiwf promote E-NN done`, then promote. The skill body, its precondition list, and the step ordering all adjust together.
4. **Wrap doesn't close the epic's named resolution gaps.** Surfaced concretely at the E-0032 wrap (2026-05-19): the epic's wrap-side spec listed *"G-0107 status: addressed. G-0126 status: addressed"* as success criteria, the wrap promoted E-0032 itself to `done` — but **G-0107 stayed `open` until a separate session-level cleanup pass** (the matching G-0126 closure happened only because M-0119's milestone-wrap prose prompted the manual `aiwf promote G-0126 addressed --by M-0119`). The mechanism is missing on the epic surface. Two design choices for the ADR to weigh:

   - **Declarative.** A forward-pointing `addressed_by:` field in the wrapping entity's frontmatter (or a `closes:` field on the milestone/epic schema), readable by the wrap verbs as the canonical list to sweep at promote-to-done. Same shape gaps already use today on the reverse direction (`addressed_by:` on the gap points back at the resolver) — extending the field to the resolver's frontmatter mirrors the relationship cleanly.
   - **Skill-driven.** `aiwfx-wrap-epic` parses the success-criteria block for `G-NNNN status: addressed`-shape claims (or `closes G-NNNN` markers) and surfaces them as a Q&A confirmation before the final promote. Lighter-weight; depends on prose discipline at wrap-prose authoring time.

E-0028's epic spec is explicit about the first three deferrals in *Scope → Out of scope*, and the M-0096 wrap referenced this gap as "to be filed at epic wrap." Concern 4 was added 2026-05-19 from the E-0032 wrap experience.

## Why it matters

The kernel currently bundles three concerns into one verb (`aiwf promote E-NN done`): FSM transition, sovereign-act declaration, and scope-end side effect. That bundle is convenient but obscures the moment a human is closing an authorize scope versus flipping an epic's status. Untangling them — scope-end first, then the trailered sovereign promote — gives each step its own audit trail and matches the principal × agent × scope model the kernel commits to.

Without this work, the start-side and wrap-side of the epic ritual are asymmetric: starting an epic is a deliberate sovereign Q&A (`aiwfx-start-epic`), but closing it remains a single verb invocation with no preflight, no sovereign-act enforcement on the actor, an opaque scope-end side effect, and **no mechanism to land the gap-status flips the epic claims to deliver** — that last one is the same "framework correctness must not depend on LLM behavior" violation the kernel rule forbids, applied to wrap-prose recall instead of code.

## Resolution paths

- **Own epic.** This is one cohesive work unit: verb change + ADR + skill update + (now) gap-closure sweep. Recommend a fresh epic (E-NNNN) rather than a milestone under E-0028, since E-0028 is closed and the change set spans both kernel and rituals.
- **Order:** ADR first (locks the timing decision, cross-verb composition, *and* the gap-closure mechanism choice — declarative vs skill-driven), then the kernel verb edit + human-only chokepoint + the gap-closure sweep wiring, then the `aiwfx-wrap-epic` skill update. Cross-repo coupling pattern from M-0090 / M-0096 applies — author the skill body in `internal/policies/testdata/aiwfx-wrap-epic/SKILL.md` and copy at wrap.
- **Migration concern.** The behavior change is observable to any caller relying on `aiwf promote E-NN done`'s auto-end-scope. The ADR should explicitly note the backwards-compatibility shape (likely: the next aiwf release surfaces it as a behavior change in `CHANGELOG.md`, with a deprecation window if anyone is actually depending on auto-end). The gap-closure sweep is additive — no caller depends on the *absence* of an auto-flip today — so no migration concern there.
