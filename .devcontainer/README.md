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

## Cross-repo plugin testing (rituals repo)

The rituals + `aiwfx-*` skills change frequently. Their canonical
location is the sibling repo at `~/Projects/ai-workflow-rituals/`
on the host, distributed via the Claude Code marketplace. Iteration
is fixture-first in this repo per CLAUDE.md *"Cross-repo plugin
testing"* — author the SKILL.md change at
`internal/policies/testdata/<skill-name>/SKILL.md`, TDD against the
fixture here, then copy the fixture into the rituals repo at wrap
time.

**The mount is free.** `devcontainer.json`'s `workspaceMount`
binds `${localWorkspaceFolder}/..` onto `/workspaces/`, so any
sibling repo under `~/Projects/` on the host is reachable inside
the container by name. With the rituals repo cloned at
`~/Projects/ai-workflow-rituals/`, the container sees it at
`/workspaces/ai-workflow-rituals/`. No additional `devcontainer.json`
config needed; M-0132's `PolicyM0132DevcontainerShape` pins the
`${localWorkspaceFolder}/..` pattern explicitly for this use case.

Sanity check inside the container:

```
ls /workspaces/ai-workflow-rituals/plugins/   # should list aiwf-extensions + wf-rituals
aiwf doctor | grep plugin-mount               # should report `ok (N plugin entries cached)`
gh auth status                                # inherited via the gh credential mount
```

**The wrap-side copy step is one command** (closes the
[CLAUDE.md flow](../CLAUDE.md) step that previously asked the
operator to construct a 6-segment path by hand):

```
make copy-skill-fixture SKILL=aiwfx-start-epic
```

The target asserts: the `SKILL` variable is set, the fixture
exists at `internal/policies/testdata/$(SKILL)/SKILL.md`, the
sibling rituals repo is reachable at `../ai-workflow-rituals`, and
the destination `plugins/<plugin>/skills/$(SKILL)/SKILL.md` exists
under it. Refuses with a clear stderr message if any precondition
fails — no partial copies.

After the copy: `cd ../ai-workflow-rituals && git diff && git
commit + git push` from the rituals repo. The container's gh
credential helper (set up by `.devcontainer/init.sh`) handles the
push; no extra auth step.

**G-0146 status.** Half-step closure: the mount + the docs + the
copy-step automation make the flow reachable end-to-end in the
container. The full end-to-end smoke (a script gating CI on
fixture → rituals-copy → drift-check round-trip) is deferred
until a forcing function names what shape the smoke assertion
should take. See G-0146 archive for the original problem framing.

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
