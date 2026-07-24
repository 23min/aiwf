---
id: M-0277
title: Add the aiwf milestone tdd policy-mutation verb
status: done
parent: E-0071
tdd: required
acs:
    - id: AC-1
      title: milestone tdd sets the policy in one trailered commit
      status: met
      tdd_phase: done
    - id: AC-2
      title: policy value validated against the closed set; unknown is a usage error
      status: met
      tdd_phase: done
    - id: AC-3
      title: 'uniform-ordinary gating: any actor flips either direction without --force'
      status: met
      tdd_phase: done
    - id: AC-4
      title: flip to required refuses when a met AC lacks tdd_phase done
      status: met
      tdd_phase: done
    - id: AC-5
      title: verb is discoverable via --help, root banner, completion, and skill
      status: met
      tdd_phase: done
    - id: AC-6
      title: milestone tdd is a selectable op in the verb-sequence stress walker
      status: met
      tdd_phase: done
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

Each AC is observable behavior paired with a mechanical assertion.

### AC-1 — milestone tdd sets the policy in one trailered commit

`aiwf milestone tdd <M-id> --policy <value>` sets the milestone's `tdd:` field
and produces exactly one commit carrying `aiwf-verb` / `aiwf-entity` /
`aiwf-actor` trailers. Verb integration test driving `run([]string{...})` and
asserting the frontmatter change, the single commit, and the trailers.

### AC-2 — policy value validated against the closed set; unknown is a usage error

`--policy` is validated against the closed set `{none, advisory, required}`; an
unknown value is a usage error (exit 2) naming the allowed values and makes no
mutation. Table test over valid and invalid values.

### AC-3 — uniform-ordinary gating: any actor flips either direction without --force

Gating is uniform-ordinary: any actor — including an `ai/` actor with a
principal — may flip the policy in either direction (including the weakening
`required -> none`) with no `--force`, and `--reason` is optional; weakening and
strengthening take the identical path. Gating test with an `ai/` actor and no
`--force` (embodies D-0048).

### AC-4 — flip to required refuses when a met AC lacks tdd_phase done

A flip to `required` that would leave an already-`met` AC without
`tdd_phase: done` is refused with an actionable error naming the offending ACs,
and aborts before committing — never auto-seeding a phase. Test: a milestone
with a met + phaseless AC, flip to `required` -> error names the AC, working
tree unmutated.

### AC-5 — verb is discoverable via --help, root banner, completion, and skill

The verb is discoverable: it appears in `aiwf milestone --help` and the root
`--help` banner, `--policy` values tab-complete, and a skill covers it. Asserted
by the existing chokepoints (`completion_drift_test`, the root-banner drift
guard from G-0285, and the `skill_coverage` policy).

### AC-6 — milestone tdd is a selectable op in the verb-sequence stress walker

`aiwf milestone tdd` is a selectable operation in the `verb-sequence` stress
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

## Work log

- **AC-1 — verb + one trailered commit.** `aiwf milestone tdd <M-id> --policy
  <value>` sets the `tdd:` field and commits once with the standard trailers;
  full discoverability wiring (root banner, `--policy` completion, `aiwf-add`
  skill mention) landed alongside so the drift guards stay green. · commit
  3e1e350f
- **AC-2 — policy validation.** Unknown `--policy` values are a clean usage
  error at the verb layer; also refreshed two now-stale check hints that told
  operators to hand-edit `tdd:` frontmatter. · commit 89614704
- **AC-3 — uniform-ordinary gating.** Binary-level test: an authorized `ai/`
  actor flips either direction with no `--force`; an unauthorized one is refused
  by the standard entity-scoped provenance gate. No sovereign carve-out. ·
  commit b1555ca6
- **AC-4 — refuse-with-hint.** A `required` flip stranding a met, phaseless AC
  is refused with a hint naming the ACs, aborting before any commit and never
  seeding a phase; the guard is precise (met + phase-done flips cleanly). ·
  commit 5eb44585
- **AC-5 — discoverability.** Named pins for the subcommand, its flag shape,
  `--policy` closed-set completion, and the root-banner line. · commit 5a2472ad
- **AC-6 — stress walker.** `milestone tdd` is a milestone-only, always-legal
  walk operation; the walk stays check-clean and the list invariant holds across
  flips (ops-table, every-operation dispatch, focused step, and full-scenario
  tests, plus a 5× harness run). · commit 53b7824b
- **Coverage completeness.** ResolveActor + lock-contention cli-guard tests; the
  defensive projection-error branch is `//coverage:ignore`d (guard-preempted). ·
  commit cd6e8669

## Validation

- `make check-fast` (vet + lint + full test suite): green — every package `ok`,
  zero failures.
- `make lint` (full `golangci-lint` set): 0 issues.
- `make coverage-gate` (diff-scoped statement coverage + firing-fixture
  meta-gate + skill-edit structural backstop): pass.
- `aiwf check`: 0 error-severity findings (2 benign warnings — the active-epic
  drafted-milestone advisory and the no-upstream provenance-audit skip).
- `go run ./cmd/stresstest run --scenario verb-sequence --repeat 5`: 5/5 passed
  with the new `milestone tdd` walk operation.

## Reviewer notes

- **Independent code-quality review** (fresh-context, adversarial, `wf-review-code`
  lens): APPROVE, no blocking findings. All six ACs verified by measurement; the
  refuse-with-hint predicate confirmed a character-exact mirror of the
  `acs-tdd-audit` detection; the `//coverage:ignore` on the projection-error
  branch confirmed genuinely unreachable for a valid policy (no check rule
  escalates a `tdd:`-field change to an introduced error once the guard preempts
  the `required` case); branch coverage and serial-test discipline confirmed.
- **Design lens** (`wf-rethink`): no new design surface — the verb mirrors the
  existing `milestone depends-on` subverb shape (no new module, abstraction, or
  data model), so there is nothing to rethink.
- **Deliberately left (non-blocking, all consistent with the `depends-on`
  sibling's precedent):**
  - The refuse-with-hint guard does not skip archived milestones, whereas the
    `acs-tdd-audit` check does (`IsArchivedPath`). Flipping an archived milestone
    with a met, phaseless AC to `required` is therefore conservatively refused
    even though the resulting tree would be check-clean. The divergence only ever
    *refuses a benign flip* — it never admits bad state — and requires the extreme
    edge of mutating a terminal, archived milestone; the walker never reaches it
    (it seeds no ACs).
  - A same-value flip (e.g. `none → none`) writes and commits redundantly; there
    is no no-op suppression, matching `MilestoneDependsOn`. Correct, just not
    minimized.

## Out of scope

- The three relation-field editors and the set-at-transition amend verbs
  (G-0442) — see D-0048 and the epic's out-of-scope list.

## Dependencies

- None blocking. D-0048 (accepted); G-0284 / G-0285 / G-0286 (all addressed).

## References

- D-0048 — governing decision (verb surface, uniform-ordinary gating).
- G-0168 — originating gap.
- ADR-0006 — skills policy (verb coverage).
