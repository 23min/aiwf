---
id: ADR-0012
title: Typed Coded error pattern for legality-pertinent verb refusals
status: accepted
---
# ADR-0012 — Typed Coded error pattern for legality-pertinent verb refusals

## Context

E-0033 produced `internal/workflows/spec/rules.go` — a declarative table of every `(kind, state, verb)` cell marked Legal or Illegal, with each illegal cell naming an `ExpectedErrorCode`. The spec deliberately ran ahead of the impl: it names codes the verbs were expected to emit, but the verbs emitted those refusals only as `fmt.Errorf` *prose*. The code lived in the message text, not as machine-readable data. The AC-5 spec→impl scanner recognized a code as impl-resolved only when it appeared as a `Code: "..."` composite literal (the `check.Finding{Code}` shape), so verb-time refusals could not satisfy the resolution arm at all. The shortfall was parked in the `deferredImplErrorCodes` IOU list — honest bookkeeping, but as long as it held legality codes the spec's "verified source of truth" claim carried asterisks.

`check.Finding` already carries its code as a first-class `Code` field: validation findings are machine-consumable data. Verb-time refusals had no equivalent. The two halves of the kernel's legality surface — findings (structural integrity) and verb refusals (FSM/precondition legality) — disagreed on whether a code is data or prose. This ADR records the pattern that closes that gap for legality-pertinent verb errors, settling open question 2 of E-0036 (`CodedError` shape) and providing the foundation M-0138 builds, against which M-0139, M-0140, and M-0141 emit their codes.

The pattern is realized in M-0138 (epic E-0036), closing G-0141 and G-0142.

## Decision

Legality-pertinent verb refusals carry a first-class, machine-readable error code as data, reachable via `errors.As` — mirroring `check.Finding{Code}` so the kernel has one mental model for "this outcome has a code," not two parallel ones. The pattern has five load-bearing parts.

### Behavioral interface — code as a method, not a struct field

Coded errors are modeled as a **behavioral interface**, `Coded interface { error; Code() string }`, in the spirit of `net.Error` and gRPC's `status` — any typed error can advertise its code without sharing a base type. Consumers extract the code with the `entity.Code(err)` helper, which uses `errors.As` to walk the `%w` chain and find the first error implementing `Coded`. The choice is deliberately *not* a struct with a public `Code` field (that couples every coded error to one base type and makes wrapping lose the code) and deliberately *not* a scanner that text-matches the message string (that leaves the code trapped in prose, the very failure this pattern retires). Because extraction goes through `errors.As`, a `Coded` error wrapped with `%w` deep in a call chain still resolves — the code travels with the error, not with the call site that happens to hold it.

### Typed errors — concrete values that preserve message text

Each refusal is a concrete typed error value carrying its own fields, and each implements `Coded`. Two such types are realized in M-0138: `FSMTransitionError` (carrying `Kind`, `From`, `To`, `Allowed`) returned by `entity.ValidateTransition` for an illegal transition of a recognized `(kind, from)`; and `AuthorizeKindError` (carrying `Kind`) returned by `aiwf authorize` when the scope-entity is not an epic or milestone. Crucially, each type's `Error()` method **preserves** the kernel's established message text verbatim. Message-matching consumers — the M-0125 negative-driver cells that assert on substrings, operators reading CLI output — keep working unchanged, while machine consumers read the structured `Code()` instead. The code is added *alongside* the prose, never at its expense; nothing that read the old message breaks.

### Named code constants — one constant per code value

Each code value lives in exactly one named `const Code... = "..."` constant, never as a scattered string literal repeated at call sites. `FSMTransitionError.Code()` returns `CodeFSMTransitionIllegal`; `AuthorizeKindError.Code()` returns `CodeAuthorizeKindNotAllowed`. This advances G-0129's direction — typed code constants as the single declaration site for each kernel code — so a code is renamed or audited in one place, and the spec→impl scanner has a stable AST shape (a `const` declaration whose name is prefixed `Code`) to collect against rather than chasing free string literals.

### Scope — legality refusals only, malformed input excluded

The pattern applies **only** to legality-pertinent verb refusals — the refusals a spec cell names with an `ExpectedErrorCode`. The roughly thirty other `fmt.Errorf` verb errors stay prose; converting them would be speculative work with no consumer, so they are left untouched per YAGNI until a spec rule or envelope consumer references them. Equally important, **malformed-input** errors stay non-`Coded`: when `entity.ValidateTransition` is handed an unknown kind or an unrecognized `from` status, it returns a plain `fmt.Errorf` error, because a malformed argument is an operator mistake, not an FSM-legality refusal the spec enumerates. Drawing the boundary at "legality refusal the spec names" keeps the `Coded` set small, intentional, and one-to-one with the spec's illegal cells.

### Scanner recognition — the AC-5 scanner extended in M-0138/AC-4

For a typed verb error to resolve as impl-side, the AC-5 spec→impl scanner `collectImplFindingCodes` (in `internal/policies/m0123_ac5_drift_test.go`) was **extended** in M-0138/AC-4 to collect `const Code... = "..."` constants alongside the pre-existing `check.Finding{Code}` composite-literal arm. Before that extension the scanner saw only `Code:` struct fields, so a code emitted through a `Coded.Code()` method would never resolve. The added `*ast.GenDecl` arm collects any `const` whose name is prefixed `Code`, which is exactly how `CodeFSMTransitionIllegal` and `CodeAuthorizeKindNotAllowed` reach the scanner without either type carrying a `Code:` field. The extension is load-bearing: removing the two codes from `deferredImplErrorCodes` keeps `TestM0123_AC5_SpecToImpl_ErrorCodesResolve` green only because the scanner now sees the constants.

## Consequences

- Verb-time legality refusals are now `errors.As`-able structured data, on the same footing as `check.Finding{Code}`. The kernel has one mental model for "this outcome carries a code."
- The two foundation codes (`fsm-transition-illegal`, `authorize-kind-not-allowed`) leave `deferredImplErrorCodes`; the IOU list shrinks toward its E-0036 target of only `ac-evidence-missing`.
- Downstream milestones (M-0139, M-0140, M-0141) emit their legality codes through this pattern rather than inventing parallel mechanisms; the pattern is the keystone the epic's later milestones depend on.
- Message-matching consumers are unaffected because `Error()` preserves the established text; the cost is the small discipline of keeping the message and the code in sync within each typed error.
- The pattern does not by itself surface codes in the `--format=json` envelope; that representation (and its exit-code treatment) is settled separately in M-0143.

## Alternatives considered

1. **(a) Typed coded error — code carried as data (adopted).** A typed error implements `Coded` and exposes its code via `Code()`; `entity.Code(err)` extracts it through `errors.As`. Mirrors `check.Finding{Code}` so there is one mental model. This is the chosen option.
2. **(b) Constant-only.** Define the code as a bare named constant and keep emitting the refusal as `fmt.Errorf` interpolating that constant into the message. Rejected: the constant tidies the declaration but the code is still trapped in prose at runtime — a consumer cannot read it back as data without text-matching the message, which is the failure mode this pattern exists to retire.
3. **(c) Permissive scanner.** Leave the verbs emitting prose and broaden the AC-5 scanner to text-match the code inside `fmt.Errorf` message strings. Rejected for the same root reason: it makes the *test* pass while leaving the *runtime* code trapped in prose, so the JSON envelope and any machine consumer still cannot read a structured code. It would rubber-stamp the gap rather than close it.

Both (b) and (c) leave the code unreachable as data; only (a) carries the code as data the way `check.Finding{Code}` already does, which is why the kernel ends with one mental model instead of two.
