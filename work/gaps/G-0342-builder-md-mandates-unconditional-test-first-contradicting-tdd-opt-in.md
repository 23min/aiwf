---
id: G-0342
title: builder.md mandates unconditional test-first, contradicting tdd opt-in
status: open
---
## Problem

The builder role-agent card
(`internal/skills/embedded-rituals/plugins/aiwf-extensions/agents/builder.md`,
materialized into every consumer's `.claude/agents/builder.md`) frames the builder
as a wholesale TDD agent, stating test-first *ordering* unconditionally:

- `:3` description — "Implements aiwf milestone acceptance criteria **via TDD**".
- `:10` — "You follow TDD."
- `:15` (Responsibilities) — "Write tests first (red → green → refactor)."

None of these condition on the milestone's `tdd:` flag. The kernel makes test-first
ordering **opt-in per milestone** (`tdd: required | advisory | none`; `CLAUDE.md:48`,
commit #8; the `acs-tdd-audit` rule fires only when `tdd: required`). So a builder
working a `tdd: none` or `tdd: advisory` milestone is told by the kernel that
test-first is optional and by its own agent card that it is mandatory — a direct
contradiction between two shipped artifacts.

Root cause: the word "TDD" conflates two separable obligations —

- **(a) test-first ordering** (red → green → refactor) — opt-in per milestone.
- **(b) coverage discipline** (every reachable branch tested; every AC backed by a
  mechanical assertion before done) — unconditional (diff-scoped coverage gate,
  AC-mechanical-evidence rule, branch-coverage hard rule).

builder.md correctly states (b) unconditionally (`:64`, branch-coverage hard rule)
but wrongly states (a) unconditionally too.

## Scope (confirmed by sweep)

- **builder.md only.** The sibling role agents were checked: `deployer.md` and
  `planner.md` have no TDD/test-first mention; `reviewer.md` references only
  *branch-coverage discipline* (`:15`, `:52`) — obligation (b), correctly
  unconditional — with no test-first mandate. `reviewer.md` is the reference pattern
  for how builder.md should read.

## Direction (for the milestone that addresses this)

- Condition the ordering language on the flag, e.g.: "On `tdd: required` milestones,
  write tests first (red → green → refactor). On `tdd: advisory | none`, no mandated
  ordering — but the unconditional coverage obligation still holds: every AC backed
  by a mechanical assertion, every reachable branch tested before done."
- Soften the blanket `:10` "You follow TDD" and `:3` "via TDD" framing so the card
  reflects opt-in ordering + unconditional coverage rather than TDD-as-identity.
- Keep `:64` branch-coverage hard rule unconditional (already correct).
- Land a referencing structural test under `internal/policies/` pinning the
  conditioned wording as the AC's mechanical evidence. Note the
  `skill-edit-structural-test-backstop` policy is `SKILL.md`-scoped, so an agent-card
  edit does not trip it — the structural test is added deliberately, not by backstop.
- Re-materialize via `aiwf update` so `.claude/agents/builder.md` regenerates.

## Not in scope

- `docs/pocv3/design/healthy-codebase-principles.md:42` ("no agile / TDD …
  **dogma**") — correctly framed by its own header as a field guide disclaiming
  ideology; consistent with the opt-in flag, no change needed.
- `CLAUDE.md` (#8, `:107`) — already the precise source of truth (opt-in ordering,
  unconditional coverage); it is the reconciler, not a party to the contradiction.

Sibling of G-0341 (the shipped-guidance batching-exception gap): same class — a
shipped artifact asserting more than the kernel's actual position.
