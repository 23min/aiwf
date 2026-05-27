---
id: G-0174
title: 'aiwf doctor: hint on cross-platform plugin-index path corruption'
status: addressed
addressed_by_commit:
    - a39d7f22
---
## What's missing

`aiwf doctor` already has a `plugin-mount:` check (`shadowMountStatus` in
`internal/cli/doctor/env.go`, landed by M-0135 / E-0035). But that check is
**presence-only**: it probes the in-container `~/.claude/plugins/` target and
reports `ok / empty / missing / error` based on whether the directory exists
and has ≥1 entry. It does **not** inspect the *contents* of the index.

So it reports `plugin-mount: ok` even when every `installLocation` /
`installPath` inside the index is a corrupt foreign-OS path — which is
exactly the cross-platform path corruption from
anthropics/claude-code#31388: a Claude Code plugin index whose entries are
macOS host paths (`/Users/...`) while the running environment is a Linux
container (or the inverse). When this happens, Claude Code's marketplace
refresh fails with:

> Marketplace '<name>' has a corrupted installLocation
> (/Users/.../.claude/plugins/marketplaces/<name>) — expected a path inside
> /home/<user>/.claude/plugins/marketplaces.

The remediation — the devcontainer plugin-index shadow-mount, or
`claude plugin marketplace remove` + re-add at project scope — lives only in
CLAUDE.md prose and the `.devcontainer/` scripts. A downstream devcontainer
consumer hitting the error has no in-tool pointer to it, and `aiwf doctor` —
the natural thing to run when "plugins failed to load" — stays green.

## Why it matters

The discoverability principle says operator capabilities must be reachable
through channels an operator routinely consults. `aiwf doctor` is the natural
diagnostic, but today it is silent on the one devcontainer failure mode that
recurs across consumers (aiwf, Liminara, FlowTime). The hint turns a
confusing upstream error into a one-line, actionable pointer.

## Proposed shape

Extend M-0135's `plugin-mount` check — keep the presence check
(`shadowMountStatus`), add a **path-prefix content check**: when the env is a
Linux container (the check already knows `env: devcontainer`) and any
`installLocation` / `installPath` in the index begins with a foreign-OS
prefix (`/Users/` under Linux), emit an **advisory** doctor line — not a hard
failure — naming the #31388 cause and pointing at the shadow-mount
remediation. Advisory severity because the cached skills often still load
while refresh fails, so it must not block. The presence-vs-content
distinction is the crux: presence is M-0135; path-correctness is this gap.

## Upstream status

As of 2026-05-27, anthropics/claude-code#31388 is **still open** (reported
2026-03-06 against v2.1.69, assigned, no fix shipped; a regression of the
closed-as-duplicate #15717, sibling of #10379). Root cause confirmed
upstream: the plugin index stores absolute paths instead of paths relative to
`~/.claude`. Until that lands, any index shared across a macOS host and a
Linux container breaks — so this hint stays warranted and the CLAUDE.md
"remove the shadow-mount once #31388 ships" condition is not yet met. The
official escape hatch (a container-owned named volume at `~/.claude` +
`CLAUDE_CONFIG_DIR`) sidesteps the bug but is a design change, not a fix.

## Discovered

Operator-reported: a downstream devcontainer consumer hit the
`corrupted installLocation` error while this repo's own container was healthy
(shadow-mount active) — so the failure mode is environment-specific and
currently undiscoverable from inside the tool.
