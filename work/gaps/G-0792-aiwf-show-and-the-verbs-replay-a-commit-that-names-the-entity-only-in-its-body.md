---
id: G-0792
title: aiwf show and the verbs replay a commit that names the entity only in its body
status: open
priority: medium
---
## What's missing

The scope replay the verbs and `aiwf show` read takes in a commit whose message body names
the entity, though its trailers name another. `cliutil.LoadEntityScopes`
(`internal/cli/cliutil/scopes.go:23`) replays the commits `readEntityScopeCommits` returns,
and that runs `git log --grep "^aiwf-entity: <id>$"` (`scopes.go:60`), which matches the
line anywhere in the message. It keeps each matched commit without checking its
`aiwf-entity` trailer, so another entity's scope commit with an `aiwf-entity: <id>` line in
its body replays as one of `<id>`'s. Its callers include the verbs' provenance gate
(`internal/cli/cliutil/provenance.go:164`), the scope ends a closing promote or cancel
writes (`provenance.go:254`), `aiwf authorize`'s pause, resume and end
(`internal/cli/authorize/authorize.go:409`) and `aiwf show`
(`internal/cli/show/scopes.go:49`). `aiwf check` groups commits by the trailer itself
(`internal/check/provenance.go:736`), and so does render
(`internal/cli/render/singlepass.go:85`), whose index is documented as reproducing
`LoadEntityScopes` (`singlepass.go:30`).

Squashing aiwf commits with `git rebase -i` makes such a commit: the trailers of every
commit but the last become body lines.

Measured 2026-11-28 in the Linux devcontainer (Go 1.25.12, git 2.51.1) with `aiwf` built
from a `git archive` of `main` at 170e9f45f, in disposable repositories: `aiwf init` with
`hosts: []`, epic `E-0001` active, milestone `M-0001` in progress under it, a second epic
`E-0002`, `proposed`, created on the milestone branch with `aiwf add epic --force --reason`,
that `M-0001` does not reach, and `milestone/M-0001-first-milestone` checked out.
Ids here name entities in those repositories.

```
$ aiwf authorize M-0001 --to ai/claude --reason "claude's delegation"   # scope 4de9f09
$ aiwf authorize E-0002 --to ai/codex --reason "codex on the other epic" --force
$ aiwf retitle M-0001 "First milestone, retitled"
$ aiwf authorize E-0002 --pause "hold codex"
$ git rebase -i HEAD~2          # squash the pause into the retitle
$ git log -1 --format=%B
  aiwf retitle M-0001 -> "First milestone, retitled"

  aiwf-verb: retitle
  aiwf-entity: M-0001
  aiwf-actor: human/t

  aiwf authorize E-0002 --pause

  hold codex

  aiwf-verb: authorize
  aiwf-entity: E-0002
  aiwf-actor: human/t
  aiwf-scope: paused
  aiwf-reason: hold codex
$ git log -1 --format='%(trailers:only)'
  aiwf-verb: authorize
  aiwf-entity: E-0002
  aiwf-actor: human/t
  aiwf-scope: paused
  aiwf-reason: hold codex
$ aiwf show M-0001
  Scopes (1):
    4de9f09  human/t → ai/claude  state: paused   opened 2023-11-14  events: 2
$ aiwf retitle M-0001 "Retitled by claude" --actor ai/claude --principal human/t
  aiwf retitle: provenance refused: actor "ai/claude" has no active scope authorizing this act (provenance-no-active-scope)
  # exit 1
$ aiwf render --format=html --out site       # site/M-0001.html, Provenance tab
  SHA Agent Principal Opened Ended State Events 4de9f09c ai/claude human/t 2023-11-14 — active 1
$ git commit --allow-empty --no-verify -m "aiwf edit-body M-0001" --trailer "aiwf-verb: edit-body" --trailer "aiwf-entity: M-0001" --trailer "aiwf-actor: ai/claude" --trailer "aiwf-principal: human/t" --trailer "aiwf-on-behalf-of: human/t" --trailer "aiwf-authorized-by: 4de9f09"
$ aiwf check
  9 findings (0 errors, 9 warnings)
  # exit 0
```

The same steps without the squash leave claude's scope active in `aiwf show M-0001`, and
its retitle exits 0.

With the opener squashed instead, the retitle followed by
`aiwf authorize E-0002 --to ai/codex --reason "codex on the other epic" --force`,
`aiwf show M-0001` lists codex's `E-0002` scope among `M-0001`'s. The gate still refuses
codex's retitle of `M-0001` as outside every scope's reach. Cancelling `M-0001`, after
cancelling its two acceptance criteria, ends codex's scope:

```
$ aiwf show M-0001
  Scopes (2):
    4cb802b  human/t → ai/claude  state: active   opened 2023-11-14  events: 1
    548182c  human/t → ai/codex  state: active   opened 2023-11-14  events: 1
$ aiwf cancel M-0001 --reason "dropped"
  aiwf cancel M-0001 -> cancelled
  # exit 0
$ git log -1 --format='%(trailers)'
  aiwf-verb: cancel
  aiwf-entity: M-0001
  aiwf-actor: human/t
  aiwf-scope-ends: 4cb802b1dc02d42b324064ca5162d4fb1bf36225
  aiwf-scope-ends: 548182ccbf311c93a7ae9a81784b408960d95597
$ aiwf show E-0002
  Scopes (1):
    548182c  human/t → ai/codex  state: active   opened 2023-11-14  events: 1
$ aiwf retitle E-0002 "Retitled by codex" --actor ai/codex --principal human/t
  aiwf retitle E-0002 -> "Retitled by codex"
  # exit 0
$ aiwf check
  error provenance-authorization-ended: commit 2519868: aiwf-authorized-by: 548182c references a scope that was already ended
  # exit 1
```

Without the squash, the cancel writes one `aiwf-scope-ends`, for claude's scope, and
`aiwf check` exits 0 after codex's retitle of `E-0002`.

## Why it matters

An agent loses a delegation no one withdrew. After the squashed pause, `ai/claude`'s
retitle of `M-0001` is refused with `provenance-no-active-scope`, and `aiwf show` reports
its scope paused, while the only pause in history names `E-0002`. Every verb that commits
through `cliutil.DecorateAndFinish` goes through the same gate (`provenance.go:309`).

Closing one entity ends a delegation on another. After the squashed opener, cancelling
`M-0001` writes an end for codex's `E-0002` scope. The verbs keep accepting codex's work on
`E-0002`, whose replay never sees the end, and `aiwf check` reports codex's retitle as
`provenance-authorization-ended`, so the pre-push hook blocks work the verb accepted.
