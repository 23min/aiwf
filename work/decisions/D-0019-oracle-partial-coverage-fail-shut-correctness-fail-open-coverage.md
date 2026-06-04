---
id: D-0019
title: 'Oracle partial-coverage: fail-shut correctness, fail-open coverage'
status: proposed
relates_to:
    - M-0161
    - G-0203
---
## Context

M-0161/AC-3 (G-0203) replaces the current `gitBranchOracle`'s all-or-nothing construction with per-ref fault tolerance and a typed `OracleErrors() []OracleErr` API. The pre-AC-3 shape conflates two states:

- **Pre-AC-3:** `newGitBranchOracle` returns `(*gitBranchOracle, error)` — if any single ref's first-parent index fails to build, the oracle is `nil` and `RunProvenanceCheck` silently swallows the error at [`internal/cli/check/provenance.go`](../../../internal/cli/check/provenance.go) line 72 (`if oracle, oErr := newGitBranchOracle(...); oErr == nil`). The isolation-escape rule then skips entirely. A single stale/corrupt ref disables the rule for the whole repo — silently. An AI escape on a healthy ref is missed because an unrelated ref broke.
- **Post-AC-3:** the oracle is constructed with per-ref tolerance. Refs whose first-parent index fails accumulate into `OracleErrors()` as typed entries (ref + underlying error). Refs that resolved cleanly populate the in-memory index normally. `FirstParentBranches(sha)` returns only the cleanly-indexed branches; the rule still runs on every healthy ref. The accumulated errors surface as a new advisory finding `isolation-escape-oracle-failure`, one per failed ref, so the partial-coverage state is mechanically visible.

The question the decision records: **what is the contract when the oracle has partial coverage?** Two extremes are wrong:

- **Pure fail-open** ("just skip the missing-data SHAs silently"): the rule would silently miss escapes on commits whose branch resolution failed, and the operator would see a clean `aiwf check` while the rule was actually blind to part of the repo. This is the bug AC-3 exists to eliminate.
- **Pure fail-shut** ("if any ref failed, refuse the whole rule"): regression to the pre-AC-3 behavior the per-ref tolerance is designed to remove. A single corrupt ref disables a kernel rule, which violates the load-bearing-rule guarantee for the rest of the repo.

The decision below is the middle position the AC-3 body's "fail-shut on rule correctness; fail-open on rule coverage" framing names but does not fully spell out.

## Decision

**Fail-shut on rule correctness; fail-open on rule coverage.**

Concretely:

1. **Correctness (per-commit firing):** the `isolation-escape` rule does NOT fire on a commit whose branch set the oracle cannot confidently classify. The pre-AC-3 algorithm already does this for the "empty branch set" case (the rule's "unknown branch — silent" branch at [`internal/check/isolation_escape.go`](../../../internal/check/isolation_escape.go) lines 240–242). Post-AC-3, the partial-coverage case is identical from the rule's vantage: refs whose index could not be built contribute zero entries to `branchesBySHA`, so `FirstParentBranches(sha)` returns nil for commits only reachable via failed refs. The rule's existing nil-branch check catches it. No false positives from partial information.

2. **Coverage (rule operation):** the rule still runs on every commit reachable via a successfully-indexed ref. A corrupt `epic/E-9999-...` does not stop the rule from policing commits on `epic/E-0030-...`. The kernel-promise of "the isolation-escape rule polices AI commits against their scope's bound branch" remains live for every ref pair that survived oracle construction.

3. **Partial-coverage visibility (mechanical):** `OracleErrors()` returns one typed entry per failed ref. `RunProvenanceCheck` reads the slice and emits one `isolation-escape-oracle-failure` advisory finding per entry, naming the ref and quoting the underlying error. The advisory is **not** an error severity — the rule's correctness contract above means partial coverage cannot silently miss escapes; the advisory exists for operator visibility, not as a blocker. Operators who see the advisory know to investigate the named ref (delete the stale ref, repack the loose objects, fetch from a remote that still has the ref, etc.).

4. **Whole-enumeration failure is a degenerate case.** When `git for-each-ref refs/heads/` itself fails (corrupted `packed-refs`, permission denied on `.git/refs/heads/`, etc.), there are no refs to name and per-ref tolerance has nothing to operate on. In that case the oracle returns the original whole-repo error from construction; `RunProvenanceCheck` continues to swallow it (matching pre-AC-3 behavior on this path) and the isolation-escape rule does not run. The `isolation-escape-oracle-failure` advisory is per-ref by contract; firing it without a ref name (or firing it once with a synthetic "whole repo" ref) would dilute the advisory's mechanical promise. The right surface for whole-enumeration failures is a separate doctor-level diagnostic, deferred out-of-scope per AC-3 body's edge-cases note.

5. **Reflog-availability and shallow-clone compose with this contract.** AC-3 documents that `OracleErrReflogDisabled` (AC-5 wiring) and shallow-clone refusals (AC-4 wiring) ride the `OracleErrors()` typed slice rather than introducing parallel finding codes. The contract above applies uniformly: a missing reflog means the AC-5 reflog-walk extension cannot fire on the affected ref, the AC-5 rule treats that ref as silent for the reflog dimension, and a single `isolation-escape-oracle-failure` advisory names the capability gap. Same for shallow-clone: the affected branch's commits cannot be classified beyond the clone's depth horizon, the rule does not fire on those classifications, the advisory names the ref + the shallow-depth reason. Each composing AC layers its own typed-error variant; the contract above stays put.

## Concrete sequencing

- **AC-3:** `OracleErr` typed slice (with `Ref` + `Err` fields and a capability tag — `ref-resolution-failed`, `reflog-disabled`, `shallow-clone`); `OracleErrors()` accumulator; new `CodeIsolationEscapeOracleFailure` advisory finding; `RunProvenanceCheck` consumes the slice + emits findings.
- **AC-4:** shallow-clone detection adds an `OracleErr` of kind `shallow-clone` for each ref whose first-parent walk hit the shallow-depth horizon.
- **AC-5:** reflog walk adds an `OracleErr` of kind `reflog-disabled` when `core.logAllRefUpdates=false` is detected at gather time.
- **AC-6, AC-7:** compose without further oracle-error-kinds; the rename/detached cases ride the existing ref-resolution failure path.

## Why not the alternatives

- **Alternative A: pure fail-open (silently skip missing-data SHAs).** Rejected — silently misses real escapes on commits whose branch resolution failed, against the kernel-rule load-bearing guarantee. This is the bug pattern AC-3 exists to eliminate.
- **Alternative B: pure fail-shut (any ref failure disables the whole rule).** Rejected — regression to pre-AC-3 behavior; a corrupt ref disables a kernel rule for the whole repo. Violates the same guarantee from the opposite direction.
- **Alternative C: one big "oracle partial coverage" error finding (single advisory regardless of failed-ref count).** Rejected — the per-ref advisory cardinality is the operator-facing affordance. One per failed ref points at the specific remediation; one rolled-up advisory says "something is wrong, go look" and operators read past it. AC-1's per-cell finding pattern (one finding per affected entity) is the parallel; this AC follows the same shape.
- **Alternative D: introduce a separate finding code per capability kind (`isolation-escape-shallow-clone`, `isolation-escape-reflog-disabled`, `isolation-escape-ref-missing`).** Rejected for AC-3 itself — the typed `OracleErr.Capability` tag carries the kind information into the advisory's hint text, which is enough mechanical surface for the operator to remediate. Introducing parallel codes multiplies the impl-to-spec drift surface (M-0123/AC-5) without buying enumerability that the single code's hint-text shape doesn't already carry. **Note:** AC-4 (shallow-clone) does introduce `isolation-escape-shallow-clone` as a separate code per its AC body; that's a deliberate exception because shallow-clone is the load-bearing surface AC-4 wraps. The general rule remains: ride the typed slice unless the AC body explicitly carves out a new code.

## References

- M-0161/AC-3 (G-0203) body — `OracleErrors() []OracleErr`, fail-shut/fail-open framing, two finding codes
- [G-0203](../gaps/G-0203-branchoracle-firstparentbranches-conflates-lookup-failed-with-no-branches.md) — the gap this decision closes the framing question for
- [`internal/check/isolation_escape.go`](../../internal/check/isolation_escape.go) lines 49–51, 240–242 — the BranchOracle interface + the existing nil-branch silent path the per-ref contract layers onto
- [`internal/cli/check/isolation_escape_oracle.go`](../../internal/cli/check/isolation_escape_oracle.go) lines 57–73 — current all-or-nothing construction
- [`internal/cli/check/provenance.go`](../../internal/cli/check/provenance.go) lines 72–75 — the call site that silently swallows the construction error pre-AC-3
- [M-0106](../epics/E-0019-poc-validation-of-isolation-escape/M-0106-kernel-finding-isolation-escape-closes-g-0099.md) — original isolation-escape landing
