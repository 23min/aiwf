---
id: G-0791
title: After a merge, aiwf check flags an agent commit the verb accepted on its branch
status: open
---
## What's missing

`aiwf check` reports an agent commit that the verb accepted, once a merge brings in a
pause or end of its scope made on another branch. `provenanceAuthorizationFindings`
(`internal/check/provenance.go:261`) judges whether a scope had ended (`provenance.go:331`)
or was paused (`provenance.go:346`) by the commit's position in `git log --reverse HEAD`
order (`internal/check/head_history.go:105`), which follows commit dates across merged
branches. The verb run on a branch cut before the pause or end found the scope active,
since that branch's history held neither, and accepted the act.

Measured 2026-11-27 in the Linux devcontainer (Go 1.25.12, git 2.51.1) with `aiwf` built
from `main` at 170e9f45f, in a disposable
repository: `aiwf init` with `hosts: []`, epic `E-0001` active, milestone `M-0001` in
progress, and branches `X` and `Y` both cut from `milestone/M-0001-first-milestone` after
the scope was opened. Commit dates are set one minute apart in the order shown. Ids in
the block name entities in that repository.

```
$ aiwf authorize M-0001 --to ai/claude --reason "claude's delegation"   # scope 0d242d4
$ git checkout X
$ aiwf authorize M-0001 --pause "hold claude"
$ git checkout Y
$ aiwf retitle M-0001 "Retitled by claude" --actor ai/claude --principal human/t
  # exit 0
$ aiwf check                                                           # exit 0
$ git checkout milestone/M-0001-first-milestone
$ git merge --no-ff X
$ git merge --no-ff Y
$ aiwf check
  error provenance-authorization-paused: commit ba0b6ec: aiwf-authorized-by: 0d242d4 references a scope that was paused
  # exit 1
```

With `aiwf authorize M-0001 --end --reason "delegation over"` on `X` in place of the
pause, the merged history draws `error provenance-authorization-ended` on the retitle, on
the same build and on one built from `main` at 0176635c3, before G-0777 added the pause
rule. Run in the other order, the retitle on `Y` a minute before the pause or end on `X`,
the same two branches merge with `aiwf check` at exit 0 in all three runs. They also merge
at exit 0, in the same three runs, when the pause or end is made first but dated a minute
after the retitle.

## Why it matters

The pre-push hook blocks work the agent did within its delegation. A human must run
`aiwf acknowledge illegal <sha>` to push it, which records as illegal an act the verb
accepted, and the finding tells them the scope had ended or was paused when the commit
landed.

Whether `aiwf check` reports the commit turns on which of two commits on separate
branches carries the later date, not on what the agent's branch held when it acted. The
same retitle passes or fails the push depending on the clock.
