---
id: M-0359
title: Generate aiwf's devcontainer from the kit with its own steps in project hooks
status: draft
parent: E-0098
tdd: advisory
acs:
    - id: AC-1
      title: The generated devcontainer mounts only the repository
      status: open
    - id: AC-2
      title: The container's Go toolchain matches CI's pinned version
      status: open
    - id: AC-3
      title: Container creation builds aiwf from source with CI-matched tools
      status: open
    - id: AC-4
      title: No reference points at a removed devcontainer file
      status: open
    - id: AC-5
      title: The parent-mount removal is recorded as a decision
      status: open
    - id: AC-6
      title: aiwf's operator notes cover the Playwright opt-in and recovery
      status: open
---

## Goal

Replace aiwf's hand-written `.devcontainer/` with one generated from the house
devcontainer kit, keep every step only this repository needs in the kit's
repository-owned `project/` hooks, and leave no test, doc or CI step pointing at a
file the move removes.

## Closes

- (none)

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

**Pass criterion**: the operator notes the milestone settles on state how to opt
into the Playwright install (`AIWF_DEVCONTAINER_E2E=true`, then rebuild) and how to
recover a container whose creation failed (the command the kit's post-create
prints), asserted structurally within the section that holds them. **Edge cases**:
the notes placed in the kit-generated README, which a template update may rewrite.
**Code references**: the chosen file, and its policy test under `internal/policies/`.

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
  aiwf's own installs and the golangci-lint match) and the devcontainer half of
  gitleaks enforcement.
- Removed: `init.sh`, the hand-written `initialize.sh` (replaced by the kit's),
  `devcontainer-lock.json` (the kit ignores it).
- Reworded: `internal/cli/doctor/env.go`'s comments and messages,
  `scripts/git-hooks/pre-commit`, the comment in `.github/workflows/gitleaks.yml`,
  and `CLAUDE.md`'s devcontainer section.

## Surfaces touched

- `.devcontainer/`
- `internal/policies/`
- `internal/cli/doctor/env.go`
- `CLAUDE.md`

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

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
