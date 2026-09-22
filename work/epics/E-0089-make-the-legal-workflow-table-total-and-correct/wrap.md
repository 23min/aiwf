# Epic wrap — E-0089

**Date:** 2026-09-22
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0089-make-the-legal-workflow-table-total-and-correct

## Milestones delivered

- M-0318 — Split every cell by target and correct the contradicted outcomes (merged `0a2a8af87`).
- M-0319 — Move authorize to GlobalRules and sweep the stale entries there (merged `433e16372`).
- M-0320 — Re-key the coverage drivers and reconcile the rejection-layer axis (merged `618a78d81`).
- M-0321 — Declare kind-by-verb applicability and enforce cell totality (merged `51b461c64`).
- M-0322 — Decide the rendered legality reference and ship it if taken (merged `b0b81b491`).

## Changelog entry

### Changed — E-0089: make TDD phase retries preserve evidence and publish workflow legality

Repeating an AC's recorded TDD phase without test metrics now succeeds without
creating a commit. Supplying metrics on a repeat is refused, including explicit
zero counts, so a successful retry cannot silently discard test evidence.

The repository provides a generated workflow legality reference at
`docs/reference/workflow-legality.md`, showing declared transitions, conditions,
outcomes, applicability exclusions and global restrictions. A freshness test
rejects drift from the specification; regenerate it from the repository root with
`go run ./cmd/workflow-reference`.

## Summary

The workflow specification implements D-0077's target-bearing transition model,
separates authorization into global restrictions, and checks declared applicability
and applicable-coordinate coverage. The drivers compare declared requests with
kernel outcomes, distinguishing mutation, convergence and refusal. ADR-0050 settles
the phase-retry behavior needed for those declarations; D-0101 governs publication
of the generated reference.

## ADRs ratified

- ADR-0050 — Converge repeated TDD phases without discarding metrics (accepted).

## Decisions captured

- D-0101 — Publish a generated workflow legality reference (accepted).

## Follow-ups carried forward

- G-0166 — Data-field mutation refusals lack workflow-spec coverage (open).
  Its rescope owns the remaining modeling decision; widening beyond status
  transitions is outside this epic.

## Doc findings

Scoped doc-lint: clean for the changed narrative documentation and generated
reference. The generated reference's links resolve and its regeneration command
matches the development executable. Narrative-guide and source-catalog disposition
remain governed by D-0101; this wrap does not claim those documents were repaired.

## Validation

Measured on 2026-09-22 in the Linux amd64 devcontainer with Go 1.25.11.
`AIWF_COVERAGE_BASE=aacd335a0 make ci` was expected to pass on the epic tree with
its integration correction. The final run exited 0: vet, lint, the full race and
coverage suite, profile-driven policy checks, build and end-to-end self-check
passed. The local integration target `main` at `aacd335a0` was an ancestor of
the epic branch; fetching origin found no missing upstream commits.

The initial run passed the race suite but failed the changed-line coverage check
at `internal/cli/promote/promote.go:188`. The in-process CLI test now exercises
explicit zero metrics and asserts the actual audit-only refusal, so an unrelated
lookup error cannot satisfy it. An overlay removing zero-payload preservation
failed that assertion; the unmodified test passed with race detection. Independent
review approved the correction and confirmed that the assignment is covered.

`bin/aiwf-diag check --since main` was expected to find no errors and exited 0
with 0 errors and 20 advisory warnings before closure. Full local CI does not
include the additional CI-only jobs listed in the Makefile. No browser visual
inspection of the generated Markdown was performed.

## Handoff

The target-bearing specification, coverage policies and generated reference are
ready for mainline integration. The reference lists declarations; it is not an
exhaustive invocation guide or a rule-resolution engine. G-0166 remains the entry
point for extending the model to data-field mutations.
