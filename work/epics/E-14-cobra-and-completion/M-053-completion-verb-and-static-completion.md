---
id: M-053
title: Completion verb and static completion
status: draft
parent: E-14
acs:
    - id: AC-1
      title: aiwf completion bash|zsh emits a sourceable shell script
      status: open
    - id: AC-2
      title: Top-level subverbs auto-complete (aiwf <TAB>)
      status: open
---

## Goal

Ship `aiwf completion <bash|zsh>`, the kubectl/gh-idiomatic verb that emits a sourceable shell completion script. Wire static completion for closed-set values: subverbs, kinds, statuses, format names. After this milestone, `aiwf <TAB>` and `aiwf <verb> --status=<TAB>` both work for fixed values.

## Approach

Cobra's built-in completion generator handles the script generation; the work is mostly adding `ValidArgs` / `ValidArgsFunction` annotations to existing flag definitions. README gets the install one-liner: `source <(aiwf completion zsh)`. Dynamic id completion is deliberately out of scope here — it's the next milestone, with its own moving parts (graceful degradation, drift test).

## Acceptance criteria

### AC-1 — aiwf completion bash|zsh emits a sourceable shell script

### AC-2 — Top-level subverbs auto-complete (aiwf <TAB>)

