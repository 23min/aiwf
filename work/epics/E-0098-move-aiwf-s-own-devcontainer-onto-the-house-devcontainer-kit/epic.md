---
id: E-0098
title: Move aiwf's own devcontainer onto the house devcontainer kit
status: active
---

## Goal

aiwf's development container is generated from the house devcontainer kit
(`gh:23min/devcontainer-kit`) like the maintainer's other repositories, mounts only
this repository plus any sibling it lists explicitly, and still builds and tests
aiwf from its own source with tools that match CI.

## Context

`.devcontainer/` is hand-written: E-0035 built it (M-0132 landed the skeleton) and
later work patched it, most recently to mount `~/.guidance`. It binds the **parent**
of the clone, which G-0524 records as exposing every sibling repository and a
parent-level `CLAUDE.md` that displaced this repository's own instructions in a live
session. Other repositories now take their containers from the kit, which mounts
only the repository, pins nothing it can take from the language's own version file,
and keeps repository-specific steps in `.devcontainer/project/*.sh` hooks that
template updates never touch. The kit mounts the same host state this container
does today: `~/.claude`, the container-only plugin index under
`~/.claude-linux/plugins`, `~/.config/gh`, `~/.codex-linux` and `~/.guidance`.

What only this repository needs lives in `.devcontainer/init.sh` today: aiwf built
from source with `go install ./cmd/aiwf`, `aiwf init --no-prompt`,
`make install-hooks`, pinned golangci-lint, gofumpt, goimports, govulncheck and
gitleaks (some cross-checked against the CI workflows), and an opt-in Playwright
install. Policy tests under `internal/policies/`, `internal/cli/doctor/env.go`,
`scripts/git-hooks/pre-commit` and `.github/workflows/gitleaks.yml` refer to the
current files.

## Scope

- Generate `.devcontainer/` from the kit. aiwf's own steps move into the
  repository-owned `project/*.sh` hooks; anything the kit cannot express lands as a
  hand edit to the generated `devcontainer.json`, which template updates merge.
- Re-point or retire every policy test, doc and CI reference that pins the current
  files, including `.devcontainer/README.md` and `CLAUDE.md`.
- Record the decision to drop the parent-folder mount, with the replacement for
  the reach it gave: explicit sibling mounts.
- Observe a rebuilt container and record the observation.

## Out of scope

- Bounding the Go build cache (G-0552). The kit moves the cache to a shared named
  volume, which changes where it grows but not whether it is bounded.
- Kit changes, except a capability this move proves it needs; each such change is
  surfaced before it is made.
- aiwf's release process.

## Constraints

- The container runs aiwf built from this checkout's source, never a released
  binary in its place.
- Tool versions the policy tests tie to the CI workflows stay tied to them.
- No step that runs unattended at container creation or start may wait for input
  (G-0446).

## Success criteria

- [ ] A container rebuilt from the kit-generated `.devcontainer/` sees this
      repository and the listed siblings only, and no parent-level `CLAUDE.md`.
- [ ] In that container, `aiwf` on PATH is built from the checkout, `make ci`
      passes, and the git hooks are installed.
- [ ] Claude Code and Codex state survive the rebuild, observed and recorded.
- [ ] `make ci` passes with no policy test, doc or CI step referring to a file the
      move removed.
- [ ] G-0524 and G-0699 are addressed.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Which sibling repositories the container mounts, and read-only or writable | no | Settled at milestone planning: none |
| Which aiwf the container runs | no | Settled: built from source, kit answer `aiwf: false`, steps in `project/post-create.sh` (M-0359) |
| Go version | no | Settled: `GOTOOLCHAIN` pinned to `go.yml`'s `GO_VERSION` (M-0359 AC-2) |
| The fate of each policy test that pins the current files | no | Settled: listed in M-0359's Design notes |
| Whether the kit's `node` language stays | no | Settled: kept, for the Playwright tests |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| The session doing the work runs inside the container being replaced, and loses reach to sibling repositories unless they are listed | med | Settle sibling access before the rebuild; save tmux sessions with `tmux-up save` first |
| A local Go newer than CI's lets lint or vet disagree between the container and CI | med | Settle the Go version before generating |

## Milestones

- M-0359 — Generate aiwf's devcontainer from the kit with its own steps in project
  hooks · depends on: —
- M-0360 — Trial-build, then rebuild aiwf's container from the kit and record the
  result · depends on: M-0359

## References

- G-0524 — the parent mount and its fix shape.
- G-0699 — Codex rebuild persistence, unverified for the current install.
- G-0552 — the unbounded Go build cache, left out.
- G-0446 — unattended steps must not wait for input.
- E-0035, M-0132 — where the current container came from.
- `gh:23min/devcontainer-kit` — `STANDARD.md` explains each kit rule.
