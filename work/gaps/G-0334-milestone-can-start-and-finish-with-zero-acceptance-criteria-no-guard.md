---
id: G-0334
title: Milestone can start and finish with zero acceptance criteria (no guard)
status: open
---
## What's missing

A milestone can traverse its entire lifecycle — `draft → in_progress → done` —
carrying zero acceptance criteria, tripping no finding. There is no kernel guard
requiring at least one AC before a milestone starts, and the one AC-completeness
rule does not cover the empty case at `done` either. So the AC-evidence discipline
is vacuous for a zero-AC milestone: it can start *and* finish with nothing to work
toward and nothing to substantiate completion.

## Evidence (traced + reproduced on v0.20.0)

- The milestone FSM edge is unconditional: `internal/entity/transition.go`
  (`"draft": {"in_progress", "cancelled"}`), and `ValidateTransition` is a pure
  table lookup with no AC inspection.
- No check rule fires on `in_progress` with zero ACs. The AC rules are
  `acs-shape`, `acs-title-prose`, `acs-tdd-audit`, `milestone-done-incomplete-acs`,
  and `acs-body-coherence`; only `milestone-done-incomplete-acs` touches
  completeness, and it keys on `status == done` with *open* ACs.
- `milestone-done-incomplete-acs` (`internal/check/acs.go`) fires only when an AC
  is `open`; `entity.MilestoneCanGoDone` returns `len(openACs) == 0`, so a zero-AC
  milestone has zero open ACs and passes `done` too.
- Reproduced: a 0-AC milestone promoted `draft → in_progress` and then
  `in_progress → done` with no error at either step.

## Contrast

The kernel has the epic analog `epic-active-no-drafted-milestones` (warning) but no
milestone sibling. The `aiwfx-start-milestone` ritual guards AC presence advisorily
— which, per "framework correctness must not depend on the LLM's behavior," is not
a guarantee.

## Direction (decision needed)

- Add a `milestone-in-progress-no-acs` check — lean: warning severity, mirroring
  `epic-active-no-drafted-milestones` — so starting a milestone with no ACs is
  surfaced.
- Decide whether `milestone-done-incomplete-acs` should also fire on a `done`
  milestone with an *empty* AC set (the vacuous-evidence case), and at what
  severity. Some milestones may legitimately start AC-less (pure coordination /
  exploratory), so severity is a genuine judgment call.

## Provenance

Surfaced by formal verification of aiwf v0.20.0; confirmed here by code trace and
measured reproduction.
