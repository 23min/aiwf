---
id: M-0140
title: Classify legality finding codes; close AC-5 bidirectional arm
status: in_progress
parent: E-0036
depends_on:
    - M-0138
tdd: required
acs:
    - id: AC-1
      title: Legality codes carry a structural Class marker enumerable in code
      status: met
      tdd_phase: done
    - id: AC-2
      title: AC-5 drift fails when a legality-classed code lacks a spec reference
      status: met
      tdd_phase: done
    - id: AC-3
      title: Existing M-0138 legality codes round-trip and resolve to spec rules
      status: open
      tdd_phase: red
---
## Goal

Add a structural `Class` marker (e.g. `ClassLegality`) so legality-pertinent finding/error codes are programmatically enumerable, and close AC-5's deferred fourth arm: every legality code is referenced by ≥1 illegal-outcome spec Rule. This makes the bidirectional-completeness guarantee live and turns the classifier into a chokepoint later milestones must satisfy.

## Context

AC-5 closes three of four drift arms; the impl→spec arm — "every legality-pertinent finding code is referenced by a spec rule" — was deferred (G-0145) because ~25 impl codes mix legality (fire on FSM/precondition violations) with structural integrity (frontmatter shape, id collisions, ref resolution). The classifier must enumerate the former. Lean from the gap: structural metadata on the code, not a hand-maintained allowlist — the classifier is a property of the code, not of the test.

## Acceptance criteria

Each AC carries an explicit **Evidence** gate — the named test or drift arm that fails if the claim breaks. "Looks right" is not evidence.

### AC-1 — Legality codes carry a structural Class marker enumerable in code

Legality-pertinent codes carry a structural `Class` marker (e.g. `ClassLegality`) that is programmatically enumerable in code — a property of the code, not a hand-maintained test allowlist (G-0145 option 2). *Evidence:* `TestFindingClass_LegalityEnumerable` — asserts the closed legality set is derived from the marker, and that a structural-integrity code (frontmatter-shape / id-collision / ref-resolution class) is *not* in it.

### AC-2 — AC-5 drift fails when a legality-classed code lacks a spec reference

The AC-5 drift test gains its deferred fourth arm: every legality-classed impl code is referenced by ≥1 illegal-outcome spec `Rule`, and the arm **fails when a legality code lacks a spec reference**. *Evidence:* the new arm in `m0123_ac5_drift_test.go`; a negative-of-the-policy fixture (a deliberately orphaned legality-classed code) drives it red — proving the policy actually fires, not just passes vacuously. A policy that cannot fail is not a chokepoint.

### AC-3 — Existing M-0138 legality codes round-trip and resolve to spec rules

The two legality codes that exist at this milestone — `fsm-transition-illegal` and `authorize-kind-not-allowed` (both M-0138) — round-trip as legality and each resolves to ≥1 illegal-outcome spec `Rule`. *Evidence:* assertion they appear in the legality set and each maps to ≥1 `Rule`. The epic's AC-3 also names the two cancel codes; those are emitted by **M-0139**, and AC-2's mechanism auto-includes them once M-0139 classifies them — chokepoint-first ordering, so they are certified there, not here. Until then the cancel codes' spec→impl direction stays covered by `deferredImplErrorCodes`.

## Constraints

- Structural metadata (G-0145 option 2), not a hand-maintained allowlist.
- `tdd: required`. AC2's negative-of-the-policy test is mandatory — a policy that can't fail is not a chokepoint.

## Out of scope

Emitting the codes (M-0138/M2); the rename (M4); reachability (M5).

## Dependencies

M-0138. Best executed after M2 so it certifies the cancel codes too (soft ordering). Closes G-0145.

## Decisions made during implementation

- **D-0011 — Classify codes with a typed `Code` descriptor carrying a `Class` field** (`accepted`). Resolves E-0036 open question 4 / G-0145's mechanism choice. A legality code becomes a `Code{ID, Class}` value (class intrinsic to the declaration); the closed legality set is enumerated by the existing AST scanner reading the `Class:` field; the behavioral `Class()` on `Coded` errors derives from the same descriptor. Realizes ADR-0012's named-code-constant decision along G-0129's typed-code trajectory. Rejected: behavioral-method + parallel list (dual source of truth), and a central `map[string]Class` registry (side-table divorced from the code).

## Work log

### AC-1 — Legality codes carry a structural Class marker enumerable in code

New leaf package `internal/codes`: `Class` enum (`ClassStructural` zero-value / `ClassLegality`) + `Code{ID, Class}` descriptor (D-0011). The two legality codes (`CodeFSMTransitionIllegal`, `CodeAuthorizeKindNotAllowed`) became descriptor `var`s carrying `ClassLegality`; `Code()` returns `.ID`, so the `Coded` interface and the preserved message text are unchanged. The AC-5 scanner `collectImplFindingCodes` now returns `map[string]codes.Class` and reads the descriptor form via `descriptorCode`/`typeNamedCode`/`classValueIsLegality`, so a single scan yields both the full code set (AC-4) and the legality subset (AC-1) — the class is a property of the declaration, no parallel allowlist. `Class()` on the errors deliberately not added (YAGNI; derivable when M-0143 needs it). RED proven load-bearing (pre-conversion `fsm-transition-illegal` classified `ClassStructural`). Implemented by the `aiwf-extensions:builder` subagent; diff + build/vet/lint/suite re-verified parent-side. commit `b035dc21` · tests: `TestFindingClass_LegalityEnumerable` + `TestDescriptorCode_Branches`; AC-4 + M-0138 stay green.

