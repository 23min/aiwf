---
id: G-0135
title: Hook path hardcoded at install time; breaks across GOPATH environments
status: open
---
## Problem

`aiwf init` and `aiwf update` materialize git hooks (`pre-commit`, `pre-push`, `post-commit`) that hardcode the absolute path to the `aiwf` binary at install time — e.g., `/Users/peterbru/go/bin/aiwf` on macOS, `/go/bin/aiwf` in a devcontainer.

When the same repo is touched from multiple environments with different GOPATH values — most concretely, a host Mac shell where `GOPATH` is `/Users/peterbru/go` and a devcontainer where `GOPATH` is `/go` — every `aiwf update` invocation overwrites the shared `.git/hooks/` with that environment's path. Subsequent hook fires from the OTHER environment fail with `<absolute-path>: No such file or directory`.

## Surfaced via

E-0033 / M-0123 / phase 2 AC-2 commit. The hook had been regenerated for `/go/bin/aiwf` (devcontainer GOPATH) by another session's `aiwf update` inside the devcontainer; my AC-2 commit attempted from the host Mac shell failed because `/go/bin/aiwf` does not exist there.

## Proposed fix shape

Three candidates, decreasing intrusiveness:

1. Hooks use `command -v aiwf` to resolve the binary at hook-execution time (PATH-relative). Portable; depends on each environment's PATH containing `aiwf`.
2. Hooks use a fixed shim path that each environment's `aiwf init`/`update` writes once (e.g., `$(git rev-parse --git-common-dir)/aiwf-wrapper`), and the wrapper resolves the actual binary per-environment.
3. Hook-update is environment-aware: skip overwrite if the existing hook already points at a binary that exists; fall back to `command -v aiwf` if not.

Option 1 is simplest and matches the "the user owns validators / binaries" posture from `design-decisions.md` §"Contracts" (the engine doesn't ship binaries; PATH lookup is the consumer's job).

## Related

- G-0094 (module path rename) — different root, but same class of "absolute reference baked at install time."
- ADR-0010 §AI chokepoint — branch-aware behavior is on this surface's roadmap; hook portability is adjacent.

## Discipline today

When the hook breaks: re-run `aiwf update` from the environment whose path should be authoritative. Cross-environment workflows pay this cost on every flip.
