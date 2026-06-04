---
id: G-0222
title: No shared conformance suites at unsigned-cheque interface seams
status: open
---
## What's missing

Shared conformance suites at the kernel's "unsigned-cheque" interface seams — places where the code declares an interface (`type Foo interface { ... }`) and lets multiple implementations satisfy it without a single test matrix proving they actually behave equivalently. Today the seams exist, the interfaces are clean, and each implementation is unit-tested in isolation. What's missing is the "one suite, parameterized over implementations" pattern the rubric calls for (D2 §"Equivalence tests at seams").

Two seams that visibly need this today:

- **`PageDataResolver`** under `internal/cli/render/` — `defaultResolver` (a stub used in unit tests) and the production resolver under `cli/render.Resolver` both claim to return identical `IndexData` / `EpicData` / `MilestoneData` / `EntityData` shapes for a given tree. They are tested separately. A fixture-tree-driven matrix would have caught any silent drift the moment one implementation grew a field the other didn't.
- **`BranchOracle`** under `internal/branchparse/` (or wherever it now lives after E-0030) — `fakeOracle` is the test double; `gitBranchOracle` is the production implementation. The G-0203–G-0207 gap cluster documents five distinct bugs in `gitBranchOracle`'s edge-case behavior (FirstParentBranches conflates failure modes; shallow-clone silence; force-push silence; rename silence; detached-HEAD silence). Each of those is a case the fake almost certainly handles "correctly" by coincidence of the fake's simpler model — a shared conformance matrix would have surfaced all five at once instead of as serial bug reports.

A third candidate worth scoping into the same suite:

- **cue↔jsonschema recipe equivalence** — the contract surface (`aiwf contract`) supports both validators. The rubric pattern "same schema + valid/invalid fixture → same pass/fail across implementations" applies cleanly. Probably skippable under `-short` because it exercises external toolchains.

Reconfirmed by the 2026-06-04 codebase health scorecard (D2 verdict: Weak; see `docs/pocv3/health-scorecard-2026-06-04.md`).

## Why it matters

The five `BranchOracle` bugs (G-0203–G-0207) are a leading indicator. Each surfaced as a one-off discovery, weeks apart, because the only thing testing the production oracle was its own unit tests — which by definition could not catch "the oracle answered the question the test asked, but a different consumer would ask a subtly different question and get a wrong answer." A conformance suite running both `fakeOracle` and `gitBranchOracle` through the same scenarios (no-branches, shallow-clone, force-push, rename, detached HEAD, ambiguous-merge-base) would have produced one failing-test commit per bug at the moment the production oracle diverged from the fake. Five gaps, one chokepoint instead.

Same shape applies to `PageDataResolver`: the rendered HTML site is the most user-facing artifact aiwf produces, and the resolver is the only thing that constructs the data the templates consume. A drift between `defaultResolver` (test) and `cli/render.Resolver` (production) would surface as "the site looks wrong" — exactly the failure mode CLAUDE.md's "Render output must be human-verified before the iteration closes" rule was added to police, and exactly the failure mode that rule still relies on human discipline to catch.

## Candidate path

1. Define the conformance scenarios as a slice of `testCase` structs (fixture root, expected shape predicates) per seam.
2. Write one `runConformance(t *testing.T, name string, impl Foo)` helper per seam that drives the implementation through every scenario and asserts the predicates.
3. Each implementation gets a thin test file that calls `runConformance(t, "fake", &fakeFoo{})` and `runConformance(t, "git", &gitFoo{})`. New scenario → both sides exercise it; new implementation → write one line.
4. Wire the conformance suites into the standard `go test ./...` run; gate the cue↔jsonschema one behind `-short` if it pulls in external binaries.
5. Cross-link from the BranchOracle gaps (G-0203–G-0207) to this one — once the conformance suite lands, each of those gaps becomes "this scenario in the conformance matrix fails until fixed."

The unifying lesson the rubric encodes: "two implementations that claim interchangeability owe the codebase a single test matrix that proves it." Without that matrix, every drift is a future gap.
