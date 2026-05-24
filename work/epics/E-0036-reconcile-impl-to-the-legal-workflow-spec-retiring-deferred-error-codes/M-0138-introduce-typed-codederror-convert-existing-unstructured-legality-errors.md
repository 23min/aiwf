---
id: M-0138
title: Introduce typed CodedError; convert existing unstructured legality errors
status: in_progress
parent: E-0036
tdd: required
acs:
    - id: AC-1
      title: CodedError carries a structured code reachable via errors.As
      status: met
      tdd_phase: done
    - id: AC-2
      title: ValidateTransition emits structured fsm-transition-illegal on illegal moves
      status: open
      tdd_phase: red
    - id: AC-3
      title: authorize refuses non-epic/milestone kinds with structured code
      status: open
      tdd_phase: red
    - id: AC-4
      title: fsm-transition-illegal and authorize-kind-not-allowed resolve as impl codes
      status: open
      tdd_phase: red
    - id: AC-5
      title: ADR records the CodedError pattern as accepted
      status: open
      tdd_phase: red
---
## Goal

Introduce a typed `CodedError` (option a) so verb-time refusals carry a first-class, `errors.As`-able structured `Code`, mirroring `check.Finding{Code}` — and apply it to the two legality errors that already fire but only as `fmt.Errorf` prose: `fsm-transition-illegal` (`entity.ValidateTransition`) and `authorize-kind-not-allowed` (`verb/authorize.go`, behavior shipped in M-0125). Extend the AC-5 spec→impl scanner to recognize the pattern. This is the keystone — M2/M3/M5 emit their codes through it.

## Context

E-0033's spec names `ExpectedErrorCode`s the verbs don't emit as structured data; the AC-5 scanner only recognizes `Code: "..."` composite literals (the `check.Finding{Code}` shape). No typed coded-error pattern exists in the codebase today — `ExpectedErrorCode` is a spec-side field only, verbs emit prose. Until a verb's refusal carries the code *as data*, the spec's "verified source of truth" claim carries asterisks. An ADR records the pattern because it governs every verb error going forward (G-0141 explicitly calls for one).

## Acceptance criteria

Each AC carries an explicit **Evidence** gate — the named test, driver cell, or drift policy that fails if the claim breaks. "Looks right" is not evidence.

### AC-1 — CodedError carries a structured code reachable via errors.As

A `CodedError` type carries a structured code reachable via `errors.As`. *Evidence:* `entity.TestCodedError_ErrorsAs` — constructs the error, asserts `errors.As` extracts the code as data; fails if the code isn't reachable structurally.

### AC-2 — ValidateTransition emits structured fsm-transition-illegal on illegal moves

`entity.ValidateTransition` returns a `CodedError` whose code is `fsm-transition-illegal` for an illegal transition and `nil` for a legal one. *Evidence:* table test across kinds, ≥1 legal + ≥1 illegal arm each; asserts the structured code on the illegal arm and no error on the legal arm (structural, not substring).

### AC-3 — authorize refuses non-epic/milestone kinds with structured code

`aiwf authorize <gap|decision|contract|adr> --to <agent>` refuses at verb-time carrying structured `authorize-kind-not-allowed`. *Evidence:* the M-0125 negative driver cells for the four authorize-kind cells un-skipped; binary-level assertion of non-zero exit + structured code + HEAD unchanged; the four authorize entries removed from `ac2KnownImplGaps`.

### AC-4 — fsm-transition-illegal and authorize-kind-not-allowed resolve as impl codes

`fsm-transition-illegal` and `authorize-kind-not-allowed` are removed from `deferredImplErrorCodes` and resolve as impl-side codes. *Evidence:* `TestM0123_AC5_SpecToImpl_ErrorCodesResolve` stays green *after* removal — only possible if the codes appear as real structured literals. (The closure that can't be claimed, only earned.)

### AC-5 — ADR records the CodedError pattern as accepted

An ADR records the `CodedError` pattern (shape; scope limited to legality-pertinent verb errors) as `accepted`. *Evidence:* structural assertion that the ADR exists with its named decision sections (scoped to the section, per CLAUDE.md doc-AC rule).

## Constraints

- Option (a) shape only — code carried as data, not a bare constant or a scanner-text-match.
- Convert **only** legality-pertinent verb errors; leave the ~30 other `fmt.Errorf` verb errors untouched (YAGNI).
- No new spec-schema expressivity.
- `tdd: required` — red-first per `wf-tdd-cycle`, branch-coverage audit before wrap.

## Design notes

`CodedError` shape (single struct vs `interface { Code() string }`) is settled in this milestone's ADR. Lean: interface, matching G-0142's proposed `FSMTransitionError{}.Code()`. The ADR is the foundation milestone's deliverable per the epic spec.

## Out of scope

Cancel guards (M2), the legality classifier (M3), the code rename (M4), scope reachability (M5). Converting non-legality verb errors.

## Dependencies

None. Closes G-0142 and G-0141.

## Work log

_(One entry per AC as it lands: `### AC-N — <title>` · outcome · commit SHA · tests N/M. The authoritative phase timeline is `aiwf history M-0138/AC-N`.)_

