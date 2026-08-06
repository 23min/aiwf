---
id: M-0293
title: Correct the surfaces that claim force enforcement
status: done
parent: E-0079
depends_on:
    - M-0291
tdd: advisory
acs:
    - id: AC-1
      title: Every surface in the table states the seam that actually refuses
      status: met
      tdd_phase: done
    - id: AC-2
      title: Every force finding hint names what the override does not relax
      status: met
      tdd_phase: done
    - id: AC-3
      title: The sovereign policy asserts a live code reference or is retired
      status: met
      tdd_phase: done
---

## Goal

Make every surface that describes force enforcement name the seam that actually
refuses, once M-0291 has made the guarantee true.

## Context

Ten surfaces assert the guarantee. Some are simply false today; others name a
chokepoint that was never built, and two code comments assert opposite stances
about where the enforcement lives. Correcting them requires M-0291 first — the
correct text names a seam that does not yet exist, so writing it earlier would
replace one false claim with another.

## Acceptance criteria

### AC-1 — Every surface in the table states the seam that actually refuses

| # | Surface | What it asserts today |
| --- | --- | --- |
| 1 | `CLAUDE.md`, engineering principles | `--force` is sovereign, humans only |
| 2 | `CLAUDE.md`, kernel commitments, provenance item | `--force` requires a human actor |
| 3 | `docs/design/design-decisions.md`, principal/agent provenance | `--force` is human-only |
| 4 | `docs/design/design-decisions.md`, the human-only bullet | the kernel refuses `--force` from any non-`human/` role |
| 5 | `docs/design/legal-workflows-audit.md`, R-RULE-076 | chokepoint column names a cliutil guard and a sovereign policy file |
| 6 | `docs/design/legal-workflows-audit.md`, R-AUDIT-0105 | module column names the cliutil guard; mechanism hard-reject |
| 7 | `docs/design/legal-workflows-audit.md`, R-AUDIT-0113 | `aiwf promote E-NNNN active` hard-rejects a non-human actor |
| 8 | `internal/cli/promote/promote.go`, the `--force` flag help | coherence checks still run |
| 9 | `internal/verb/promote_sovereign_act.go`, the helper comment | the coherence chokepoint makes the override human-only by construction |
| 10 | `internal/verb/add.go`, the force branch comment | the check-time audit is the backstop every force path relies on, rather than a verb-time gate |
| 11 | [ADR-0029](../../../docs/adr/ADR-0029-verb-shape-correctness-comes-from-pre-write-projection.md), the pre-write-projection rationale | `verb.Apply` performs no validation and adding a check inside it would duplicate the real gate |
| 12 | `internal/entity/sovereign.go`, the `SovereignActShape` doc comment | the sovereign gesture is a `human/` actor by default, or `--force --reason` from a non-human actor |
| 13 | `internal/policies/aiwf_promote_epic_active_audit_test.go`, the policy's failure message | tells the reader to append `--force --reason` or have a human run the verb |

Rows 12 and 13 are the same shape as each other: both offer `--force` by a
non-human actor as a working remedy, which M-0291 made false. The runtime
message that said the same thing was corrected there, because it is reachable
only for the population the advice is wrong for; these two are prose and a test
message, so they wait here.

Row 11 is the mirror of the rest: a surface asserting the *absence* of a
guarantee that now exists. Its scope was content and shape validation, so it
does not literally contradict the coherence guard — but its stated reasoning
transfers, and a reader meeting it beside ADR-0040 gets opposite guidance on
whether `verb.Apply` may refuse anything. It was already inaccurate before this
epic, since three other guards live there. The correction names what `Apply`
does and does not validate, rather than removing the rule.

Rows 9 and 10 contradict each other inside one package, and row 9 is a reasoned
handoff to a guard that its own verb never calls — the sovereign-act gate steps
aside for `--force` precisely where nothing else was checking.

An eleventh site, `CLAUDE.md`'s gate-discipline bullet, calls `--force`
"additionally human-only". Its subject is how the human batches approvals rather
than what the kernel enforces, so it is reviewed and left alone unless the wording
reads as an enforcement claim on a second pass.

Evidence: structural assertions scoped to the named section or table row. A grep
for a literal proves it exists somewhere, not in the right place — and several of
these phrases are short and generic enough to match elsewhere in the same file.

### AC-2 — Every force finding hint names what the override does not relax

The finding-hint half folded in from G-0333. A hint that offers `--force` as the
remedy must say what the override relaxes and what it leaves standing, so an
operator cannot read it as a general escape from the finding it is attached to.

Evidence: an assertion over the hint table, not a spot-check of one hint.

### AC-3 — The sovereign policy asserts a live code reference or is retired

G-0534's subject narrows once the guard sits at the seam: routing becomes
structural rather than policed, and what is left to assert is that no production
path reaches a commit off-seam — which is M-0291/AC-3, not this policy.

Either outcome is acceptable, decided by measuring after M-0291 rather than
chosen now: re-aim the policy at a reference that exists, or retire it with a
recorded decision. G-0535 records the caution that makes this worth deciding
late — a policy re-aimed at whatever still passes keeps its name and loses its
subject.

Evidence: the re-aimed policy failing against a fixture that violates it, or the
decision entity recording the retirement.

## Constraints

- No surface gains a claim the kernel does not keep. If a correction cannot be
  made true, the claim is removed rather than softened.

## Out of scope

- The Tier-1 / Tier-2 override boundary itself, which stays with G-0333. Only
  that gap's finding-hint half is folded in here.

## Dependencies

- M-0291 — the corrected text names the seam that milestone builds.

## Surfaces touched

- `docs/design/legal-workflows-audit.md` — R-RULE-076, R-RULE-077, R-RULE-078,
  R-AUDIT-0070, R-AUDIT-0105, R-AUDIT-0113.
- `docs/design/design-decisions.md` — the human-only bullet.
- [ADR-0029](../../../docs/adr/ADR-0029-verb-shape-correctness-comes-from-pre-write-projection.md)
  — Decision and Consequences.
- `internal/cli/promote/promote.go`, `internal/cli/cancel/cancel.go`,
  `internal/cli/add/add.go`, `internal/cli/authorize/authorize.go` — the
  `--force` help.
- `internal/verb/promote_sovereign_act.go`, `internal/verb/add.go` — the two
  comments that contradicted each other.
- `internal/entity/sovereign.go` — the `SovereignActShape` doc comment.
- `internal/check/hint.go` — the derived force caveat, its predicate, and the
  composer fold.
- `internal/policies/aiwf_promote_epic_active_audit_test.go` — the failure
  message's remedy.
- `CHANGELOG.md`.

Removed: `internal/policies/sovereign.go` and its tests, and
`internal/cli/cliutil/sovereign_force.go` and its tests, per D-0061.

`CLAUDE.md` is not touched. Rows 1 and 2 of AC-1's table were made true by
M-0291 and needed no edit; they are pinned instead, so the claims cannot soften
back.

## Work log

### AC-1 — Every surface in the table states the seam that actually refuses

Ten of the thirteen rows corrected, plus two the table did not enumerate —
`R-RULE-077`'s citation and `R-RULE-078`'s "requires `--force` AND a human
actor", the rule-catalogue twins of rows 6 and 7. Correcting one catalogue and
leaving the other is the drift this milestone ends · commits `c25483278`,
`78fd5c769` (ADR-0029, through `aiwf edit-body`), and the review-round commit
below.

Row 4 was corrected rather than merely pinned: review measured that the kernel
refuses the `aiwf-force:` *trailer*, not the `--force` *flag*, and the original
sentence was false in exactly the case `aiwf add --force` produces.

### AC-2 — Every force finding hint names what the override does not relax

The caveat is appended in `HintFor` from one constant, folded together with the
ratification sentence M-0292 added there · commit `527caf70d` and the review
round below.

### AC-3 — The sovereign policy asserts a live code reference or is retired

Both, in that order. The predicate was re-aimed from a substring scan to a
call-expression scan, the guards were written, and measurement then showed the
layer cannot hold the rule — so the policy and the guards were retired and the
reasoning recorded as D-0061 · commits `c25483278` and the review round below.

## Decisions made during implementation

- **D-0061 — the sovereign-dispatcher policy is retired; the force rule lives at
  the apply seam.** The operator chose re-aiming over retirement when the
  question was open. Building it answered the question G-0534 said had to be
  settled first: whether `--force` denotes a sovereign act depends on verb and
  on tree state, and the dispatcher layer has neither, so a flag-keyed check
  refuses invocations the kernel permits. The decision entity carries the
  measurements.
- **The actor constraint was removed from `provenance-force-non-human`'s hint.**
  The derived caveat supplies it, and stating it twice in adjacent sentences is
  what an operator read otherwise.

## Validation

- `make check-fast` — clean (unit tests + full `golangci-lint`).
- `AIWF_COVERAGE_BASE=epic/E-0079-… make coverage-gate` — clean.
- `aiwf check` — 0 errors, 1 warning
  (`provenance-untrailered-scope-undefined`; the branch has no upstream).
- Doc-lint, scoped to the change-set — clean. Every symbol and file path the
  corrected docs cite resolves, including after the retirement.
- Behaviour measured against a binary built from `HEAD`, in disposable repos:
  an in-scope agent with full provenance forcing a real transition is refused
  with `HEAD` unmoved; a converging forced request by an agent returns exit 0
  and "nothing to change", per ADR-0036; `--format=json` carries the error
  envelope; `aiwf add milestone --force` as a non-human actor succeeds and its
  commit carries no `aiwf-force` trailer.
- Vacuity probes, per claim: reverting the hint predicate, dropping the caveat
  from `HintFor`, and making `add`'s force trailer unconditional each reddened
  the assertion that claims to hold it. Every `mustNotSay` needle was confirmed
  to match the pre-edit text, independently re-verified by review.

## Deferrals

None. Every blocking finding was fixed in-branch; the checks that pin them
landed in the same commits. The judgment findings that survived the retirement
are recorded below rather than deferred.

## Reviewer notes

Two independent reviewers, one code-quality and one design, over the full
change-set. Four blocking findings, all confirmed by measurement before being
acted on. Three are worth a later reader's attention:

- **The milestone introduced three false claims while correcting false claims.**
  The caveat sentence asserted that `--force` relaxes the FSM transition rule and
  nothing else; measured, it gates six non-FSM preconditions in `promote` and
  relaxes no FSM rule at all in `cancel`. Row 4 claimed the kernel refuses the
  flag rather than the trailer. Three surfaces claimed the dispatcher guard was
  "a second moment, not a second opinion". A green suite, a clean coverage gate,
  and binary-driven checks all certified the first two — because none of them
  re-measured the claim the prose made.
- **The dispatcher guard broke two ratified contracts and nothing caught it.**
  It refused converging requests that ADR-0036 specifies as exit 0, and dropped
  the `--format=json` error envelope. The convergence contract is pinned in
  `internal/verb`, one layer below where the guard sat, so a full-suite pass was
  compatible with breaking it. A contract pinned only below the layer a change
  touches is not pinned against that change.
- **The exemption criterion was applied inconsistently, and that was the tell.**
  `add` was exempted because its `--force` is conditional on state the
  dispatcher cannot see. The same is true of `promote` and `cancel` on the
  converging path. Had the criterion been applied evenly when it was written,
  the layer's unsuitability would have surfaced before the guards were built
  rather than at review.

Judgment findings accepted and acted on:

- The hint composers duplicated their ordering at both `HintFor` return sites.
  Folded into one ordered list, so a third composer is one edit.
- Two assertion rationales described the defect the same commit fixed, in the
  present tense.

Judgment findings declined, so a later reviewer meets a decision:

- **Shortening the force caveat to the actor constraint alone** was proposed,
  on the grounds that "every other check still runs" repeats what
  `milestone-done-incomplete-acs` already says. Declined: that hint is one of
  four carrying the caveat, and the clause is the half that stops a reader
  taking the override for a general escape.
- **Keeping the re-aimed policy with every dispatcher exempt** was the middle
  path between re-aiming and retiring. Declined: a policy whose whole population
  is enumerated asserts nothing today, and reads as coverage. D-0061 records the
  reasoning instead, where a future contributor asking the same question will
  meet it.
