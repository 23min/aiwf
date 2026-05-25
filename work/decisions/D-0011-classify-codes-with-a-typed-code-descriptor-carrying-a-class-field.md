---
id: D-0011
title: Classify codes with a typed Code descriptor carrying a Class field
status: proposed
relates_to:
    - ADR-0012
    - M-0140
    - G-0145
---
## Context

E-0036's AC-5 fourth arm (impl→spec completeness) must enumerate the *legality-pertinent* kernel codes — the codes a verb emits when it refuses an FSM/precondition-illegal action — **independently of the spec**, so the drift policy can assert each is named by ≥1 illegal `spec.Rule`. The enumeration cannot be sourced from the spec or the check is circular (the spec can't be the source of "which codes are legality" while we check the spec against that set). Today codes are bare `string` constants (`const CodeFSMTransitionIllegal = "..."`) scattered across `entity`, `verb`, `contractverify`, and `check`; `check.Finding` carries a `Code` but no class. There is no impl-side way to tell a legality code (`fsm-transition-illegal`, `authorize-kind-not-allowed`) from a structural-integrity code (provenance, contract-verification, frontmatter-shape, id-collision). G-0145 / E-0036 open question 4 deferred the mechanism choice to this milestone, leaning "structural `Class` property of the code, not a hand-maintained allowlist in the drift test."

## Resolution

A legality-pertinent kernel code is a typed **descriptor value** that carries its class intrinsically:

```go
type Class int
const (
    ClassStructural Class = iota // integrity findings: frontmatter shape, id collision, ref resolution, provenance, contract verification
    ClassLegality                // verb-time FSM / precondition refusals named by illegal spec cells
)

type Code struct {
    ID    string
    Class Class
}

var CodeFSMTransitionIllegal = Code{ID: "fsm-transition-illegal", Class: ClassLegality}
```

- The class is a **property of the code value** — one local source of truth at the declaration site. No parallel list and no central registry can drift from it.
- The closed legality set is **enumerated by the existing AST scanner** (`collectImplFindingCodes`), extended to read the `Class:` field — so the set is derived from the same declaration it classifies and cannot diverge. One scan yields both "all codes" (AC-4 / AC-5 spec→impl) and "legality codes" (the fourth arm).
- The behavioral classifier is **derivable** from the descriptor: a `Coded` typed error answers `Class()` by returning its code's `.Class` (`FSMTransitionError.Class()` → `CodeFSMTransitionIllegal.Class`). This gives the idiomatic per-instance classification (à la `net.Error.Timeout()`) with no second source of truth.
- `Code() string` keeps returning the code's `.ID`, so the `Coded` interface (ADR-0012) is unchanged and message / JSON consumers see the same string.

Scope discipline (YAGNI): only legality codes migrate to the descriptor now. Structural-integrity codes stay bare strings until a consumer needs their class; converting them is mechanical when that arrives.

This **realizes** ADR-0012's "named code constants" decision along G-0129's typed-code trajectory — the descriptor *is* the typed code constant ADR-0012 gestured at, not a competing architecture. It governs how M-0139 (cancel codes) and any later legality code declare themselves: as `Code{..., Class: ClassLegality}`, which the fourth-arm chokepoint then forces to be referenced by an illegal spec cell.

## Alternatives considered

1. **Behavioral `Class()` on `Coded` + a co-located `LegalityCodes()` list (rejected).** Idiomatic on the behavioral axis, but carries *two* sources of truth — the methods and the list — reconciled by a consistency test. A consistency test guarding duplicated truth is a band-aid, not rigor; M-0139 would have to update both. Impure.
2. **Central `map[string]Class` registry in a new `internal/codes` package (rejected).** Single-sourced, but the class lives in a side-table divorced from the code declaration (action at a distance) — exactly the "registry / hand-maintained allowlist" shape G-0145 leaned away from, merely relocated from the test to production.
3. **Typed `Code` descriptor (adopted).** The class is intrinsic to the code value; enumeration and behavior both derive from the one declaration; aligned with G-0129. Cost: legality `const`→`var`, `Code()` returns `.ID`, the AC-4 scanner learns the descriptor shape, and a few comparisons gain `.ID` — bounded, and the scanner change is a net improvement (it reads structured codes).

A lighter `type LegalityCode string` subtype (the *type* as the marker) was noted as a fallback — less ripple, codes stay string-like — but rejected for conflating a code's Go type with its semantic class and being binary-only (a third class later would need a third string type).

## Consequences

- M-0139's cancel codes and any future legality code must be declared as `Code{..., Class: ClassLegality}`; the AC-5 fourth arm fails them if no illegal spec cell references them — the intended chokepoint.
- The AC-4 spec→impl scanner is extended once to recognize the descriptor form alongside `const Code* = "..."` and `Code:` composite-literal fields; AC-5's existing arms keep resolving the legality codes.
- `entity.Code(err)` and the `--format=json` envelope (M-0143) are unaffected — they consume the `.ID` string, which is unchanged.
