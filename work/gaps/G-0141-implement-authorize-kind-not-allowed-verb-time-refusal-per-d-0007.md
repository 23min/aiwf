---
id: G-0141
title: Implement authorize-kind-not-allowed verb-time refusal per D-0007
status: open
discovered_in: M-0123
---
## What's missing

Per **D-0007** (committed in M-0123 phase 1), `aiwf authorize` should refuse
when the scope-entity is not a `KindEpic` or `KindMilestone`. The other four
kinds (gap, decision, contract, ADR) are not delegation targets — they have
no "in-flight" state for an agent to advance.

The spec's `authorizeKindRestrictionRules()` in
`internal/workflows/spec/rules.go` encodes four illegal cells (one per
disallowed kind) with `ExpectedErrorCode: "authorize-kind-not-allowed"`.

### Status: partially addressed (M-0125/AC-2)

The **verb-time refusal landed** during M-0125/AC-2 (commit d5abcf51 on
the merged milestone branch). `internal/verb/authorize.go::authorizeOpen`
now checks the entity's kind and returns:

```go
fmt.Errorf("aiwf authorize: kind %q is not allowed (authorize-kind-not-allowed); only epic and milestone carry autonomous-work scopes", e.Kind)
```

…before any FSM work, for any kind ∉ {Epic, Milestone}.
`TestAuthorize_Open_RefusesNonScopeEntityKind` in
`internal/verb/authorize_test.go` pins the new guard via a 4-case table
test (gap/decision/contract/adr). The behavioral chokepoint the spec
asked for is in place.

The implementation also surfaces in M-0125/AC-2's negative driver: the
four cells participate in end-to-end coverage (verb returns non-zero,
HEAD unchanged, the error substring includes "authorize" and "not
allowed"). The kernel matches the spec at the *behavior* level.

### Why the gap stays open

The AC-5 spec→impl drift policy (`TestM0123_AC5_SpecToImpl_ErrorCodesResolve`
in `internal/policies/m0123_ac5_drift_test.go`) treats a code as
"impl-resolved" only when it appears as a **`Code: "X"` composite-literal
field** in non-test `.go` files under `internal/` — typically a
`check.Finding{Code: "..."}` initialization. Verb-time errors that emit
the code in `fmt.Errorf` text don't match this pattern; the scanner
doesn't see them.

So `"authorize-kind-not-allowed"` is still listed in
`deferredImplErrorCodes` (m0123_ac5_drift_test.go:261). Removing it
would fail the AC-5 test, because the code resolves to neither a
structured-impl literal nor a deferred entry. The behavioral chokepoint
is closed; the **structured-emission contract** the drift policy
enforces is not.

This is the same shape as every other entry in `deferredImplErrorCodes`:
all are verb-time errors that emit codes in `fmt.Errorf` text. Closing
G-0141 cleanly would require a structured-emission pattern for verb
errors — likely either:

  (a) A typed error wrapper (`type CodedError struct { Code string; ... }`)
      that the verb returns instead of `fmt.Errorf`, with the `Code` field
      visible to the AC-5 scanner.
  (b) A code-constant convention (e.g., `const CodeAuthorizeKindNotAllowed
      = "authorize-kind-not-allowed"`) referenced from the verb, with the
      AC-5 scanner extended to recognize the constant declaration.
  (c) Refine the AC-5 scanner to also detect verb-error-message mentions
      (more permissive; risk of false positives if a typo in error text
      happens to contain a code).

(a) is the cleanest design — it parallels `check.Finding{Code: "..."}` and
makes the code first-class data, not a string in human prose. But it
affects every verb error going forward; it's a design choice that warrants
its own discussion (likely a new gap or a follow-up ADR), not a one-off
fix bundled into M-0125 wrap.

### Proposed fix shape

Two-phase closure:

**Phase 1 (done in M-0125/AC-2):** Verb-time refusal at
`internal/verb/authorize.go`, behavior-level chokepoint, pinned by
`TestAuthorize_Open_RefusesNonScopeEntityKind`. Behavior matches the
spec.

**Phase 2 (remaining):** Structured-code emission so the AC-5 spec→impl
drift policy recognizes the impl-side resolution. Pick a pattern from
(a)/(b)/(c) above; apply uniformly across all five `deferredImplErrorCodes`
entries; remove the deferred entries; close the corresponding gaps
(G-0139, G-0140, G-0141, G-0142, G-0143) in one sweep.

Phase 2 is shared work across the five deferred-impl gaps. A new
umbrella entity (epic or gap) that picks up all five in one design
pass is the natural home for it.

## Why it matters

`aiwf authorize G-NNNN --to ai/claude` succeeds today, opening a scope on
a gap entity that has no FSM-driven work surface. The agent has nothing to
advance; the scope sits there until manually paused. A verb-time refusal
saves the operator from the silent-success failure mode.

**Behavioral problem: resolved in M-0125/AC-2.**
**Structured-emission contract: pending Phase 2.**
