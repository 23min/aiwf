---
id: M-0230
title: Make roadmap regeneration zero-friction in the wrap rituals
status: draft
parent: E-0049
tdd: advisory
---
## Goal

Make ROADMAP.md regeneration add **zero friction** to the ritual workflow — no gate, no
step the operator has to reason about — while keeping the committed roadmap in sync.
Addresses G-0350: `aiwf render roadmap --write` emits its own commit, but the wrap /
release ritual bodies describe the regen as ungated "housekeeping."

## Context

See G-0350 for the finding and root cause: the verb couples generate + commit and does a
`git stash` dance that refuses to run on a dirty tree. The guiding constraint is
**zero-friction** — the regen should be streamlined *away*, not enumerated as another
gated step. Directions the drafter should weigh (G-0350 lists them; none prescribed):
auto-regenerate on the state-changing mutations, fold the ROADMAP.md refresh into an
already-gated wrap commit, or decouple the verb into write-only so the caller's existing
gate carries the commit.

## Scope

To be scoped by the drafter — acceptance criteria are intentionally deferred; this
milestone is a scaffold capturing the work under E-0049. Likely surfaces: the
`render-roadmap` verb implementation, the wrap/release ritual bodies under
`internal/skills/embedded-rituals/`, and — if the auto-regen route is chosen — the
post-commit hook materialized by `aiwf init` / `aiwf update`. Any ritual-body edit must
land with a referencing structural test under `internal/policies/` (the
skill-edit-structural-test backstop), and the position chosen must be measured against
the zero-friction bar: the operator never sees a roadmap gate.
