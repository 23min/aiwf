---
id: M-0201
title: Planning-ritual body-fill via edit-body and Next-step routing
status: in_progress
parent: E-0048
depends_on:
    - M-0196
tdd: advisory
acs:
    - id: AC-1
      title: Planning and record-decision skills route body-fill through aiwf edit-body
      status: met
    - id: AC-2
      title: aiwfx-record-decision carries the ADR authoring no-gate-language discipline
      status: met
    - id: AC-3
      title: Planning-ritual Next-step routing is status-aware through start-epic
      status: met
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

## Work log

### AC-1 — body-fill routes through aiwf edit-body

`aiwfx-plan-epic` step 5 and `aiwfx-plan-milestones` step 5 gained an `aiwf edit-body`
route for the template body fill; `aiwfx-record-decision` step 7 was reframed from a
plain `git commit -m "docs(adr): ..."` to `aiwf edit-body` (bless mode), dropping the
untrailered route; plan-epic's "does not commit the body fill" bullet was reframed to
"does not merge to main." · commit f81a140d

### AC-2 — ADR authoring discipline

`aiwfx-record-decision` step 3 gained an "ADR authoring discipline" note citing
CLAUDE.md §"Authoring an ADR" — *decision is decision*, no gate/schedule language in
the ADR body. · commit f81a140d

### AC-3 — status-aware Next-step routing

`aiwfx-plan-milestones` `## Next step` became a two-case, status-aware fork
(`aiwfx-start-epic` for a `proposed` epic; `aiwfx-start-milestone` for an `active`
one); the same leapfrog in `aiwfx-plan-epic`'s `## Next step` was corrected inline. ·
commit f81a140d

Review-response hardening (branch-coverage self-sufficiency for `findNumberedStep`,
a symmetric AC-3 assertion pinning both halves of G-0248's contract, and an
`aiwf edit-body --body-file` footgun note) landed as commit 72f38842.

## Decisions made during implementation

No ADR/decision entity was warranted. One empirically-verified fact shaped the
`aiwfx-record-decision` guidance: `aiwf edit-body` **bless mode refuses** a working
copy with frontmatter changes, but **`--body-file` mode preserves** the working-copy
frontmatter (e.g. a `supersedes:` cross-reference) while taking the body from the
draft file. So the frontmatter-cross-ref path routes through `--body-file`. Verified
against the real verb and reproduced independently by the reviewer; mechanism in
`internal/verb/editbody.go`.

## Validation

- `go test ./internal/policies/ -count=1` — green (the four M-0201 AC/helper tests
  plus the full package).
- `make check-fast` — exit 0 (full lint + vet + unit suite).
- `aiwf check` via a branch-accurate diag binary (built from this branch, including
  the G-0299 `skill-body-id` rule) — **0 errors**; no `skill-body-id` finding on the
  edited skills; the skill-edit backstop is satisfied for all three edited skills.
- Empirical verification of the `--body-file` frontmatter-preservation claim against
  the real verb (bless refuses; `--body-file` retains `supersedes:` in a trailered
  commit).

## Reviewer notes

Independent fresh-context review (code-quality lens; the change introduced no new
module/abstraction/data model, so `wf-rethink` had no design surface) returned
**APPROVE**. It verified all four AC tests redden on pre-fix content by revert-and-test,
confirmed the plan-milestones AC-1 assertion is step-5-scoped and therefore
non-vacuous (old content named `aiwf edit-body` only in step 6, and the step-5-scoped
test failed on it), independently reproduced the `--body-file` frontmatter claim
against the verb, and confirmed skill coherence, `skill-body-id` cleanliness, backstop
satisfaction, and full G-0300 / G-0248 / G-0331 scope. Its three non-blocking findings
were all addressed inline in commit 72f38842: (1) the `findNumberedStep`
branch-coverage test was made self-sufficient for the second-loop fence-skip branch;
(2) the AC-3 plan-milestones assertion now pins both halves of G-0248's two-case
contract; (3) a `--body-file` footgun note (the draft file supplies the body, not the
in-place step-3 edit).

## Deferrals

None. G-0300, G-0248, and G-0331 are fully addressed. Per this epic's convention,
the source gaps stay `open` until the E-0048 wrap sweep. G-0331 is closed as a
consequence — `aiwfx-record-decision` now carries a referencing structural test
(`internal/policies/aiwfx_record_decision_test.go`), clearing the M-0196 skill-edit
backstop for it; the plan-epic half was already cleared by M-0200. Verified at
implementation time (per G-0300's scope note) that no pre-existing gap already
covered the edit-body routing.
