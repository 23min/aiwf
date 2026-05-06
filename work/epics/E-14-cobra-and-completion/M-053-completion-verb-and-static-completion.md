---
id: M-053
title: Completion verb and static completion
status: in_progress
parent: E-14
acs:
    - id: AC-1
      title: aiwf completion bash|zsh emits a sourceable shell script
      status: met
    - id: AC-2
      title: Top-level subverbs auto-complete (aiwf <TAB>)
      status: met
    - id: AC-3
      title: --status= auto-completes per-kind closed-set values
      status: met
    - id: AC-4
      title: --format= auto-completes text|json values
      status: met
    - id: AC-5
      title: 'README has install one-liner: source <(aiwf completion zsh)'
      status: open
---

## Goal

Ship `aiwf completion <bash|zsh>`, the kubectl/gh-idiomatic verb that emits a sourceable shell completion script. Wire static completion for closed-set values: subverbs, kinds, statuses, format names. After this milestone, `aiwf <TAB>` and `aiwf <verb> --status=<TAB>` both work for fixed values.

## Approach

Cobra's built-in completion generator handles the script generation; the work is mostly adding `ValidArgs` / `ValidArgsFunction` annotations to existing flag definitions. README gets the install one-liner: `source <(aiwf completion zsh)`. Dynamic id completion is deliberately out of scope here — it's the next milestone, with its own moving parts (graceful degradation, drift test).

## Acceptance criteria

### AC-1 — aiwf completion bash|zsh emits a sourceable shell script

### AC-2 — Top-level subverbs auto-complete (aiwf <TAB>)

### AC-3 — --status= auto-completes per-kind closed-set values

### AC-4 — --format= auto-completes text|json values

### AC-5 — README has install one-liner: source <(aiwf completion zsh)

