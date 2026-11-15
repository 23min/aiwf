---
id: G-0766
title: aiwf add admits a scoped ai/ create that aiwf check reports out of scope
status: open
---
## What's missing

`aiwf add` and `aiwf check` judge an `ai/` actor's create against its authorized
scope by different things.

- At the verb, `scopeAllowsAct` (`internal/verb/allow.go`) passes
  `ReachesScopeAny` the reference flags the create was given, as
  `addCreationRefs` (`internal/cli/add/add.go`) collects them: `--epic`,
  `--discovered-in`, `--relates-to`, `--linked-adr` and `--depends-on`, whatever
  the kind stores. A flag naming the scope entity, or anything under it, admits
  the create; `--linked-adr` never can, since it names an ADR, which neither
  holds a scope nor sits under one.
- At check time, the `provenance-authorization-out-of-scope` rule
  (`internal/check/provenance.go`) asks whether the created entity reaches the
  scope through `ReachesScope` (`internal/tree/tree.go`), which follows only its
  stored parent chain and `discovered_in`.

A create admitted only by some other link commits and then fails the check: a
link the entity stores that is not one of those two, such as a decision's
`relates_to` or a milestone's `depends_on`, or a flag the entity's kind does not
store at all, such as `--relates-to` on a gap.

Both sides are meant to apply one rule. D-0014 says the verb-time and check-time
gates "must share one predicate"; the check's comment calls them "one predicate,
two enforcement times", and `scopeAllowsAct`'s says governance edges, `relates_to`
among them, "do not punch through a scope boundary". The scope check in
`docs/design/provenance-model.md` judges a creation act by "the new entity's
references", as the verb does for a link the entity stores. The comment on
`addCreationRefs` says an agent cannot create a top-level epic under any active
scope, and the verb admits one whose `--relates-to` names the scope entity
(G-0638 measures it for a forced create on a ritual branch).

Measured at `3c5bd7546` (git 2.51.1, Linux) in an empty directory, with an
`aiwf` built from that commit on `PATH` so aiwf's hooks run. Output from the
setup commands, the warnings `aiwf check` raises, and the check output the
pre-push hook repeats are left out.

```
$ git init -q --bare remote.git && git init -q -b main r && cd r
$ git config user.name t && git config user.email t@example.invalid
$ aiwf init --no-prompt && git add -A && git commit -qm init
$ aiwf add epic --title Scoped && aiwf promote E-0001 active
$ git remote add origin ../remote.git && git push -q -u origin main
$ git checkout -qb epic/E-0001-scoped && aiwf authorize E-0001 --to ai/claude
$ printf "## Question\n\nQ.\n\n## Decision\n\nD.\n\n## Reasoning\n\nR.\n" > ../d.md
$ printf "## What's missing\n\nX.\n\n## Why it matters\n\nY.\n" > ../g.md
$ aiwf add decision --title D1 --body-file ../d.md --relates-to E-0001 --actor ai/claude --principal human/t
aiwf add decision D-0001 "D1"
$ grep -A1 relates_to work/decisions/D-0001-d1.md
relates_to:
    - E-0001
$ aiwf add gap --title G1 --body-file ../g.md --relates-to E-0001 --actor ai/claude --principal human/t
aiwf add gap G-0001 "G1"
$ sed -n 1,5p work/gaps/G-0001-g1.md
---
id: G-0001
title: G1
status: open
---
$ aiwf add gap --title G2 --body-file ../g.md --discovered-in E-0001 --actor ai/claude --principal human/t
aiwf add gap G-0002 "G2"
$ aiwf check
error provenance-authorization-out-of-scope: commit 69cb55c: aiwf-authorized-by: 5928f9b — target D-0001 does not reach scope-entity E-0001 — hint: …
error provenance-authorization-out-of-scope: commit d56f0a2: aiwf-authorized-by: 5928f9b — target G-0001 does not reach scope-entity E-0001 — hint: …
7 findings (2 errors, 5 warnings)
$ git push -q -u origin epic/E-0001-scoped
error: failed to push some refs to '../remote.git'
```

All three creates exit 0, `aiwf check` exits 1, and the push is refused.
`aiwf acknowledge illegal` on each of the two commits clears its error.

In the same repository, after a human adds `M-0001` under `E-0001` and opens a
scope on it for `ai/other` from `milestone/M-0001-m1`,
`aiwf add milestone --epic E-0001 --tdd none --title M2 --depends-on M-0001 --actor ai/other --principal human/t`
exits 0, the same command without `--depends-on` is refused at the verb, and
`aiwf check` reports `target M-0002 does not reach scope-entity M-0001`.

## Why it matters

An assistant acting under an active scope gets a commit from `aiwf add` and a
refused push for the same act, with nothing at the verb to say the link it chose
does not count. Getting past the refusal takes rewriting the branch or a human's
`aiwf acknowledge illegal`. A consumer who delegates through `aiwf authorize`
meets it whenever the assistant's only link from new work to the scope is
something other than a stored parent or `discovered_in`.

## Related

- G-0638 — a forced epic create on a ritual branch is admitted the same way.
