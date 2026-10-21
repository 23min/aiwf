# Epic wrap — E-0098

**Date:** 2026-10-01
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0098-move-aiwf-s-own-devcontainer-onto-the-house-devcontainer-kit

## Milestones delivered

- M-0359 — Generate aiwf's devcontainer from the kit with its own steps in project hooks
  (merged 9c7fdcd42)
- M-0360 — Trial-build, then rebuild aiwf's container from the kit and record the result
  (merged 11334a7ba)

## Changelog entry

### Changed (internal) — E-0098: aiwf's devcontainer is generated from the house devcontainer kit

Nothing changes for repositories that use aiwf. aiwf's own development container:

- is generated from the house devcontainer kit, on an Ubuntu 24.04 base in place of Debian 12;
- opens only at `/workspaces/aiwf` on this checkout: the folder above it is no longer mounted,
  so sibling repositories and any `CLAUDE.md` beside the clone are out of reach, and git does not
  work in a container opened on a sibling worktree, whose `.git` points into the main checkout;
  a sibling that is needed is added through the kit's `siblings` or `writable_siblings` answers;
- runs Go at CI's `GO_VERSION`, builds `aiwf` from the checkout, and installs golangci-lint,
  govulncheck and gitleaks at the versions CI runs, plus gofumpt and goimports; Node is the
  kit's current LTS rather than a pinned 22;
- takes git identity from the host's global git config;
- keeps the Go module and build caches and the npm cache in named volumes shared by every kit
  container, so a rebuild no longer clears them;
- no longer preinstalls the Go extension's helper tools, the `dlv` debugger among them; install
  them with **Go: Install/Update Tools**;
- before each start, removes a `core.hooksPath` naming this repository's own hooks directory by
  its full host path from `.git/config` when that changes nothing on the host, so git hooks run
  in the container too;
- checks Codex at each start and installs it with the kit's standalone installer when it is
  missing or behind, instead of through npm.

`.devcontainer/project/README.md` covers aiwf's additions, the Playwright opt-in and recovery
from a failed container creation.

## Summary

aiwf's development container now comes from the same kit as the maintainer's other repositories,
with aiwf's own steps — tools pinned to CI, `aiwf` built from source, its git hooks — in
repo-owned project hooks checked by policy tests. It mounts only the checkout, which closes
G-0524's exposure of sibling repositories and a rival `CLAUDE.md` (D-0104). The new container
was built beside the old one first, then in its place, and its checks, including Codex state
(G-0699) and Claude Code sessions surviving the rebuild, are recorded in M-0360. The epic branch
merged into `main` once before that rebuild, since VS Code builds from the checkout it opens.

## ADRs ratified

- none

## Decisions captured

- D-0104 — Mount only the checkout in aiwf's devcontainer

## Follow-ups carried forward

- G-0725 — `make install-hooks` reports success when it cannot create the hook links.
- G-0726 — the go workflow skips the policy suite when only a non-Go file it reads changes.
- G-0727 — `aiwf doctor --self-check` hangs when its input is a terminal, so `make ci` typed at
  one stops at its last step.
- G-0728 — `CLAUDE.md` points the completion drift test at files that do not hold it.
- G-0729 — `design-decisions.md` and ANTI-0008 describe a `doctor` setting no code reads.

## Doc findings

Scoped to every file the epic branch changed since `9c262dc53`, plus a search of the Normative
docs for the removed container. No doc describes the hand-written container, its `init.sh`, the
parent mount or the npm Codex install as current. Fixed in the epic: M-0359's AC-3 named the
retired check and its deleted file; the epic's Context described the starting container in the
present tense; and the project notes misnamed a `CLAUDE.md` section. The pre-commit hook comment
that repeats the claim G-0726 shows false is carried in G-0726. Outside the epic's subject, filed
as G-0728 and G-0729.

## Handoff

The container is in use and `main` carries it. What is open is recorded above: the gaps this
epic found. Two defects outside this repository are not filed here, since each belongs to its own:
devcontainer-kit's post-create prints "post-start installs/refreshes Codex and aiwf" even where
the kit answer `aiwf` is false, and tmux-up's `save` finds a Claude conversation only through
`~/.claude/sessions/<pid>.json`, a file Claude does not write for every running session.
