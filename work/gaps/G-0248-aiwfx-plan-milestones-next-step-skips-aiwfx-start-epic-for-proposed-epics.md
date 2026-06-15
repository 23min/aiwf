---
id: G-0248
title: aiwfx-plan-milestones Next step skips aiwfx-start-epic for proposed epics
status: open
---
## What's missing

The `aiwfx-plan-milestones` skill's `## Next step` section points
unconditionally at `aiwfx-start-milestone <M-NNNN>`. That pointer is
only correct when the parent epic is already `active` (re-planning or
adding milestones mid-epic). In the first-time flow — `plan-epic` →
`plan-milestones`, where the epic is still `proposed` — the correct
next step is `aiwfx-start-epic`, which performs the sovereign
`proposed → active` promote and cuts the `epic/E-NNNN-<slug>` branch
that milestone branches fork from. The skill jumps over `start-epic`
entirely. The same text ships in the embedded ritual snapshot at
`internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-plan-milestones/SKILL.md`,
so this is not a materialization artifact — the authoring fix lands
there and `aiwf update` refreshes the materialized copy.

## Why it matters

The three skills' own cross-references establish the intended order as
`plan-epic` → `plan-milestones` → `start-epic` → `start-milestone`:
`aiwfx-start-milestone`'s preconditions require the parent epic to be
`active` with its `epic/E-NNNN-<slug>` branch existing locally and
checked out ("If the parent epic isn't active or its branch doesn't
exist locally, use `aiwfx-start-epic E-NNNN` first"), and
`aiwfx-start-epic`'s preconditions require the epic to be `proposed`
with at least one `draft` milestone — i.e. it is designed to run
*after* `plan-milestones`. The bad pointer mis-routes an operator or an
LLM driving the flow to the wrong skill. Severity is Low–Medium:
following it lands in `aiwfx-start-milestone`, whose step-1 preflight
detects the non-active epic / missing branch and redirects to
`aiwfx-start-epic`, so the worst case is a wasted hop plus confusion,
not a corrupted tree. The fix makes the `## Next step` status-aware
(run `aiwfx-start-epic E-NNNN` when the epic is still `proposed`;
`aiwfx-start-milestone <M-NNNN>` only when it is already `active`) and
is best pinned by a structural policy assertion scoped to the
`## Next step` section so a future edit cannot silently revert it.
