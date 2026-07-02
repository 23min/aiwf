---
id: G-0339
title: contract subverbs stamp aiwf-verb values their own commit-msg hook rejects
status: addressed
addressed_by_commit:
    - 1853ee7f
---
## What's missing

Four `contract`-family verbs stamp an `aiwf-verb:` trailer whose value omits an ancestor path segment, so it does not match the closed set the trailer-verb check derives from the Cobra command tree. `aiwf contract recipe install` stamps `recipe-install` (`internal/verb/contractrecipe.go:59`), `contract recipe remove` stamps `recipe-remove` (`:100`), `contract bind` stamps `bind` and `contract unbind` stamps `unbind` (`internal/verb/contractbind.go:125,185`). But `enumerateRegisteredVerbs` / `walkVerbs` (`internal/cli/check/verbs.go:84`) hyphen-joins the *full* path, yielding `contract-recipe-install`, `contract-recipe-remove`, `contract-bind`, `contract-unbind`. The stamped value is therefore absent from `registeredVerbs`, and it is not a ritual verb, so both the `commit-msg` hook (`internal/cli/check/commit_msg.go:69`) and `RunTrailerVerbUnknown` reject it.

The root cause is two independent sources of truth for a verb's trailer value: the verb layer hand-writes the string, and the check layer derives it from the Cobra path. They disagree only for the `contract` subverbs whose stamp drops `contract-`. No consistency test asserts that every stamped trailer value is a member of `enumerateRegisteredVerbs(root)`, so the mismatch shipped; `internal/cli/integration/trailer_shape_test.go:161` actively pins the buggy value `recipe-install` as expected.

## Why it matters

Since G-0218 installed the `commit-msg` hook via `aiwf init`/`aiwf update`, and `gitops.Commit` runs `git commit -m` without `--no-verify` (`internal/gitops/gitops.go:113`), each affected verb's own commit trips its own hook and aborts with `git commit: exit status 1`. All four verbs are fully non-functional in any initialized consumer repo — reproduced end-to-end against a fresh `aiwf init` tree. This breaks the framework's own "correctness must not depend on LLM behavior" invariant at the kernel level: a shipped verb cannot complete its single-commit mutation. Before G-0218 the same mismatch was a latent pre-push `trailer-verb-unknown` warning; the hook promoted it to a hard composition-time block. A downstream user hit it on `contract recipe install` against v0.21.0.
