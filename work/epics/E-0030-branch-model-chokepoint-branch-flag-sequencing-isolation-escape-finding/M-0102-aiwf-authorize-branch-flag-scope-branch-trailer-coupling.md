---
id: M-0102
title: aiwf authorize --branch flag + scope-branch trailer coupling
status: draft
parent: E-0030
tdd: required
---

## Goal

Add `aiwf authorize --branch <name>` flag and a new commit trailer key recording the scope-branch coupling on the `authorize` commit. Pure additive: optional flag, new trailer, no behavior change when the flag is absent.

## Context

[ADR-0010](../../../docs/adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md) says AI multi-commit work belongs on named ritual branches; this milestone adds the kernel surface that records *which branch* a scope is bound to. Foundation for M-0103's preflight (which refuses dispatch when the coupling is absent) and M-0106's kernel finding (which detects drift away from the recorded branch).

The flag wires through Cobra completion per CLAUDE.md's auto-completion-friendly rule. The trailer key is added to CLAUDE.md § "Commit conventions" so `aiwf history` consumers and downstream tooling see it.

## Out of scope

- Refusing the dispatch when `--branch` is absent (that's M-0103, the preflight).
- Auto-creating the branch if absent — default is "require the named branch already exists" per ADR-0010's promote-then-cut sequencing rule. Deferred unless friction surfaces.
- Updates to `aiwfx-start-epic` / `aiwfx-start-milestone` rituals (M-0104 / M-0105).
- Kernel finding for post-hoc detection (M-0106).
- Changes to human-actor `aiwf authorize` flows (sovereignty preserved).

## Dependencies

None — foundational.

## Open questions for AC drafting

- **Trailer key name:** `aiwf-branch:` or `aiwf-scope-branch:`? Pick one; consistency check: existing trailer keys are `aiwf-verb:`, `aiwf-entity:`, `aiwf-actor:`, `aiwf-to:`, `aiwf-scope:`. `aiwf-branch:` is the shorter natural extension; `aiwf-scope-branch:` is more explicit about *whose* branch.
- **Flag completion behavior:** When the operator tabs through `--branch`, what do they see? Existing local branches matching ritual patterns (`epic/E-*`, `milestone/M-*`, `fix/*`, `patch/*`, `doc/*`, `chore/*`)? Or the full local-branch list?
- **Behavior when `--branch` is omitted:** Backward-compatible no-op (current behavior, no trailer emitted)? Or always emit the trailer with the current branch name?

## Acceptance criteria

<!-- Drafted at `aiwfx-start-milestone M-0102` time. -->
