---
id: G-0744
title: The untrailered-entity audit trusts any commit carrying an aiwf-verb trailer
status: open
priority: high
---
## What's missing

`RunUntrailedAudit` (`internal/check/provenance.go:487`) skips every commit whose git-parsed `aiwf-verb` trailer is non-empty (the `if` at line 513) before it walks the entity paths the commit touched. A change to an entity the commit's trailers do not name is therefore never reported by this audit. The same change in a commit with no trailers raises `provenance-untrailered-entity-commit` at error severity, so another entity's trailer is enough to silence a blocking finding. Other rules still apply to the change: an illegal status move is caught by `fsm-history-consistent`.

Ordinary git use reaches this. Git parses trailers only from a message's last paragraph, and `git rebase -i` ends a combined commit's message with the trailer block of whichever commit's message comes last. Reproduced 2026-10-25 with aiwf built from `1e8e17e6c`, git 2.51.1, Linux, in a fresh `aiwf init` repository holding two gaps:

```
git tag base
echo "edited by hand" >> work/gaps/G-0002-second.md
git add work/gaps/G-0002-second.md
git commit --no-verify -m "tweak second gap"
aiwf edit-body G-0001 --body-file <file>
aiwf check --since base | tail -1
GIT_SEQUENCE_EDITOR="sed -i '2s/^pick/squash/'" GIT_EDITOR=true git rebase -i base
aiwf check --since base | tail -1
git show --name-only --format= HEAD
git log -1 --format=%B | git interpret-trailers --parse
aiwf history G-0002
```

```
before squash: 1 findings (1 errors, 0 warnings)
after squash:  ok — no findings
  touches:  work/gaps/G-0001-first.md work/gaps/G-0002-second.md
  trailers: aiwf-verb: edit-body aiwf-entity: G-0001 aiwf-actor: human/peter
  aiwf history G-0002:
    2026-10-25T13:16:58Z  human/peter       add         -             aiwf add gap G-0002 "Second"  e6f9418
```

With an upstream on a bare remote and the installed hooks, `git push` exits 1 before the squash and 0 after it.

Whether the combined commit keeps the verb's trailers depends on the order and the rebase command. Of the four combinations of a hand edit and a verb commit, two keep them and pass the audit, and two drop them and raise the error:

```
hand-then-verb, squash: commits=1 trailers=[edit-body] check: ok — no findings
hand-then-verb, fixup: commits=1 trailers=[] check: 2 findings (2 errors, 0 warnings)
verb-then-hand, squash: commits=1 trailers=[] check: 2 findings (2 errors, 0 warnings)
verb-then-hand, fixup: commits=1 trailers=[edit-body] check: ok — no findings
```

Some verbs touch entities their trailers do not name. In the same environment, an `aiwf reallocate` commit rewrote a referring gap's file while its trailers named only the renumbered entity (`aiwf-entity` with the new id, `aiwf-prior-entity` with the old one), and an `aiwf archive --apply` commit carried no `aiwf-entity` trailer at all. Which entities a trailered commit may legitimately touch depends on its verb.

## Why it matters

A hand edit to an entity, which the audit exists to stop at push, reaches the default branch once a rebase folds it into a verb commit whose message ends the combined message: a squash into a later verb commit, or a fixup into an earlier one, the shape `git commit --fixup` followed by `git rebase -i --autosquash` makes.

Afterwards `aiwf history` of the edited entity has no record of the change, so a reader of that history cannot tell the entity was edited or by whom.

## Relates to

G-0480 (wontfix) states the same skip for frontmatter fields, including one unnamed-entity route: a nested milestone's frontmatter riding along inside a parent epic's directory move. It was closed with "declined: needs the verb-to-owned-fields map ADR-0038 defers, and G-0475 covers the one route that silences a blocking rule". The reproduction here is a body edit, outside G-0480's frontmatter scope, and it silences a blocking rule without G-0475's rename-and-status commit.
