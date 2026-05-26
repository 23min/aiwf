---
id: M-0141
title: Enforce three-edge scope reachability at verb-time
status: in_progress
parent: E-0036
depends_on:
    - M-0138
tdd: required
acs:
    - id: AC-1
      title: Scope reachability traverses exactly D-0006's three edges
      status: met
      tdd_phase: done
    - id: AC-2
      title: Out-of-scope authorized-agent verb refuses with structured code
      status: met
      tdd_phase: done
    - id: AC-3
      title: 'discovered_in reverse: agent promotes a gap it filed in scope'
      status: met
      tdd_phase: done
---
## Goal

Narrow scope reachability to D-0006's exact three-edge tree at **both** enforcement sites (verb-time `verb.Allow` and check-time `check/provenance.go`), closing the over-permissive scope-leak through governance edges, and emit a structured `errors.As`-able out-of-scope code at verb-time (M-0138 pattern), distinct from the "no active scope" case. Recorded in D-0014.

## Context

The mandated reviewed-reconcile read (D-0014) overturned this milestone's original "greenfield" premise. Reachability is **already enforced** at two sites — verb-time (`verb.Allow` → `scopeAllowsAct` → `tree.Reaches`) and check-time (`check/provenance.go` → `provenance-authorization-out-of-scope`, same `tree.Reaches`) — but both walk the **full reference graph** (`entity.ForwardRefs`: parent, depends_on, addressed_by, relates_to, supersedes, discovered_in, linked_adrs). D-0006 mandates exactly three edges and explicitly excludes the governance ones.

So the work is **tighten + structure**, not a greenfield build:

- The over-permissive bug is real: an agent scoped on E-1 can reach a cross-epic milestone via `depends_on`, or an ADR via a `supersedes`/`linked_adrs` chain, and the verb is wrongly allowed.
- D-0006's `discovered_in` friction case already passes today (discovered_in is a forward edge) — there is no regression to fix there, only a behavior to preserve under the narrower walk.
- The verb-time refusal carries no structured code and conflates "no active scope" with "scope exists but target out of reach."

`tree.Reaches`/`ReachesAny` have exactly two production callers, both scope-reachability; the broad walk has no other consumer.

The **formal-model arm is split out** (D-0014, G-0171): making `scope-reach` an executable spec predicate (`EvaluatePredicate` + `cellcoverage` + a global-precondition schema), reclassifying the code to `codes.ClassLegality`, and the AC-5 fourth-arm round-trip are genuinely greenfield spec-schema design (D-0006 deferred the encoding) and warrant their own ADR — per E-0036 open-question-1. This milestone ships the behavior (the substance of G-0143); G-0171 ships the certification.

## Acceptance criteria

Each AC carries an explicit **Evidence** gate — the named test that fails if the claim breaks. "Looks right" is not evidence.

### AC-1 — Scope reachability traverses exactly D-0006's three edges

A scope-reachability function resolves exactly the three D-0006 edges — target's `parent` chain to the scope-entity (transitive), composite-id containment (`M/AC-N` rolls up to `M`), and a single `discovered_in` hop followed by parent-climb — and reaches a target through **no** governance edge. *Evidence:* a table test over a fixture tree with one **reachable** case per included edge and one **not-reachable** case per excluded governance edge (`depends_on`, `addressed_by`, `relates_to`, `supersedes`, `superseded_by`, `linked_adrs`); every branch of the function exercised (branch-coverage audit).

### AC-2 — Out-of-scope authorized-agent verb refuses with structured code

An authorized agent invoking a verb on an in-scope target succeeds; on an out-of-scope target the verb refuses with the structured `provenance-authorization-out-of-scope` code (`errors.As`-able via M-0138's `Coded`; surfaced as `error.code` under `--format=json` via the M-0143 envelope), exits non-zero, and leaves HEAD unchanged. The refusal distinguishes "out of reach" from "no active scope" (each emits its own code). *Evidence:* a binary-level positive + negative test (agent actor, open-scope fixture) asserting the code by structural field access + exit code + HEAD-unchanged on the negative arm, plus a unit assertion that the two refusal reasons carry distinct codes.

### AC-3 — discovered_in reverse: agent promotes a gap it filed in scope

The D-0006 friction case holds: an agent authorized on E-NN can promote a gap it filed with `--discovered-in M-K` where M-K is in E-NN's subtree — the case strict-parent-only reachability would wrongly refuse. *Evidence:* a dedicated test of the `discovered_in`-reverse arm (gap → `discovered_in` M-K → `parent` E-NN) asserting the verb is allowed.

## Constraints

- Out-of-scope code emitted via M-0138's `Coded` pattern; reuse the existing `provenance-authorization-out-of-scope` string (one code per violation, shared verb-time + check-time — the M-0143 / C2 unification).
- Edges exactly per D-0006 — closed set, expressed explicitly so no governance edge is reachable by construction (not a filtered graph walk that could silently re-broaden).
- Both enforcement sites repoint to the single narrowed predicate; the dead full-graph `tree.Reaches` is removed.
- **Verify the narrowed check-time rule does not newly-flag this repo's own authorized history** before wrapping; if it does, surface it (don't paper over).
- Keep the code `codes.ClassStructural` — reclassification to legality travels with G-0171 (it is inseparable from the spec-Rule round-trip).
- `tdd: required`.

## Out of scope (→ G-0171, recommend its own epic)

- Reclassifying `provenance-authorization-out-of-scope` to `codes.ClassLegality`.
- Implementing `scope-reach` in `EvaluatePredicate` (verb-invocation context through `EvalContext`).
- Extending `internal/cellcoverage` to exercise a scope-reach precondition.
- Representing/drift-certifying a **global** precondition in the spec `Rule` table + the AC-5 fourth-arm extension.
- The `CodedError` foundation (M-0138).

## Design note

The split follows E-0036 open-question-1's resolution (*"split ... if its verb-time-refusal design warrants its own ADR"*). The reconcile finding (D-0014) relocated the unsized risk from the verb-layer hook (which already exists) to the spec-model representation of a global precondition (which D-0006 deferred and never settled). The behavior fix is pure and self-contained; the formal-model certification is the greenfield-design part and lives in G-0171.

## Dependencies

M-0138 (the `Coded` pattern). Closes **G-0143** (three-edge reachability + verb-time out-of-scope refusal with a structured code). Formal-model certification tracked in **G-0171**. Reconcile + narrow/unify/split recorded in **D-0014**.

## Work log

### AC-1 — Scope reachability traverses exactly D-0006's three edges

Explicit three-edge `tree.ReachesScope`/`ReachesScopeAny` (parent-climb + composite rollup + one `discovered_in` hop), reading `e.Parent`/`e.DiscoveredIn` directly so no governance edge is reachable by construction; `parentChainReaches` carries a cycle-terminating visited guard. The dead full-graph `tree.Reaches`/`ReachesAny` were deleted; both `verb.Allow` and `check/provenance.go` repointed. commit `0a8a6bf1` · tests: `TestReachesScope` / `TestReachesScopeAny` / `TestReachesScope_ParentCycleTerminates` (`internal/tree`), 100% branch coverage. RED-first (stub); mutation-proven (parent-only impl → discovered_in cases RED).

### AC-2 — Out-of-scope authorized-agent verb refuses with structured code

`verb.Allow` splits the denial into `*ScopeOutOfReachError` (active scope, target unreachable) and `*NoActiveScopeError` (no scope), each carrying its code via `AllowResult.Err`; `gateAndDecorate` wraps with `%w` so `entity.Code` surfaces `provenance-authorization-out-of-scope` as `error.code`. commit `0a8a6bf1` · tests: `TestScopeReach_OutOfScopeRefusal_AC2` (binary: structural `error.code` + exit 1 + HEAD-unchanged), `TestAllow_DenialCodes` (distinct codes), `TestAllow_NilTreeRefuses` (guard). RED-first (empty code on the pre-change binary); mutation-proven (forced code-empty → RED).

### AC-3 — discovered_in reverse: agent promotes a gap it filed in scope

End-to-end friction test: an agent authorized on E-0001 files a gap `--discovered-in M-0001` (in-scope creation), then promotes it — reachable ONLY via the `discovered_in`-reverse edge (gaps have no parent). commit `0a8a6bf1` · test: `TestScopeReach_DiscoveredInFriction_AC3`. Mutation-proven (parent-only impl → the promote is refused with `provenance-authorization-out-of-scope`).

## Decisions made during implementation

- **D-0014 — Narrow scope reachability to D-0006 three edges; split formal-model arm** (`accepted`). The reviewed-reconcile read found reachability already enforced at two sites via the full reference graph (over-permissive vs D-0006). The decision: one explicit three-edge predicate at both sites; the verb-time denial split into two coded errors (out-of-reach vs no-active-scope); the legality-classification + executable `scope-reach` spec predicate split to G-0171. Rejected: keeping the full-graph walk (the leak bug), narrowing only verb-time (two sources of truth), doing the spec-model arm now (undecided global-precondition schema, would force a hack).

## Validation

```
CGO_ENABLED=0 go build ./...            # exit 0
go vet ./...                            # exit 0
golangci-lint run                       # 0 issues
go test ./... -count=1 -parallel 8      # green (runs #1 and #3 clean; run #2 hit one
                                        #   pre-existing parallel-contention flake in
                                        #   internal/contractverify — text-file-busy
                                        #   write-then-exec — passing in isolation,
                                        #   untouched by this diff)
aiwf check                              # 0 errors · 9 warnings (pre-existing: M-0102 ×5,
                                        #   G-0061 ×3; + epic-active-no-drafted-milestones
                                        #   on E-0036, expected — last milestone, clears
                                        #   at epic wrap)
```

Per-AC mechanical evidence (all green): `TestReachesScope` / `…Any` / `…ParentCycleTerminates` (AC-1, 100% branch coverage); `TestScopeReach_OutOfScopeRefusal_AC2` + `TestAllow_DenialCodes` + `TestAllow_NilTreeRefuses` (AC-2); `TestScopeReach_DiscoveredInFriction_AC3` (AC-3). The narrowed check-time rule was verified against this repo's own authorized history: **no new `provenance-authorization-out-of-scope` finding**.

## Deferrals

- **G-0171** — executable `scope-reach` global precondition + legality classification (the formal-model certification arm), split per D-0014 and recommended as its own epic. No deferred or cancelled ACs (all three `met`).

## Reviewer notes

- **The reconcile finding is the substance.** The milestone's "greenfield" premise was wrong: the verb-layer enforcement hook (`verb.Allow`) already existed and worked; the genuinely-unsized risk was the spec-model representation of a *global* precondition, which D-0006 explicitly deferred and never settled. That arm split to G-0171.
- **One predicate, two enforcement sites.** Both `verb.Allow` and `check/provenance.go` now call `tree.ReachesScope`; the full-graph `tree.Reaches` was deleted (it had no non-scope caller). The check-time out-of-scope tests are all `parent:`-based fixtures, so the narrowing is behavior-preserving for them (verified at the fixture level, not just by the suite passing).
- **Code kept `codes.ClassStructural`.** `codes.go` files "provenance" under structural, and the check finding has always been structural. Reclassifying to `ClassLegality` is inseparable from the spec-Rule round-trip (the AC-5 fourth arm requires every legality code to be named by an illegal spec cell), so it travels with G-0171 — not a skip, a dependency.
- **Two provenance integration tests updated, root-caused not papered.** `TestProvenance_AgentRefusedOutOfScope` and `…AgentAddMilestoneInScope` asserted the old conflated `"no active scope"` message on the out-of-reach path; D-0014 split that case off with its own code, so both now assert `provenance-authorization-out-of-scope` (a strengthening). The authorize verb's own pause "no active scope" assertion (`authorize_cmd_test.go`) is a different message and untouched.
- **Flakes observed are pre-existing, not from this diff.** `internal/contractverify` (text-file-busy write-then-exec) and `internal/cli/integration` (repo-lock "another aiwf process") are the documented G-0097/G-0127 parallel-contention class; both pass in isolation. The AC-2 `--phase green` promote was blocked once by the same policies-hook flake and retried after confirming policies green in isolation.
