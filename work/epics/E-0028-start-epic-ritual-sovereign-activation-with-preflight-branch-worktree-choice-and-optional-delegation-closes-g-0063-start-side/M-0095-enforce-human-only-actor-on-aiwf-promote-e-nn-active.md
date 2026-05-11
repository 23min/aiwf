---
id: M-0095
title: Enforce human-only actor on aiwf promote E-NN active
status: in_progress
parent: E-0028
tdd: required
acs:
    - id: AC-1
      title: non-human actor promoting epic to active is refused with override-hint error
      status: met
      tdd_phase: done
    - id: AC-2
      title: human actor promoting epic to active succeeds without override
      status: met
      tdd_phase: done
    - id: AC-3
      title: rule scoped to proposed-to-active edge; other epic transitions unaffected
      status: met
      tdd_phase: done
    - id: AC-4
      title: rule scoped to epic kind; other kinds unaffected by sovereign-act rule
      status: open
      tdd_phase: green
---

# M-0095 — Enforce human-only actor on `aiwf promote E-NN active`

## Goal

Make `aiwf promote E-NN active` a sovereign act: refuse non-human actors by default, mirroring the existing `--force` rule. The standard `--force --reason <text>` override remains available for any legitimate non-human invocation. This operationalizes G-0063's preflight row 5 ("`aiwf promote <epic> active` actor is `human/...` — refusal") and brings the principal × agent × scope provenance model to the epic-activation moment.

## Context

Today `aiwf promote E-NN active` accepts any actor without enforcement, which collapses the sovereign delegation moment into a routine FSM flip. The kernel's `--force` machinery already implements the "human-only with `--reason` override" pattern; this milestone reuses that machinery for the specific `epic → active` transition rather than inventing a parallel mechanism.

This is a standalone kernel rule. It does not depend on M-0094 (drafted-milestone finding); the two land in parallel and the skill in M-0096 consumes both.

## Acceptance criteria

(ACs allocated at `aiwfx-start-milestone` time per the planner-skill convention.)

## Expected shape

- The verb's per-kind transition handling (in `internal/verb/promote*.go`) gains a sovereign-act check on the `epic / proposed → active` edge. Refusal returns a typed error referencing the rule and the `--force --reason` override.
- Test coverage spans: refusal path (non-human actor), explicit-`--force` path (succeeds with proper trailers), and the unaffected paths (human actor; non-epic kinds; other transitions on epic).
- A pre-implementation audit of automation paths in this repo (CI workflows, hooks, scripts) confirms no legitimate non-human caller exists that would break — if any surface; it either gains `--force --reason` or its invocation moves to a human actor.

## Dependencies

- None. Standalone kernel rule; lands in parallel with M-0094.

## References

- E-0028 epic spec.
- G-0063 — preflight checks table, row 5.
- CLAUDE.md *Provenance is principal × agent × scope* — the model this rule operationalizes.
- `internal/verb/promote*.go` — verb path for the transition; existing `--force` handling.

### AC-1 — non-human actor promoting epic to active is refused with override-hint error

Invoking `aiwf promote <epic-id> active` with an actor that does not start with `human/` returns a Go error from `verb.Promote`. The error message references the new sovereign-act rule (so a reader understands *why* the verb refused) and points at the `--force --reason "..."` override path (so a sovereign actor knows the unblock path). Test drives `verb.Promote` directly with a non-human actor against an in-memory fixture; asserts non-nil error and substring presence of the override hint.

### AC-2 — human actor promoting epic to active succeeds without override

Invoking `aiwf promote <epic-id> active` with a `human/...` actor succeeds in default mode (no `--force`, no `--reason` required). The commit's standard trailers land (`aiwf-verb: promote`, `aiwf-entity: <epic-id>`, `aiwf-actor: human/...`). Test drives `verb.Promote` with a human actor end-to-end against a `proposed` epic; asserts no error, status flipped to `active`, trailer set well-formed.

### AC-3 — rule scoped to proposed-to-active edge; other epic transitions unaffected

The rule fires only on the `epic / proposed → active` edge. Other epic transitions executed by a non-human actor — `active → done`, `proposed → cancelled`, `active → cancelled`, `done → active` (with `--force`), etc. — are not refused by *this* rule (other rules may still apply via separate mechanisms). Table-driven test covering each non-`proposed-to-active` transition with a non-human actor; asserts the rule's error message does not appear.

### AC-4 — rule scoped to epic kind; other kinds unaffected by sovereign-act rule

The rule is scoped to `entity.KindEpic`. Non-human actors invoking promote on other kinds — milestone (`draft → in_progress`), contract (`proposed → active`), gap (`open → addressed`), ADR (`proposed → accepted`), decision — are not blocked by this rule. Table-driven test covering each non-epic kind reaching its respective active/accepted/in_progress/addressed state with a non-human actor; asserts no rule-fired error (other unrelated errors are tolerated; we assert the absence of the sovereign-act message).

