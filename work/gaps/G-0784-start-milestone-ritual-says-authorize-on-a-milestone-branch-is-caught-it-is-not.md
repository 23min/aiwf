---
id: G-0784
title: Start-milestone ritual says authorize on a milestone branch is caught; it is not
status: open
discovered_in: M-0364
---
## What's missing

The start-milestone ritual says the kernel stops an authorize run from a milestone branch, at the
verb and again at `aiwf check`, and neither does.
`internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-milestone/SKILL.md`
§"Anti-patterns", *Cutting the milestone branch before the sovereign acts*: "The kernel's
preflight refuses authorize-on-milestone-branch with `branch-context-required` at the verb
layer; the `isolation-escape` kernel finding catches the same shape post-hoc at `aiwf check`
(warning severity)." A ritual-shape checkout satisfies the preflight (M-0103/AC-3: "AI-actor
authorize from ritual-shape checkout (no --branch) accepts"), and the scope then binds to that
branch, so the agent's work there is on its bound branch. The scope audit of M-0364 recorded it
(`spec/scope/audit/decisions.md`, Q18).

Measured 2026-11-21 in the Linux devcontainer (Go 1.25.12, git 2.51.1) with `aiwf` built from a tree whose code is `main` at 0c00b37c4 (`aiwf version`: `v0.42.1-0.20261121183124-0c00b37c47b3+dirty`), in a disposable repository: `aiwf init` with `hosts: []`, epic `E-0001` active, milestone `M-0001` in progress with two ACs, and `milestone/M-0001-first-milestone` checked out. Ids in the block name entities in that repository.

```
$ git branch --show-current
milestone/M-0001-first-milestone
$ aiwf authorize M-0001 --to ai/claude --reason go      # exit 0; aiwf-branch: milestone/M-0001-first-milestone
$ aiwf retitle M-0001 "Renamed by agent" --actor ai/claude --principal human/t   # exit 0
$ aiwf check --since main                                # no isolation-escape, 0 errors
```

From `main`, the same authorize is refused with `branch-context-required`.

## Why it matters

The ritual's order (promote and authorize on the epic branch, then cut the milestone branch)
rests on this sentence. An operator who cuts the milestone branch first is told the kernel will
refuse the authorize or report it, and it does neither: the authorize commit lands on the milestone
branch instead of the epic branch, with no refusal and no finding.
