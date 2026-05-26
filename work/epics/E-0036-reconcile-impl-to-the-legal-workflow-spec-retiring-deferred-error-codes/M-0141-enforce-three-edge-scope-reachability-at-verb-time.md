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
      status: open
      tdd_phase: done
    - id: AC-2
      title: Out-of-scope authorized-agent verb refuses with structured code
      status: open
      tdd_phase: red
    - id: AC-3
      title: 'discovered_in reverse: agent promotes a gap it filed in scope'
      status: open
      tdd_phase: red
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
