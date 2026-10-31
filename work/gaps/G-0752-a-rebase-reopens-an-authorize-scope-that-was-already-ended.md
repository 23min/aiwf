---
id: G-0752
title: A rebase reopens an authorize scope that was already ended
status: open
---
## What's missing

A rebase reopens an `aiwf authorize` scope that was already ended. A scope ends by a commit whose `aiwf-scope-ends` names the SHA of the scope's `authorize` commit, written either by `aiwf authorize <id> --end` or automatically by the promote or cancel that takes the scope-entity to a terminal status (`docs/design/provenance-model.md`, §"Scope states", line 158). The readers of that trailer count it only where it equals the opener's SHA exactly: the replay the verbs use to find an active scope (`ReplayScopes`, `internal/cli/cliutil/scopes.go:98`, reached from `loadActiveScopesForActor`, `internal/cli/cliutil/provenance.go:145`, through `LoadEntityScopes`), and the indexes `aiwf check` builds for `provenance-authorization-ended` (`buildEndedAtIndex` and `buildEndedByIndex`, `internal/check/provenance.go:663` and `:689`) and for `isolation-escape` (`internal/check/isolation_escape.go:284`). A `git rebase` re-creates the `authorize` commit under a new SHA and leaves the end commit naming the old one, so the re-created scope has no end. The verbs treat it as active, `aiwf show` reports it `active`, and the check finds no end to hold an agent's later commit against, although the model says "`ended` is terminal" and "the human must issue a new authorization" (line 159).

Reproduced 2026-10-28 with aiwf built from `main` at `c06ffdaf9` (`vcs.modified=false`), git 2.51.1, Linux, with the hooks `aiwf init` installs. On `main`, `aiwf promote E-0001 active`. On `epic/E-0001-spike-epic`, `aiwf promote M-0001 in_progress`, then `aiwf authorize M-0001 --to ai/claude --branch milestone/M-0001-spike-milestone`. On the milestone branch, `aiwf authorize M-0001 --end --reason "delegation over"`. A second clone then pushes an unrelated commit to `main`:

```
$ aiwf retitle M-0001 "Spike milestone, retitled" --actor ai/claude --principal human/peter; echo "exit $?"
aiwf retitle: provenance refused: actor "ai/claude" has no active scope authorizing this act (provenance-no-active-scope)
exit 1
$ git fetch -q origin && git rebase -q origin/main && echo rebased
rebased
$ git log --format='%h %s | %(trailers:key=aiwf-scope-ends,valueonly,separator=%x2C)' origin/main..HEAD | grep -v '^$'
89d27b3 aiwf authorize M-0001 --end --scope ea660c0 (ai/claude) | ea660c0aad25b0346f415cfe184514e56ab5f91a
ba40756 aiwf authorize M-0001 --to ai/claude | 
2177cd1 aiwf promote M-0001 draft -> in_progress | 
$ aiwf retitle M-0001 "Spike milestone, retitled" --actor ai/claude --principal human/peter; echo "exit $?"
aiwf retitle M-0001 -> "Spike milestone, retitled"
exit 0
$ git log -1 --format='%h %s%n%(trailers:key=aiwf-actor)%(trailers:key=aiwf-authorized-by)' | grep -v '^$'
2a7d505 aiwf retitle M-0001 -> "Spike milestone, retitled"
aiwf-actor: ai/claude
aiwf-authorized-by: ba40756fe9320b5a9e3f836df6a099b62033997f
$ git push -u origin milestone/M-0001-spike-milestone     # the pre-push hook runs aiwf check
…
7 findings (0 errors, 7 warnings)
…
 * [new branch]      milestone/M-0001-spike-milestone -> milestone/M-0001-spike-milestone
```

The seven warnings are the scratch tree's own (empty body sections, no drafted milestone, a `promote-on-wrong-branch` on the rebased promote, no upstream before the first push); `aiwf check` and `aiwf check --since origin/main` on the pushed branch report no error either. Without the rebase the check does catch the act: an agent commit made by hand after `--end`, naming the original `authorize` commit, draws `provenance-authorization-ended` at error severity.

The automatic end comes back the same way. With `aiwf promote M-0001 done` in place of `--end`, `aiwf show M-0001` reports the scope `ended` before the rebase and `active` after it, and the agent's retitle is refused before (exit 1) and succeeds after (exit 0) with `aiwf check` at no error. On the `aiwfx-wrap-milestone` path, with the agent's verb on the milestone branch, the wrap merge, then `aiwf promote M-0001 done` on the epic branch, a plain rebase of the epic branch onto `main` turns the scope from `ended` to `active` and the agent's retitle succeeds. There `aiwf check` raises one error, the `provenance-authorization-missing` G-0751 records, on the agent's earlier commit; nothing reports the later act.

## Why it matters

Ending a scope, by hand or by closing its entity, is how a human's delegation is withdrawn, and the closing promote of a delegated milestone writes such an end. After a rebase the withdrawal no longer holds: the agent's verbs succeed again, each carries an `aiwf-authorized-by` naming the re-created scope, and `aiwf check` accepts it, so the history records an act made after the delegation was withdrawn as authorized, and the pre-push hook passes it.

## Relates to

G-0751 records the same rebase leaving `aiwf-authorized-by` naming an `authorize` commit that is no longer on the branch. The provenance model addresses a scope by the SHA of its `authorize` commit and leaves force-pushes to "the read-time stale-SHA detection (`provenance-authorization-missing`)" (§"Scope id", line 257); the end commit's `aiwf-scope-ends`, left naming a commit no longer on the branch, draws no finding.
