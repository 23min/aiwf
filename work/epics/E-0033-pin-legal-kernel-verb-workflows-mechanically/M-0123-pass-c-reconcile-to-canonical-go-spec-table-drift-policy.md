---
id: M-0123
title: Pass C reconcile to canonical Go spec table + drift policy
status: done
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
      status: met
      tdd_phase: done
    - id: AC-4
      title: 'LookupRule helper: hit, miss, no-duplicates semantics'
      status: met
      tdd_phase: done
    - id: AC-5
      title: Bidirectional drift policy (impl->spec and spec->impl)
      status: met
      tdd_phase: done
    - id: AC-6
      title: Every Sources.Decision resolves to an existing D-NNNN entity
      status: met
      tdd_phase: done
    - id: AC-7
      title: Rules() slice not exported; LookupRule is the only access
      status: met
      tdd_phase: done
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


## Work log

### AC-1 — spec package declares Rule, enums, Predicate, RuleSource, AntiRule types

`internal/workflows/spec/spec.go` declares the type system: `Rule` struct (9 fields, declaration order pinned by structural test); `Outcome` enum with `OutcomeUnspecified` zero-sentinel; `RejectionLayer` enum with `RejectionLayerNone` zero-sentinel; `Predicate`, `RuleSource`, `AntiRule` structs; and the two `entity.Kind`-typed extensions `KindAC` and `KindTDDPhase` for sub-FSM cells. Schema invariants pinned in the package doc; their enforcement is AC-5's drift-policy concern. Mechanical evidence: 7 reflect-based structural tests in `internal/policies/m0123_ac1_spec_types_test.go` covering field-order, enum-distinctness, and the two `Kind*` constants. Commit `11662cdc`.

### AC-2 — Rules() covers Q1-Q15 cells; schema invariants hold

`internal/workflows/spec/rules.go` populated with ~60 cells covering every (Kind, FromState) the kernel FSM recognizes plus 12 terminal-state illegal cells via the `terminalIllegal` helper. Schema adjustment from phase 1: the cell uniqueness key relaxed from `(Kind, FromState, Verb)` to `(Kind, FromState, Verb, Outcome, Preconditions)` to admit the preconditioned-pair pattern (Q5/Q6/Q7/Q8 each have legal + illegal companion cells at the same key, distinguished by Preconditions). 9 invariant tests in `internal/policies/m0123_ac2_rules_test.go`. Commit `93846e75`.

### AC-3 — AntiRules() carries 12 entries (Pass B §10 plus Q10 addition)

`internal/workflows/spec/antirules.go` returns 12 entries (ANTI-0001..ANTI-0012). ANTI-0001..ANTI-0011 mirror R-FP-0166..R-FP-0176 from Pass B §10. ANTI-0012 is the Q10 reconciliation addition: an epic MAY transition `proposed → active` with zero milestones, distinct from the `epic-active-no-drafted-milestones` warning whose guard ("all milestones drafts") is satisfied vacuously by the zero case but is deliberately allowed. 5 structural tests in `internal/policies/m0123_ac3_antirules_test.go`. Commit `e5ad99e5`.

### AC-4 — LookupRule helper: hit, miss, no-duplicates semantics

`internal/workflows/spec/lookup.go` declares `LookupRules(kind, fromState, verb string) []Rule`. Plural slice return per the AC-2 schema relaxation; the caller resolves which cell applies by walking `Preconditions`. 5 tests in `internal/policies/m0123_ac4_lookuprules_test.go`: hit-single, hit-preconditioned-pair, miss (3 sub-cases), match-all-inputs, no-duplicates-within-result. One sharp edge: the no-duplicates test initially asserted "at most 1 per Outcome" which broke on `(KindAC, open, promote)`'s two-legal-cell shape (open→met and open→deferred, distinguished by Preconditions); corrected to mirror AC-2's actual `(Outcome, Preconditions)` uniqueness contract. Commit `e1c19562`.

### AC-5 — Bidirectional drift policy (impl→spec and spec→impl)

`internal/policies/m0123_ac5_drift_test.go` with 8 sub-tests across two arms. **impl→spec:** every kind/from-state in `entity.AllKinds() × AllowedStatuses()` covered (strengthens AC-2's hardcoded list by walking the exported enumerators); every AC status in `AllowedACStatuses()` covered; every TDD phase in `AllowedTDDPhases() ∪ {""}` covered; every top-level Cobra verb covered or in `nonLegalityVerbAllowlist` (25 entries, each with one-line rationale). **spec→impl:** every Rule's Kind, FromState, Verb, and ExpectedErrorCode resolves. The error-code resolver walks `Code: "..."` literals across `internal/` and admits a `deferredImplErrorCodes` allowlist of 5 codes citing the tracking D-NNNN. Two drift findings surfaced during authoring: (a) spec missing the `(KindAC, "cancelled", "promote")` terminal cell — fixed; (b) allowlist key bug `editbody` vs Cobra-Use `edit-body` — fixed. Commit `070bd761`.

### AC-6 — Every Sources.Decision resolves to an existing D-NNNN entity

`internal/policies/m0123_ac6_decision_resolves_test.go` with two tests: one over `spec.Rules()` and one over `spec.AntiRules()`. Both use the existing `sharedRepoTree` helper for loader-based resolution per CLAUDE.md "Policy tests that read entity files must resolve via the loader" rule — `Tree.ByID` transparently resolves active and archive paths (ADR-0004). Tests passed green on first run because D-0002..D-0007 were committed during phase 1; the assertion's bite is for future deletion / rename / wrong-kind drift. Commit `bc553f59`.

### AC-7 — Rules() slice not exported; LookupRule is the only access

`internal/policies/m0123_ac7_rules_encapsulation_test.go` enforces the encapsulation invariant via static analysis rather than Go visibility. Given AC-2/5/6 were already authored as out-of-package drift tests that legitimately iterate `spec.Rules()`, the "Rules() slice not exported" AC text is read as a behavioral contract ("Rules() is not the consumer-facing access path"). The walker AST-parses every non-test `.go` file under `internal/` outside `internal/workflows/spec/` and `internal/policies/`, tracks the local name of the workflows/spec import (honoring aliases), and reports any `SelectorExpr` to `Rules` or `AntiRules`. Five tests: live-repo positive (zero production callers today), synthetic violation detection, alias honoring, LookupRules exemption, _test.go exemption. Commit `a833744f`.

## Decisions made during implementation

- **AC-2 schema relaxation: `(Kind, FromState, Verb, Outcome, Preconditions)` uniqueness key.** The phase 1 concretization doc specified `(Kind, FromState, Verb)` uniqueness. Reality (Q5/Q6/Q7/Q8 preconditioned-pair pattern) required loosening: same key + different Outcome + identical Preconditions is a duplicate (bad); same key + different Preconditions is a legitimate refined-cell. `LookupRule` adapted to `LookupRules` (plural, slice) to accommodate. Captured in AC-2's commit body.
- **AC-7 encapsulation reading: static policy, not Go visibility.** Three readings considered (rename to lowercase + move tests in-package; iterator-only API; static policy). Chose option 3 because the literal-unexport reading would retroactively break AC-2/5/6's already-shipped out-of-package tests. The "Rules() not exported" AC text is read as a behavioral guarantee enforced by `internal/policies/m0123_ac7_rules_encapsulation_test.go`. Captured in AC-7's commit body.
- **AC-5 deferred impl-side classifier.** The "impl→spec finding-code coverage" arm of the bidirectional drift policy (every legality-pertinent finding code referenced by ≥1 illegal-outcome Rule) requires a classifier distinguishing "verb-time legality" findings from "structural integrity" findings — wider than M-0123. Deferred via gap (G-0145) and documented in the test file's package-comment block.

## Validation

- `aiwf check` against the milestone branch — 0 errors, 24 warnings, none pertaining to M-0123 (all pre-existing on the branch).
- `make test-race` against the full module — every package green; `?` for `internal/workflows/spec/` (no test files in the spec package itself; tests live in `internal/policies/`).
- `golangci-lint run ./internal/policies/ ./internal/workflows/spec/...` — no findings on M-0123-authored files.
- `go build ./...` — green.
- Per-AC test counts: AC-1 (7 reflect tests), AC-2 (9 invariants), AC-3 (5 structural), AC-4 (5 hit/miss), AC-5 (8 drift sub-tests), AC-6 (2 resolver), AC-7 (5 encapsulation including 4 fixture-driven branch-coverage tests). Total: 41 tests across the seven ACs.

## Deferrals

All deferrals are tracked on `main` as follow-up gaps with `discovered_in: M-0123`:

- **G-0139** — Implement cancel-cascade per D-0003 and D-0004. The spec table references `epic-cancel-non-terminal-children` and `milestone-cancel-non-terminal-acs` as verb-time refusal codes; impl emits neither today. Listed in AC-5's `deferredImplErrorCodes` allowlist.
- **G-0140** — Implement `--evidence` flag on `aiwf promote AC met` per D-0005. The spec encodes the preconditioned legal + illegal companion cells; impl doesn't yet enforce the evidence binding. Listed in AC-5's `deferredImplErrorCodes` allowlist.
- **G-0141** — Implement `authorize-kind-not-allowed` verb-time refusal per D-0007. Spec encodes four illegal cells (one per disallowed kind); impl accepts authorize for any kind. Listed in AC-5's `deferredImplErrorCodes` allowlist.
- **G-0142** — Structured `fsm-transition-illegal` error from `entity.ValidateTransition`. Today's free-form `fmt.Errorf` doesn't carry the structured code; every terminal-illegal spec cell references it. Listed in AC-5's `deferredImplErrorCodes` allowlist.
- **G-0143** — Implement scope-tree three-edge reachability per D-0006. The spec's `scope-reach` predicate references the formal model; impl in `internal/scope/` not yet reconciled.
- **G-0144** — Rename `gap-resolved-has-resolver` to match Q8 addressed-by semantics. The code name predates the gap FSM's vocabulary shift from "resolved" to "addressed"; impl-side rename + spec-cell update + hint update needed.
- **G-0145** — Classifier for legality-pertinent finding codes (AC-5 impl→spec arm). Required to close the fourth arm of the bidirectional drift policy.

## Reviewer notes

- **Pacing.** Phase 2 ran end-to-end per-AC: ~4 commits per AC (feat + phase-green + phase-done + AC-met). Human approval at the AC-met boundary; the user signed off on each AC before the next started. The total commit count on this branch is 28 (7 ACs × 4 commits each) plus phase 1 setup commits.
- **Host → devcontainer handoff at AC-2 met.** Phase 1 + AC-1 + AC-2 landed on a macOS host. Sonoma 14.8.x syspolicyd throttle (G-0134) and hook-path ping-pong (G-0135) made further work slow and fragile; AC-3..AC-7 completed in the Linux devcontainer (E-0035 / M-0132). Branch state was clean at the handoff boundary (no work-in-progress).
- **Drift policy surfaced two real spec gaps during AC-5 authoring.** (a) `(KindAC, "cancelled", "promote")` terminal cell missing — the spec had the symmetric `deferred` terminal cell but not `cancelled`. (b) `nonLegalityVerbAllowlist` keyed by package name `editbody` rather than Cobra `Use: "edit-body"`. Both fixed in AC-5's feat commit. The drift tests working as designed.
- **`deferredImplErrorCodes` allowlist is the chokepoint for spec-vs-impl impedance.** Five codes (fsm-transition-illegal, epic-cancel-non-terminal-children, milestone-cancel-non-terminal-acs, ac-evidence-missing, authorize-kind-not-allowed) sit in the allowlist with D-NNNN justification. As each follow-up gap (G-0139..G-0142) lands, the allowlist entry comes out and the M-0123/AC-5 drift test re-binds the spec cell to the impl-side `Code: "..."` literal. The wiring is mechanical; the test fires automatically on PRs that don't remove the allowlist entry when the impl lands.
- **AC-7 is the most interpretive of the seven.** "Rules() slice not exported" had three defensible readings; the chosen one (static policy, not Go visibility) is documented in the AC-7 commit body and in the test file's package comment. A future contributor who disagrees can move the drift tests in-package and unexport — but that work was outside M-0123's scope and would have been a non-trivial reorganization.
- **`aiwf list --kind ac --status met` doesn't surface the work log structure.** The Work log section above is hand-maintained narrative; the structured truth is in `aiwf history M-0123/AC-N`. The narrative complements the history with the "why this commit" framing the trailers don't carry.
