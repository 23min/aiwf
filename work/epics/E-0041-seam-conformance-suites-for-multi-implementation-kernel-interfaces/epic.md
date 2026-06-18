---
id: E-0041
title: Seam conformance suites for multi-implementation kernel interfaces
status: proposed
---
# E-0041 — Seam conformance suites for multi-implementation kernel interfaces

## Goal

Make every kernel "unsigned-cheque" interface seam — a `type Foo interface { … }` that admits more than one implementation claiming interchangeability — own a single conformance matrix that proves the implementations actually agree, so silent drift between a production impl and its test double (or between two production impls) fails CI instead of shipping. This converts the kernel's per-implementation-isolated-test posture into the rubric's "one suite, parameterized over implementations" pattern (D2 §"Equivalence tests at seams"), and adds the drift policy that keeps the discipline mechanical for future seams. Closes [G-0222](../../gaps/G-0222-no-shared-conformance-suites-at-unsigned-cheque-interface-seams.md).

## Context

D2 ("equivalence tests at seams") is the recurring **Weak** verdict across the codebase health scorecards (2026-06-04 and 2026-06-16). The leading indicator is the `BranchOracle` cluster: [G-0203](../../gaps/G-0203-branchoracle-firstparentbranches-conflates-lookup-failed-with-no-branches.md)–[G-0207](../../gaps/G-0207-detached-head-handling-untested-in-preflight-and-oracle.md) were **five distinct silent false-negatives in a safety check** (`FirstParentBranches` conflating failure modes; shallow-clone silence; force-push-orphan silence; rename false-positive; detached-HEAD untested), each surfaced as a one-off discovery weeks apart. They surfaced serially because the only thing exercising the production `gitBranchOracle` was its own unit tests — which by construction cannot catch "the oracle answered the question the test asked, but a different consumer asks a subtly different question and gets a wrong answer."

[M-0161](../archive/E-0030-branch-model-chokepoint-branch-flag-sequencing-isolation-escape-finding/M-0161-imagination-driven-hardening-shallow-force-push-rename-detached-trunk.md) (E-0030, done) fixed all five bugs and shipped the typed-error contract ([D-0019](../../decisions/D-0019-oracle-partial-coverage-fail-shut-correctness-fail-open-coverage.md)). But each fix is pinned only by its own per-scenario test; **nothing pins that `fakeOracle` and `gitBranchOracle` continue to agree** as the rule and its consumers evolve. A future change that drifts the production oracle from the test double would reopen the same class one bug at a time. This epic lands the matrix that would have caught the whole class at once, and generalizes it.

The same shape applies to the `PageDataResolver` seam: the rendered HTML site is the most user-facing artifact aiwf produces, and the resolver is the only thing constructing the data the templates consume. Drift between `defaultResolver` (htmlrender) and `cli/render.Resolver` (production) surfaces as "the site looks wrong" — exactly the failure mode CLAUDE.md's "Render output must be human-verified before the iteration closes" rule exists to police, and exactly the one that rule still relies on human discipline to catch. This seam carries a subtlety the matrix must respect (see §"Open question").

## Scope

### In scope

- **`BranchOracle` conformance matrix.** One scenario table parameterized over both implementations (`internal/check`'s `fakeOracle` and `internal/cli/check`'s `gitBranchOracle`), asserting identical answers from `FirstParentBranches`, `OracleErrors`, and `BranchOfSHA` across the M-0161 scenario set: no-branches, shallow clone, force-push orphan, branch rename, detached HEAD, ambiguous merge-base, and per-ref oracle-construction failure. Each scenario expresses its topology once; the git fixture builds the `gitBranchOracle` and the equivalent in-memory state builds the `fakeOracle`.
- **`PageDataResolver` conformance.** A matrix over `defaultResolver` and `cli/render.Resolver` asserting agreement on the methods that are genuinely interchangeable (`IndexData`, `EpicData`, `MilestoneData`, `EntityData`, `KindIndexData`), with the equivalence contract for the deliberately-divergent `StatusData` recorded as a decision (see §"Open question").
- **cue↔jsonschema recipe-validator equivalence.** The contract surface (`aiwf contract`) supports both validators; the matrix asserts "same schema + valid/invalid fixture → same pass/fail across validators." Gated behind `-short` because it exercises external toolchains.
- **Seam-conformance drift policy.** An `internal/policies/` chokepoint that fails CI when an interface with two or more production implementations (or one production impl plus a test double that stands in for production behavior) has no conformance suite. This is the part that earns G-0222's "catches the whole class" framing — without it, the matrices are just more tests rather than a guarantee that future seams inherit the discipline.
- **Cross-link the BranchOracle gaps.** [G-0203](../../gaps/G-0203-branchoracle-firstparentbranches-conflates-lookup-failed-with-no-branches.md)–[G-0207](../../gaps/G-0207-detached-head-handling-untested-in-preflight-and-oracle.md) are already `addressed` (M-0161); this epic re-pins them as named cells in the conformance matrix so a future regression of any one fails a named scenario rather than going silent.

### Out of scope

- **Net-new behavior in any implementation.** This epic is test-architecture plus a chokepoint, not feature work. If the matrix uncovers a real divergence bug, that bug is fixed under the milestone that found it; the goal is not to add capability.
- **Reworking the `StatusData` divergence itself.** The `defaultResolver`'s git-free `nil` return is deliberate. The decision in §"Open question" records what equivalence means at that method; it does not change the resolvers' behavior unless it uncovers a genuine bug.
- **`branchparse` prefix-id coherence** ([G-0198](../../gaps/G-0198-branchparse-regex-accepts-prefix-id-mismatch-epic-m-milestone-e.md)). A separate, unrelated `wf-patch` (regex tightening + the one consumer); not a conformance-seam concern.
- **Discovering new seams beyond the three named here.** The drift policy will surface any uncovered seam mechanically; this epic delivers suites for the three known seams and the policy. A seam the policy flags that this epic did not anticipate is a follow-up, not a scope expansion.

## Constraints

- **No implementation behavior change to satisfy a matrix.** If `fakeOracle` and `gitBranchOracle` disagree, the matrix has found either a real bug (fix it, with the fix as the RED→GREEN evidence) or an over-broad equivalence claim (narrow the contract and record why). The matrix is not "make the fake match the production code by editing the fake until green" — that papers over the signal the suite exists to surface (CLAUDE.md §"Don't paper over a test failure").
- **AC promotion requires mechanical evidence** (CLAUDE.md). Every AC has a Go test or policy that fails if its claim breaks. For the conformance suites, the evidence is the suite itself plus a sabotage check: reverting one implementation's correct behavior makes a named scenario fail.
- **Test the seam, not just the layer** (CLAUDE.md). The conformance suites are themselves the realization of this rule for these interfaces; the drift policy is its generalization.
- **Structural, not substring, assertions** (CLAUDE.md). Resolver-output assertions name the field/section they expect a value in, not a flat substring match over the rendered page.
- **Diff-scoped branch-coverage gate** (G-0067) applies to all new Go code in this epic.

## Open question

**What is the equivalence contract at the `PageDataResolver` seam, given `defaultResolver.StatusData()` returns `nil` by design (so htmlrender needs no git)?** A naive "both impls return identical shapes for every method" matrix would be wrong here — the two impls are intentionally not fully interchangeable. The resolution is a recorded decision (allocated via `aiwfx-record-decision` as a `D-NNN`) during the PageDataResolver milestone, choosing among: (a) the matrix asserts equivalence only on the git-independent methods and pins `StatusData`'s divergence as an explicit documented asymmetry; (b) the matrix drives `defaultResolver` with a fixture that does supply status data; or (c) the seam is re-typed so the capability difference is in the type system rather than a runtime `nil`. This question is resolved inside its milestone, not pre-committed here.

## Success criteria

- Every milestone listed in *Milestones* below is `done`.
- A single conformance suite drives both `BranchOracle` implementations through one scenario table; reverting any one of the M-0161 hardening fixes (shallow detection, rename SHA-fallback, typed `OracleErrors`, …) makes a **named** scenario in that suite fail, not a silent pass.
- A single conformance suite drives both `PageDataResolver` implementations through the methods the recorded decision deems interchangeable; the `StatusData` asymmetry is pinned by an explicit assertion rather than left implicit.
- The cue↔jsonschema recipe equivalence suite runs in the normal `go test` matrix under `-short`-gating and asserts cross-validator pass/fail agreement on a shared fixture set.
- The seam-conformance drift policy fails CI when a qualifying interface lacks a conformance suite, and passes against the live tree once this epic's suites land. Removing one suite makes the policy fire, naming the uncovered interface.
- [G-0222](../../gaps/G-0222-no-shared-conformance-suites-at-unsigned-cheque-interface-seams.md) is `addressed` with a resolver pointing at this epic.
- The PageDataResolver equivalence decision is recorded as a `D-NNN` (or an ADR if it warrants one) and referenced from its milestone.

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| The drift policy's "interface with ≥2 implementations" detection is hard to do statically in Go (finding all concrete types satisfying an interface is non-trivial via AST alone). | Medium | Scope the policy to a curated registry of seams the epic knows about, asserting each named seam has a suite, rather than attempting full static impl-discovery. A registry the epic maintains is a real chokepoint (adding a seam without a suite fails the registry's own coverage check) and avoids the AST impl-resolution rabbit hole. The "discover unknown seams" ambition is explicitly a follow-up, not this epic. |
| The `fakeOracle`/`gitBranchOracle` dual-setup (map state vs git fixture) makes scenarios verbose and the two sides drift in what they model. | Medium | Each scenario defines its topology once as data; helper builders derive both the git fixture and the fake's map from that single description, so the two sides cannot disagree about the scenario's intent. |
| The matrix surfaces a real divergence that is expensive to fix, expanding scope mid-epic. | Low–Medium | Per the constraint above, a real divergence is a bug fixed under the finding milestone with its own RED→GREEN evidence; if the fix is large, it splits into its own milestone rather than inflating an existing one. The matrices ship at the scenario set known today; an expensive newly-found divergence is tracked as a gap, not silently absorbed. |
| The PageDataResolver decision stalls the epic if the equivalence contract is genuinely contested. | Low | The decision is local to its milestone and has three pre-identified options; it does not block the BranchOracle milestone (which lands first and carries the core value). |

## Milestones

<!--
Milestone ids are allocated by `aiwfx-plan-milestones` (aiwf add milestone --epic E-0041);
the candidates below are named, not id-labelled, until then. Sequenced: the BranchOracle
matrix lands first (keystone, best-understood, highest value); the PageDataResolver matrix
reuses its harness shape and carries the equivalence decision; the drift policy plus the
cue↔jsonschema suite plus consolidation land last, generalizing over the established shape.
-->

- **BranchOracle conformance matrix** — one scenario table over `fakeOracle` and `gitBranchOracle`, covering the M-0161 hardening cases; re-pins G-0203–G-0207 as named cells. · depends on: —
- **PageDataResolver conformance + equivalence-semantics decision** — matrix over `defaultResolver` and `cli/render.Resolver`; records the `StatusData`-divergence decision. · depends on: BranchOracle matrix (reuses harness shape)
- **Seam-conformance drift policy + cue↔jsonschema suite + consolidation** — the generalizing chokepoint plus the recipe-validator equivalence suite; closes G-0222. · depends on: both matrices above

## ADRs produced

This epic likely produces no new ADRs — it realizes the rubric's D2 pattern and the kernel's existing "test the seam" rule. The one decision it forces (the `PageDataResolver` equivalence contract) is recorded as a `D-NNN` during its milestone, or promoted to an ADR only if the choice turns out to be architectural (e.g., re-typing the resolver seam).

## References

- [G-0222](../../gaps/G-0222-no-shared-conformance-suites-at-unsigned-cheque-interface-seams.md) — the gap this epic closes (source of the candidate path)
- [G-0203](../../gaps/G-0203-branchoracle-firstparentbranches-conflates-lookup-failed-with-no-branches.md)–[G-0207](../../gaps/G-0207-detached-head-handling-untested-in-preflight-and-oracle.md) — the five BranchOracle bugs the matrix re-pins (all `addressed` via M-0161)
- [D-0019](../../decisions/D-0019-oracle-partial-coverage-fail-shut-correctness-fail-open-coverage.md) — oracle partial-coverage fail-shut/fail-open contract the matrix asserts
- [E-0030](../archive/E-0030-branch-model-chokepoint-branch-flag-sequencing-isolation-escape-finding/epic.md) / M-0161 — the epic that built and hardened the BranchOracle subsystem
- `internal/check/isolation_escape.go` — the `BranchOracle` interface definition
- `internal/cli/check/isolation_escape_oracle.go` — the production `gitBranchOracle`
- `internal/check/isolation_escape_test.go` — the `fakeOracle` test double
- `internal/htmlrender/htmlrender.go` — the `PageDataResolver` interface and `defaultResolver`
- `internal/cli/render/resolver.go` — the production `cli/render.Resolver`
- `docs/pocv3/health-scorecard-2026-06-16.md` — D2 Weak verdict and priority action #4
- CLAUDE.md §"Test the seam, not just the layer" / §"Render output must be human-verified before the iteration closes"
