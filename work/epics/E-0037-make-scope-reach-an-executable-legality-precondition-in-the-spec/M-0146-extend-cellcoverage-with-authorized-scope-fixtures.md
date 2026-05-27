---
id: M-0146
title: Extend cellcoverage with authorized-scope fixtures
status: done
parent: E-0037
depends_on:
    - M-0144
    - M-0145
tdd: required
acs:
    - id: AC-1
      title: Fixture framework builds an authorized-scope context a driver consumes
      status: met
      tdd_phase: done
    - id: AC-2
      title: Positive and negative scope-reach scenarios exercisable via the driver path
      status: met
      tdd_phase: done
---
## Goal

Extend the `internal/cellcoverage` fixture framework to stand up an **authorized-scope context** (an `authorize` commit + an agent actor) so the `m0124`/`m0125` positive/negative drivers can exercise a `scope-reach`-gated rule like any other cell.

## Context

`cellcoverage` today builds entity-state fixtures and evaluates entity-side predicates; it has no notion of an authorized-scope context (who is authorized on what). A `scope-reach` rule cannot be exercised by an entity fixture alone — it needs an open scope + an agent actor. This is E-0037's **unsized-risk** piece; M-0144's ADR sizes it and records the fallback (a dedicated test + AC-4 exemption) should it prove its own epic.

## Acceptance criteria

(ACs allocated separately via `aiwf add ac` after milestone creation; bodies seeded at allocation time.)

## Constraints

- Per M-0144's sizing decision; if the extension exceeds this milestone, the ADR's recorded fallback applies.
- `tdd: required`.

## Out of scope

The global `scope-reach` rule + the reclassification + AC-5 (M-0147). This milestone is the test-infrastructure, not the rule.

## Dependencies

M-0144 (ADR sizing), M-0145 (the evaluable predicate the driver exercises).

### AC-1 — Fixture framework builds an authorized-scope context a driver consumes

The fixture framework can build an authorized-scope context (an open `aiwf authorize` scope on an entity + an agent actor) that a driver consumes.

*Evidence:* a test that builds the scope fixture and asserts the scope is active / loadable.

### AC-2 — Positive and negative scope-reach scenarios exercisable via the driver path

A positive and a negative `scope-reach` scenario are exercisable through the driver path (in-scope agent verb succeeds; out-of-scope refused).

*Evidence:* a driver-level test exercising both arms against the real binary. The global rule itself lands in M-0147; this milestone proves the *machinery* exercises a scope-gated cell (via the existing runtime `provenance-authorization-out-of-scope` gate).

## Work log

### AC-1 — Fixture framework builds an authorized-scope context a driver consumes
Added `CellFixture.AuthorizeScope(t, entityID, agent) *scope.Scope` (`internal/cellcoverage/authorized_scope.go`) — opens a real `authorize` scope in-process (`verb.Authorize` + `verb.Apply`) and round-trips it through `cliutil.LoadEntityScopes`. RED→GREEN (`AuthorizeScope undefined` → implemented) · commit `e950f559` · `TestCellFixture_AuthorizeScope` green.

### AC-2 — Positive and negative scope-reach scenarios exercisable via the driver path
Driver-level test (`internal/policies/m0146_scope_machinery_test.go`): agent authorized on E-0001; promote of an in-scope milestone (under E-0001) succeeds, promote of an out-of-scope milestone (under E-0002) is refused with `provenance-authorization-out-of-scope` and lands no commit · commit `8c43f717` · `TestM0146_ScopeReachMachinery` green.

## Decisions made during implementation

- **Full integration confirmed; no fallback invoked.** M-0144's ADR sized this as tractable full integration with a documented fallback (dedicated test + cellcoverage exemption) only if it proved its own epic. The extension landed as one fixture method (`AuthorizeScope`) plus a driver-level test reusing the existing `authorize` verb and `testutil.RunBin` subprocess path — no new framework. The fallback was **not** needed.
- **AC-2 exercises the existing runtime gate, not the M-0147 spec rule.** The global `scope-reach` spec `Rule` lands in M-0147; this milestone proves the *machinery* (authorized-scope fixture + subprocess driver) can drive a scope-gated cell both ways, against the M-0141 `provenance-authorization-out-of-scope` runtime gate. The AC text pre-decided this scoping.

## Validation

- `go test ./...` — 56 packages ok, 0 non-flake failures. `go build ./...` — clean. `aiwf check` — 0 errors.
- `go test ./internal/cellcoverage/ ./internal/policies/` — green (`TestCellFixture_AuthorizeScope`, `TestM0146_ScopeReachMachinery`).
- TDD: AC-1 RED (`AuthorizeScope undefined`) → GREEN; AC-2 characterizes existing+AC-1 machinery (its red was shared with AC-1). Both `met` at `phase: done`.
- Branch coverage: `AuthorizeScope` happy path covered by both tests; its two defensive guards (`LoadEntityScopes` error, no-active-scope-after-authorize) are `//coverage:ignore` with rationale — unreachable in a well-formed fixture, and `errcheck` forbids dropping the error handling.

## Deferrals

None. The global `scope-reach` rule + the `provenance-authorization-out-of-scope` reclassification + the AC-5 fourth arm are M-0147 (the final milestone), not deferred scope.

## Reviewer notes

- **The negative arm is the load-bearing assertion.** It asserts the specific code `provenance-authorization-out-of-scope` (distinct from the `NoActiveScopeError` code), proving the refusal is the *scope-reach* gate — the agent has an active scope, it just doesn't reach the out-of-scope target. The positive arm then proves an in-scope target is allowed *given* the gate is active.
- **AC-2 relies on sequential id allocation** (`M-0001` under E-0001, `M-0002` under E-0002) — consistent with the codebase's fixture convention. It fails *loud* if the assumption breaks (a mis-scoped M-0001 would make the positive promote refuse and `Fatalf`), so it cannot silently test the wrong thing.
- **`AuthorizeScope` round-trips through `cliutil.LoadEntityScopes`** (the git-log loader the cmd layer uses) rather than hand-constructing a `scope.Scope`, so a driver consuming the scope sees exactly what production resolves.

