---
id: G-0780
title: D-0007 offers authorize --force on other kinds; the verb refuses them first
status: open
discovered_in: M-0364
---
## What's missing

D-0007 offers an override the verb does not have. D-0007 §Resolution: "Sovereign override:
`aiwf authorize <id> --to <agent> --force --reason "..."` remains available for genuinely
exceptional scope-entity kinds." `authorizeOpen` (`internal/verb/authorize.go:306`) returns
`AuthorizeKindError` for any kind but epic and milestone (`authorize.go:312`) before it reads
`--force`.

Measured 2026-11-21 in the Linux devcontainer (Go 1.25.12, git 2.51.1) with `aiwf` built from a tree whose code is `main` at 0c00b37c4 (`aiwf version`: `v0.42.1-0.20261121183124-0c00b37c47b3+dirty`), in a disposable repository: `aiwf init` with `hosts: []`, epic `E-0001` active, milestone `M-0001` in progress with two ACs, and `milestone/M-0001-first-milestone` checked out. Ids in the block name entities in that repository. A gap `G-0001` was added with `aiwf add gap --body-file`.

```
$ aiwf authorize G-0001 --to ai/claude --force --reason "try"
aiwf authorize: aiwf authorize: kind "gap" is not allowed (authorize-kind-not-allowed); only epic and milestone carry autonomous-work scopes
```

Exit 1.

## Why it matters

An accepted decision and the verb state opposite rules for the same input, so a reader cannot
learn from the decision record whether a human may open a scope on a gap, decision, contract or
ADR, and one following D-0007 is refused.
