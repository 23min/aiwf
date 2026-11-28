---
id: G-0793
title: A multi-line --reason breaks the trailer block of the commit the verb writes
status: open
priority: high
---
## What's missing

A `--reason` that spans lines breaks the trailer block of a verb that writes the reason
into a trailer. Every verb commit composes its message in `gitops.CommitMessage`
(`internal/gitops/gitops.go:36`), which writes each trailer's value unchanged, so each
further line of the reason that starts with a non-space character lands in the trailer
block on its own. A line that is not a trailer leaves the last paragraph unparseable as
trailers, and git then reads the commit as carrying none. A line shaped like a trailer
becomes one. A line that starts with whitespace continues the trailer, and the block
survives. The write-time check `aiwf authorize` and `aiwf acknowledge illegal` run,
`gitops.ValidateTrailer`, requires only that the value be non-empty after trim
(`internal/gitops/trailers.go:260`); `aiwf promote` does not call it. The commit-msg hook
passes the result: `aiwf check --commit-msg` exits 0 on the broken pause message below.

The verbs measured below write the reason into a trailer: `aiwf authorize` when it opens,
pauses or ends a scope (`internal/verb/authorize.go:436`, `:486`, `:557`),
`aiwf acknowledge illegal` (`internal/verb/acknowledgeillegal.go:84`), and a forced
`aiwf promote` through `aiwf-force` (`internal/verb/promote.go:351`). A plain
`aiwf promote` of an acceptance criterion and a closing `aiwf cancel` write the reason into
the message body alone, and a two-line reason left their trailers intact.

Measured 2026-11-28 in the Linux devcontainer (Go 1.25.12, git 2.51.1) with `aiwf` built
from a `git archive` of `main` at 170e9f45f, each run in a fresh disposable repository:
`aiwf init` with `hosts: []`, epic `E-0001` active, milestone `M-0001` in progress with
two open acceptance criteria, and `milestone/M-0001-first-milestone` checked out. Ids here
name entities in those repositories.

```
$ aiwf authorize M-0001 --to ai/claude --reason "claude's delegation"   # scope 89ce265
$ aiwf authorize M-0001 --pause $'hold claude\nthe release is frozen until Monday'
  aiwf authorize M-0001 --pause
  # exit 0
$ git log -1 --format=%B
  aiwf authorize M-0001 --pause

  hold claude
  the release is frozen until Monday

  aiwf-verb: authorize
  aiwf-entity: M-0001
  aiwf-actor: human/t
  aiwf-scope: paused
  aiwf-reason: hold claude
  the release is frozen until Monday
$ git log -1 --format='%(trailers)'

$ aiwf show M-0001
  Scopes (1):
    89ce265  human/t → ai/claude  state: active   opened 2026-11-28  events: 1
$ aiwf retitle M-0001 "Retitled by claude" --actor ai/claude --principal human/t
  aiwf retitle M-0001 -> "Retitled by claude"
  # exit 0
$ aiwf check
  6 findings (0 errors, 6 warnings)
  # exit 0
```

`aiwf history M-0001` does not list that pause. With scope 84227fe open instead and a
second line shaped like a trailer:

```
$ aiwf authorize M-0001 --pause $'hold\naiwf-scope: resumed'
  aiwf authorize M-0001 --pause
  # exit 0
$ git log -1 --format='%(trailers)'
  aiwf-verb: authorize
  aiwf-entity: M-0001
  aiwf-actor: human/t
  aiwf-scope: paused
  aiwf-reason: hold
  aiwf-scope: resumed
$ aiwf show M-0001
  Scopes (1):
    84227fe  human/t → ai/claude  state: active   opened 2026-11-28  events: 1
```

With a second line that starts with spaces, `$'hold claude\n  until Monday'`, git unfolds
the trailer to `aiwf-reason: hold claude until Monday` and the scope is paused.

Ending a scope, here 4c5e933, fails the same way, and `aiwf history M-0001` does not list
the end:

```
$ aiwf authorize M-0001 --end --reason $'delegation over\nthanks claude'
  aiwf authorize M-0001 --end --scope 4c5e933 (ai/claude)
  # exit 0
$ git log -1 --format='%(trailers)'

$ aiwf show M-0001
  Scopes (1):
    4c5e933  human/t → ai/claude  state: active   opened 2026-11-28  events: 1
$ aiwf retitle M-0001 "Retitled by claude" --actor ai/claude --principal human/t
  aiwf retitle M-0001 -> "Retitled by claude"
  # exit 0
```

Opening one opens nothing, and `aiwf history M-0001` does not list the opener:

```
$ aiwf authorize M-0001 --to ai/claude --reason $'claude takes it\nuntil Friday'
  aiwf authorize M-0001 --to ai/claude
  # exit 0
$ git log -1 --format='%(trailers)'

$ aiwf retitle M-0001 "Retitled by claude" --actor ai/claude --principal human/t
  aiwf retitle: provenance refused: actor "ai/claude" has no active scope authorizing this act (provenance-no-active-scope)
  # exit 1
```

An acknowledgement records nothing. Commit f6fd5df is a hand-made `ai/claude` commit under
scope 23c4c58, made after that scope was paused:

```
$ aiwf acknowledge illegal f6fd5df --reason $'accepted\nreviewed with the team'
  aiwf acknowledge illegal f6fd5df
  # exit 0
$ git log -1 --format='%(trailers)'

$ aiwf check
  error provenance-authorization-paused: commit f6fd5df: aiwf-authorized-by: 23c4c58 references a scope that was paused
  7 findings (1 errors, 6 warnings)
  # exit 1
```

A forced promote loses its `aiwf-force`, and `aiwf history M-0001` does not list it:

```
$ aiwf promote M-0001 draft --force --reason $'back to planning\nscope changed'
  aiwf promote M-0001 in_progress -> draft
  # exit 0
$ git log -1 --format='%(trailers)'

$ aiwf check
  work/epics/E-0001-engine/M-0001-first-milestone.md: error fsm-history-consistent/illegal-transition: entity M-0001 status changed in_progress → draft in commit 4732a582 — not a legal milestone FSM
  6 findings (1 errors, 5 warnings)
  # exit 1
```

## Why it matters

A pause or an end the human ran does not take, and nothing reports it. The verb prints its
success line at exit 0, the scope stays active, the agent's next verb succeeds, and neither
the commit-msg hook nor `aiwf check` reports an error or a warning; only `aiwf show` still
lists the scope as active. An opener fails the same way; the agent finds out at its first
verb.

A sovereign act does not count. A forced promote's commit carries no `aiwf-force`, so
`aiwf check` reports the human's override as an illegal transition and the pre-push hook
refuses it. An acknowledgement carries no trailers, so the error it was run to clear
stays.

The history loses the act. A pause, an end, an opener and a forced promote made this way
are missing from `aiwf history M-0001`.
