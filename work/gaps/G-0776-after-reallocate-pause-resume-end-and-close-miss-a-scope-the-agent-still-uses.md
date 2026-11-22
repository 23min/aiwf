---
id: G-0776
title: After reallocate, pause, resume, end and close miss a scope the agent still uses
status: open
discovered_in: M-0364
---
## What's missing

After `aiwf reallocate` gives a milestone a new id, the paths that look up a milestone's scopes
by its id no longer see a scope opened under the old id, while the agent's verbs still do.
`LoadEntityScopes` (`internal/cli/cliutil/scopes.go:40`) selects the commits whose
`aiwf-entity` is the id it is given (`readEntityScopeCommits`, `scopes.go:151`). Pause and
resume (`internal/cli/authorize/authorize.go:409`), the end a closing promote writes
(`loadEndableScopeAuthSHAsForEntity`, `internal/cli/cliutil/provenance.go:254`) and
`aiwf show` (`internal/cli/show/scopes.go:49`) pass it the milestone's current id. The
agent's path, `loadActiveScopesForActor` (`provenance.go:146`), starts from the opener's own
`aiwf-entity` and maps it forward with `ResolveCurrentEntityID` (`scopes.go:209`).

Measured 2026-11-21 in the Linux devcontainer (Go 1.25.12, git 2.51.1) with `aiwf` built from a tree whose code is `main` at 0c00b37c4 (`aiwf version`: `v0.42.1-0.20261121183124-0c00b37c47b3+dirty`), in a disposable repository: `aiwf init` with `hosts: []`, epic `E-0001` active, milestone `M-0001` in progress with two ACs, and `milestone/M-0001-first-milestone` checked out. Ids in the block name entities in that repository.

```
$ aiwf authorize M-0001 --to ai/claude --reason go                          # exit 0
$ aiwf add ac M-0001 --title three --actor ai/claude --principal human/t   # exit 0
$ aiwf reallocate M-0001
aiwf reallocate M-0001 -> M-0002
$ aiwf show M-0002                                                          # lists no scope
$ aiwf add ac M-0002 --title four --actor ai/claude --principal human/t
aiwf add ac M-0002/AC-4 "four"                                               # exit 0
$ aiwf authorize M-0002 --pause hold
aiwf authorize: no active scope on M-0002 to pause                           # exit 2
$ aiwf authorize M-0002 --resume go
aiwf authorize: no paused scope on M-0002 to resume                          # exit 2
$ aiwf promote M-0002 done              # every AC met first; exit 0, no aiwf-scope-ends
$ aiwf show M-0002
    c2b24f1  human/t → ai/claude  state: active   opened 2026-11-21  events: 1
$ aiwf retitle M-0002 "Renamed by agent" --actor ai/claude --principal human/t   # exit 0
```

The agent's `add ac` carried `aiwf-authorized-by` naming the opening commit. `aiwf check`
reported no provenance finding. With `aiwf authorize M-0001 --end --reason over` before the
reallocate, the agent's `add ac` on `M-0002` was refused with `provenance-no-active-scope`.
After the reallocate, `--end` is refused too:

```
$ aiwf authorize M-0002 --end --reason over
aiwf authorize: no non-ended scope on M-0002 to end                          # exit 2
$ aiwf authorize M-0002 --end --scope <opening commit> --reason over
aiwf authorize: no scope on M-0002 matches --scope "d04a1c5"; …              # exit 2
```

and the agent's `add ac` on `M-0002` still exits 0. `authorize.go:409` serves `--end` as well as
pause and resume. The later `aiwf show M-0002` listing comes from show's second path, which finds
scopes through the `aiwf-authorized-by` of commits on the entity (`show/scopes.go:58`–`67`); the
agent's commits on `M-0002` name the opening commit.

## Why it matters

The human loses control of a live delegation while the agent keeps working: the scope cannot be
paused, resumed or ended, the close that ends every scope on its milestone leaves this one active on a
`done` milestone, and the agent's verbs still succeed after that close. Nothing reports any of
it.

A reallocate after work under a scope is how an id collision found when two branches meet is
resolved; G-0118 records one at an epic's wrap.
