---
id: M-0280
title: Regression-guard structural test for the verb scaffolds
status: done
parent: E-0072
depends_on:
    - M-0279
tdd: advisory
acs:
    - id: AC-1
      title: Structural test asserts no verb hand-rolls the diagnostic block
      status: met
    - id: AC-2
      title: Structural test asserts no verb hand-rolls the root/actor prelude
      status: met
    - id: AC-3
      title: Guard test fails red if either scaffold is re-inlined
      status: met
---
# M-0280 — Regression-guard structural test for the verb scaffolds

## Goal

Add an `internal/policies` structural test asserting that no verb hand-rolls either the diagnostic block or the `ResolveRoot → ResolveActor` prelude, so the convergence the two prior milestones achieved cannot silently reappear.

## Context

Once both scaffolds are single-sourced, nothing mechanical stops a future verb from re-inlining them — the property would exist by reviewer vigilance alone. This milestone lands the chokepoint, mirroring the existing `skill-edit-structural-test-backstop` pattern. Sequenced last, since it pins the end state the two migration milestones produce.

## Acceptance criteria

<!-- ACs created via `aiwf add ac`; contracts filled below. -->

### AC-1 — Structural test asserts no verb hand-rolls the diagnostic block

A test under `internal/policies` scans the verb sources and fails if any verb reconstructs the diagnostic block inline (the `ResolveLogger` / `EmitVerbOutcome` wiring appearing outside `cliutil.BeginVerbDiag`), allowing a named, rationale-carrying allowlist for any documented intentional non-member.

### AC-2 — Structural test asserts no verb hand-rolls the root/actor prelude

The same (or a sibling) structural test fails if any verb reconstructs the `ResolveRoot → ResolveActor` prelude inline instead of calling the shared helper, with the same allowlist affordance.

### AC-3 — Guard test fails red if either scaffold is re-inlined

The guard is demonstrably non-vacuous: re-inlining either scaffold into a verb turns the test red, and the test is green on the post-migration tree. (Established the wf-vacuity way — break it, watch it fail — not merely asserted.)

## Constraints

- The test asserts against source structure (the inline `ResolveLogger` / `EmitVerbOutcome` and `ResolveRoot → ResolveActor` patterns appearing outside the shared helpers), not against runtime behavior.
- An intentional non-member (a verb documented as legitimately not routing through a helper) is representable — the guard names an allowlist with a rationale rather than a blanket ban, matching the `internal/cellcoverage` precedent in the layering policy test.

## Out of scope

- Any further extraction — this milestone adds only the guard.
- The G-0227 relocation (the guard should be written so it survives the later `cliutil` split, but accommodating that split is not this milestone's work).

## Dependencies

- Both migration milestones — the guard pins the post-migration state, so it lands after the verbs are migrated (otherwise it fails on the very duplication it is meant to prevent).

## References

- E-0072 — parent epic.
- `skill-edit-structural-test-backstop` — the existing structural-guard pattern this mirrors.
- G-0447 — the convergent-duplication cleanup this seam completes.

## Work log

### AC-1 — Diagnostic-block guard

`PolicyVerbScaffoldSingleSeam` (`internal/policies/verb_scaffold_convergence.go`) detects a hand-rolled diagnostic block — a direct `cliutil.ResolveLogger` / `EmitVerbOutcome` call from a verb under `internal/cli/` instead of routing through `cliutil.BeginVerbDiag` — with `upgrade` allowlisted as the documented non-member and a relocation anchor pinning the primitives to package `cliutil`. Detection is AST-based, keyed on the `cliutil.`-qualified selector, scoped to `internal/cli/`. · commit 8c267f90

### AC-2 — Root/actor prelude guard

Extended the policy with the prelude scaffold: a direct `cliutil.ResolveActor` — or the `ResolveActorWithSource` sibling, closing a same-package dodge — from a verb not routing through `cliutil.ResolvePrelude`. `importcmd` (three-way actor precedence), `whoami`, and `doctor` (identity display via `ResolveActorWithSource`) are allowlisted with rationale; `ResolveRoot` is deliberately unkeyed, since read verbs resolve the root alone. · commit 5e4c3b81

### AC-3 — Non-vacuity

Green-on-migrated-tree assertion wired via `runPolicy` (`TestPolicy_VerbScaffoldSingleSeam` → zero violations, pinning both allowlists to exactly the documented non-members); the red-when-re-inlined half is the per-scaffold fire tests; `TestPolicyVerbScaffold_RelocationAnchor` proves the guard fails loud — naming the relocated primitive — rather than rotting green if a primitive leaves `cliutil`. · commit acb9d26e

Wrap-review corrections landed in commit b3837b00 (see Reviewer notes).

## Decisions made during implementation

No decision recorded as an ADR or `D-NNNN` entity. The two design forks — keying the prelude on `ResolveActor` + the `ResolveActorWithSource` sibling rather than a `ResolveRoot → ResolveActor` co-occurrence, and adding the relocation anchor so a future `cliutil` split fails loud rather than green-vacuously — were resolved by independent review and are recorded under Reviewer notes as the design rationale.

## Validation

- `go build ./...`: clean.
- `go test ./internal/policies/...` (incl. the guard, the real-tree assertion, and the relocation-anchor test): green.
- `make check-fast` (race tests + full `golangci-lint` set + `go vet`): clean.
- `make coverage-gate` (diff-scoped branch coverage + firing-fixture-presence + no-stale-allowlist): green. `PolicyVerbScaffoldSingleSeam` at 97.5% statement coverage — the one uncovered arm is the `WalkGoFiles` filesystem-error path, `//coverage:ignore`d.
- `aiwf check`: 0 error-severity findings on the milestone.

## Reviewer notes

- Independent fresh-context two-lens review before wrap. **Code-quality** (`wf-review-code`): APPROVE — all six load-bearing claims verified by measurement (real-tree green, fire-on-re-inline non-vacuous, both allowlists exactly the documented non-members, 97.5% branch coverage, conventions, allowlist-rationale accuracy). **Design-quality** (`wf-rethink`) on the `PolicyVerbScaffoldSingleSeam` / `verbScaffold` unit: DESIGN-SOUND — the abstraction shape, single-primitive keying, and G-0227 durability all sound, with the relocation anchor called the strongest part.
- A pre-implementation independent design analysis shaped three choices adopted before the first commit: the relocation anchor; keying the prelude on `ResolveActor` + the `ResolveActorWithSource` sibling (a concrete same-package dodge, not a speculative one); and scoping the walk to `internal/cli/` so a non-verb package legitimately calling a primitive cannot false-positive. The rejected alternative — `ResolveRoot → ResolveActor` co-occurrence — would be weaker, since `ResolveRoot` alone is legitimate for read verbs.
- Wrap-review corrections: the finding message now names both seams per scaffold (`BeginVerbDiag` / `BeginReadVerbDiag`; `ResolvePrelude` / `ResolvePreludeEnvelope`) so it can't misdirect a read- or envelope-verb author; `containsString` was replaced by stdlib `slices.Contains`; a walk-continuation test now asserts the per-file parse-error skip does not abort the walk.
- Accepted, precedent-consistent trade-offs (not defects): the allowlist is per-file and blankets both of a scaffold's primitives (matching `atomic_write_chokepoint.go`); a stale allowlist entry can only suppress a finding in an already-exempt file, never hide a re-inline elsewhere. The documented blind spots — an aliased `cliutil` import and a from-scratch reimplementation of a primitive's internals — match the sibling AST policies and are not the copy-paste regression this guard protects against.

## Deferrals

- None. The guard adds no deferred work. G-0447 — the convergent-duplication cleanup this epic executes — remains open by design and closes when the epic wraps, not at this milestone. G-0456 (filed in M-0279) is unrelated to this guard and stays open.
