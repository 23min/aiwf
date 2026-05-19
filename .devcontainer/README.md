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
(`mcr.microsoft.com/devcontainers/go:1-1.25-bookworm`) and the three
declared features. Subsequent builds use cached layers. The build
generates `.devcontainer/devcontainer-lock.json` pinning resolved
feature SHAs — commit this file once it lands so future builds
reproduce exactly.

Repo location: this repo cloned at `~/Projects/aiwf/` (or any
sibling-tree path — the workspace mount goes one level up so
siblings like `~/Projects/ai-workflow-rituals/` are reachable
inside).

## Reopen in Container

In VS Code at this repo's root:

1. Install the **Dev Containers** extension if you haven't already.
2. Command Palette → `Dev Containers: Reopen in Container`.
3. The first open builds the image (slow) and runs
   `.devcontainer/init.sh` (idempotent install of golangci-lint,
   gofumpt, govulncheck, Claude Code CLI, aiwf binary, framework
   hooks). Subsequent opens reuse the cached image.
4. After init completes, the banner in init.sh prints the manual
   step you still need: install both rituals plugins at PROJECT
   scope via the `/plugin` menu inside Claude Code. The CLI form
   defaults to USER scope (wrong); use the interactive menu.

Verify the container is set up correctly:

```
aiwf doctor          # No recommended-plugin-not-installed warnings.
make ci              # vet + lint + test-race + coverage + selfcheck green.
```

## Environment variables

The container reads these from the host VS Code session or from
`.devcontainer/devcontainer.env` (gitignored):

| Variable | Default | What it does |
|---|---|---|
| `AIWF_DEVCONTAINER_E2E` | `false` | When `true`, `init.sh` runs `npm install` in `e2e/playwright/` and installs Chromium (~100MB). Default off because most contributors aren't touching the HTML renderer. Set to `true` and rebuild the container to opt in. |
| `AIWF_DEVCONTAINER` | (set by `containerEnv`) | Always `1` inside the container; the eventual `aiwf doctor` containerized-env awareness (sibling milestone) keys on this. |

Outside those, the container inherits `$PATH` and standard host
environment from VS Code's remote session.

## Recovery prompt

If the container fails to start, fails postcreate, or otherwise hits
a first-boot failure mode, drop a clean Claude Code session into this
prompt to pick up where the previous session left off:

> You are continuing devcontainer milestone M-0132. Read the milestone
> spec at `work/epics/E-0035-devcontainer-based-dev-loop/M-0132-*/*.md`.
> The container failed to {start | finish postcreate | run `make ci` |
> install plugins | mount workspace correctly | …}. Diagnose per the
> `## First-boot recovery` section of that spec. If the failure isn't
> listed there, add it as a new entry before fixing, so the next
> failure of the same shape is one-shot.

The `## First-boot recovery` section in the milestone body is the
durable handoff payload. It grows as new failure modes are discovered
— **add the entry before fixing**, not after, so the next session
hitting the same shape has the answer in hand.
