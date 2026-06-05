---
id: D-0024
title: 'M-0162/AC-4 bijection split architecture: static AST + runtime post-hook'
status: proposed
---
## Question

M-0162/AC-4 body line 236 specifies: *"A new bijection meta-test at `internal/policies/branch_cell_bijection_test.go` (under `//go:build testpins`) enforces four invariants between `branch.Rules()` and the `branchtest.Pins()` registry."*

The body names a single test location, a single build tag, and a single data source. The implementation can only deliver this if the test binary running the bijection check is also the test binary that populates `branchtest.Pins()`. Should M-0162/AC-4 ship as specified, or as a different architecture that delivers the same 4 invariants via different mechanisms?

## Decision

**Split the bijection enforcement across static AST scan (in policies) + runtime registry read (in integration).** The shipped architecture:

- **Static (policies, no build tag)**: `internal/policies/m0162_ac4_bijection_test.go::TestM0162_AC4_Bijection` runs under default `go test`. Walks every `*_test.go` file under `internal/` via `go/parser` AST and extracts Pin call literals (`pinCell("...", ...)`, `branchtest.Pin("...", ...)`, dynamic-prefix forms via `BinaryExpr`, and `CellID:` struct field literals). Enforces invariants 1, 2, 3 against this static mirror of the runtime registry.

- **Runtime (integration, `//go:build testpins`)**: `internal/cli/integration/bijection_posthook_testpins_test.go::bijectionPostHook` runs after `m.Run()` returns in `setup_test.go`'s `TestMain` epilogue. Reads `branchtest.Pins()` directly. Enforces invariant 4 (no test function pins 2+ cells) at the `t.Name()` granularity that static AST cannot resolve.

- **Sabotage discrimination**: synthetic-data subtests in `internal/policies/m0162_ac4_sabotage_testpins_test.go` cover invariants 1, 2, 3 against `evaluateBijection`. Invariant 4 sabotage proved via live demonstration during the AC-4 audit (deliberate double-pin produced expected post-hook violation + `os.Exit(1)`).

- **Allowlist**: 14 entries at `bijectionAllowlist()` for M-0158/M-0161-era named cells whose primary tests live outside `internal/cli/integration/`. Each entry's "primary test TestX in internal/<dir>/" claim is mechanically verified by `TestM0162_AC4_AllowlistClaimsResolve`.

## Reasoning

- **Process-isolation constraint.** `branchtest.Pins()` is a per-process registry. The policies test binary is a different process from the integration test binary. A policies-package test reading `Pins()` sees an empty map because no integration tests ran in its process — would emit false positives on every cell, defeating the bijection check entirely. Static AST sidesteps the process boundary by reading source files, not process state.

- **Invariant 4 inherent runtime nature.** Invariant 4's identifier (`t.Name()`) is a runtime value. Static analysis can attribute pin calls by source position (file:line) but cannot resolve which `t.Run` subtest's name a call records at runtime. Static-only enforcement of invariant 4 would require duplicating Go's subtest semantics in the AST walker — significantly more complex than running the actual tests and reading the actual registry. The runtime portion stays in integration, where Pin calls happen.

- **Body's "branchtest.Pins() registry" phrasing is honored by the runtime portion.** The post-hook reads `Pins()` exactly as the body specifies. The cell-side invariants (1, 2, 3) are statically equivalent because every pinCell/branchtest.Pin call recorded to the registry also appears as an AST literal — the two views are isomorphic by construction.

- **CI exercise pattern.** `.github/workflows/go.yml` runs `go test -tags testpins ./...` which exercises BOTH portions: the static check via the policies test binary, the runtime check via the integration test binary's TestMain post-hook. Local devs run the same via `make test-pins`.

- **Architectural honesty.** The deviation from the body is significant enough to deserve an explicit decision record rather than living only in test-file doc-comments. Recording the choice here makes the engineering trade-off discoverable by future contributors (and AI assistants) reading the entity tree.

## Alternatives considered

- **(a) Single test in policies reading `Pins()` per body literal text.** Rejected: process isolation makes this impossible without spawning a sub-process. The sub-process approach was prototyped briefly during AC-4 — it requires duplicating Go's test runner, materializing fixtures, and reading exit status. Cost-prohibitive vs the split.

- **(b) Move all bijection enforcement to integration package.** Rejected: the body's location hint at `internal/policies/` is meaningful — policies is the AI-discoverable "framework correctness invariants" home. Burying the static cell-side bijection in integration would hide it from grep-for-policies workflows.

- **(c) Drop invariant 4 entirely; document as static-only scope.** Initially considered (the AC-4 first close shipped this way). The audit flagged the silent defer as a half-finished implementation. The runtime portion was added in `e4b22935`; this decision formalizes that fix as the architecture, not a patch.

## Status

Recorded at M-0162 milestone-wide reviewer audit (R1-B2, R2-B1, R2-B3 findings consolidated). Status starts at `proposed`; promotion to `accepted` recommended at M-0162 milestone wrap since the architecture is now load-bearing in shipped code.

## References

- M-0162 body §"Closure notes" — the milestone's reconciliation between body and shipped code.
- M-0162/AC-4 body §"Observable behavior" line 236 — the original single-test specification this decision reconciles against.
- `internal/policies/m0162_ac4_bijection_test.go` — static portion.
- `internal/cli/integration/bijection_posthook_testpins_test.go` — runtime portion.
- `internal/cli/integration/setup_test.go` — TestMain epilogue wiring.
- `internal/policies/m0162_ac4_sabotage_testpins_test.go` — sabotage subtests.
- `internal/policies/m0162_ac4_allowlist_verification_test.go` — allowlist prose verification (R1-T4 fix).
- M-0162 milestone-wide reviewer audit (3 subagents, 2026-06-05) — the audit that surfaced the body/code divergence as a deserving-its-own-entity decision.
