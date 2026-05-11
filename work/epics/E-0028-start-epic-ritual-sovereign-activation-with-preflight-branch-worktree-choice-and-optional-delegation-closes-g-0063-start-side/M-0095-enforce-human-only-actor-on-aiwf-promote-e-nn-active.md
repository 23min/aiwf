---
id: M-0095
title: Enforce human-only actor on aiwf promote E-NN active
status: draft
parent: E-0028
tdd: required
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
