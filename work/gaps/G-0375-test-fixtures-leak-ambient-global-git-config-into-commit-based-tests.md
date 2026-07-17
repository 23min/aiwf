---
id: G-0375
title: Test fixtures leak ambient global git config into commit-based tests
status: open
priority: high
discovered_in: M-0186
---
## What's missing

`gitops.Commit`/`gitops.CommitAllowEmpty` shell out to literal `git commit`, which has always consulted `commit.gpgsign` from the full merged config (system → global → local) — this is baseline git behavior, not anything aiwf added. Nineteen test files across `internal/gitops`, `internal/verb`, `internal/cli/integration`, and `internal/policies` build fixture repos via `gitops.Init` and then commit through this path (or, as of M-0186/AC-3, through `gitops.CommitTree`, which AC-4 made equally `commit.gpgsign`-aware for parity with `git commit`) without ever pinning `commit.gpgsign` to a known value in the fixture. Only 4 pre-existing test files defensively set `commit.gpgsign=false` locally. Every other fixture inherits whatever the invoking machine's real global (`~/.gitconfig`) or system config says.

Reproduced directly: pointing `HOME` at a directory whose `.gitconfig` sets `commit.gpgsign = true` (a realistic, not exotic, personal default for a security-conscious contributor) with no working signing key/agent produces 221 failures in `internal/verb` and 62 in `internal/gitops` — nearly the entire suite that touches a verb-commit or gitops-commit path. `git stash`-ing every M-0186 change and re-running confirmed `TestCommitAllowEmpty` (code untouched by this milestone) fails identically, proving the exposure predates this epic entirely; it has existed for as long as `gitops.Commit`/`gitops.CommitAllowEmpty` have.

## Why it matters

CI has never been affected — GitHub Actions runners start from a fresh VM with no global git config, and the workflows only ever set `user.name`/`user.email` globally, never `commit.gpgsign`. But any contributor running `go test` locally with `commit.gpgsign=true` set globally (and no default signing key configured, or a revoked/expired one, or no agent running) would see the bulk of the test suite fail with `gpg failed to sign the data` — a confusing, unrelated-looking failure with no connection to whatever change they were actually testing.

A first attempt at a blanket fix (`GIT_CONFIG_GLOBAL=/dev/null` + `GIT_CONFIG_SYSTEM=/dev/null` in `testsupport.HardenGitTestEnv`, the existing single chokepoint already used for exactly this class of "insulate fixtures from the invoking environment" concern) was tried and reverted: `internal/policies`'s cell-coverage fixtures (`internal/cellcoverage.NewCellFixture`, used by dozens of tests) deliberately never set `user.email`/`user.name` locally — they rely on inheriting identity from the real global config, because they exist specifically to exercise aiwf's own actor-resolution feature ("Identity is runtime-derived from `git config user.email`, not stored in `aiwf.yaml`" — a load-bearing design decision, not an oversight). Cutting off global config entirely broke that intentional dependency (`no actor: pass --actor ... or set git config user.email`) while fixing the `commit.gpgsign` leak. `GIT_CONFIG_COUNT`/`GIT_CONFIG_KEY_n`/`GIT_CONFIG_VALUE_n` (the mechanism `HardenGitTestEnv` already uses for `gc.auto`) isn't a usable substitute either — it's `-c`-equivalent, the highest config precedence tier, so it overrides even a fixture's own repo-local config; forcing `commit.gpgsign=false` that way would make it impossible for a test to ever locally opt into signing to test it (confirmed empirically: an env-injected `false` beat a repo-local `true`).

So the fix is genuinely per-key, not a single process-wide toggle: `user.email`/`user.name` must keep resolving from real global config (by design); `commit.gpgsign` must not leak from it (by test hygiene). No existing mechanism in this codebase does "safe default, locally overridable, single specific key" across the whole test binary — only per-fixture explicit local config does, and most fixtures don't do it.

## Possible directions (not decided)

- Audit every fixture that builds a repo via `gitops.Init` and have it explicitly set `commit.gpgsign=false` locally (matching the 4 files that already do this), rather than depending on inherited global config for a key the fixture doesn't care about. Mechanical but touches many files across several packages.
- Give `internal/cellcoverage.NewCellFixture` (and any other shared fixture builder relying on inherited identity) its own explicit local `user.name`/`user.email`, decoupling actor-resolution tests from the *global* config tier specifically while still exercising the "resolve from `git config`" behavior via local config — then a process-wide `commit.gpgsign=false`-only default (via a real redirected global-config *file* rather than `/dev/null`, which local config would still be free to override) becomes safe to add centrally.
- Cheapest, narrowest: scope the fix to just the newly-exposed surface (`internal/gitops`'s own `CommitTree`-based tests) and leave the wider, pre-existing `Commit`/`CommitAllowEmpty` exposure as accepted, documented risk for a dedicated follow-up.
