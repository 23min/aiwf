---
id: G-0783
title: Provenance docs name pause and resume refusal codes the verb never sends
status: open
discovered_in: M-0364
---
## What's missing

Two docs give the pause and resume refusals codes the verb does not send.
`docs/design/provenance-model.md` §"Verb behavior": "If none active, refuses with
`provenance-no-active-scope-to-pause`" and "If none paused, refuses with
`provenance-no-paused-scope-to-resume`"; `docs/design/legal-workflows-first-principles.md`
R-FP-0135 and R-FP-0136 say the same. The refusals are plain messages
(`internal/verb/authorize.go`, "no active scope on %s to pause" and "no paused scope on %s to
resume"), as `docs/audits/normative-docs-drift-audit.md` (2026-08-06) and
`docs/audits/entity-truth-audit.md` (2026-08-19) found. The scope audit of M-0364 recorded it (`spec/scope/audit/decisions.md`, Q18).

Measured 2026-11-21 in the Linux devcontainer (Go 1.25.12, git 2.51.1) with `aiwf` built from a tree whose code is `main` at 0c00b37c4 (`aiwf version`: `v0.42.1-0.20261121183124-0c00b37c47b3+dirty`), in a disposable repository: `aiwf init` with `hosts: []`, epic `E-0001` active, milestone `M-0001` in progress with two ACs, and `milestone/M-0001-first-milestone` checked out. Ids in the block name entities in that repository. No scope is opened.

```
$ aiwf authorize M-0001 --pause hold --format json
{"tool":"aiwf",…,"status":"error","findings":[],"error":{"message":"no active scope on M-0001 to pause"},…}
```

Exit 2; the error carries no code.

## Why it matters

A tool or reader matching on the documented codes never sees them, and the two docs present as
the rule two codes no part of aiwf defines.
