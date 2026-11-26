---
id: G-0789
title: promote and cancel --audit-only record a status only the working copy holds
status: open
discovered_in: M-0364
---
## What's missing

`aiwf promote --audit-only` and `aiwf cancel --audit-only` record a status that only the
uncommitted working copy holds. `PromoteAuditOnly` (`internal/verb/auditonly.go:43`) "Refuses
when the entity is not already at newStatus" (`auditonly.go:36-37`) and compares `e.Status`
(`auditonly.go:58`) from the tree `aiwf promote` loads from the working copy
(`tree.Load(ctx, rootDir)`, `internal/cli/promote/promote.go:168`). `CancelAuditOnly`
(`auditonly.go:113`) checks `e.Status` (`auditonly.go:145`) of the tree `aiwf cancel` loads the
same way (`internal/cli/cancel/cancel.go:128`) and passes to it (`cancel.go:141`). The
commit-side write guard returns for each "before examining anything"
(`internal/policies/verb_write_guard_coverage.go:29-30`), since `verb_write_guard_coverage.go:71-73`
marks `CancelAuditOnly`, `PromoteAuditOnly` and `PromoteACPhaseAuditOnly` record-only: "there is
nothing for an operator's working copy to be laundered into" (`verb_write_guard_coverage.go:31-32`).
The legal-workflows audit gives the flag as the backfill
"when state was reached via manual commit; entity must already be at target state"
(`docs/design/legal-workflows-audit.md`, R-RULE-077; R-AUDIT-0215 says the same).

Measured 2026-11-24 in the Linux devcontainer (Go 1.25.12, git 2.51.1) with `aiwf` built from
`main` at f068fa5a1 (`aiwf version`: `(devel)`), in a disposable repository: `aiwf init` with
`hosts: []`, epic `E-0001` active, milestone `M-0001` in progress with two ACs,
`milestone/M-0001-first-milestone` checked out, and `aiwf authorize M-0001 --to ai/claude
--reason go` run. For the promote, both ACs were met first. Ids in this section name entities in
that repository. Lines shown as `…` are left out of the output.

```
$ sed -i 's/^status: in_progress$/status: done/' work/epics/E-0001-engine/M-0001-first-milestone.md   # not committed
$ aiwf promote M-0001 done --audit-only --reason record
aiwf promote M-0001 done [audit-only]                                        # exit 0
$ git show --stat --format=%B HEAD                                           # lists no file
aiwf promote M-0001 done [audit-only]

record

aiwf-verb: promote
aiwf-entity: M-0001
aiwf-actor: human/t
aiwf-to: done
aiwf-audit-only: record
aiwf-scope-ends: ab50698f28d8e405ba4b2e94fe08cd2de6a87df4
$ git show HEAD:work/epics/E-0001-engine/M-0001-first-milestone.md | grep '^status'
status: in_progress
$ git checkout -- work/epics/E-0001-engine/M-0001-first-milestone.md
$ aiwf show M-0001
M-0001 · First milestone · status: in_progress · tdd: none
…
  Scopes (1):
    ab50698  human/t → ai/claude  state: ended    opened 2026-11-24  ended 2026-11-24  events: 2
…
    2026-11-24  promote     → done        aiwf promote M-0001 done [audit-only]
…
$ aiwf check --since main
…
5 findings (0 errors, 5 warnings)                                            # exit 0
$ aiwf retitle M-0001 "Renamed by agent" --actor ai/claude --principal human/t
aiwf retitle: provenance refused: actor "ai/claude" has no active scope authorizing this act (provenance-no-active-scope)
```

`aiwf cancel M-0001 --audit-only --reason record`, run on an uncommitted edit to
`status: cancelled` in a second such repository, does the same: exit 0, a commit that lists no
file and carries `aiwf-scope-ends`, and after `git checkout --` the milestone reads `in_progress`
with its scope `ended`. `PromoteACPhaseAuditOnly` has the same record-only treatment and was not
measured. With the edit committed by `git commit` before the audit-only promote, the same commit
is the backfill the flag is for.

## Why it matters

History then records a move to `done`, or a cancel, that no commit holds. Once the edit is
discarded, as an operator backing out of a hand edit does, the milestone is `in_progress` while
its log says it was promoted or cancelled, and `aiwf check`, which CLAUDE.md makes the authority,
reports nothing about it.

The same commit ends every scope on the milestone, as a close does, so the agent's delegation is
gone on a milestone that never closed, and the agent's verbs are refused until a human opens a
new scope.
