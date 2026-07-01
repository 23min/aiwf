---
id: G-0331
title: aiwfx-plan-epic and aiwfx-record-decision lack structural-test backstop
status: open
discovered_in: M-0197
---
## What's missing

M-0195 (`c67a8457`, the skill-body-id sweep) edited two ritual skill bodies —
`internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-plan-epic/SKILL.md`
and `.../aiwfx-record-decision/SKILL.md` — before M-0196 landed the
skill-edit → structural-test backstop policy (G-0220). Neither skill's path is
referenced by any structural test under `internal/policies/`, so both are
flagged by `PolicySkillEditStructuralTestBackstop` when the coverage-gate's
base includes M-0195's edits.

They are 2 of the ~10 unbacked ritual skills M-0196's reviewer notes named
("the next edit to such a skill must add a structural test; never
retroactively"). Because M-0195 predates the backstop, no test was added at
sweep time.

## Why it matters

The backstop is diff-scoped. On CI-per-push to the epic branch the base is the
previous tip, so M-0195's already-landed edits are not re-flagged (matching the
"never retroactively" intent). But:

- The local `make coverage-gate` (base = merge-base with `origin/main`) sweeps
  in M-0195's edits and fails today — surfaced during M-0197's self-review.
- The E-0048 → `main` merge push (base = old `main` tip) brings the whole
  epic's diff against `main`, so the backstop will fire and block the epic
  wrap.

So the epic cannot merge to `main` cleanly until both skills gain a
referencing structural test.

## Proposed fix

Add a structural test under `internal/policies/` for each skill — a
heading-walk assertion pinning some prescribed content (the
`aiwfx_wrap_epic_test.go` template) — so each skill's path is referenced and
the backstop clears.

Sequencing: M-0201 (planning-ritual body-fill, G-0300) is expected to edit
`aiwfx-plan-epic` and would then be forced by the ratchet to add its test —
but M-0201's scope is not yet written, and `aiwfx-record-decision` is not
clearly in any planned milestone's scope. The receiving milestone is recorded
when this gap is picked up.

Discovered during M-0197 (its coverage-gate run surfaced the two).
