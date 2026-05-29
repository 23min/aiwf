---
id: G-0176
title: 'aiwf doctor: warn when PATH binary is older than mainline kernel HEAD'
status: addressed
addressed_by_commit:
    - 78366bdc
---
## What's missing

`aiwf doctor` does not warn the operator when their PATH `aiwf`
binary is older than the kernel-source HEAD on mainline. Kernel
fixes land on `main`; the operator's daily-flow binary remains
built from earlier source until they explicitly `make install` /
`aiwf upgrade`. Today there's no in-tool signal that the gap
exists.

## Why it matters

This violates the CLAUDE.md kernel principle *"framework
correctness must not depend on the LLM's/operator's behavior."*
Right now, framework correctness *does* depend on the operator
remembering to redeploy after every kernel fix.

Concrete demonstration from 2026-05-28: G-0170 (`apply rollback
restores pre-Apply worktree state, not HEAD`) landed at 10:34 UTC.
G-0150's close-out at ~13:55 UTC ran `aiwf promote` against the
PATH binary (built 2026-05-26 14:23 — two days before G-0170 even
existed) and hit a `.git/index.lock` race. The PATH binary's old
rollback logic could not recover, leaving the gap entity file half-
applied on disk (`status: addressed` written; commit failed; no
captured-bytes restore in the old code). Required manual `git
checkout --` to recover.

With the up-to-date kernel binary, the captured-bytes restore is
pure filesystem and would have run regardless of the lock. **The
fix designed to prevent exactly this failure mode was undeployed
when the failure mode struck.**

## Prior art and scope distinction

**G-0147** added `make diag-aiwf` and CLAUDE.md's *Worktree binary
discipline* section. That covers the **diagnostic case** — when
the operator is debugging *uncommitted* kernel source against the
PATH binary, and runs the wrong binary by accident.

This gap covers the complementary **deployment case** — when a
kernel fix has *landed* on mainline but the operator's PATH
binary is still from before the landing.

Both share a root cause (PATH binary != current kernel source) but
the failure modes and triggers differ:

| | Diagnostic case (G-0147) | Deployment case (this gap) |
|---|---|---|
| Operator state | Editing kernel source, uncommitted | Done editing, fix merged to main |
| PATH binary built from | Earlier commit, OR an unrelated worktree | Pre-fix mainline commit |
| Trigger | Operator runs `aiwf <verb>` against worktree | Operator runs `aiwf <verb>` for daily work |
| Mitigation | `make diag-aiwf` + invoke by absolute path | Redeploy via `make install` |

## Proposed shape

Extend `aiwf doctor` (the natural surface — it's the "is my install
healthy?" verb) with a **`binary:` row staleness check**:

- Read the running binary's build SHA / time (already surfaced by
  `aiwf version` via `runtime/debug.ReadBuildInfo` + the
  ldflags-stamped `Version`).
- Read mainline kernel HEAD's SHA (from
  `refs/remotes/origin/main`, or the project's configured trunk
  ref — `aiwf.yaml` already names a trunk).
- When the binary's commit SHA is not a descendant-of (or equal
  to) the trunk HEAD's SHA, emit an **advisory** line on the
  binary: row: `binary: ... (stale: N commits behind <trunk>; run make install to refresh)`.

The check is advisory (not a `Finding`) — it goes on the
informational row, never increments doctor's problem count. The
operator's daily flow keeps working; the signal is "your binary
is older than the fix you might be depending on."

Skip the check when:

- The running binary is a dirty-tree build (`+dirty` suffix) — the
  operator is mid-edit; the check is meaningless and the warning
  would be noisy.
- The trunk ref is unavailable (no upstream configured, no
  network) — degrade silently, mirroring how the existing
  `latest:` row degrades when GOPROXY is unreachable.

## Out of scope

- **Auto-deploy** (a verb that re-`go install`'s the binary mid-
  operation). Intrusive; the `aiwf upgrade` verb already exists for
  the manual case.
- **A pre-push hook** that blocks pushes when the PATH binary is
  stale. The hook runs *inside* the binary; if the binary is stale
  the hook itself is stale. Wrong layer.
- **Cross-platform binary-staleness mechanics** (Darwin
  notarization, Windows code-signing). Per CLAUDE.md the kernel
  targets Claude Code only; signing is a separate gap surface
  (G-0133/G-0134).

## Related

- **G-0147** — diagnostic-case companion. Same root cause class.
- **G-0170** — the fix that surfaced this gap by being undeployed
  when it mattered.
- **CLAUDE.md §"Worktree binary discipline"** — the operator-side
  prose this gap would complement with a mechanical chokepoint.
- **G-0149** — the gap whose close-out (and `aiwf upgrade --check`
  audit) confirmed the proxy-lag class is real; informs why
  proxy-based staleness checks aren't load-bearing.

## Discovered

2026-05-28, during G-0150's close-out. The `aiwf promote G-0150`
verb call hit a `.git/index.lock` race; the PATH binary's old
rollback left the entity file in a half-applied state on disk.
Recovery required manual `git checkout --`. The G-0170 fix that
shipped earlier the same day would have prevented the half-applied
state but had not been redeployed to the operator's PATH binary.
