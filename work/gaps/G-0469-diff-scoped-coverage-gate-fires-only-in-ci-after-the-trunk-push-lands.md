---
id: G-0469
title: Diff-scoped coverage gate fires only in CI, after the trunk push lands
status: open
priority: medium
---
## What's missing

The diff-scoped coverage gate (`PolicyBranchCoverageAudit`, G-0067) runs only in the `go.yml` `test` job. `scripts/git-hooks/pre-push` runs the lint boundary, the gitleaks secret scan, and the comment history-attrition scan — not the coverage gate. `make coverage-gate` exists but is invoked by hand, is documented in `CLAUDE.md` alone, and appears in no ritual: no wrap skill mentions it.

A statement on a changed line with no covering test is therefore undiscoverable until after the push has landed on trunk.

## Why it matters

8 of the 100 `go.yml` runs preceding 2026-07-30 failed on this gate — a quarter of the red runs in that window, and a cause G-0457 does not name. The gate is correct: its findings name real untested statements. The defect is timing. The finding arrives after the change has crossed the trunk boundary, which is the position the repo's own chokepoint principle exists to avoid — fire as early as the class allows.

The comment history-attrition scan is the direct precedent. It shares the diff-scoped shape and the same base expression, and it is wired into pre-push on the stated grounds that an offender is *"caught here rather than after the trunk push has landed"*. The identical argument applies to coverage and was not applied.

Diagnosis after the fact is awkward too. Once the merge has landed on mainline, `make coverage-gate`'s local base — the merge-base with `origin/main` — resolves to HEAD, so the run reports nothing and the CI finding cannot be reproduced without setting `AIWF_COVERAGE_BASE` by hand.

## Resolution shape

Decide the tier. Pre-push matches the comment scan's precedent but costs a full coverage build, measurably heavier than the comment scan's couple of seconds, so the Go-surface short-circuit carries more weight here. A wrap-ritual step is cheaper to adopt but is operator discipline rather than a chokepoint, and this gap's evidence is that operator discipline alone has not held. The two are not mutually exclusive: the ritual prompts, the hook enforces.

Whichever tier is chosen, the post-merge base-resolution wrinkle should be addressed so a CI finding is locally reproducible.

## Where to fix

- `scripts/git-hooks/pre-push` — the gate chain and its Go-surface short-circuit.
- `internal/policies/branch_coverage_audit.go` — base resolution, if a post-merge base needs to be resolvable without a hand-set environment variable.
- `Makefile` — the cost of `coverage-gate` if it is to run at push time.
- The wrap rituals under `internal/skills/embedded-rituals/`, if the chosen tier is ritual rather than hook.
