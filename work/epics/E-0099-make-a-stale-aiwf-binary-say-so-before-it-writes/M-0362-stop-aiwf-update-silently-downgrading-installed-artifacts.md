---
id: M-0362
title: Stop aiwf update silently downgrading installed artifacts
status: in_progress
parent: E-0099
tdd: required
acs:
    - id: AC-1
      title: An older release leaves installed artifacts unchanged
      status: cancelled
    - id: AC-2
      title: An unorderable pair refreshes and reports it could not compare
      status: cancelled
    - id: AC-3
      title: A newer release or an unstamped tree refreshes as before
      status: open
---

## Goal

`aiwf update` run by a release older than the one that last wrote a repository's
artifacts leaves them unchanged and says so, instead of rolling them back and reporting
`updated`.

## Closes

- G-0600 — `aiwf update` no longer rolls installed artifacts back when run by an older
  release.

## Context

`aiwf update` writes every materialized artifact from the running binary's embedded
snapshot through `initrepo.RefreshArtifacts`, without consulting what is on disk
(G-0600). The guidance fragment, which every refresh rewrites, carries the writing
binary's version as `aiwf-version:` (`internal/skills/guidance.go`), so its stamp stands
for the last refresh. `decideStatuslineRefresh` (`internal/skills/statusline.go`) already
guards the statusline with `version.Compare(binary, installed)`.

## Acceptance criteria

### AC-1 — An older release leaves installed artifacts unchanged

**Pass criterion**: in a repository whose guidance fragment is stamped with a release
newer than the running binary's (for example stamp `v0.42.0`, binary `v0.41.0`, both
tagged), `aiwf update` writes nothing the refresh would otherwise write, exits 0, and
prints a line naming both versions and that it did not downgrade. **Edge cases**: the
same with `aiwf upgrade`'s re-executed update; a stamp equal to the binary's version is
not a downgrade (AC-3). **Code references**: the comparison reuses `version.Compare` as
`decideStatuslineRefresh` does; it lands where `RefreshArtifacts` starts or in its
`update` caller (`internal/cli/update/update.go`); a binary-built test in
`internal/cli/integration/` stamps the binary through `-ldflags`.

### AC-2 — An unorderable pair refreshes and reports it could not compare

**Pass criterion**: when `version.Compare` returns `SkewUnknown` for the fragment's stamp
and the running binary's version (for example stamp `main@v0.40.0-677-g4ba3a6b2b`,
binary `v0.41.0`, or a pseudo-version on either side), `aiwf update` refreshes as it does
today and prints a line naming both versions and that it could not compare them.
**Edge cases**: a pre-release tag on either side; a `+dirty` or `-dirty` version.
**Code references**: the same comparison site as AC-1; the integration test of AC-1.

### AC-3 — A newer release or an unstamped tree refreshes as before

**Pass criterion**: a binary newer than or equal to the fragment's stamp, a repository
with no guidance fragment, and a fragment with no `aiwf-version:` stamp all refresh
exactly as today, with no downgrade or comparison line. **Edge cases**: `aiwf init` in
an empty repository; a fragment whose stamp line is malformed (treated as unstamped).
**Code references**: the same comparison site; the integration test of AC-1, plus the
existing update and init tests, unchanged.

## Constraints

- Reuse `version.Compare`; no new version ordering.
- A skipped refresh writes nothing it would have refreshed, and the run's exit status
  stays 0.
- No network access for the comparison.

## Design notes

- The fragment's stamp stands for the whole refresh; no other artifact family gains a
  stamp.
- The skip covers the whole `aiwf update` run: it returns before `RefreshArtifacts`, so
  an older binary leaves every artifact, hook and the health file untouched.

## Out of scope

- An override that forces a downgrade.
- A guard in `aiwf worktree add`, and stamps on skills, templates or agent cards
  (E-0099's *Out of scope*).

## Dependencies

- None.

## References

- G-0600
- E-0099
