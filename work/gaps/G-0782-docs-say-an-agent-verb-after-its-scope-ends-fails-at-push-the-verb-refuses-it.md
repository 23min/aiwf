---
id: G-0782
title: Docs say an agent verb after its scope ends fails at push; the verb refuses it
status: open
discovered_in: M-0364
---
## What's missing

Two rituals and the provenance model say an agent's verb run after its scope ended writes a commit naming that scope, which
then fails at push. The verb refuses it instead, with `provenance-no-active-scope`, and writes
nothing; every function that writes `aiwf-authorized-by` must run the allow rule
(`internal/policies/authorized_by_via_allow.go`). The scope audit of M-0364 settled it
(`spec/scope/audit/decisions.md`, Q7): "the verb refuses (`provenance-no-active-scope`) and
writes no commit, so a later `check` passes."

- The wrap-milestone ritual,
  `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-milestone/SKILL.md`,
  step 10: "A verb-driven commit produced after that point risks an ended-scope
  `aiwf-authorized-by:` trailer on push", and its anti-patterns: "closing gaps after
  `aiwf promote M-NNNN done` risks an ended-scope `aiwf-authorized-by:` trailer on the
  gap-promote commit."
- The wrap-epic ritual, `…/aiwfx-wrap-epic/SKILL.md`, §"Why promote is last among verb-driven
  commits": "Any commit produced *after* this that goes through a kernel verb … would carry
  `aiwf-authorized-by:` referencing the just-ended scope and trigger the kernel's
  `provenance-authorization-ended` finding on push"; its §Constraints: "any commit produced
  after it that routes through a kernel verb carries an ended-scope `aiwf-authorized-by:` and
  fails the kernel's `provenance-authorization-ended` check on push"; and its anti-patterns:
  "subsequent verb-driven commits carry an ended-scope `aiwf-authorized-by:` and fail
  `provenance-authorization-ended` on push."
- `docs/design/provenance-model.md` §"Worked examples", Example 5: "Subsequent agent verbs
  referencing it as `aiwf-authorized-by: 4b13a0f` produce `provenance-authorization-ended`
  findings."

Measured 2026-11-21 in the Linux devcontainer (Go 1.25.12, git 2.51.1) with `aiwf` built from a tree whose code is `main` at 0c00b37c4 (`aiwf version`: `v0.42.1-0.20261121183124-0c00b37c47b3+dirty`), in a disposable repository: `aiwf init` with `hosts: []`, epic `E-0001` active, milestone `M-0001` in progress with two ACs, and `milestone/M-0001-first-milestone` checked out. Ids in the block name entities in that repository.

```
$ aiwf authorize M-0001 --to ai/claude --reason go
$ aiwf promote M-0001 done               # both ACs met first; commit carries aiwf-scope-ends
$ aiwf retitle M-0001 "Renamed by agent" --actor ai/claude --principal human/t
aiwf retitle: provenance refused: actor "ai/claude" has no active scope authorizing this act (provenance-no-active-scope)
```

Exit 1, no commit.

## Why it matters

The wrap rituals give an operator the wrong failure to expect after the closing promote: a push
blocked by `provenance-authorization-ended`, where an agent actually meets a refusal at the verb.
Example 5 teaches the provenance model's reader that the verb writes such a commit.
