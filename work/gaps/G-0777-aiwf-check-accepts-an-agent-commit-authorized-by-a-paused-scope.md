---
id: G-0777
title: aiwf check accepts an agent commit authorized by a paused scope
status: open
discovered_in: M-0364
---
## What's missing

`aiwf check` has no rule for pause state, so it accepts an agent commit whose
`aiwf-authorized-by` names a paused scope. Its provenance pass builds an index of openers, one of ends and a rename chain
(`buildAuthOpenerIndex`, `buildEndedAtIndex` and `buildRenameChain`, called at
`internal/check/provenance.go:103`, `:108` and `:110`), and none of pause state;
`provenanceAuthorizationFindings` (`provenance.go:257`) judges each commit against those. The verb refuses the same act with `provenance-no-active-scope`, so only a commit
made around the verb reaches this.

Measured 2026-11-21 in the Linux devcontainer (Go 1.25.12, git 2.51.1) with `aiwf` built from a tree whose code is `main` at 0c00b37c4 (`aiwf version`: `v0.42.1-0.20261121183124-0c00b37c47b3+dirty`), in a disposable repository: `aiwf init` with `hosts: []`, epic `E-0001` active, milestone `M-0001` in progress with two ACs, and `milestone/M-0001-first-milestone` checked out. Ids in the block name entities in that repository.

```
$ aiwf authorize M-0001 --to ai/claude --reason go
$ aiwf authorize M-0001 --pause hold
$ git commit --allow-empty --no-verify -m "aiwf edit-body M-0001" \
    --trailer "aiwf-verb: edit-body" --trailer "aiwf-entity: M-0001" \
    --trailer "aiwf-actor: ai/claude" --trailer "aiwf-principal: human/t" \
    --trailer "aiwf-on-behalf-of: human/t" --trailer "aiwf-authorized-by: <opening commit's SHA>"
$ aiwf check                       # exit 0, no provenance finding
```

The same hand-made commit, made after `aiwf authorize M-0001 --resume go` and
`aiwf authorize M-0001 --end --reason over`, draws `error provenance-authorization-ended` and
`aiwf check` exits 1.

## Why it matters

A pause holds only while the agent goes through a verb. An agent commit written another way
during a pause passes `aiwf check` and the pre-push hook, which CLAUDE.md makes the authority
("Skills are advisory; the pre-push hook and `aiwf check` are authoritative"). G-0755 notes the
same absence: "`aiwf check` has no rule for pauses".
