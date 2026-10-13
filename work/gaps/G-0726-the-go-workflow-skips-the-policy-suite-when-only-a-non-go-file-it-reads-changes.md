---
id: G-0726
title: The go workflow skips the policy suite when only a non-Go file it reads changes
status: open
discovered_in: M-0359
---
## What's missing

The `go` workflow runs the `internal/policies` suite, but its `on.push.paths` and
`on.pull_request.paths` filters in `.github/workflows/go.yml` list only Go sources, `go.mod`,
`go.sum`, `.golangci.yml`, `go.yml` and `gitleaks.yml`, `.devcontainer/**`, the `Makefile`, the
embedded trees and the development guidance (`CLAUDE.md`, `AGENTS.md`, `.guidance/**`).
Several files the suite reads match none of those patterns, so a push that changes only such a
file starts no run of the policies that judge it:

- `README.md`, `CONTRIBUTING.md`, `docs/adr`, `docs/design`, `docs/reference` and `scripts/` —
  read by `PolicyDevcontainerPathsResolve` (`internal/policies/devcontainer_paths_resolve.go`);
- `.github/workflows/flake-hunt.yml` — read by `internal/policies/race_parallel_cap.go` and
  `internal/policies/flake_hunt_package_isolation.go`;
- `.github/workflows/changelog-check.yml` — read by
  `internal/policies/changelog_audit_wiring_test.go` and
  `internal/policies/release_heading_matches_tag_gate_test.go`;
- `scripts/git-hooks/pre-push` — its gitleaks install hint is compared with CI's pin by
  `TestGitleaksEnforcement_PinnedVersionConsistent`
  (`internal/policies/gitleaks_enforcement_test.go`);
- every file the development guidance cites, such as `docs/workflows.md` and
  `scripts/sign-and-run.sh` — checked to exist by `PolicyGuidancePathsResolve`
  (`internal/policies/guidance_paths_resolve.go`), so moving or deleting one outside the
  filtered paths starts no run.

A change to one of these alone can break a policy with nothing on the push to say so; the red
run appears on the next push that touches a filtered path, attached to a commit that did not
cause it. `CLAUDE.md`'s local cadence rule lets a contributor skip `make ci` for the same change, since
none of these is on its list.

Nothing holds the filter to what the suite reads: deleting the three guidance entries from both
filters leaves `go test -count=1 ./internal/policies/` passing, measured 2026-10-11 in the aiwf
devcontainer.

This is a claim about the filter's shape: compare the `paths:` lists in
`.github/workflows/go.yml` with the files the named policies open. GitHub's run history was not
queried; `gh run list --workflow go --commit <sha>` on a push that touches only `CLAUDE.md`
would show it.

The comment in `scripts/git-hooks/pre-commit` that lists the inputs it leaves to CI (`CLAUDE.md`,
`docs/`, `.gitleaks.toml` and others) says an edit to one "falls through to CI's unconditional
run"; the filter above makes that run conditional on the paths it lists.

## Why it matters

The policy suite is the CI tier that holds the repository's invariants. Where a push can change
an input without running the suite, the invariant is checked late and blamed on the wrong
commit, and a reviewer reading a green push has no signal that the policies never ran.
