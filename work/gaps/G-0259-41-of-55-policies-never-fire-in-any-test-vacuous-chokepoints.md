---
id: G-0259
title: 41 of 55 policies never fire in any test (vacuous chokepoints)
status: open
---
## What's missing

A test-quality audit (2026-06-19) found that **41 of 55 policies in `internal/policies/` are "fully dark": their `Violation{}`-construction code never executes in any test.** 3 more are partially dark; only **11 are fully lit** (every firing branch exercised) — and those 11 include the three chokepoints added in the session that surfaced this (`validate-check-is-never-writes`, `layering-direction`, `no-time-now-in-core`), which carry firing fixtures by discipline.

The dark set includes load-bearing chokepoints: `verbs-validate-then-write`, `no-timestamp-manipulation`, `principal-write-sites-guard-human`, `sovereign-dispatchers-guard-human-actor`, `fsm-invariants`, `no-history-rewrites`, `no-signature-bypass`, `no-actor-fields-in-aiwfyaml`, `test-setup-presence`, `trailer-parser-uniqueness`, `read-only-verbs-do-not-mutate`, and the whole milestone-structural family (`m0132-*`, `m0134-*`, `m0137-*`).

These policies pass CI **only because the live tree is clean**. Each is registered as `runPolicy(t, PolicyX)` — a live-tree scan that returns no violations on a clean repo — but **none has a fixture that drives it to *fire***. So there is zero evidence they can detect the regression they exist to catch: if a refactor silently turned one into a no-op, CI stays green and the chokepoint is gone with no signal.

## Why it matters

This is the framework's own principle turned on itself. "Framework correctness must not depend on the LLM's behavior" is enforced *by these policies* — but the policies themselves are unverified, so their correctness currently **does** depend on no one having broken them. A vacuous chokepoint is worse than no chokepoint: it reads as a guarantee in the enforcement table while guarding nothing.

## Root cause

The diff-scoped coverage gate (G-0067) enforces firing-line coverage only on **changed** lines, going forward. Every policy predating G-0067 — the bulk of the corpus — never had its firing path gated. New policies pick up the discipline because the gate forces coverage on their changed firing lines; the back-catalog never paid that toll.

## Proposed fix (two-pronged — complements mutate-hunt, does not duplicate it)

1. **Firing-fixture meta-chokepoint.** A policy test asserting that every registered `PolicyX` has at least one test driving it to return ≥1 violation (a firing fixture). This makes the property total and permanent, not diff-scoped. Trivial for the fixture/file/tree-scanning majority.
2. **Structure-auditors need a different mechanism.** `fsm-invariants`, `trailer-order-matches-constants`, `closed-set-status-via-constants`, etc. audit *hardcoded Go structures*; their firing path can only be exercised by mutating the audited structure. Options per policy: refactor to input-driven (accept the structure as a parameter so a broken fixture can be injected), cover via `mutate-hunt`, or allowlist-with-rationale in the meta-chokepoint. This is `mutate-hunt`'s home turf — `wf-vacuity`'s probe 1 explicitly defers to it.
3. **`mutate-hunt` sweep over `internal/policies/...`** as mechanical corroboration of this audit and the ongoing tool for the structure-auditor subset.

Backfilling ~41 firing tests plus the meta-chokepoint is milestone/epic-scale, not a single patch.

## Method (regenerate the dark list)

Run `go test ./internal/policies/ -coverprofile=cov.out`, expand every count-0 block to its line span, then intersect with the source lines carrying a `Policy: "..."` Violation field. A policy whose every `Policy:` line falls in a count-0 block is fully dark (never fires in any test).

## Source

Surfaced while building the chokepoint trio (`validate-check-is-never-writes`, `layering-direction`, `no-time-now-in-core`); the contrast — new policies lit, back-catalog dark — prompted the audit. Relates to G-0258 (the `wf-vacuity` ritual) and the `mutate-hunt` workflow. Sibling forward-work: strengthening `wf-vacuity` into a mechanical chokepoint and a full-corpus vacuity audit beyond `internal/policies/`.
