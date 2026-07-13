---
id: G-0412
title: ResolveRoot coverage-ignore rationale text is inaccurate across multiple files
status: open
discovered_in: M-0253
---
## What's missing

The blessed `//coverage:ignore` rationale for `cliutil.ResolveRoot`'s
error branch — "cliutil.ResolveRoot only fails on missing aiwf.yaml +
non-existent --root path" — is factually wrong on both counts.
Reading `resolveroot.go`: a missing `aiwf.yaml` is not a failure (the
function silently falls back to cwd), and a non-existent `--root`
path is not statted (`filepath.Abs` succeeds regardless of whether the
path exists). The branch is genuinely unreachable in a normal test
harness — it only fires on a `filepath.Abs`/`os.Getwd` fault — so the
ignore itself is legitimate; only the stated reason is loose.

This wording originates at `internal/cli/archive/archive.go:127` and
has since been copied verbatim into `internal/cli/renamearea`,
`internal/cli/setarea`, and every M-0252/M-0253 file that ignores this
branch (`add`, `cancel`, `promote`, `reallocate`, `rename`, `retitle`,
`milestone`, `update`, `editbody` — nine files and counting as of
M-0253's wrap, with M-0254 through M-0256 still to land more).
Independently flagged by two reviewers during M-0253's wrap.

## Why it matters

A `//coverage:ignore` is a trust boundary: an honest one records real
unreachability, and each new instance widens the surface a future
reader inherits without re-verifying. The finding isn't blocking
today's milestones — the branches genuinely aren't triggerable — but
the copied-verbatim inaccuracy will keep propagating into M-0254
through M-0256, and a repo-wide sweep gets more expensive the longer
it waits. A single corrected rationale string, applied everywhere this
exact ignore text appears, closes it in one small patch: something
like "ResolveRoot's error path (a filepath.Abs/os.Getwd fault) is not
portably reproducible in a unit-test harness."