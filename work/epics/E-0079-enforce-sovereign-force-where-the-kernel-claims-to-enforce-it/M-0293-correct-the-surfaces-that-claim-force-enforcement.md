---
id: M-0293
title: Correct the surfaces that claim force enforcement
status: in_progress
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

