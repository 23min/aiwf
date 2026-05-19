---
id: M-0123
title: Pass C reconcile to canonical Go spec table + drift policy
status: in_progress
parent: E-0033
depends_on:
    - M-0121
    - M-0122
tdd: required
acs:
    - id: AC-1
      title: spec package declares Rule, enums, Predicate, RuleSource, AntiRule types
      status: met
      tdd_phase: done
    - id: AC-2
      title: Rules() covers Q1-Q15 cells; schema invariants hold
      status: met
      tdd_phase: done
    - id: AC-3
      title: AntiRules() carries 12 entries (Pass B §10 plus Q10 addition)
      status: open
      tdd_phase: green
    - id: AC-4
      title: 'LookupRule helper: hit, miss, no-duplicates semantics'
      status: open
      tdd_phase: red
    - id: AC-5
      title: Bidirectional drift policy (impl->spec and spec->impl)
      status: open
      tdd_phase: red
    - id: AC-6
      title: Every Sources.Decision resolves to an existing D-NNNN entity
      status: open
      tdd_phase: red
    - id: AC-7
      title: Rules() slice not exported; LookupRule is the only access
      status: open
      tdd_phase: red
---
## Goal

Reconcile M-0121's audit catalog with M-0122's first-principles catalog into the canonical **Go spec table** that downstream tests (M-0124, M-0125) drive against the binary, plus the **drift policy** that ensures the spec stays closed-set against the impl in both directions.

This is where the epic's deliverable lands. The two catalogs are evidence; this milestone produces the actual spec.

## Reconciliation classes

Each rule from either catalog falls into one of:

| Class | M-0121 says | M-0122 says | Resolution |
|---|---|---|---|
| **Agreement** | legal | legal | Spec entry, marked agreed |
| **Agreement** | illegal | illegal | Spec entry, marked agreed |
| **Audit-only** | rule X | silent | Spec entry citing audit source |
| **FP-only** | silent | rule X | Decision needed: do we ratify? |
| **Conflict** | rule X | rule Y (different) | Decision needed: which wins, why |
| **Undefined-by-both** | silent | silent | Surface as known-undecided; this milestone decides the posture (see *Open question to settle here* below) |

FP-only and Conflict entries each surface a decision via `aiwf add decision --relates-to E-0033`, body shape pinned under *Decision body template* below. Agreement and Audit-only entries go straight to the spec.

## Workflow

This milestone runs in two phases inside a single milestone shell. Status flow: `draft` → promote `active` at start of phase 1 (no ACs yet — see note below) → ACs added at end of phase 1 → phase 2 implementation → `done`.

**Phase 1 — Reconciliation walk + schema design.** Research and authoring; not TDD-shaped.

1. Sit with both catalogs open simultaneously. Walk rule-by-rule, classifying each as Agreement / Audit-only / FP-only / Conflict / Undefined-by-both.
2. For each FP-only and Conflict entry, open a decision via `aiwf add decision --relates-to E-0033` using the *Decision body template* below.
3. Design the Go schema in concrete form, informed by the walk — the catalog shape informs the schema, not the other way round. The load-bearing shape is pinned under *Canonical Go spec* below.
4. Add the milestone's real AC list via `aiwf add ac`. AC-1 is the structural assertion that `internal/workflows/spec/` carries the agreed `Rule` struct shape; subsequent ACs cover the `Rules()` table itself, the bidirectional drift policy, and the meta-coverage tests that M-0124 and M-0125 will consume.

The milestone is `active`-with-zero-ACs through phase 1. That's intentional — `aiwf status` will show M-0123 as active without an AC list until the walk completes. The kernel does not forbid this; `acs-tdd-audit` only fires on ACs that exist with missing `tdd_phase`.

**Phase 2 — TDD-shaped implementation of the phase-1 ACs.** Per CLAUDE.md *AC promotion requires mechanical evidence*, every AC carries a Go test under `internal/policies/` or `internal/workflows/spec/`. The drift policy lands in the same milestone as the spec table (not split out).

## Canonical Go spec

Under `internal/workflows/spec/` (exact package name decided during phase 1):

- A `Rule` struct capturing one legality cell.
- A `Rules() []Rule` function exposing the closed-set table.
- A `LookupRule(kind, fromState, verb)` helper for tests to call.

The schema's load-bearing shape:

```go
type Rule struct {
    Kind              entity.Kind       // typed; entity.KindEpic etc.
    FromState         string            // bare type, but uses entity.Status* constants at cell sites
    Verb              string            // bare; closed by spec→impl test
    Preconditions     []Predicate
    Outcome           Outcome           // legal | illegal
    ExpectedErrorCode string            // bare; uses internal/check Code* constants where typed (provenance family) and bare strings elsewhere — G-0129 follow-up to typed-ify the rest
    RejectionLayer    RejectionLayer    // verb-time | check-time; zero value for legal cells
    BlockingStrict    bool              // whether aiwf check --strict blocks
    Sources           RuleSource
}
```

### Predicate vocabulary (closed set)

`Preconditions` carry `Predicate`s expressed against the planning tree at verb-time. Subject vocabulary is constrained to four shapes:

- `self.<field>` — the entity the verb operates on
- `parent.<field>` — the entity's parent (e.g., milestone's epic, AC's milestone)
- `all-children.<field>` — every child satisfies the predicate
- `any-child.<field>` — at least one child satisfies it

Cross-verb sequence legality manifests as predicates on the later verb (e.g., "AC promote requires `parent.status != entity.StatusDone`"). There is no separate `SequenceRule` type. If reconciliation surfaces a rule that cannot be expressed as a state predicate, that is evidence the entity model is missing a state — surface it as a decision against E-0033, do not grow a second schema.

Widening the predicate subject vocabulary beyond the four shapes above requires its own decision entity.

### Severity model (two axes, not a scalar)

Illegal cells carry two independent fields:

- `RejectionLayer`: `verb-time` (verb returns non-zero, no commit, no side effect) | `check-time` (verb succeeds, `aiwf check` reports a finding)
- `BlockingStrict`: `true` if the cell is blocked at the chosen layer when `aiwf check --strict` runs; `false` if the cell stays advisory (warning-only)

This faithfully encodes the impl's existing layered severity. Phase 1 validates that the second axis carries real information across enough cells to justify it; if only a handful of `(check-time, non-blocking)` cells exist, collapse to a single `Severity` enum with three named values (`HardReject | CheckError | CheckWarning`) and record the rationale via decision entity. The default lean is the two-axis form.

### Rule sources

```go
type RuleSource struct {
    Audit    []string // R-AUDIT-NNN ids; slice in case multiple audit rows back this cell
    FP       []string // R-FP-NNN ids
    Decision string   // D-NNNN, populated only for FP-only and Conflict cells
}
```

The `Decision` field's zero value is the cheapest "this cell did not require ratification" signal. Per the reconciliation table:

- Agreement → `{Audit: [...], FP: [...]}` (both citations, no decision)
- Audit-only → `{Audit: [...]}` (audit citation only; rule is already in the impl, no ratification needed)
- FP-only → `{FP: [...], Decision: "D-NNNN"}` (we adopted a rule the impl doesn't have)
- Conflict → `{Audit: [...], FP: [...], Decision: "D-NNNN"}` (both citations + the ratifying decision)

The forward-link from `Sources.Decision` to its D-NNNN, paired with the *Spec cell* section in each decision body, makes the decision↔spec mapping bidirectionally traceable via `aiwf history` alone.

### CUE considered

A CUE-based spec was considered and rejected. Three reasons: (a) it re-opens M-0120 §3's "canonical form is Go data structures" decision, which the methodology ADR is supposed to lock; (b) it makes drift symmetry strictly worse — CUE cannot reference Go symbols, so the compiler-catches-removal closure on every column that resolves to a constant name is forfeit and every column needs an explicit spec→impl test; (c) it violates `design-decisions.md`'s "external schema languages deferred until a real consumer needs to customize the vocabulary" stance. Re-open only if a consumer materializes who needs the constraint-language affordance.

## Drift policy

Closure runs in **both directions** between the spec and the impl. Tests under `internal/policies/` assert:

**impl → spec** (impl cannot grow without spec growing):

- Every kind/state pair the impl's FSM tables recognize has at least one corresponding rule in the spec.
- Every top-level Cobra verb is referenced by at least one rule.
- Every `aiwf check` finding code that pertains to verb legality is referenced by at least one illegal-cell rule (advisory codes are exempt).

**spec → impl** (spec cannot reference dead symbols):

- `Kind` is closed by the compiler (`entity.Kind` is a named type with typed constants).
- `FromState` is closed by the compiler when cells reference `entity.StatusFoo` constants rather than bare string literals — the `enum-literal-adoption` policy (G-0126) already requires this at comparison sites, and spec cells are one such site.
- `ExpectedErrorCode` is closed by the compiler for the ~13 codes that carry typed constants today (`CodeProvenance*` family in `internal/check/provenance.go`). The remaining ~32 bare-string codes are closed by a dedicated spec→impl test that asserts every cell's value appears in the impl's emitted-code set. G-0129 tracks the follow-up to typed-ify these so the compiler closes them too.
- `Verb` is closed by a dedicated test that asserts every cell's `Verb` value appears in the root Cobra command's registered command set.

Failure of any of these is a hard CI block. The impl cannot grow a new verb / state / finding without the spec growing too, and the spec cannot retain a rule whose verb / state / code has been deleted.

The drift policy is scoped to this kernel repo. It does not push down to consumer-side `aiwf check`, since consumers do not carry a workflows-spec artifact.

## Decision body template

Each reconciliation decision allocated under this milestone (FP-only or Conflict, via `aiwf add decision --relates-to E-0033`) carries three required sections:

```
## Sources
- First-principles: R-FP-NNN (legal-workflows-first-principles.md L<line>)
- Audit: R-AUDIT-NNN (legal-workflows-audit.md L<line>)        ← omit for FP-only

## Resolution
<chosen direction + rationale; for Conflict, explain why this catalog wins>

## Spec cell
internal/workflows/spec — Rule{Kind: ..., FromState: ..., Verb: ...}
```

The *Spec cell* forward-link is non-negotiable: paired with each `Rule.Sources.Decision` slot pointing back at the D-NNNN, it makes the decision↔spec mapping mechanically traceable via `aiwf history` alone.

Enforcement is reviewer-level. A structural finding-rule under `internal/check/` to assert these sections present-and-non-empty is deferred until the volume of reconciliation decisions warrants the chokepoint.

## Acceptance criteria

(Added via `aiwf add ac` at the end of phase 1 — see *Workflow* above.)

## Approach

- Phase 1 produces: the concrete Go schema, one decision entity per FP-only / Conflict case, and the AC list.
- Phase 2 implements the ACs in TDD order; the drift policy lands in the same milestone as the spec table.

## Open question to settle here

The "negative-of-undefined" posture (cells the spec deliberately leaves silent) is decided in this milestone, based on whether reconciliation actually surfaces any genuinely undecidable cells. Default lean: closed spec (every cell decided one way or the other). Decide otherwise only if forced by reality; record the call via `aiwf add decision --relates-to E-0033`.

### AC-1 — spec package declares Rule, enums, Predicate, RuleSource, AntiRule types

### AC-2 — Rules() covers Q1-Q15 cells; schema invariants hold

### AC-3 — AntiRules() carries 12 entries (Pass B §10 plus Q10 addition)

### AC-4 — LookupRule helper: hit, miss, no-duplicates semantics

### AC-5 — Bidirectional drift policy (impl->spec and spec->impl)

### AC-6 — Every Sources.Decision resolves to an existing D-NNNN entity

### AC-7 — Rules() slice not exported; LookupRule is the only access

