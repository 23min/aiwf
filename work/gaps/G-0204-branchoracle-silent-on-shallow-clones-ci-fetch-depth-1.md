---
id: G-0204
title: BranchOracle silent on shallow clones (CI fetch-depth=1)
status: open
discovered_in: M-0158
---
M-0106's `BranchOracle` indexes per-commit first-parent
reachability via `git rev-list --first-parent <branch>` per ritual
branch at oracle construction time
([`internal/cli/check/isolation_escape_oracle.go`](../../internal/cli/check/isolation_escape_oracle.go)).
The command honors the repository's shallow boundary: if the
clone is shallow (e.g. GitHub Actions default `actions/checkout`
uses `fetch-depth: 1`), `rev-list` returns only the commits within
the shallow window. Commits beyond the boundary are absent from
the oracle's index.

The rule treats an unknown SHA as "silent" — no finding fires. In
shallow-clone environments, this means **every AI-actor commit
beyond the shallow boundary silently bypasses the
`isolation-escape` check.**

## Concrete failure mode

A GitHub Actions workflow that runs `aiwf check` after `actions/checkout`
with default settings indexes only HEAD's first-parent. Any AI
commit older than HEAD-1 returns empty from
`FirstParentBranches(sha)` → silent.

For a repo with regular CI, this means:
- Branch protection that relies on `aiwf check` doesn't actually
  enforce branch-binding for the bulk of history.
- A subagent that escaped on an older commit lands on the
  protected branch silently.

## What's needed

The oracle should detect the shallow state and either:
- Refuse construction with a typed error the caller can surface
  as an `oracle-incomplete` finding (preferred — fail loud).
- Fall back to a different reachability strategy (`git merge-base
  --is-ancestor`) for individual SHAs (slower but works on shallow
  clones).
- Document the requirement: `aiwf check` requires a complete
  history; CI workflows must use `fetch-depth: 0`.

The detection itself is straightforward: `git rev-parse --is-shallow-repository`
returns "true" on shallow clones.

## Why parked

The M-0158 honest-scope audit surfaced this. The current oracle
ships with no shallow-clone detection; the M-0106 spec didn't
enumerate this scenario and tests run on full clones. Address as
part of the real-world hardening milestone.
