---
id: M-0201
title: Planning-ritual body-fill via edit-body and Next-step routing
status: draft
parent: E-0048
depends_on:
    - M-0196
tdd: advisory
acs:
    - id: AC-1
      title: Planning and record-decision skills route body-fill through aiwf edit-body
      status: open
    - id: AC-2
      title: aiwfx-record-decision carries the ADR authoring no-gate-language discipline
      status: open
    - id: AC-3
      title: Planning-ritual Next-step routing is status-aware through start-epic
      status: open
---
## Goal

Route the three planning/decision-recording rituals' entity-body fills through the
trailered `aiwf edit-body` verb, make the planning Next-step routing status-aware so
it no longer leapfrogs `aiwfx-start-epic`, and give `aiwfx-record-decision` the
ADR authoring discipline it currently lacks. Each fix is pinned by a structural test
against the embedded ritual snapshot (the source of truth per ADR-0016).

The three defects catalogued in G-0300 and G-0248:

- **`aiwfx-record-decision`** (G-0300) commits the ADR/decision body with a plain
  `git commit -m "docs(adr): ..."` at step 7 — no `aiwf-verb` / `aiwf-entity` /
  `aiwf-actor` trailers, so every recorded decision trips
  `provenance-untrailered-entity-commit`. The skill never mentions `aiwf edit-body`.
  Separately, though it is the ADR authoring skill, it carries none of CLAUDE.md
  §"Authoring an ADR" discipline — nothing warns the author against writing
  gate/schedule language ("ratify after X", "status stays proposed through Y") into
  an ADR body.
- **`aiwfx-plan-epic`** (step 5) and **`aiwfx-plan-milestones`** (step 5) rewrite the
  entity body from a template but specify no trailered commit route — plan-epic
  explicitly says "the user commits when ready" without saying how, which invites the
  same untrailered `git commit`.
- **`aiwfx-plan-milestones`** `## Next step` (G-0248) points unconditionally at
  `aiwfx-start-milestone`, skipping `aiwfx-start-epic` for a still-`proposed` epic.
  The correct first-time route is `plan-epic → plan-milestones → start-epic →
  start-milestone`; the bad pointer mis-routes an operator or LLM to a skill whose
  own preflight then redirects back — a wasted hop.

No kernel-code change: this milestone edits skill prose only, plus the structural
tests that pin it. All three edited skills are rituals under
`internal/skills/embedded-rituals/**`, so the edits are subject to the M-0196
skill-edit → structural-test backstop (G-0220). `aiwfx-plan-epic` (via M-0200's
`prose_description_polish_test.go`) and `aiwfx-plan-milestones` (via its own test
file) are already referenced; `aiwfx-record-decision` gains its first referencing
structural test here, which clears the backstop for it and closes **G-0331**
(the plan-epic half of G-0331 was already cleared by M-0200 — the epic spec's
parenthetical naming a M-0200 edit to `aiwfx-record-decision` is inaccurate; M-0200
edited `aiwfx-plan-epic`, not record-decision, but the net "half already done"
holds).

## Acceptance criteria

### AC-1 — Planning and record-decision skills route body-fill through aiwf edit-body

`aiwfx-plan-epic` (step 5), `aiwfx-plan-milestones` (step 5), and
`aiwfx-record-decision` (the body-fill step) each instruct filling the entity body
via `aiwf edit-body <id> --body-file` — the canonical trailered route the
`aiwf-edit-body` skill exists for — instead of a plain `git commit`.
`aiwfx-record-decision` drops the untrailered `git commit -m "docs(adr): ..."`
body-fill route entirely, and `aiwfx-plan-epic`'s "does not commit the body fill —
the user commits when ready" bullet is reframed to the edit-body route (the body
fill lands as one trailered commit on the ritual branch; promotion and merge-to-main
stay separate). **Test:** a structural assertion that each of the three skill bodies
names `aiwf edit-body` in its body-fill step, and that `aiwfx-record-decision` no
longer carries the `git commit -m "docs(adr):` untrailered route — so a future edit
that reintroduces the plain-commit path reddens.

### AC-2 — aiwfx-record-decision carries the ADR authoring no-gate-language discipline

`aiwfx-record-decision` carries a note pointing at CLAUDE.md §"Authoring an ADR" and
warns against writing gate/schedule language into an ADR body — the "decision is
decision" discipline (no "ratify after X", no "status stays proposed through Y";
when to *act on* a decision is a planning concern, not ADR body content). **Test:** a
structural assertion that the record-decision body references the "Authoring an ADR"
discipline and the no-gate-language rule, so a future edit cannot silently drop it.

### AC-3 — Planning-ritual Next-step routing is status-aware through start-epic

`aiwfx-plan-milestones`'s `## Next step` routes to `aiwfx-start-epic E-NNNN` when the
parent epic is still `proposed` (the first-time `plan-epic → plan-milestones` flow)
and to `aiwfx-start-milestone <M-NNNN>` only when the epic is already `active`
(re-planning mid-epic). The same leapfrog in `aiwfx-plan-epic`'s `## Next step` —
which skips `start-epic` on the way to `start-milestone` — is corrected inline
(closely-related, same routing-defect class; fixed in place rather than filed as a
twin gap). **Test:** a section-scoped assertion (over `## Next step`) that
plan-milestones names both `aiwfx-start-epic` and the proposed/active condition — no
longer an unconditional `start-milestone` pointer — and that plan-epic's `## Next
step` names `aiwfx-start-epic`.
