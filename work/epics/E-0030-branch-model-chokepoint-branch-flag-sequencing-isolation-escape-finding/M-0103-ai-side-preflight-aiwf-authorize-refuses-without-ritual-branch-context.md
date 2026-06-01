---
id: M-0103
title: 'AI-side preflight: aiwf authorize refuses without ritual branch context'
status: in_progress
parent: E-0030
depends_on:
    - M-0102
tdd: required
acs:
    - id: AC-1
      title: AI-actor authorize on main without --branch refuses (branch-context-required)
      status: open
      tdd_phase: red
    - id: AC-2
      title: AI-actor authorize with --branch <missing> refuses (branch-not-found)
      status: open
      tdd_phase: red
    - id: AC-3
      title: AI-actor authorize from ritual-shape checkout (no --branch) accepts
      status: open
      tdd_phase: red
    - id: AC-4
      title: AI-actor authorize with --branch <existing> accepts
      status: open
      tdd_phase: red
    - id: AC-5
      title: --force --reason bypasses preflight (override path)
      status: open
      tdd_phase: red
    - id: AC-6
      title: --force without --reason refuses (regression guard)
      status: open
      tdd_phase: red
    - id: AC-7
      title: Non-AI authorize is unaffected by the preflight
      status: open
      tdd_phase: red
---

## Goal

Make `aiwf authorize <id> --to ai/<agent>` refuse the dispatch when no ritual branch context is in play — either `--branch <name>` is passed naming an existing ritual-shape branch, or the current checkout is already on a recognized ritual-shape branch (matched via `internal/branchparse/` from M-0102). Refusal produces an actionable error pointing at the ritual surface to use and naming the override path explicitly.

## Context

M-0102 added the `--branch` flag, the `aiwf-branch:` trailer, and the `internal/branchparse/` package; this milestone wires the chokepoint behavior that makes [ADR-0010](../../../docs/adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md)'s AI-isolation rule enforceable at the verb level. Together with M-0106's post-hoc kernel finding, this is defense in depth — the preflight blocks the bad dispatch at the source; the kernel finding catches drift that slips through.

Human-actor `aiwf authorize` invocations are unaffected — the preflight only fires when `--to ai/<id>` is in play. Author sovereignty is preserved per ADR-0010.

## Pre-decided design

Per E-0030 §"Design decisions":

- **Branch-context detection:** accept either signal. The preflight passes when (a) `--branch <name>` was supplied and `git show-ref --verify refs/heads/<name>` succeeds, *or* (b) `git symbolic-ref --short HEAD` yields a branch whose name matches one of the ritual shapes recognized by `internal/branchparse/`. Both signals are checked; failure is "neither matched."
- **Error message text** (to be tightened during AC drafting; the seed is below):

  > *"`aiwf authorize <id> --to ai/<agent>` requires a ritual branch context. Either run `aiwfx-start-epic <id>` / `aiwfx-start-milestone <id>` first to land on a recognized ritual branch (`epic/E-NNNN-<slug>` / `milestone/M-NNNN-<slug>` / `patch/g-NNNN-<slug>`), or pass `--branch <name>` naming an existing branch. To override this preflight as a sovereign act, use `--force --reason \"<one-sentence justification>\"`."*

- **Sovereign override:** `--force --reason "..."` bypasses the preflight. The existing trailer-shape rule (`internal/gitops/trailers.go::ValidateTrailer`) refuses `--force` from an `ai/` actor and requires a non-empty `--reason` after trim — so the override is structurally human-sovereign by reuse, not by new code in this milestone.
- **Error code** (for spec-cell coverage in the consolidation milestone): `branch-context-required` (for case 1 in the epic's corner-case catalog) and `branch-not-found` (for case 2). Both surface as typed `Coded` errors per [ADR-0012](../../../docs/adr/ADR-0012-typed-coded-error-pattern-for-legality-pertinent-verb-refusals.md) so machine consumers see them in the JSON envelope.

## Out of scope

- Rituals reorder (M-0104 / M-0105).
- Kernel finding for post-hoc detection (M-0106).
- Spec-cell registration in `internal/workflows/spec/branch/` — that's the consolidation milestone's work.
- Branch *creation* — the preflight only checks existence; cutting the branch is the ritual's job.
- Any changes to the trailer key or flag itself (already shipped in M-0102).
- Changes to human-actor `aiwf authorize` flows (sovereignty preserved).

## Dependencies

- **M-0102** — provides the `--branch` flag, the trailer, and the `internal/branchparse/` helpers this milestone reads.

## Acceptance criteria

<!-- Drafted at `aiwfx-start-milestone M-0103` time. AC seed set:
1. `aiwf authorize <id> --to ai/<agent>` on main (no `--branch`) refuses with a `branch-context-required` Coded error; exit code 1; JSON envelope status=error.
2. `aiwf authorize <id> --to ai/<agent> --branch epic/E-NN-X` against a non-existent branch refuses with a `branch-not-found` Coded error.
3. `aiwf authorize <id> --to ai/<agent>` from a checkout on `epic/E-NNNN-<slug>` (no `--branch`) accepts; the trailer records the current branch.
4. `aiwf authorize <id> --to ai/<agent> --branch epic/E-NNNN-<slug>` against an existing branch accepts.
5. `aiwf authorize <id> --to ai/<agent> --force --reason "..."` bypasses the preflight (override path); commit carries `aiwf-force:` and `aiwf-reason:` trailers.
6. `aiwf authorize <id> --to ai/<agent> --force` without `--reason` refuses (existing rule, regression guard).
7. Human-actor `aiwf authorize <id> --to ai/<agent>` is *unaffected*: a human-actor invocation on main, no `--branch`, succeeds (this preflight only fires when the implicit-or-explicit actor is `ai/`, but in practice `--to ai/` is what triggers it — the preflight branches on the target agent's role, not on the verb's actor).

Note: AC-7 is the kernel-correctness guard — the chokepoint must not regress the existing legitimate human-driven AI-delegation path that doesn't pass --branch (because there isn't one yet today). Once M-0104/M-0105 land, no real-world ritual flow leaves --branch unset; this AC documents the back-compat seam during the migration window.
-->

### AC-1 — AI-actor authorize on main without --branch refuses (branch-context-required)

### AC-2 — AI-actor authorize with --branch <missing> refuses (branch-not-found)

### AC-3 — AI-actor authorize from ritual-shape checkout (no --branch) accepts

### AC-4 — AI-actor authorize with --branch <existing> accepts

### AC-5 — --force --reason bypasses preflight (override path)

### AC-6 — --force without --reason refuses (regression guard)

### AC-7 — Non-AI authorize is unaffected by the preflight

