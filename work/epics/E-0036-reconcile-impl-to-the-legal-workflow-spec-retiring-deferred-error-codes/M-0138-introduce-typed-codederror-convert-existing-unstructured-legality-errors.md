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
      status: met
      tdd_phase: done
    - id: AC-3
      title: authorize refuses non-epic/milestone kinds with structured code
      status: met
      tdd_phase: done
    - id: AC-4
      title: fsm-transition-illegal and authorize-kind-not-allowed resolve as impl codes
      status: open
      tdd_phase: done
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

`aiwf authorize <gap|decision|contract|adr> --to <agent>` refuses at verb-time with a typed `AuthorizeKindError` carrying structured `authorize-kind-not-allowed` (extractable via `entity.Code`). *Evidence:* `verb.TestAuthorize_Open_RefusesNonScopeEntityKind` extended to assert `entity.Code(err) == verb.CodeAuthorizeKindNotAllowed` structurally across all four disallowed kinds; M-0125's existing authorize cells stay green on the preserved message text.

*Evidence correction (made during implementation):* the originally-drafted gate — "un-skip the four authorize cells in `ac2KnownImplGaps`; binary-level structured-code assertion" — was inaccurate. Those cells were **never** in `ac2KnownImplGaps` (they have run end-to-end since G-0141 Phase 1, asserting substrings), and the `--format=json` envelope surfaces **no** code today. Surfacing the code in the envelope (E-0036's "errors.As-able for the JSON envelope" goal clause) is split to its own milestone **M-0143**, where the envelope representation + exit-code treatment is settled via a D-NNNN. AC-3 here is scoped to verb-layer `errors.As` extractability, which is the part actually mechanically checked.

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

### AC-1 — CodedError carries a structured code reachable via errors.As

`entity.Coded` behavioral interface (`error` + `Code() string`) + `entity.Code(err)` helper that extracts the code structurally by walking the `%w` chain with `errors.As`. Anti-cheat test confirms a code present only in an error's *message text* does not resolve. `coded.go` `Code()` at 100% branch coverage (both `errors.As` arms). commit `bdfd26e3` · tests: `TestCodedError_ErrorsAs` (7 cases) + `TestCode_EmptyCodeStillFound`.

### AC-2 — ValidateTransition emits structured fsm-transition-illegal on illegal moves

`ValidateTransition` now returns a typed `FSMTransitionError{Kind,From,To,Allowed}` (implements `Coded`; `Code()` → `CodeFSMTransitionIllegal`) for illegal transitions of a recognized `(kind, from)`; `Error()` preserves the kernel's terminal/not-allowed message text verbatim. Malformed input (unknown kind / unrecognized from) stays a plain, non-`Coded` error. Seam verified: only caller `verb/promote.go:93` returns the error unwrapped, and `verb` + M-0125's binary driver stay green (no flattening). `ValidateTransition`/`Error`/`Code` at 100% branch coverage. commit `325b49a6` · tests: `TestValidateTransition_FSMTransitionIllegalCode` (cross-kind: not-allowed, terminal, legal, malformed).

### AC-3 — authorize refuses non-epic/milestone kinds with structured code

`verb.AuthorizeKindError{Kind}` (implements `entity.Coded`; `Code()` → `CodeAuthorizeKindNotAllowed`) replaces the prior `fmt.Errorf`; `Error()` builds the message from the constant, so the text — including `(authorize-kind-not-allowed)` — is preserved and M-0125's substring driver stays green. `Error`/`Code` at 100% coverage. Scoping discovery: the envelope-surfacing (E-0036 goal clause) was not actually part of this AC and is split to **M-0143**; the original AC-3 evidence wording was corrected (see the AC-3 section above). commit `1d499b38` · tests: `TestAuthorize_Open_RefusesNonScopeEntityKind` (extended with the structural `entity.Code` assertion, 4 kinds).

