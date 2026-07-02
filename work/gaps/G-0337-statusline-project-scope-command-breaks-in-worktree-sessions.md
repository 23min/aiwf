---
id: G-0337
title: statusline project-scope command breaks in worktree sessions
status: open
---
## Problem

`aiwf`'s **project-scope** statusline wiring (`--statusline` / `aiwf update --statusline`)
writes a `statusLine.command` that does not resolve from a git worktree — the working
directory aiwf's own epic/milestone rituals run in (ADR-0023: in-repo worktrees under
`.claude/worktrees/`). Claude Code runs the statusline command from the session cwd, which
follows the session into the worktree. The scaffolded script lives only at
`<root>/.claude/statusline.sh` and is gitignored in consumer repos, so it is absent from
every fresh worktree checkout. Result: the statusline silently renders blank in exactly the
sessions aiwf's own workflow produces — no error, no `aiwf doctor` signal.

Two command forms have been observed, both fragile:

1. **cwd-relative** `.claude/statusline.sh` (what `internal/skills/statusline.go`
   `statuslineDest` emits today for project scope) — resolves against the worktree cwd →
   file absent → blank.
2. **`${CLAUDE_PROJECT_DIR:-<root>}/.claude/statusline.sh`** (a consumer hand-fix, proposed
   for upstreaming) — depends on `CLAUDE_PROJECT_DIR` being (a) exported to the statusline
   process [undocumented; observed *unset* for a child process in a `CLAUDE_CODE_CHILD_SESSION`]
   and (b) equal to the workspace root rather than the worktree subdir [undocumented]. Its
   baked `<root>` fallback also calcifies: in the admin-sorter corpus the fallback is
   `/workspace` while the repo has since been mounted at `/workspaces/admin-sorter`, so with
   the var unset the command resolves to a non-existent path and the statusline is blank even
   though the "fix" is applied.

## Why aiwf's own dogfooding masks it

aiwf's repo *tracks* `.claude/statusline.sh` (`.gitignore` carries `!.claude/statusline.sh`),
so `git worktree add` materializes it into every worktree and the cwd-relative command works.
A consumer running `aiwf update --statusline` instead gets the script added to `.gitignore`
as a plain ignore (untracked, absent from worktrees). The bug is therefore consumer-only and
structurally invisible to aiwf's self-test — it was filed by a consumer (admin-sorter), not
caught internally. This is the "works-for-us / breaks-for-consumers" class the CLAUDE.md
"test the seam" rule warns about.

## doctor reports healthy while broken

`internal/cli/doctor/statusline.go` `hasStatusLineKey` checks only that a `statusLine` key
exists, not that its command resolves. A repo broken in every worktree session reports as
wired/healthy. `statuslineCmdPathForScope` additionally recomputes the project command string
independently, a second copy that can drift from what wiring writes.

## Resolution direction (agreed with operator)

- **User scope is robust and is already aiwf's recommended container path** (the doctor
  container nudge). The statusline script self-resolves the repo/entity at runtime from cwd +
  `git rev-parse --git-common-dir` (it explicitly derives the real repo from a worktree) +
  stdin JSON; it does *not* depend on its own install location, so a single
  `$HOME/.claude/statusline.sh` renders correctly for every repo and worktree. `~/.claude` is
  a persistent host bind-mount in the standard devcontainer, so it survives rebuilds.
- **The user-scope command must use `$HOME` (or `~`), not a baked-absolute home path.**
  `~/.claude/settings.json` is shared host<->container via the mount, and a baked
  `/home/<user>/...` resolves under only one `$HOME`. `$HOME` is POSIX-guaranteed and
  per-environment-correct — the reliable analogue of the `CLAUDE_PROJECT_DIR` bet that does
  not hold. Current code (`statusline.go` user branch) bakes the absolute path via
  `filepath.Join(home, ...)`.
- **doctor must catch the fragility**: warn when a project-scope `statusLine.command` is a
  bare relative path or resolves to a non-existent file, and nudge toward user scope.
- **Single source of truth** for the command string: carry it on
  `StatuslineScaffoldResult` and consume it in cliutil + doctor, removing the two duplicate
  hardcoded copies.

## Open decisions (deferred — not made in this gap)

1. Should the **container default** flip to user scope? (doctor only nudges today.)
2. Project scope's command emission — leave as-is (fragile, doctor-warned) or stop
   auto-wiring it for worktree-heavy consumers?
3. Refresh dimension: both the baked fallback and the scaffold-once script calcify (see
   G-0312 — materialized statusline never refreshes on update). Re-derive on every
   `aiwf update`?

## Evidence

- `internal/skills/statusline.go` `statuslineDest` — project branch returns the relative
  path; user branch returns a baked absolute.
- `internal/cli/doctor/statusline.go` `hasStatusLineKey` / `statuslineCmdPathForScope`.
- `internal/cli/cliutil/statusline.go` `statuslineCmdPath` — parses the command back out of
  the snippet and carries a hardcoded relative fallback.
- admin-sorter `.claude/settings.local.json` (anchored form) with a stale `/workspace`
  fallback while the repo is mounted at `/workspaces/admin-sorter`; worktree `epic-E-0010`
  had no script, `epic-E-0013` a stray copy.
- Claude Code docs: the statusline command "runs in a shell" (documented); its cwd and
  whether `CLAUDE_PROJECT_DIR` reaches it are undocumented.

## Scope of the patch tied to this gap

1. User-scope command -> `$HOME/.claude/statusline.sh`.
2. doctor warning on fragile project-scope commands (bare-relative or non-resolving).
3. Consolidate the command string to a single source (result field), removing the duplicate
   copies in doctor + cliutil.

The default-flip and project-scope-emission questions above are deferred to a separate
decision.
