---
id: G-0446
title: aiwf init consent prompt blocks non-interactively while holding the repo lock
status: open
discovered_in: E-0071
---
`aiwf init` runs its hook-consent gate (`GateHookDecisions`, `hooks.go:38`)
while holding the repo flock: `initcmd.go:104` acquires the lock, `defer
release()` at 108, then `gateAndPersistHookDecisions` prompts `[y/N]` per
undecided registry hook at 197 — inside the lock's critical section. A blocking
human-input read is IO, so this violates the repo's own "never hold a lock
across an IO call" rule.

The `render.IsTTY(os.Stdin)` guard is necessary but not sufficient: a
devcontainer `postCreateCommand` can allocate a pty with no interactive human
behind it, so `IsTTY` returns true, `promptYN`'s `ReadString` blocks forever
waiting for input that never arrives, and the flock is held the entire time —
wedging every subsequent `aiwf` invocation on the repo (observed: a 2h+
`futex_wait` hang holding `.git/aiwf.lock`, blocking an unrelated `aiwf
edit-body` hours later).

The non-interactive path is wrong on three counts:

1. **Blocks under a fake TTY**, holding the repo lock (the hang above).
2. **Defaults undecided hooks to `false`** (`GateHookDecisions`'s `default:`
   arm writes `false` into aiwf.yaml). That records a *decided*-declined state,
   which `HookDrift` treats as fully-synced (green) — so even without the hang,
   non-interactive init silently declines and *hides* the missed config, the
   opposite of surfacing it. The ADR-0032 doctor/statusline machinery already
   renders an *absent* (undecided) hook as a yellow `SeverityWarn` via
   `.claude/health.aiwf.json`; init just needs to leave undecided hooks absent
   rather than defaulting them.
3. **Ignores existing recorded decisions.** Unlike `aiwf update` (which
   pre-filters to hooks absent from the current aiwf.yaml), `init` treats the
   whole registry as fresh, so a container rebuild re-decides/clobbers instead
   of honoring the `true`/`false` already recorded.

## Fix direction

- **Honor existing** — init reads `doc.Hooks()` first and only considers hooks
  absent from it; recorded `true`/`false` pass through untouched (mirror
  `update`'s pre-filter).
- **Undecided stays undecided** — in a non-interactive run, init leaves
  undecided hooks absent from what it writes (never defaults to `false`), so
  the existing doctor→`health.aiwf.json`→statusline path surfaces them yellow.
  No assumption, no timeout default.
- **Recommended: a `--no-prompt` flag** as the non-interactive signal, passed
  by `.devcontainer/init.sh`. A fake pty defeats `isatty`, so init cannot
  *detect* "no human" — it must be *told*. A flag is deterministic (works under
  a fake pty), discoverable/tab-completable per the CLI convention, and testable
  without a pty library — unlike `aiwf init </dev/null` (relies on `isatty`,
  undiscoverable) or running `aiwf update` on rebuild (shares the same
  `promptYN`-under-TTY path, so it does not fix the hang alone).
- **Defense-in-depth (optional):** move the remaining genuinely-interactive
  prompt out of the flock's critical section, so no human-input read is ever
  held across the repo lock.

Surfaced during E-0071 planning, when the wedged `aiwf init` blocked an `aiwf
edit-body`. Call sites: `initcmd.go:104–197`, `hooks.go:38`,
`GateHookDecisions`'s `default` arm, and the `aiwf init` call in
`.devcontainer/init.sh`.
