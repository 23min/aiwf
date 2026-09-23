# Code health — hold these while writing or changing code

Stack-agnostic forces, and the **single home for rules that apply in every language**. The
per-language modules cover only language-specific tooling and idioms — everything here is
assumed across all of them. Priming, not a manual — reach for the full rubric in the
**[code-health rubric](rubric.md)** when designing a module or reviewing a non-trivial diff.

## Design discipline
- **KISS.** Prefer the boring solution. A few similar lines beat a premature abstraction.
  Avoid cleverness — reflection, metaprogramming, deep generics, control-flow tricks —
  unless the simple version is demonstrably worse.
- **YAGNI.** Don't build for a future that hasn't arrived. No speculative interfaces, no
  "might need it later" knobs, no plugin systems for a single implementation. Add the second
  case when it shows up; abstract on the third.
- **No half-finished implementations.** If a feature lands, it lands tested. Stubs and TODOs
  in shipped code are a smell, not a milestone. No compatibility shims without a named
  removal trigger.
- **The smaller change is usually the right change.**
- **Minimal dependencies.** Justify each new one against real need; don't add a package
  without checking first.

## Data & state
- **Single source of truth.** Each fact lives in one place; derive the rest with pure
  functions. Every cache names its invalidation rule. Two parallel sources drift.
- **Atomic writes.** Persisted state is fully-old or fully-new, never half-written: write a
  sibling temp and rename; for multi-file updates write all temps, then rename in order.
- **Typed, validated boundaries.** Shapes crossing a module or process boundary are named
  types, declared once, validated where they're read — not loose maps or unchecked payloads
  flowing inward, and no blind casts on external data.
- **Internal vs wire format.** Use the language-idiomatic internal casing; the serialization
  boundary owns the conversion to the wire shape and back.

## Errors & logs
- **Errors carry context and identity.** Wrap an error with context as it crosses a
  boundary; compare errors by identity/type, never by string-matching the message.
- **Self-explaining operator errors.** A failure at an operator-facing seam says what
  failed, the current state, and what to do next.
- **Structured logs.** Events have a name and fields, not interpolated prose. One logger per
  process; bare `print`/`console.log` only in throwaway scripts.

## Testing
- **TDD by default for logic/API/data code** — red → green → refactor. Before calling
  anything done, confirm **every reachable branch** has a test that exercises it.
- **Pin behavior, not implementation.** Tests assert what the code does for given inputs,
  not which internals ran. Mock only at process/network/filesystem boundaries — never
  internal functions.
- **Real dependencies over mocks.** A transitional mock needs a tracked reason and a removal
  trigger. Temp dirs, env-var seams, and fakes at the process edge are not mocks.
- **Deterministic.** No network or wall-clock dependence in unit tests; push time and
  randomness to the edges. Test names read as specifications, not implementation notes.
- **Zero warnings.** Treat compiler and linter warnings as errors; a clean build is the
  floor, not the goal.
