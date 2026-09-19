# aiwf devcontainer

The aiwf dev loop runs in a Linux devcontainer. macOS-specific bugs
that bite the host path (G-0127 fork/exec deadlock under `-race` +
parallel; G-0128/G-0133 syspolicyd crashes on unsigned Mach-O
binaries) simply don't exist in Linux, so `make ci` is green without
the host-side workaround discipline.

The macOS host-fallback path (`scripts/sign-and-run.sh`, in-test
`codesign` blocks, `-parallel 8` cap) stays available for the rare
case you must run on the host. The container is the default.

See `work/epics/E-0035-devcontainer-based-dev-loop/M-0132-*` for the
per-decision rationale (Q1–Q7 of the design conversation) and the
`## First-boot recovery` section of the milestone spec for anticipated
failure modes.

## Build

Two paths to build the container image:

**VS Code (primary path).** Install Docker Desktop and the
[Dev Containers](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
extension. Open this repo in VS Code, then Command Palette →
"Dev Containers: Reopen in Container". The extension drives image
build, container start, and `init.sh` execution. **No standalone
CLI install needed** — the extension carries the devcontainer spec
implementation internally.

**Standalone CLI (terminal-first builds, future CI).** Install
Docker Desktop and `@devcontainers/cli`:

```
npm install -g @devcontainers/cli
devcontainer build --workspace-folder /path/to/aiwf
```

Only needed when scripting the build outside VS Code. The future
CI matrix (sibling milestone under E-0035) uses this path; the
operator path doesn't need it.

Either path: the first build downloads the base image
(`mcr.microsoft.com/devcontainers/go:2-1.25-bookworm`) and the three
declared features. Subsequent builds use cached layers. The build
generates `.devcontainer/devcontainer-lock.json` pinning resolved
feature SHAs — commit this file once it lands so future builds
reproduce exactly.

Repo location: this repo cloned at `~/Projects/aiwf/` (or any
sibling-tree path — the workspace mount goes one level up so
sibling repos under the same parent directory are reachable
inside).

## Reopen in Container

In VS Code at this repo's root:

1. Install the **Dev Containers** extension if you haven't already.
2. Command Palette → `Dev Containers: Reopen in Container`.
3. The first open builds the image (slow) and runs
   `.devcontainer/init.sh` (idempotent install of golangci-lint,
   gofumpt, govulncheck, Claude Code CLI, Codex CLI, aiwf binary, framework
   hooks). Subsequent opens reuse the cached image.
4. `aiwf init` (run by init.sh) materializes skills, guidance and templates
   for the selected hosts: `.claude/` and `CLAUDE.md` for Claude;
   `.agents/` and `AGENTS.md` for Codex. An explicit `hosts` list in
   `aiwf.yaml` controls selection; otherwise init detects installed CLIs.
   Claude also receives role agents. No separate plugin install is needed.

Verify the container is set up correctly:

```
aiwf doctor          # rituals: line confirms the skills are materialized.
make ci              # vet + lint + test-race + coverage + selfcheck green.
```

## Codex CLI

`init.sh` installs `@openai/codex` globally through the existing Node feature
on container creation/rebuild. It skips installation when npm's Codex binary
already exists, even if the VS Code extension also supplies a binary.
To upgrade explicitly, run `npm install -g @openai/codex@latest` inside the
container.

The host directory `~/.codex-linux` is mounted at `/home/vscode/.codex`.
It retains files stored there, including Codex configuration and sessions,
across rebuilds, separately from the host's `~/.codex`. File-backed login state
is retained too; credentials stored elsewhere are outside this mount.
Keep this directory outside Git. A custom `CODEX_HOME` must use its own
persistent mount; this setup mounts the default location only.

**Before the first rebuild**, preserve any existing container-only state you
want to keep. Quit active Codex sessions, then run these commands from a
**Docker-host terminal** (the remote host for an SSH workflow), provided
`~/.codex-linux` does not already exist:

```sh
mkdir -m 700 "$HOME/.codex-linux" &&
  docker cp aiwf-dev:/home/vscode/.codex/. "$HOME/.codex-linux/"
```

If the destination already exists, reconcile it before copying; do not overwrite
an existing Codex state directory blindly. Without this copy, the new mount
starts with its own state and does not contain the old container's sessions.

Apply the configuration with VS Code's **Dev Containers: Rebuild Container**.
Then, in the container terminal:

```sh
npm_codex_prefix=$(npm prefix -g)
"$npm_codex_prefix/bin/codex" --version
command -v codex
codex login status
# If not logged in:
codex login --device-auth
# From the repository directory:
codex resume
```

If device login is unavailable for your account, follow the CLI's login guidance.
Run `codex` for a new session. Installing the CLI and materializing aiwf artifacts
are separate steps: init.sh performs both. For an existing checkout, run
`aiwf update` with the intended host selection, then start a fresh session to
load its generated guidance and skills. See [host setup](../README.md#2-host-setup-and-embedded-rituals).

To verify persistence, record the npm binary version and login status, retain a
known session and configuration value, then rebuild and check them again.
Repeating initialization must skip the npm install when that binary exists.
The host mount preserves files; it does not keep a running session or tmux
server alive through container replacement.

## Environment variables

The container reads these from the host VS Code session or from
`.devcontainer/devcontainer.env` (gitignored):

| Variable | Default | What it does |
|---|---|---|
| `AIWF_DEVCONTAINER_E2E` | `false` | When `true`, `init.sh` runs `npm install` in `e2e/playwright/` and installs Chromium (~100MB). Default off because most contributors aren't touching the HTML renderer. Set to `true` and rebuild the container to opt in. |
| `AIWF_DEVCONTAINER` | (set by `containerEnv`) | Always `1` inside the container; the eventual `aiwf doctor` containerized-env awareness (sibling milestone) keys on this. |

Outside those, the container inherits `$PATH` and standard host
environment from VS Code's remote session.

## Ritual authoring

The rituals (`aiwfx-*` / `wf-*` skills, role agents, templates) are
authored in-repo in the embedded snapshot at
`internal/skills/embedded-rituals/`, embedded into the `aiwf` binary
via `go:embed`, and materialized for the selected hosts into `.claude/`
or `.agents/` by `aiwf init` / `aiwf update` (ADR-0014, ADR-0016).
A ritual edit is one commit in this repo — there is no separate marketplace repo and no cross-repo
copy step; the upstream marketplace channel that predated this is
archived (ADR-0016, G-0193).

Verify the materialized rituals inside the container with:

```
aiwf doctor          # the `rituals:` line reports materialization status
```

See CLAUDE.md §"Ritual content authoring" for the authoring
workflow and the structural-test discipline that accompanies a
ritual edit.

## Recovery prompt

If the container fails to start, fails postcreate, or otherwise hits
a first-boot failure mode, drop a clean Claude Code session into this
prompt to pick up where the previous session left off:

> You are continuing devcontainer milestone M-0132. Read the milestone
> spec at `work/epics/E-0035-devcontainer-based-dev-loop/M-0132-*/*.md`.
> The container failed to {start | finish postcreate | run `make ci` |
> materialize rituals | mount workspace correctly | …}. Diagnose per the
> `## First-boot recovery` section of that spec. If the failure isn't
> listed there, add it as a new entry before fixing, so the next
> failure of the same shape is one-shot.

The `## First-boot recovery` section in the milestone body is the
durable handoff payload. It grows as new failure modes are discovered
— **add the entry before fixing**, not after, so the next session
hitting the same shape has the answer in hand.
