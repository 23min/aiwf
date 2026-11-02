---
id: G-0755
title: A copy of an authorize opening commit opens a second scope
status: open
priority: high
---
## What's missing

aiwf reads every copy of an `aiwf authorize <id> --to <agent>` commit as a scope of its own. `ReplayScopes` (`internal/cli/cliutil/scopes.go`), through which the verbs and `aiwf show` read scopes, opens one scope for each commit carrying `aiwf-verb: authorize` and `aiwf-scope: opened`, keyed by that commit's SHA, and `buildAuthOpenerIndex` (`internal/check/provenance.go`) indexes the openers `aiwf check` resolves the same way. A copy of the opening commit has the same trailers and another SHA, so one grant reads as two active scopes. `aiwf authorize --end --scope` ends the one it names, and `--pause`, which names none, pauses the most recently opened active scope (`mostRecent`, in the same file as `ReplayScopes`); either way the other stays active.

A copy arises without a cherry-pick. Reproduced 2026-11-01 with aiwf built from `main` at `c06ffdaf9` (`vcs.modified=false`), git 2.51.1, Linux, with the hooks `aiwf init` installs (the `ok — no findings` lines are its pre-commit hook). On `main`, `aiwf promote E-0001 active`; then, on `epic/E-0001-some-epic`:

```
$ aiwf authorize E-0001 --to ai/claude
aiwf authorize E-0001 --to ai/claude
exit 0
$ aiwf retitle E-0001 "Retitled by the agent" --actor ai/claude --principal human/peter
aiwf retitle E-0001 -> "Retitled by the agent"
exit 0
$ git checkout -q -b other && git commit -q --allow-empty -m "work on other"
ok — no findings
exit 0
$ git checkout -q main && git commit -q --allow-empty -m "main moves"
ok — no findings
exit 0
$ git checkout -q epic/E-0001-some-epic && git rebase -q main
exit 0
$ git merge -q --no-edit other
exit 0
$ git log --format='%h %s' --grep='authorize E-0001 --to'
f0b4e48 aiwf authorize E-0001 --to ai/claude
cef0246 aiwf authorize E-0001 --to ai/claude
$ aiwf show E-0001 | grep -A3 Scopes
  Scopes (2):
    cef0246  human/peter → ai/claude  state: active   opened 2026-11-01  events: 1
    f0b4e48  human/peter → ai/claude  state: active   opened 2026-11-01  events: 1
$ aiwf authorize E-0001 --pause "hold on"
aiwf authorize E-0001 --pause
exit 0
$ aiwf show E-0001 | grep -A3 Scopes
  Scopes (2):
    cef0246  human/peter → ai/claude  state: active   opened 2026-11-01  events: 1
    f0b4e48  human/peter → ai/claude  state: paused   opened 2026-11-01  events: 2
$ aiwf retitle E-0001 "Retitled while paused" --actor ai/claude --principal human/peter
aiwf retitle E-0001 -> "Retitled while paused"
exit 0
$ git log -1 --format='%(trailers:key=aiwf-authorized-by)'
aiwf-authorized-by: cef0246198984bcb944668250d0ed4edd42d8175
$ aiwf check --since 11a6fd914afc382f23b0bb4f755a6985e34a5a59
epic-active-no-drafted-milestones (warning) × 1 — epic E-0001 is active but has no milestones at status draft
isolation-escape-orphaned-ai-commit (warning) × 1 — AI-actor commit 1ebff00b was orphaned by a non-fast-forward update on "epic/E-0001-some-epic" at 2026-11-01 11:48:54 +0000; entity E-0001, actor "ai/claude"
2 findings (0 errors, 2 warnings)
exit 0
```

The rebase re-created the opening commit as `f0b4e48` and the agent's first retitle as a commit still naming `cef0246`, and the merge brought `cef0246` back, so that pointer resolves. The warning names the agent's pre-rebase retitle `1ebff00`, which the merge also brought back; it is reachable from `HEAD`.

The end behaves the same. In the same environment, with the opening commit cherry-picked onto another branch and merged back, `aiwf authorize E-0001 --end --reason "delegation over"` refused, listing both SHAs as `2 non-ended scopes`; `--end --scope` naming the original ended it (exit 0) and left the copy `active`; the agent's retitle then succeeded (exit 0) with `aiwf-authorized-by` naming the copy, and `aiwf check` reported no error. The automatic end covers both: `aiwf promote E-0001 done` wrote an `aiwf-scope-ends` for each copy, and `aiwf show` reported both `ended`.

## Why it matters

Pausing a scope, or ending it while the work goes on, is how a human withdraws a delegation without closing the entity. With a copy of the opening commit in history, either reaches one copy only. `--pause` pauses one and reports nothing about the other; `--end` without `--scope` lists the two as separate scopes and asks for one, and ending that one leaves the other active. The agent's verbs then keep succeeding under the copy still active.

After an end, `aiwf check` accepts those verbs too, since `provenance-authorization-ended` judges each commit by the copy it names, so the history records acts made after the delegation was withdrawn as authorized and the pre-push hook passes them. `aiwf check` has no rule for pauses, so on that path what the copy defeats is the verb's own refusal.

Nobody has to copy the commit on purpose: in the reproduction, a rebase of the branch carrying the opening commit and a merge of a branch built on its earlier version were enough.

## Relates to

G-0460 records two active scopes on one entity from running `aiwf authorize` twice, with the same consequence for a pause; here one grant is read twice because git copied its commit. G-0752 records a rebase that leaves an ended scope with no end. G-0751 records a rebase that leaves `aiwf-authorized-by` naming an opening commit no longer on the branch; here the merge brings that commit back, and its error does not fire.
