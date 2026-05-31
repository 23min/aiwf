---
id: M-0102
title: aiwf authorize --branch flag + scope-branch trailer coupling
status: in_progress
parent: E-0030
tdd: required
acs:
    - id: AC-1
      title: --branch flag wired on aiwf authorize with Cobra completion
      status: open
      tdd_phase: red
    - id: AC-2
      title: aiwf-branch trailer constant plus git-ref-shape ValidateTrailer rule
      status: met
      tdd_phase: done
    - id: AC-3
      title: aiwf-branch trailer emitted iff --branch passed (backward-compatible)
      status: open
      tdd_phase: red
    - id: AC-4
      title: aiwf-branch sorts between aiwf-scope and aiwf-scope-ends in trailerOrder
      status: open
      tdd_phase: red
    - id: AC-5
      title: internal/branchparse/ package extracted from worktrees.go
      status: met
      tdd_phase: done
    - id: AC-6
      title: --branch completion returns ritual-shape local branches
      status: open
      tdd_phase: red
    - id: AC-7
      title: completion_drift_test passes on new flag without allowlist entry
      status: open
      tdd_phase: red
    - id: AC-8
      title: --branch against non-existent branch not refused at this milestone
      status: open
      tdd_phase: red
---

## Goal

Add `aiwf authorize --branch <name>` flag and the new `aiwf-branch:` commit trailer key recording the scope-branch coupling on the `authorize` commit. Lift `parseEntityFromBranch` and the ritual-shape regexes from `internal/cli/status/worktrees.go:485` into a new `internal/branchparse/` package so M-0103's preflight and the existing `aiwf status --worktrees` correlation share one regex set. Pure additive: optional flag, new trailer, no behavior change when the flag is absent.

## Context

[ADR-0010](../../../docs/adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md) says AI multi-commit work belongs on named ritual branches; this milestone adds the kernel surface that records *which branch* a scope is bound to. Foundation for M-0103's preflight (which refuses dispatch when the coupling is absent) and M-0106's kernel finding (which detects drift away from the recorded branch).

The flag wires through Cobra completion per CLAUDE.md's auto-completion-friendly rule. The trailer key is added to CLAUDE.md § "Commit conventions" so `aiwf history` consumers and downstream tooling see it.

## Pre-decided design

Per E-0030 §"Design decisions":

- **Trailer key:** `aiwf-branch:`. Constant lands in `internal/gitops/trailers.go` alongside `TrailerActor`, `TrailerTo`, `TrailerScope`; included in `trailerOrder` between `TrailerScope` and `TrailerScopeEnds` (the scope's branch is metadata *about* the scope opening). `ValidateTrailer` shape rule: git-ref-shape regex (`^[A-Za-z0-9._/-]+$`, no leading slash, no embedded `..`); a refusal at write time is preferable to a soft find later.
- **Behavior when `--branch` is absent:** backward-compatible no-op. The trailer is emitted *only when* `--branch <name>` is passed. M-0103's preflight is what enforces the chokepoint — this milestone keeps the surface additive.
- **Completion behavior:** `RegisterFlagCompletionFunc("branch", ...)` returns local branches matching the ritual-shape regexes from `internal/branchparse/`. Full-branch-list completion is a smaller hammer (better UX when the operator is intentionally naming a custom branch) but defeats the discoverability win; ritual-shape-only is the right default.
- **`internal/branchparse/` extraction:** lifts `parseEntityFromBranch` and the ritual-shape compiled regexes from `internal/cli/status/worktrees.go:485` plus the helper that maps `branch → (kind, entity-id)`. Both this milestone's flag-completion and M-0103's preflight detection consume it; `worktrees.go` rewires to consume from the new package. One source of truth — by construction, not by review.

## Out of scope

- Refusing the dispatch when `--branch` is absent (that's M-0103, the preflight).
- Auto-creating the branch if absent — default is "require the named branch already exists" per ADR-0010's promote-then-cut sequencing rule. Deferred unless friction surfaces.
- Updates to `aiwfx-start-epic` / `aiwfx-start-milestone` rituals (M-0104 / M-0105).
- Kernel finding for post-hoc detection (M-0106).
- Spec-cell registration in `internal/workflows/spec/branch/` — that's M-0158's consolidation.
- Changes to human-actor `aiwf authorize` flows (sovereignty preserved).

## Dependencies

None — foundational.

## Acceptance criteria

<!-- Drafted at `aiwfx-start-milestone M-0102` time. The catalog below is the AC seed set:
1. `--branch <name>` flag is wired on `aiwf authorize <id> --to ai/<agent>` and round-trips through Cobra completion.
2. `aiwf-branch:` trailer constant lands in `internal/gitops/trailers.go` with the git-ref-shape `ValidateTrailer` rule.
3. The trailer is emitted on the authorize commit iff `--branch` was passed; absent flag = no trailer (backward-compatible).
4. Trailer ordering: `aiwf-branch:` sorts between `aiwf-scope:` and `aiwf-scope-ends:` per `trailerOrder`.
5. `internal/branchparse/` package exists; `parseEntityFromBranch` and the ritual-shape regexes are lifted from `internal/cli/status/worktrees.go:485`; `worktrees.go` consumes from the new package.
6. The flag's completion returns local branches matching ritual-shape regexes from `internal/branchparse/`.
7. `internal/cli/integration/completion_drift_test.go` recognizes the new flag without an allowlist entry.
8. `--branch <name>` against a non-existent branch is *not* refused at this milestone — that's M-0103's job. This milestone's behavior is "record whatever name was passed, validated only against trailer-shape rules."
-->

### AC-1 — --branch flag wired on aiwf authorize with Cobra completion

### AC-2 — aiwf-branch trailer constant plus git-ref-shape ValidateTrailer rule

### AC-3 — aiwf-branch trailer emitted iff --branch passed (backward-compatible)

### AC-4 — aiwf-branch sorts between aiwf-scope and aiwf-scope-ends in trailerOrder

### AC-5 — internal/branchparse/ package extracted from worktrees.go

### AC-6 — --branch completion returns ritual-shape local branches

### AC-7 — completion_drift_test passes on new flag without allowlist entry

### AC-8 — --branch against non-existent branch not refused at this milestone

