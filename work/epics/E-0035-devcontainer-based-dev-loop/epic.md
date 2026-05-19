---
id: E-0035
title: Devcontainer-based dev loop
status: active
---
## Goal

Move aiwf's primary dev loop from the macOS host into a reproducible
Linux devcontainer, where macOS-specific bugs (G-0127 fork/exec
deadlock under `-race` + parallel; G-0128/G-0133 syspolicyd crashes
on unsigned Mach-O binaries) simply don't exist. The existing
host-side workarounds — `scripts/sign-and-run.sh`, in-test
`codesign` blocks, the `-parallel 8` cap — stay as graceful
fallbacks for the rare case where host execution is necessary,
but the canonical dev surface becomes the container. Success
means a fresh checkout + "Reopen in Container" gives any
contributor the same green `make ci` without remembering the
macOS DO/DON'T rules.

## Scope

- **First milestone (this epic's skeleton):** land `.devcontainer/`
  using a features-first composition on
  `mcr.microsoft.com/devcontainers/go:1-1.25-bookworm`, HTTPS + gh
  credential helper for auth, project-scope plugin install via
  postcreate banner, env-gated Playwright install, and the
  plugin-index shadow-mount workaround for
  [claude-code#31388](https://github.com/anthropics/claude-code/issues/31388).
  Container builds clean from cold cache; `make ci` is green
  inside it via VS Code's "Reopen in Container" + integrated
  terminal; plugin install completes at PROJECT scope;
  shadow-mount preserves the host plugin index across container
  restarts.
- **Later milestone — CI matrix integration (Docker-in-Docker).**
  The operator path (VS Code "Reopen in Container") needs no
  automation. A CI matrix that runs `devcontainer build` +
  `devcontainer exec ... make ci` against the same `.devcontainer/`
  shape catches regressions the unit-test suite can't see (a
  feature-SHA drift, an init.sh idempotency bug, a workspace-mount
  path change, a stale `image:` reference). This milestone owns
  the standalone `@devcontainers/cli` install in CI, the smoke
  scripts the operator path doesn't need, and the workflow YAML
  wiring them up. Smoke-script content originally drafted as
  AC-7/AC-8 of the first milestone lands here.
- **Later milestone — CLAUDE.md DO/DON'T refresh.** Once the
  container is the default surface, demote the macOS host
  wrapper from "primary path with DO/DON'T discipline" to
  "fallback when you must run on macOS." The structural fix has
  landed; the documentation should reflect it.
- **Later milestone — `aiwf doctor` containerized-env awareness.**
  Recognize when the verb is running inside a devcontainer
  (e.g., `/.dockerenv` present, `AIWF_DEVCONTAINER=1` from
  containerEnv) and surface container-specific advice
  (shadow-mount status, plugin-index sanity) instead of the
  generic host-side guidance.
- **Later milestone — cross-repo dogfooding hardening.** Verify
  the rituals plugin testing pattern (per CLAUDE.md's
  "Cross-repo plugin testing" section) works inside the
  container against the sibling `~/Projects/ai-workflow-rituals/`
  checkout — the one-level-up workspace mount makes this
  reachable, but the actual flow (fixture authoring → wrap-side
  copy into rituals repo → drift-check test against marketplace
  cache) hasn't been exercised end-to-end inside a container yet.

## Out of scope

- **Production / deployment containers.** Aiwf ships as a Go
  binary via `go install`, never as a Docker image. The
  devcontainer is dev loop only.
- **Proliminal.net devcontainerization.** Host-only by maintainer
  choice; downstream consumers run their own setup.
- **Liminara / FlowTime devcontainer changes.** We mirror their
  patterns (features-first composition, gh credential helper
  repair, /tmp symlink dance, plugin shadow-mount) but don't
  modify their `.devcontainer/` files. The shadow-mount cleanup,
  once claude-code#31388 closes, happens in each repo
  independently.
- **Removing the macOS host-fallback path entirely.**
  `scripts/sign-and-run.sh`, the per-binary `codesign` blocks
  in `internal/cli/cliutil/testutil/proc.go` and
  `internal/policies/m080_test.go`, and the `-parallel 8` cap
  stay in place. Deprecating them is a future call, not part
  of this epic.
- **Adapter support for non-Claude-Code IDEs.** Per
  `docs/pocv3/design/design-decisions.md`, aiwf targets Claude
  Code only; the devcontainer is therefore VS Code + Claude
  Code, not a multi-IDE artifact.
