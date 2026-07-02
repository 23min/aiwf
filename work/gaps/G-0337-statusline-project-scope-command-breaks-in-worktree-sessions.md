---
id: G-0337
title: statusline project-scope command breaks in worktree sessions
status: open
---
## Problem

`aiwf`'s statusline wiring (`--statusline` / `aiwf update --statusline`) writes a
`statusLine.command` that does not resolve from a git worktree — the working directory
aiwf's own epic/milestone rituals run in (ADR-0023: in-repo worktrees under
`.claude/worktrees/`). Claude Code runs the statusline command from the session cwd, which
follows the session into the worktree. The scaffolded script lives only at
`<root>/.claude/statusline.sh` and is gitignored in consumer repos, so it is absent from
every fresh worktree checkout. Result: the statusline silently renders blank in exactly the
sessions aiwf's own workflow produces — no error, no `aiwf doctor` signal.

Two command forms were observed, both fragile:

1. **cwd-relative** `.claude/statusline.sh` (what `statuslineDest` emitted for project
   scope) — resolves against the worktree cwd, file absent, blank.
2. **`${CLAUDE_PROJECT_DIR:-<root>}/.claude/statusline.sh`** (a consumer hand-fix) — depends
   on `CLAUDE_PROJECT_DIR` being (a) exported to the statusline process [undocumented;
   observed *unset* for a child process] and (b) equal to the workspace root, not the
   worktree subdir [undocumented]. Its baked `<root>` fallback also calcifies: in the
   admin-sorter corpus the fallback was `/workspace` while the repo had since been mounted at
   `/workspaces/admin-sorter`, so with the var unset the command resolved to a non-existent
   path — blank even with the "fix" applied.

## Why aiwf's own dogfooding masks it

aiwf's repo *tracks* `.claude/statusline.sh` (`.gitignore` carries `!.claude/statusline.sh`),
so `git worktree add` materializes it into every worktree and the cwd-relative command works.
A consumer running `aiwf update --statusline` gets the script added to `.gitignore` as a
plain ignore (untracked, absent from worktrees). The bug is therefore consumer-only and
structurally invisible to aiwf's self-test — it was filed by a consumer (admin-sorter).

## doctor reports healthy while broken

`hasStatusLineKey` checks only that a `statusLine` key exists, not that its command resolves.
A repo broken in every worktree session reports as wired/healthy. `statuslineCmdPathForScope`
additionally recomputes the project command string independently — a second copy that can
drift from what wiring writes.

## Decisions (settled)

1. **Default scope -> user, everywhere.** `--scope` default flips project -> user in
   `update.go` + `initcmd.go`; `--scope project` is the explicit opt-in. User scope is the
   only form independent of repo location / worktree / mount layout.
2. **User-scope command = `$HOME/.claude/statusline.sh`** (env-var anchored; on-disk dest
   stays the resolved absolute path for scaffolding). `$HOME` is POSIX-guaranteed and
   per-environment-correct across the shared host<->container `~/.claude` mount — the
   reliable analogue of the `CLAUDE_PROJECT_DIR` bet that does not hold.
3. **Project-scope command (opt-in) = `${CLAUDE_PROJECT_DIR:-<root>}/.claude/statusline.sh`.**
   Best available for the deliberate in-repo choice; its baked-fallback calcification is
   mitigated by the always-refresh below (re-run `--statusline` re-derives `<root>`) and
   surfaced by the new doctor check.
4. **Script: always byte-refresh `statusline.sh` idempotently** on every `aiwf update`
   (drop scaffold-once; keep the name; no rename; no marker). Same contract as every other
   aiwf-materialized artifact. Flips M-0155/AC-3.
5. **doctor gains:** a precedence-conflict warning (project + user both wired -> project
   silently wins, the trap that hid the bug); a fragile-project-command warning
   (bare-relative, or `${...:-<fallback>}` whose `<fallback>/.claude/statusline.sh` does not
   resolve); and the "not wired" hint now suggests `--scope user`.
6. **`--statusline` stays the opt-in trigger** (now defaulting to user scope); no
   auto-scaffold on `init`/`update`. Keeps the ADR-0015 consent surface unchanged.
7. **Single source of truth:** a `Command` field on `StatuslineScaffoldResult`; consumed by
   cliutil + doctor; the two duplicate hardcoded command strings are deleted.

## AC impact

M-0155/AC-3 (scaffold-once -> always-refresh), AC-4 (project command repo-relative ->
`${CLAUDE_PROJECT_DIR:-<root>}`), and AC-5 (user command baked-absolute -> `$HOME`-anchored)
all flip — their original premises were the bug. The patch revises the `internal/policies/`
M-0155 tests in place; the milestone stays done.

## Evidence

- `internal/skills/statusline.go` `statuslineDest`; `ScaffoldStatuslineWithHome` scaffold-once.
- `internal/cli/doctor/statusline.go` `hasStatusLineKey` / `statuslineCmdPathForScope`.
- `internal/cli/cliutil/statusline.go` `statuslineCmdPath` (parses the command back out of the
  snippet; hardcoded relative fallback).
- `internal/cli/update/update.go:61` + `internal/cli/initcmd/initcmd.go:58` — `--scope`
  defaults to project.
- admin-sorter `.claude/settings.local.json` anchored form with a stale `/workspace` fallback
  while mounted at `/workspaces/admin-sorter`; worktree `epic-E-0010` had no script,
  `epic-E-0013` a stray copy.
- Claude Code docs: the statusline command "runs in a shell" (documented); its cwd and
  whether `CLAUDE_PROJECT_DIR` reaches it are undocumented.

## Vehicle

`wf-patch` on a dedicated in-repo worktree (`.claude/worktrees/G-0337`), branch
`patch/G-0337-statusline-scope-robustness`, revising the M-0155 policy tests in place.
