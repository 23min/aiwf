---
id: G-0111
title: 'Wrap-side ritual: scope ends before done, human-only on done, wrap-epic update'
status: open
discovered_in: M-0096
---

## What's missing

E-0028 shipped the start-epic ritual proper (`aiwfx-start-epic` skill plus kernel chokepoints `epic-active-no-drafted-milestones` and the M-0095 sovereign-act refusal on `aiwf promote E-NN active`). Three wrap-side concerns were deliberately deferred from E-0028's scope:

1. **Scope-end-before-`done` behavior change.** Today `aiwf promote E-NN done` auto-ends any `aiwf authorize E-NN --to ai/<id>` scope as a side effect. The sub-decision in G-0063 (the original gap E-0028 closes) argued scope-end should happen *before* `done`, not as part of it — so the authorize scope is a deliberate close, not a status-flip artifact. Requires a verb-behavior change plus an ADR capturing the timing decision.
2. **Human-only enforcement on `aiwf promote E-NN done`.** Pairs with the start-side rule M-0095 ships. `done` is also a sovereign moment (declaring completion); a non-human actor flipping an epic to `done` should require `--force --reason` the same way activation does. The kernel rule lives in `internal/verb/promote_sovereign_epic_active.go` today; a sibling helper or extension covers the `done` edge.
3. **`aiwfx-wrap-epic` update for the new wrap-timing.** The rituals-plugin skill needs to coordinate with the verb-side timing change — running `aiwf authorize E-NN --end` *before* `aiwf promote E-NN done`, then promote. The skill body, its precondition list, and the step ordering all adjust together.

E-0028's epic spec is explicit about this deferral in *Scope → Out of scope*, and the M-0096 wrap referenced this gap as "to be filed at epic wrap."

## Why it matters

The kernel currently bundles three concerns into one verb (`aiwf promote E-NN done`): FSM transition, sovereign-act declaration, and scope-end side effect. That bundle is convenient but obscures the moment a human is closing an authorize scope versus flipping an epic's status. Untangling them — scope-end first, then the trailered sovereign promote — gives each step its own audit trail and matches the principal × agent × scope model the kernel commits to.

Without this work, the start-side and wrap-side of the epic ritual are asymmetric: starting an epic is a deliberate sovereign Q&A (`aiwfx-start-epic`), but closing it remains a single verb invocation with no preflight, no sovereign-act enforcement on the actor, and an opaque scope-end side effect.

## Resolution paths

- **Own epic.** This is one cohesive work unit: verb change + ADR + skill update. Recommend a fresh epic (E-NNNN) rather than a milestone under E-0028, since E-0028 is closed and the change set spans both kernel and rituals.
- **Order:** ADR first (locks the timing decision and the cross-verb composition), then the kernel verb edit + human-only chokepoint, then the `aiwfx-wrap-epic` skill update. Cross-repo coupling pattern from M-0090 / M-0096 applies — author the skill body in `internal/policies/testdata/aiwfx-wrap-epic/SKILL.md` and copy at wrap.
- **Migration concern.** The behavior change is observable to any caller relying on `aiwf promote E-NN done`'s auto-end-scope. The ADR should explicitly note the backwards-compatibility shape (likely: the next aiwf release surfaces it as a behavior change in `CHANGELOG.md`, with a deprecation window if anyone is actually depending on auto-end).
