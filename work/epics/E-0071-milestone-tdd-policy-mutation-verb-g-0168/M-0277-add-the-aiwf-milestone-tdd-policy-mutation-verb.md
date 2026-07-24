---
id: M-0277
title: Add the aiwf milestone tdd policy-mutation verb
status: draft
parent: E-0071
tdd: required
---

# M-0277 — Add the aiwf milestone tdd policy-mutation verb

## Goal

Add `aiwf milestone tdd <M-id> --policy none|advisory|required`, the
post-creation mutator for a milestone's TDD policy — closing the `tdd:` portion
of G-0168's verb-chokepoint hole so changing the policy is a first-class,
trailered, discoverable act instead of a hand-edit.

## Context

D-0048 settled the verb surface: `tdd:` gets this verb now (uniform-ordinary
gating), the three relation-field editors are deferred, and the
set-at-transition pair went to G-0442. The prerequisites are all resolved —
G-0285 (root-banner drift guard), G-0284 (skill-coverage for namespace
subverbs), and G-0286 (an absent `tdd_phase` is legal until an AC is `met`). The
verb follows the one existing subverb precedent, `aiwf milestone depends-on`.

## Acceptance criteria

<!-- Prose shape; formalized via `aiwf add ac` at aiwfx-start-milestone.
     Each is observable behavior with a mechanical assertion. -->

1. `aiwf milestone tdd <M-id> --policy <value>` sets the milestone's `tdd:`
   field and produces exactly one commit carrying `aiwf-verb` / `aiwf-entity` /
   `aiwf-actor` trailers — verb integration test driving `run([]string{...})`
   and asserting the frontmatter change, the single commit, and the trailers.
2. `--policy` is validated against the closed set `{none, advisory, required}`;
   an unknown value is a usage error (exit 2) naming the allowed values and
   makes no mutation — table test over valid and invalid values.
3. Gating is uniform-ordinary: any actor — including an `ai/` actor with a
   principal — may flip the policy in either direction (including the weakening
   `required -> none`) with no `--force`, and `--reason` is optional; weakening
   and strengthening take the identical path — gating test with an `ai/` actor
   and no `--force` (embodies D-0048).
4. A flip to `required` that would leave an already-`met` AC without
   `tdd_phase: done` is refused with an actionable error naming the offending
   ACs, and aborts before committing — never auto-seeding a phase. Test: a
   milestone with a met + phaseless AC, flip to `required` -> error names the
   AC, working tree unmutated.
5. The verb is discoverable: it appears in `aiwf milestone --help` and the root
   `--help` banner, `--policy` values tab-complete, and a skill covers it —
   asserted by the existing chokepoints (`completion_drift_test`, the root-banner
   drift guard from G-0285, and the `skill_coverage` policy).
6. `aiwf milestone tdd` is a selectable operation in the `verb-sequence` stress
   walker — milestone-only (like `move`), classified as an always-legal simple
   step — and a walk keeps `aiwf check` clean against its baseline and the
   list-vs-ground-truth invariant intact across policy flips. Asserted by the
   walker's operation-table test (`TestWalkOperationsFor_*` naming the op) plus a
   scenario run. This covers only the uniform-ordinary legal path; the
   refuse-with-hint branch stays owned by AC-4's targeted test — the walker seeds
   no ACs, so it cannot reach a met-phaseless flip-to-`required`.

## Constraints

- Uniform-ordinary gating is non-negotiable: no directional or sovereign
  carve-out, and no new entry in the FSM-status-keyed sovereign-act tier
  (`internal/entity/sovereign.go`).
- The verb never auto-seeds an AC's `tdd_phase`; it refuses with a hint. A
  seeded `red` or `done` on an untouched or already-passed AC is false state.
- Standard kernel conventions: one commit per mutation, trailers, completion
  wiring, skill coverage per ADR-0006.

## Design notes

- Governed by D-0048 (verb surface, uniform-ordinary gating). Mirrors the
  `aiwf milestone depends-on` subverb shape.
- Verb spelling is `milestone tdd --policy <x>` (the flag form): it mirrors the
  `milestone depends-on` subverb precedent and completes a closed-set value in a
  flag rather than a bare positional.

## Surfaces touched

- The `milestone` command group (where `depends-on` is wired) — add the subverb.
- The verb layer — the verb body (mutation + validation + refuse-with-hint).
- `internal/check/acs.go` — read path for the met-phaseless detection.
- `cmd/aiwf/` root banner + completion wiring; the covering skill under the
  embedded rituals, or a `skill_coverage` allowlist entry.
- `internal/stresstest/verb_sequence.go` — add `milestone tdd` to the
  `verb-sequence` walker's operation table as a milestone-only simple step.

## Out of scope

- The three relation-field editors and the set-at-transition amend verbs
  (G-0442) — see D-0048 and the epic's out-of-scope list.

## Dependencies

- None blocking. D-0048 (accepted); G-0284 / G-0285 / G-0286 (all addressed).

## References

- D-0048 — governing decision (verb surface, uniform-ordinary gating).
- G-0168 — originating gap.
- ADR-0006 — skills policy (verb coverage).
