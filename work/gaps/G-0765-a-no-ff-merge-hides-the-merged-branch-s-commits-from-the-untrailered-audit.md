---
id: G-0765
title: A --no-ff merge hides the merged branch's commits from the untrailered audit
status: open
---
## What's missing

When a branch is merged with `git merge --no-ff`, `aiwf check` on the receiving
branch never puts the merged branch's own commits through the untrailered-entity
audit. `ReadUntrailedCommits` (`internal/cli/check/provenance.go`) reads the push
range with `git log -m --first-parent`, which leaves those commits off the walk,
and `RunUntrailedAudit` (`internal/check/provenance.go`) then skips the merge
commit itself as an ordinary merge. The comment on that skip names the assumption
both rest on: the branch's commits "are audited by the feature branch's own
pre-push hook before the merge can land". Nothing makes it hold. `wf-patch` merges
its branch locally and says the patch branch is normally never pushed, and the
hand edit below commits with aiwf's hooks installed.

`trailer-verb-unknown` reads the same first-parent list and misses the merged
commits too, but the commit-msg hook refuses an unregistered `aiwf-verb:` value
when the commit is made, so that half reaches only a commit made without the
hooks.

Measured at `3c5bd7546` (git 2.51.1, Linux) in an empty directory, with an
`aiwf` built from that commit on `PATH` so aiwf's hooks run. Hook output from
the other setup commands is left out.

```
$ git init -q --bare remote.git && git init -q -b main w && cd w
$ git config user.name t && git config user.email t@example.invalid
$ aiwf init --no-prompt >/dev/null && git add -A && git commit -qm init
$ printf "## What's missing\n\nX.\n\n## Why it matters\n\nY.\n" > ../g.md
$ aiwf add gap --title "A gap" --body-file ../g.md >/dev/null
$ git remote add origin ../remote.git && git push -q -u origin main
$ git checkout -qb patch/G-0001-x
$ echo extra >> work/gaps/G-0001-a-gap.md && git commit -qam "hand edit to the gap"
ok — no findings
$ git checkout -q main
$ git merge -q --ff-only patch/G-0001-x && aiwf check
error provenance-untrailered-entity-commit: commit ba1b215 touched G-0001 with no aiwf-verb: trailer — hint: …

1 findings (1 errors, 0 warnings)
$ git reset -q --hard origin/main && git merge -q --no-ff --no-edit patch/G-0001-x && aiwf check
ok — no findings
$ git push -q origin main
ok — no findings
$ aiwf history G-0001
2026-11-15T12:06:11Z  human/t           add         -             aiwf add gap G-0001 "A gap"  031a5f5
```

## Why it matters

A hand edit to an entity, the case the untrailered-entity audit exists for,
reaches `main` and the remote with no finding at the commit, the merge or the
push, and `aiwf history` for that entity, which lists only trailered commits,
does not list it either. A consumer following the shipped `wf-patch` ritual
merges this way every time, so the pre-push check, the chokepoint the rituals
defer to, misses the edit on the ritual's own path.

## Related

- G-0539 — the same hand edit passes the pre-commit hook.
- G-0679 — a branch's first push, made before it has an upstream, skips the
  audit as well.
- G-0032 — merge commits bypassing this audit, addressed; the G-0231 merge skip
  in `RunUntrailedAudit` passes over an ordinary `--no-ff` merge again.
