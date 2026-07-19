---
id: M-0267
title: Relax acs-shape/tdd-phase to allow absent phase until AC met
status: done
parent: E-0068
tdd: required
acs:
    - id: AC-1
      title: 'Absent tdd_phase is legal on a non-met AC under tdd: required'
      status: met
      tdd_phase: done
    - id: AC-2
      title: 'Regression: tdd-phase closed-set and met-requires-done checks unchanged'
      status: met
      tdd_phase: done
---

# M-0267 — Relax acs-shape/tdd-phase to allow absent phase until AC met

## Goal

Stop `acs-shape/tdd-phase` from forcing every AC in a `tdd: required` milestone to carry a `tdd_phase` from the moment it's created — an absent phase should be legal until the AC reaches `status: met`, matching what the design actually commits to.

## Context

`internal/check/acs.go`'s `acsShape` function (~line 155) currently fires `CodeACsShape`/`tdd-phase` whenever `ac.TDDPhase == "" && tddRequired`, regardless of the AC's own status. That's stricter than CLAUDE.md #8 and `design-decisions.md`, which commit only to "`AC met` requires `tdd_phase: done`" when the milestone is `tdd: required` — not "every AC is phase-tracked from creation." The practical effect (G-0286): strengthening a milestone `advisory → required` reddens the tree for every pre-existing AC, and the only way to clear it today is to seed untouched ACs to `red`, which misrepresents them (`red` means "a failing test exists").

## Acceptance criteria

### AC-1 — Absent tdd_phase is legal on a non-met AC under tdd: required

Under a `tdd: required` milestone, an AC with `status` anything other than `met` and an absent `tdd_phase` produces no `acs-shape/tdd-phase` finding. This is the core relaxation: `tdd_phase` becomes optional until the AC is claimed `met`, not mandatory from creation. Covers `status: open` explicitly (the case G-0286 traces — a milestone freshly upgraded `advisory → required` with pre-existing open ACs that have never had a phase set).

### AC-2 — Regression: tdd-phase closed-set and met-requires-done checks unchanged

Two existing behaviors must survive the relaxation untouched: (1) a *present* `tdd_phase` value outside the closed set (`red|green|refactor|done`) still fires `acs-shape/tdd-phase` exactly as today; (2) `acsTDDAudit`'s rule — an AC with `status: met` under `tdd: required`/`advisory` still requires `tdd_phase: done`, firing at error/warning severity respectively when it doesn't — is completely unaffected by this milestone's change. This AC exists because the relaxation touches the same conditional block as the closed-set check, and the audit rule (in a different function, `acsTDDAudit`) is the property this milestone must not weaken.

## Constraints

- The `acsTDDAudit` function itself is out of scope for this milestone — no code changes there, only regression coverage confirming it still fires as before.
- No change to `entity.IsAllowedTDDPhase` or the phase enum (`red|green|refactor|done`) — this milestone doesn't add a "not started" phase value; absence itself now means "not started."

## Design notes

- Per G-0286's fork: the "strict reading" (every AC is phase-tracked from creation) is explicitly rejected in favor of the "committed reading" (every AC reaches `done` before `met`) — see G-0286's own body for the full argument. No new decision entity was needed for this milestone; the gap itself is the settled design.

## Surfaces touched

- `internal/check/acs.go` — `acsShape` (the `tdd-phase` subcode's conditional).
- `internal/check/hint.go` — the `acs-shape/tdd-phase` operator hint, coupled to the finding whose semantics changed.

## Out of scope

- G-0168's set-policy verb (auto-seed-vs-refuse on `advisory → required`) — this milestone only fixes the check-layer rule; per G-0286, relaxing this rule means that verb no longer needs an auto-seed decision at all, but building the verb itself is tracked separately in G-0168.
- Any change to `wf-tdd-cycle` or other ritual-content guidance on when to set `tdd_phase`.

## Dependencies

- None. Independent of M-0268 and of D-0039 (M-0267 predates neither).

## References

- [G-0286](../../gaps/G-0286-acs-shape-tdd-phase-over-demands-a-phase-on-every-ac-under-tdd-required.md) — source gap, fully specifies the fix and the design fork's resolution.
- CLAUDE.md §"What aiwf commits to", item 8 — the committed reading this milestone brings the check in line with.

---

## Work log

### AC-1 — Absent tdd_phase is legal on a non-met AC

Dropped `acsShape`'s presence requirement, keeping only the closed-set validity check · commit 88a32e3c · tests 4/4 new (plus 1 regression on `acsTDDAudit`'s previously-untested absent-phase branch).

Branch-coverage audit: the single compound condition's three reachable combinations (absent → no finding, present+valid → no finding, present+invalid → finding) are each hit by an existing or new test. Vacuity audit (`wf-vacuity`): 3 mutations attempted (flip `!=`/`==`, drop the closed-set conjunct, invert it), all killed; no weak or tautological assertions found.

### AC-2 — Regression: tdd-phase closed-set and met-requires-done checks unchanged

Both claims were already correctly implemented; AC-1's own cycle had already locked the "present-but-invalid still errors" and "required+met+absent still errors" halves. The one untested cell was tdd: advisory + met + absent phase (only advisory+wrong-value was covered) · commit e95e5b94 · tests 1/1 new, test-only diff, no production change.

Vacuity audit (`wf-vacuity`): mutated `acsTDDAudit`'s severity switch (advisory → error instead of warning); both advisory tests went red as expected. No weak or tautological assertions found.

## Decisions made during implementation

- None — all decisions are pre-locked above (G-0286's own body already settles the design fork).

## Validation

`go build ./...` — clean. `go vet ./...` — clean. `go test ./...` (full tree) — all packages pass, no failures. `make lint` (`golangci-lint run`) — 0 issues. `aiwf check` — 0 errors (5 pre-existing warnings unrelated to this milestone: archive-sweep-pending and terminal-entity-not-archived on D-0005, plus a provenance-scope-undefined note from the worktree having no upstream configured).

## Deferrals

- (none)

## Reviewer notes

Independent two-lens review (dispatched fresh-context, no authorship attachment):

- **Code-quality** (`wf-review-code`): **APPROVE**. Verified by measurement, not by trusting this spec — reran build/vet/test/lint independently, confirmed via the diff itself (not the spec's prose) that `acsTDDAudit` is untouched, confirmed all three branches of the relaxed conditional are covered by name-checked passing tests. One non-blocking nit: this spec's `## Surfaces touched` originally omitted `internal/check/hint.go` — fixed in place.
- **Design-quality** (`wf-rethink`): no rethink exercise run, by design — independently confirmed the change introduces no new package boundary, abstraction, or data model (it removes one arm of an existing conditional; the "met requires done" invariant was already owned by a separate, untouched function). Applying the trigger correctly means recognizing when nothing needs rethinking, not manufacturing an exercise.

`wf-doc-lint` (scoped to this milestone's changeset): 2 non-blocking findings, neither introduced by this milestone's own file changes — both are pre-existing docs describing the old behavior this milestone changed. `docs/pocv3/plans/acs-and-tdd-plan.md:206` now states a stale requirement ("tdd_phase is required when milestone tdd: required"); `docs/initiatives/tdd-cycle-subagent-boundaries.md:112` describes G-0286 in present tense, which will read as historical once G-0286 is `addressed`. Left as follow-up, not fixed here — out of this milestone's scope (design docs, not the shipped kernel behavior or its operator-facing hints).

- (none)
