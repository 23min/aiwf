---
id: G-0400
title: Stress scenario catalog exercises only 10 of 38 aiwf verbs
status: open
priority: medium
discovered_in: M-0249
---
## What's missing

The 12 scenarios registered in `cmd/stresstest/registry.go` collectively
invoke only 10 of the 38 leaf `aiwf` CLI verbs: `add`, `promote`,
`authorize`, `show`, `check`, `reallocate`, `edit-body`, `cancel`,
`history`, `acknowledge` (verified by grepping every `runAiwfJSON`/
`exec.Command` call site in `internal/stresstest/*.go` against the
CLI's own `root.go` wiring; `lock-kill` drives the separate lockholder
binary and calls no `aiwf` verb at all, and the two `worktree add`
calls in `cross_worktree_edit_body_race.go` are raw `git worktree
add` fixture setup, not the `aiwf worktree add` verb).

Of the verbs wired for diagnostic logging (E-0061, extended by this
milestone's own follow-up work), 15 are never exercised by any
scenario: `move`, `upgrade`, `rename`, `rename-area`, `set-area`,
`retitle`, `rewidth`, `archive`, `import`, `worktree-add`,
`contract-bind`, `contract-unbind`, `contract-verify`,
`contract-recipe-install`, `contract-recipe-remove`. A further 13
verbs (`init`, `update`, `doctor`, `render`, `whoami`, `status`,
`list`, `schema`, `template`, `version`, `contract-recipes`,
`contract-recipe-show`, `milestone-depends-on`) are neither wired nor
exercised.

## Why it matters

The stress harness exists to catch concurrency, durability, and
isolation regressions under real subprocess load. A verb the harness
never calls can regress silently — the diagnostic-logging
instrumentation carries no mechanical assurance for the 15 wired-but-
unexercised verbs, since no scenario drives them concurrently or under
fault injection. Deciding which of those 15 (and which of the 13
untouched verbs, if any) warrant a dedicated scenario — versus which
are read-only or low-risk enough to skip — is a scoping decision for
whoever next extends E-0062's scenario catalog.

## Notes

M-0250 (E-0062) closed part of this: the registered verb-sequence walker (commit `59f00c89`) and a concurrent-move scenario (commit `4b4d14fa`) added real coverage for `move`, `archive`, `rename`, and `retitle`. The registry now carries 16 scenarios (up from 12).

`import`, the `contract-*` sub-verbs, and `worktree add` as a driven verb (as opposed to raw fixture setup) remain unexercised — M-0250's own spec explicitly scopes these out as "open questions for a future milestone," and E-0065's wrap re-confirmed them as still deliberately deferred.
