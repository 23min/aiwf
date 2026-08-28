---
id: G-0646
title: Closing an epic is not a sovereign act, and the gate reaches only promote
status: open
priority: medium
---
## What's missing

Promoting an epic to `done` accepts a non-human actor. `sovereignActShapes` in `internal/entity/sovereign.go` holds exactly one entry — epic `proposed → active` — so `requireHumanActorForSovereignAct` in `internal/verb/promote_sovereign_act.go` returns nil for the `active → done` edge, and nothing downstream gates it.

Measured in a throwaway fixture repo, against its only epic, `active` with a live `aiwf authorize` scope delegated to `ai/claude`: `aiwf promote <epic> done --actor ai/claude --principal human/fixture` exited 0 and flipped the epic to `done`. The resulting commit carries `aiwf-actor: ai/claude` and no `aiwf-force` trailer. `aiwf check` on that tree reported 0 errors. Expected a refusal naming a required `human/` actor — what the same actor gets on the `proposed → active` edge.

One instance already sits in this repo's history: commit `c030cb926`, `aiwf promote E-0029 active -> done`, `aiwf-actor: ai/claude`, no force trailer. It is the only one across all refs.

The gate also reaches only one verb. `requireHumanActorForSovereignAct` has a single call site, in `internal/verb/promote.go`, so `aiwf cancel` never consults the closed set. `active → cancelled` is the other terminal edge out of the same state, and the same kind of declaration by a principal — yet it cannot be covered by adding a second entry. With no call site in `cancel`, that entry would go unenforced at verb time while `forcedUntraileredFindings` still fired on the landed commit: the cancel would exit 0 and fail the next push. That is refusal after the act, the shape ADR-0040 closed for the force route.

## Why it matters

Activating an epic is gated because it commits a human to the work. Declaring that work finished is the same kind of claim pointed the other way, and it is the one that ends the delegation scope, retires the entity from the active tree, and stands afterwards as the record that the epic's success criteria were met. An autonomous agent can make that claim today and no surface objects, at the verb or afterwards.

The cost is to the guarantee more than to any single tree. The kernel states plainly that sovereign moments require a human; that statement is true of one transition and false of both edges that close an epic, so a reader who trusts it will be wrong about which acts an agent can perform unassisted — and being wrong in that direction is exactly what the rule exists to prevent.
