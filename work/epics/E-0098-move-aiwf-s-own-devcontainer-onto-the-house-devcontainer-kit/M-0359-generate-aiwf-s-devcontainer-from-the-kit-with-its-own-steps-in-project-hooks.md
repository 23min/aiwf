---
id: M-0359
title: Generate aiwf's devcontainer from the kit with its own steps in project hooks
status: in_progress
parent: E-0098
tdd: advisory
acs:
    - id: AC-1
      title: The generated devcontainer mounts only the repository
      status: met
    - id: AC-2
      title: The container's Go toolchain matches CI's pinned version
      status: met
    - id: AC-3
      title: Container creation builds aiwf from source with CI-matched tools
      status: met
    - id: AC-4
      title: No reference points at a removed devcontainer file
      status: met
    - id: AC-5
      title: The parent-mount removal is recorded as a decision
      status: met
    - id: AC-6
      title: aiwf's operator notes cover the Playwright opt-in and recovery
      status: met
---

## Goal

Replace aiwf's hand-written `.devcontainer/` with one generated from the house
devcontainer kit, keep every step only this repository needs in the kit's
repository-owned `project/` hooks, and leave no test, doc or CI step pointing at a
file the move removes.

## Closes

- G-0524 — the workspace mount narrows to the checkout, with explicit sibling
  mounts recorded as the replacement for the reach the parent mount gave.

## Context

The epic's open questions are settled: no sibling mounts; kit answers
`languages: [go, node]` and `aiwf: false`; the Go toolchain pinned through
`GOTOOLCHAIN` to `go.yml`'s `GO_VERSION`; each policy test that pins the current
files retired or re-pointed as listed under Design notes. The kit (v0.4.2 at
planning) mounts the same host state the current container does.

## Acceptance criteria

### AC-1 — The generated devcontainer mounts only the repository

**Pass criterion**: a policy test parses `.devcontainer/devcontainer.json` and
fails unless `workspaceMount` binds `${localWorkspaceFolder}` itself at
`/workspaces/aiwf`, and no entry in `mounts` binds the checkout's parent.
**Edge cases**: a `/..` suffix on the source; a mount whose source is
`${localWorkspaceFolder}/..` added under `mounts` rather than `workspaceMount`; the
`//` comment lines the kit writes, which the parser must strip before decoding.
**Code references**: a new policy under `internal/policies/`, replacing
`m0132_devcontainer_shape.go`.

### AC-2 — The container's Go toolchain matches CI's pinned version

**Pass criterion**: a policy test fails unless `containerEnv.GOTOOLCHAIN` in
`.devcontainer/devcontainer.json` equals `go` followed by the `GO_VERSION` value in
`.github/workflows/go.yml`. **Edge cases**: `GOTOOLCHAIN` absent; `GO_VERSION`
absent or not an exact patch version; a `+auto` or `local` suffix on the value.
**Code references**: the same new policy file as AC-1, or its own, under
`internal/policies/`.

### AC-3 — Container creation builds aiwf from source with CI-matched tools

**Pass criterion**: the re-pointed `m0132-init-script` and gitleaks-enforcement
checks read `.devcontainer/project/post-create.sh` and fail unless it runs
`go install ./cmd/aiwf`, `aiwf init --no-prompt` with stdin from `/dev/null`, and
`make install-hooks`, and pins golangci-lint and gitleaks to the versions in
`.github/workflows/go.yml` and `.github/workflows/gitleaks.yml`. **Edge cases**: a
tool installed without a pin; a pin that differs from CI; `aiwf init` without the
`/dev/null` redirect. **Code references**: `internal/policies/m0132_init_script.go`,
`internal/policies/gitleaks_enforcement_test.go`.

### AC-4 — No reference points at a removed devcontainer file

**Pass criterion**: a policy test collects every `.devcontainer/<path>` named in
`CLAUDE.md`, `README.md`, `CONTRIBUTING.md`, the Normative docs tier, `scripts/`,
`.github/` and Go source outside tests' fixtures, and fails on any path that does
not exist; and no policy inventory lists a retired policy id. **Edge cases**: a
path inside a code fence; a glob-shaped path (`project/*.sh`); archival and
entity-tree files, which are excluded. **Code references**: a new policy under
`internal/policies/`; the inventories in `internal/policies/` that name policy ids.

### AC-5 — The parent-mount removal is recorded as a decision

**Pass criterion**: a decision entity is `accepted`, cites G-0524, and states the
replacement for sibling reach: explicit entries in the kit's `siblings` or
`writable_siblings` answers. `aiwf show` on it reports no finding. **Edge cases**:
none. **Code references**: `work/decisions/`.

### AC-6 — aiwf's operator notes cover the Playwright opt-in and recovery

**Pass criterion**: `.devcontainer/project/README.md`, which template updates never
touch, states how to opt into the Playwright install (`AIWF_DEVCONTAINER_E2E=true`,
then rebuild) and how to recover a container whose creation failed (the command the
kit's post-create prints), asserted structurally within the sections that hold
them. Each section of the current `.devcontainer/README.md` is carried over while
still true or dropped where the kit now covers it. **Edge cases**: a section whose
subject the kit's generated README already states. **Code references**:
`.devcontainer/project/README.md`, and its policy test under `internal/policies/`.

## Constraints

- The container runs aiwf built from this checkout, never a released binary.
- Every unattended step reads stdin from `/dev/null` (G-0446).
- Kit-owned files are not hand-edited except `devcontainer.json`, whose hand edits
  template updates merge; everything else aiwf needs lives under `project/`.

## Design notes

- Kit answers: `languages: [go, node]`, `aiwf: false`, `dood: false`, no
  `siblings` or `writable_siblings`, no `data_dir`, aiwf's current extensions in
  `extra_extensions`.
- `devcontainer.json` hand edits: `containerEnv.GOTOOLCHAIN`,
  `containerEnv.AIWF_DEVCONTAINER=1` (read by `aiwf doctor`), and the Go
  formatter and YAML settings the current file carries.
- Policy tests retired: `m0132-devcontainer-shape`, `m0132-initialize-script`,
  `m0132-devcontainer-lock`, `m0132-devcontainer-readme`,
  `m0202-devcontainer-onboarding`, `TestDevcontainerCodexInstall`,
  `TestDevcontainerCodexStateMount`, `TestDevcontainerGuidanceMount`. The kit owns
  what they pinned; its own tests cover the guidance step.
- Policy tests re-pointed to `project/post-create.sh`: `m0132-init-script` (only
  aiwf's own installs, with golangci-lint and govulncheck matched to CI) and the
  devcontainer half of gitleaks enforcement.
- Removed: `init.sh`, the hand-written `initialize.sh` (replaced by the kit's),
  `devcontainer-lock.json` (the kit ignores it).
- Reworded: the `containerEnv` comment in `internal/cli/doctor/env.go`, the comments in
  `.github/workflows/gitleaks.yml` and `.github/workflows/go.yml`, the doc comment in
  `internal/policies/m0210_trailer_commit_drift.go`, and `CLAUDE.md`'s devcontainer section.

## Surfaces touched

- `.devcontainer/`
- `internal/policies/`
- `internal/cli/doctor/env.go`
- `.github/workflows/`
- `CLAUDE.md`, `TODO.md`

## Out of scope

- Building or opening the new container: M-0360.
- Kit changes. A capability this milestone proves missing is surfaced before any
  kit change is made.

## Dependencies

- devcontainer-kit v0.4.2 or later on `gh:23min/devcontainer-kit`.

## References

- E-0098, G-0524, G-0446, G-0292.

---

## Release note

aiwf's own development container is generated from the house devcontainer kit, and opens only at
`/workspaces/aiwf` on this checkout: the folder above it is no longer mounted, so sibling
repositories and any `CLAUDE.md` beside the clone are out of reach, and the container can no
longer be opened on a sibling worktree. A sibling that is needed is added through the kit's
`siblings` or `writable_siblings` answers. The container runs Go at CI's `GO_VERSION`, builds
`aiwf` from the checkout, and installs golangci-lint, govulncheck and gitleaks at the versions CI
runs, plus gofumpt and goimports. Node is the kit's current LTS rather than a pinned 22. Git
identity comes from the host's global git config. The Go module and build caches and the npm
cache live in named volumes shared by every kit container, so a rebuild no longer clears them.
The Go extension's helper tools, the `dlv` debugger among them, are no longer preinstalled;
install them with **Go: Install/Update Tools**. Before each start, a `core.hooksPath` naming this
repository's own hooks directory by its full host path is removed from `.git/config` when that
changes nothing on the host, so git hooks run in the container too. Codex installs through the kit at each
start instead of through npm. `.devcontainer/project/README.md` covers aiwf's additions, the Playwright
opt-in and recovery from a failed container creation. Nothing changes for repositories that use
aiwf.

## Decisions made during implementation

- D-0104 — the container mounts only the checkout; siblings come through the kit's answers.
- The re-pointed `m0132-init-script` policy is `devcontainer-project-post-create`, named for the
  hook it checks, since `.devcontainer/init.sh` no longer exists.
- govulncheck is pinned to CI's version (v1.6.0, down from the v1.7.0 the container ran) and
  compared with `.github/workflows/go.yml`, as golangci-lint is.
- A `core.hooksPath` naming the repository's own hooks directory by its absolute host path is
  removed on the host, before every start, by `.devcontainer/project/hooks-path.sh` (run from
  `project/initialize.sh`): only when every value set is that directory, a trailing slash
  allowed, and none comes from the global or system git config, so git's default on the host is
  the same directory and nothing changes there, while the hooks come back in the container, where
  that path does not exist. Anything else is left alone, a spelling with `~` among it, and the
  script never stops a start (`TestDevcontainerHooksPathRepair`). The decision is made on the host
  because only the host sees its own global git config.
- Python 3 comes from the kit's `apt_packages` answer (`python3`), which brings the full standard
  library `scripts/growth-report.py` imports; the `python3-minimal` the Node feature installs
  lacks `dataclasses`, so `make ci` would fail without it.

## Validation

Run on the milestone branch at `294ced2fc`, in aiwf's pre-move devcontainer (Linux, Go 1.25.11):

- `make ci` — exit 0: lint 0 issues; `go test -race` 74 packages ok, 0 failing; the diff-scoped
  coverage gate and the firing-fixture gate pass; total statement coverage 91.9%; self-check
  passes all 29 steps.
- `aiwf check` — 0 errors. Warnings: `acs-tdd-audit` for each met criterion, which records no TDD
  phase under `tdd: advisory`; `provenance-untrailered-scope-undefined`, since the branch has no
  upstream.
- Each new policy was run against the real tree with its subject broken and failed as stated:
  the parent mount, a `GOTOOLCHAIN` of `go1.27.1`, the recovery command shortened in the notes,
  the pre-change `gitleaks.yml` comment naming `.devcontainer/init.sh`, govulncheck pinned at
  v1.7.0 against CI's v1.6.0, gitleaks and govulncheck installed at `@latest`. Fixtures fail on
  each required hook command removed, on `make install-hooks` present only in a comment, on a
  parent bind spelled `src=` in `mounts` or passed as `-v` in `runArgs`, on a removed path
  named in `docs/design`, the `Makefile` or `.devcontainer/project/README.md`, on strict mode or
  a version pin present only in a comment, on a required command inside a trailing comment, on
  another step's `version:` ahead of the golangci-lint action's, on that action's version
  carrying a suffix (`v2.11.4-rc1`), on a suffixed pin in the hook or on govulncheck's or
  gitleaks's install line, and on a gitleaks install or pin present only in a comment.
- `hooks-path.sh`, run in throwaway repositories with an isolated global config
  (`TestDevcontainerHooksPathRepair`): the repository's own absolute hooks directory is unset,
  with a trailing slash too and when set twice; it is kept when the global config also sets
  `core.hooksPath`, for another directory, for a relative path, and when the unset fails, and the
  script exits 0 in each case. Dropping the global-scope guard, dropping the own-directory match,
  or exiting 1 on a failed unset each fails that test. `project/initialize.sh`, copied into such
  a repository and started from an unrelated directory, exits 0 and repairs that repository
  (`TestDevcontainerHooksPathRepairRunsOnTheHost`); without its `cd` to the checkout, or with the
  repair call commented out, that test fails. Both tests also pass with `TMPDIR` behind a symlink,
  as macOS's is, and fail there when the throwaway repository's path is not resolved through it.
- `shellcheck -x .devcontainer/project/*.sh` — clean.
- `gitleaks git --config=.gitleaks.toml`, the scan CI and the pre-push hook run, at `2d9599a0e` —
  no leaks found, exit 0; the macOS home paths with an invented user name that earlier commits on this branch
  recorded are accepted by fingerprint in `.gitleaksignore`.
- Not verified: that `AIWF_DEVCONTAINER_E2E=true` set in `containerEnv` reaches the project hook
  at container creation; `make e2e-install`, which the notes give first, does not depend on it.
  Nor that the built image's `python3` imports `dataclasses`: Ubuntu 24.04's `python3` depends on
  `python3.12`, which depends on `libpython3.12-stdlib`, but no image was built. Building the
  container and running `make ci` in it is M-0360. Nor that `make e2e` runs on the new Ubuntu
  24.04 base after `make e2e-install`, which installs Chromium without its system libraries; the
  opt-in flow is the old container's, unchanged.

## Deferrals

- G-0725 — `make install-hooks` reports success when it cannot create the hook links.

## Reviewer notes

**Obligations on later changes.** Each rule below is held by the named check, run with
`go test -count=1 -run '<pattern>' ./internal/policies/`; its owner is whoever makes the change
named, and it retires when the fact it guards leaves the repository.

- A Go bump in `go.yml`'s `GO_VERSION` also edits `containerEnv.GOTOOLCHAIN`
  (`DevcontainerGoToolchain`); retires when the container stops pinning Go there.
- A golangci-lint or govulncheck bump in `go.yml`, or a gitleaks bump in `gitleaks.yml`, also
  edits the pin in `project/post-create.sh` (gitleaks's in the `scripts/git-hooks/pre-push`
  install hint too), and that hook keeps each required step as a command, not a comment
  (`DevcontainerProjectPostCreate|Gitleaks`); retires when the container stops installing its
  own copies.
- `workspaceMount` stays the checkout at `/workspaces/aiwf`, and `devcontainer.json` never names
  `${localWorkspaceFolder}/..` (`DevcontainerWorkspaceMount`); retires with D-0104.
- Every `.devcontainer/` path named in the root guides, the Makefile, the Normative docs,
  scripts, workflows, Go source and `.devcontainer/project/` exists
  (`DevcontainerPathsResolve`); retires with the directory.
- `project/README.md` names the Playwright variable and the kit's recovery command
  (`DevcontainerProjectNotes`); retires with the notes.
- `hooks-path.sh` unsets `core.hooksPath` only when every value is this repository's own hooks
  directory, all from its local config, and exits 0 in every case; `project/initialize.sh` runs
  it on its own checkout from any directory (`DevcontainerHooksPath`). Retires when the kit
  handles a host-only hooks path itself, or the repair is removed.
- Unchecked, held at review: `uvx copier update` must keep the `devcontainer.json` hand edits
  (only `GOTOOLCHAIN` is checked); gofumpt and goimports pins must build with the Go in
  `GOTOOLCHAIN` (`project/post-create.sh` says so); the host-side scripts stay bash 3.2-safe,
  which the tests, run under bash 5, do not show.

**Findings declined.**

- The parent-mount check is a text search for `${localWorkspaceFolder}/..`; a JSON-escaped
  `\/..` or `/./..` passes it. It guards against the parent mount returning as someone would
  type it, and its comment says what it searches for.
- `.devcontainer/` paths are matched without a left boundary, so a URL into another repository's
  `.devcontainer/` is flagged; that failure is loud, and no scanned file has one.
- `AGENTS.md` is not among the files the paths check scans; it names no `.devcontainer/` path.
- The hooks-path tests inherit `GIT_CONFIG_PARAMETERS`; a value there can only make them fail,
  never pass.
- The failed-unset case checks the exit status and the kept value, not the warning's wording.
- Each new policy test repeats the package's write-the-fixture-then-run-the-policy loop rather
  than sharing a helper; every policy test in the package is written that way.
- Nothing runs the hooks-path tests with a symlinked temporary directory in CI, which runs on
  Linux; `hooksPathSandbox` is the one place that builds their repositories and resolves it.

**Left as they are.** A `core.hooksPath` written with `~` is not repaired. Container commits take
their identity from the host's global git config, so `aiwf-actor` follows that address; M-0360
observes it.

**Verdict.** (written after the deciding review)
