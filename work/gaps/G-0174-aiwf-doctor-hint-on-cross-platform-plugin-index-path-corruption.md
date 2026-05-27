---
id: G-0174
title: 'aiwf doctor: hint on cross-platform plugin-index path corruption'
status: open
---
## What's missing

`aiwf doctor`'s `plugin-mount` check reports `ok` whenever the index is
readable, but it does not detect the cross-platform path corruption from
anthropics/claude-code#31388: a Claude Code plugin index whose
`installLocation` / `installPath` entries are macOS host paths
(`/Users/...`) while the running environment is a Linux container (or the
inverse). When this happens, Claude Code's marketplace refresh fails with:

> Marketplace '<name>' has a corrupted installLocation
> (/Users/.../.claude/plugins/marketplaces/<name>) — expected a path inside
> /home/<user>/.claude/plugins/marketplaces.

The remediation — the devcontainer plugin-index shadow-mount, or
`claude plugin marketplace remove` + re-add at project scope — lives only in
CLAUDE.md prose and the `.devcontainer/` scripts. A downstream devcontainer
consumer hitting the error has no in-tool pointer to it.

## Why it matters

The discoverability principle says operator capabilities must be reachable
through channels an operator routinely consults. `aiwf doctor` is the natural
thing to run when "plugins failed to load," but today it stays silent on the
one devcontainer failure mode that recurs across consumers (aiwf, Liminara,
FlowTime). The hint turns a confusing upstream error into a one-line,
actionable pointer.

## Proposed shape

Extend the existing `plugin-mount` check: when the env is a Linux container
(the check already knows `env: devcontainer`) and any `installLocation` /
`installPath` in the index begins with a foreign-OS prefix (`/Users/` under
Linux), emit an **advisory** doctor line — not a hard failure — naming the
#31388 cause and pointing at the shadow-mount remediation. Advisory severity
because the cached skills often still load while refresh fails, so it must
not block.

## Discovered

Operator-reported: a downstream devcontainer consumer hit the
`corrupted installLocation` error while this repo's own container was healthy
(shadow-mount active) — so the failure mode is environment-specific and
currently undiscoverable from inside the tool.
