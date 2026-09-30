---
id: M-0360
title: Trial-build, then rebuild aiwf's container from the kit and record the result
status: in_progress
parent: E-0098
depends_on:
    - M-0359
tdd: none
acs:
    - id: AC-1
      title: A trial container built alongside passes every check
      status: met
    - id: AC-2
      title: The rebuilt aiwf container passes every check and resumes sessions
      status: open
---

## Goal

Bring the kit-generated container up twice, first as a trial beside the current
container and then in place of it, and record what each showed.

## Closes

- G-0699: records Codex's configuration, sessions and login surviving a rebuild of the
  kit-built container (M-0360 AC-2).

## Context

M-0359 lands the kit-generated `.devcontainer/`. VS Code builds a container from
the files in the checkout it opens, and a rebuild removes the running container
before creating its replacement; the session doing this work runs inside that
container. The trial proves the new container before the current one is replaced.

## Acceptance criteria

### AC-1 — A trial container built alongside passes every check

**Pass criterion**, met by a record of command, expectation, observation and
environment: in a container built from a separate clone of the milestone's branch,
`/workspaces` holds only the repository and no `CLAUDE.md` outside it; `aiwf
version` reports a build of that clone's commit; `go version` reports the Go named by
`GO_VERSION` in `.github/workflows/go.yml`; golangci-lint, govulncheck and gitleaks report
the versions pinned in `.devcontainer/project/post-create.sh`; `make ci` passes; `aiwf
doctor` reports the git hooks installed; a commit made in the trial clone carries the
same `aiwf-actor` as the commits made before the move, now that the container's git identity
comes from the host's global git config; Claude Code and Codex start logged in, with
their existing sessions listed. **Edge cases**: the trial clone's container name, which
must differ from the running container's, changed only in the trial clone and
never committed. **Code references**: none; the record lives in this milestone's
Validation section.

### AC-2 — The rebuilt aiwf container passes every check and resumes sessions

**Pass criterion**, met by a record: after the current container is rebuilt from
the merged `.devcontainer/`, the AC-1 checks hold, and `claude --resume` lists
sessions from before the rebuild. The record covers what G-0699 left unobserved:
Codex configuration, sessions and login surviving a rebuild. **Edge cases**: a
failed rebuild, which triggers the rollback in Design notes and is itself
recorded. **Code references**: none.

## Constraints

- The running container is not replaced until AC-1 is met.
- Credential contents are never read or recorded; only whether a login holds.

## Design notes

- Before the rebuild: `tmux-up save` in the running container.
- Trial: clone the milestone branch on the host into its own folder, change the
  container name in its `runArgs` locally, open it in VS Code.
- Rollback: on the host, `git revert` the merge that brought the new
  `.devcontainer/` in, rebuild, and resume sessions with `claude --resume`.

## Out of scope

- Changes to the generated files; a defect found here is filed as a gap, or fixed in this
  milestone when it is small.

## Dependencies

- M-0359.

## References

- E-0098, G-0524, G-0699.

---

## Release note

## Decisions made during implementation

- (none)

## Validation

**AC-1, the trial container** — observed 2026-09-30. Environment: a container built by VS Code
Dev Containers, over Remote-SSH, on the Mac that runs Docker, from a fresh clone of
`epic/E-0098-move-aiwf-s-own-devcontainer-onto-the-house-devcontainer-kit` at `314af2657` (the
same `.devcontainer/` this milestone's branch carries), with only the container name and
hostname changed locally to `aiwf-trial` so it could run beside `aiwf-dev`; Ubuntu 24.04.3 LTS,
x86_64. Each check, run in the trial container's terminal, with what was expected and seen:

- `ls -A /workspaces; ls /workspaces/CLAUDE.md /CLAUDE.md` — expected the repository alone and
  no `CLAUDE.md` outside it; saw `aiwf` alone, and both `CLAUDE.md` paths absent.
- `aiwf version; git rev-parse --short HEAD` — expected a build of the clone's commit; saw
  `v0.40.1-0.20260929232416-314af2657387+dirty` at `314af2657`, dirty from the local name edit.
- `go version` against `GO_VERSION` in `.github/workflows/go.yml` — expected equal; saw
  `go1.25.12` and `"1.25.12"`.
- `golangci-lint version`, `govulncheck -version`, and
  `go version -m "$(command -v gitleaks)"` against the pins in
  `.devcontainer/project/post-create.sh` — expected v2.11.4, v1.6.0 and v8.30.1; saw 2.11.4,
  v1.6.0 and `github.com/zricethezav/gitleaks/v8 v8.30.1`. `gitleaks version` itself prints
  "version is set by build process", since `go install` stamps no version.
- `make ci </dev/null` — expected exit 0; saw exit 0, lint 0 issues, 73 packages ok, no failure
  lines, statement coverage 91.9%, self-check passed all 29 steps. Run from the interactive
  terminal without `</dev/null`, it stopped in the self-check for over 40 minutes, with the
  self-check's `init` waiting on a prompt it could not show; the same happens in the pre-move
  container (G-0727).
- `aiwf doctor` — expected the hooks installed; saw the pre-commit and pre-push hooks resolve to
  the built `aiwf` and chain to their `.local` hooks.
- `aiwf whoami`, then an empty commit's author address — expected the actor commits carried
  before the move; saw `human/peter` from `git config user.email`, and the same address domain.
- `codex login status` and `claude auth status` — expected both logged in; saw "Logged in using
  ChatGPT" and `"loggedIn": true`. `claude --resume` listed the sessions from the running
  container.
- `python3 -c 'import dataclasses'` — saw Python 3.12.3 import it, which settles the point
  M-0359's Validation left open. The shell is zsh.

**AC-2, the rebuilt container** — observed 2026-09-30. Environment: `aiwf-dev` rebuilt by VS Code
Dev Containers on the same Docker Mac, from the main checkout at `f13a6aa26`, the merge of the
epic branch into `main` (CI's go, gitleaks, markdown-lint, link-check and scrub workflows passed
on it); Ubuntu 24.04.3 LTS, x86_64. The AC-1 checks, run again in this container:

- `/workspaces` holds `aiwf` alone, and neither `/workspaces/CLAUDE.md` nor `/CLAUDE.md` exists.
- `aiwf version` reports `v0.40.1-0.20260930102944-f13a6aa269f3+dirty` at `f13a6aa26`, dirty from
  an untracked folder in the checkout.
- `go1.25.12` against `GO_VERSION` `"1.25.12"`; golangci-lint 2.11.4, govulncheck v1.6.0 and
  gitleaks v8.30.1 (from `go version -m`) against the hook's pins.
- `make ci </dev/null` — exit 0, lint 0 issues, 73 packages ok, no failure lines, statement
  coverage 91.9%, self-check passed all 29 steps.
- `aiwf doctor` reports the pre-commit and pre-push hooks installed, chaining to their `.local`
  hooks; `aiwf whoami` reports `human/peter` from `git config user.email`, the address domain
  of the commits before the move.
- `codex login status` reports "Logged in using ChatGPT"; `claude auth status` reports
  `"loggedIn": true`. This conversation, started before the rebuild, was resumed in the rebuilt
  container with `claude --resume`.
- Terminals open in zsh through the kit's `terminal.integrated.defaultProfile.linux`, with Oh My
  Zsh installed; the account's login shell is `/bin/bash`.

What G-0699 left unobserved, compared with M-0343's AC-5 baseline without reading a credential
file:

- `findmnt` shows `~/.codex` backed by the host's `~/.codex-linux`, as in the baseline.
- The baseline's saved session,
  `sessions/2026/09/19/rollout-2026-09-19T15-53-49-01a0ba5f-b597-79e1-9cff-3ac155e4b537.jsonl`,
  has the baseline SHA-256 `307c6b8cf71c43abe0741fb56184d47a934a97b64ecc845c2e80149b5a0f62ec`;
  the session a Codex terminal ran before the rebuild is present too; 109 saved sessions in all.
- `config.toml` keeps the baseline's `alternate_screen = "never"` and `animations = false`, and
  its SHA-256 is now `969f247a126b405b22a0edeb7014b163fe452845a28ef047afe7739592b33a94`, not the
  baseline's: Codex rewrites the file itself, at 10:54:16, two seconds after `codex --yolo`
  started at 10:54:14, and none of the kit's scripts writes it. `auth.json` was last written
  2026-09-28, before either build; only its timestamp was read.
- Login: `codex login status` as above.
- Install: npm no longer installs Codex, so the baseline's npm checks no longer apply. The kit's
  post-start runs the standalone installer when `codex --version` does not report the latest
  release; in a new container the `~/.local/bin/codex` link is absent, so it ran, re-linked the
  0.159.2 release the trial had installed at 00:49, and added no release. `codex --version`
  reports `codex-cli 0.159.2`; Claude Code reports 2.1.286.
- `.devcontainer/README.md` says the same: Codex from the standalone installer into the shared
  `~/.codex`, checked at every start, with its state in the host's `~/.codex-linux`.

## Deferrals

- G-0727 — `aiwf doctor --self-check` hangs when its input is a terminal, so `make ci` typed
  at one stops at its last step.

## Reviewer notes

- (none)
