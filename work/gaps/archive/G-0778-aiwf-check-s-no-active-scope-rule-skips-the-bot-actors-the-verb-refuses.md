---
id: G-0778
title: aiwf check's no-active-scope rule skips the bot/ actors the verb refuses
status: addressed
discovered_in: M-0364
addressed_by_commit:
    - eeef04807
---
## What's missing

`aiwf check`'s no-active-scope rule tests only the `ai/` prefix:
`if strings.HasPrefix(actor, "ai/") && !hasOnBehalfOf && idx[gitops.TrailerVerb] != "authorize" {`
(`internal/check/provenance.go:278`). The verb gate, `Allow` (`internal/verb/allow.go:138`),
refuses any actor that is not `human/` without an active scope, `bot/` included, so the two
layers disagree on `bot/` actors.

Measured 2026-11-21 in the Linux devcontainer (Go 1.25.12, git 2.51.1) with `aiwf` built from a tree whose code is `main` at 0c00b37c4 (`aiwf version`: `v0.42.1-0.20261121183124-0c00b37c47b3+dirty`), in a disposable repository: `aiwf init` with `hosts: []`, epic `E-0001` active, milestone `M-0001` in progress with two ACs, and `milestone/M-0001-first-milestone` checked out. Ids in the block name entities in that repository. No scope is opened.

```
$ aiwf add ac M-0001 --title x --actor bot/ci --principal human/t
aiwf add ac: provenance refused: actor "bot/ci" has no active scope authorizing this act (provenance-no-active-scope)
$ git commit --allow-empty --no-verify -m "aiwf edit-body M-0001" \
    --trailer "aiwf-verb: edit-body" --trailer "aiwf-entity: M-0001" \
    --trailer "aiwf-actor: bot/ci" --trailer "aiwf-principal: human/t"
$ aiwf check                       # exit 0, no provenance finding
```

The same commit with `aiwf-actor: ai/claude` draws
`error provenance-no-active-scope: … has no aiwf-on-behalf-of: (no active scope authorized this act)`,
exit 1.

## Why it matters

A `bot/` commit made with no scope, by hand or by any path that skips the verb, passes
`aiwf check` and the pre-push hook, which CLAUDE.md names as the guarantee: "Skills are
advisory; the pre-push hook and `aiwf check` are authoritative."
