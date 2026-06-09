---
id: D-0021
title: M-0161/AC-7 doctor JSON envelope deferred; substring match ships now
status: proposed
relates_to:
    - M-0161
    - G-0207
---
## Context

M-0161/AC-7 (G-0207) closes detached-HEAD handling across four surfaces (preflight, oracle, check, doctor). The AC-7 body matrix at lines 476-484 specifies the doctor surface as:

> | Detached HEAD + `aiwf doctor --format=json` | exit 0; JSON envelope's `findings[]` contains a finding with `code: "detached-head"`, `severity: "advisory"` (asserted structurally against the parsed envelope, NOT via stdout substring) |
> | NOT detached + `aiwf doctor --format=json` | exit 0; JSON envelope's `findings[]` contains NO finding with `code: "detached-head"` (baseline) |

And line 498 calls out the structural-vs-substring distinction:

> The `aiwf doctor` side of this AC uses the JSON envelope's structured findings array, which IS a structural assertion (per the matrix rows above).

The discrepancy with what AC-7 actually shipped: `aiwf doctor` does NOT currently support `--format=json`. The verb emits human-readable text only ([`internal/cli/doctor/doctor.go`](../../internal/cli/doctor/doctor.go) DoctorReport returns `[]string` lines). Adding a JSON envelope would require:

1. Defining a structured `DoctorFinding` schema (code + severity + message + hint).
2. Refactoring every existing doctor check (binary version, env, plugin-mount, config, actor, skills, ids, filesystem, hook, render, rituals, etc.) to emit both text rows AND structured findings.
3. Wiring `--format=json` as a flag on the doctor command and routing emission through it.
4. Updating every doctor test (~30) to assert against the JSON envelope on top of the existing text-row assertions.

That's a substantial restructure — a milestone unto itself, not a sub-AC of M-0161.

## Decision

**Ship AC-7 with substring-match-on-stdout for the doctor surface. Defer the `--format=json` requirement to a future doctor-shape milestone.**

Concrete acceptance trade-off:

- AC-7's preflight + oracle + check surfaces ship at the structural-assertion bar the body specifies (substring-against-stderr-scoped-to-the-error-context per the AC-7 body line 498 exception for verb-time errors; structural exit-code + envelope assertions for `aiwf check`).
- AC-7's doctor surface ships with substring match against the canonical token `detached-head`. The emitted line at [`internal/cli/doctor/doctor.go`](../../internal/cli/doctor/doctor.go) DoctorReport carries `head: detached-head: advisory ...` so the substring is stable and identifying. The E2E tests assert presence/absence of the substring.
- A future milestone landing `aiwf doctor --format=json` will tighten the doctor-side assertions to structural envelope queries (the AC-7 body's original ask). Until then the substring is the available signal.

Rationale for deferring rather than expanding AC-7 to include `--format=json`:

1. **Scope discipline.** AC-7's load-bearing claim is detached-HEAD detection across the surfaces. JSON output shape is orthogonal — a delivery format for a finding the rule already surfaces. The shape isn't where the kernel-correctness contract lives.
2. **Refactor cost.** Doctor JSON output touches every existing check (~12 separate concerns). The refactor reasoning belongs to a milestone that scopes the doctor's emission API explicitly, not a sub-AC that adds one more check.
3. **No immediate operational gap.** Operators get the advisory via human-readable output today; CI pipelines parsing JSON would benefit but no CI pipeline currently consumes `aiwf doctor`'s envelope (no such envelope exists). When the consumer appears, the milestone follows.
4. **AC-5 precedent (D-0020).** The same shape: AC-5's body assumed `aiwf acknowledge-illegal` composition; that composition turned out to be unavailable until a verb extension lands. We deferred to G-0226 + D-0020 rather than expanding AC-5. AC-7 follows the pattern.

## Concrete sequencing

- **AC-7 wrap (now):** record this decision; the AC-7 body matrix rows 6-7 (the JSON-envelope assertions) point at this D-0021; the substring-based E2E tests stay in place.
- **Future doctor-shape milestone:** add `--format=json` to `aiwf doctor`; restructure every check's emission to a typed `DoctorFinding`; tighten the detached-head AC's assertions to structural envelope queries; close out the substring carve-out.
- **In between:** if a real consumer of doctor envelope shape surfaces, file a gap with the use case to bring the milestone forward.

## Why not the alternatives

- **Alternative A: implement `aiwf doctor --format=json` under M-0161/AC-7.** Rejected — conflates the detached-HEAD detection scope (the load-bearing claim of AC-7) with a separate doctor-emission-shape concern that touches ~12 unrelated checks.
- **Alternative B: drop the doctor surface from AC-7 entirely.** Rejected — the detached-HEAD detection genuinely belongs in `aiwf doctor` as a proactive surface. The substring-match assertion is weaker than structural but is the tightest pin available without expanding scope.
- **Alternative C: write the AC-7 body to not promise JSON envelope.** Rejected — the body was written before the implementation revealed the doctor-shape constraint. Updating the body in-flight with a strikethrough + `D-NNN` pointer (the AC-5 cell-5 pattern) is the honest closure.

## References

- M-0161/AC-7 (G-0207) — the cycle that surfaced this constraint
- M-0161/AC-7 body matrix lines 476-484 — the JSON-envelope assertions that this decision defers
- M-0161/AC-7 body line 498 — the existing substring-exception framing this decision extends to the doctor surface
- [`internal/cli/doctor/doctor.go`](../../internal/cli/doctor/doctor.go) DoctorReport — the verb that needs the future JSON output
- D-0019 (AC-3 oracle contract) — fail-shut-on-correctness pattern this composes with
- D-0020 (AC-5 cell-5 deferral) — the precedent of deferring a sub-AC scope claim via `D-NNN`
