---
id: G-0554
title: The statusline CI cache mints a file per commit and deletes none
status: open
priority: medium
---
## What's missing

The statusline's CI-verdict cache creates one file per commit and deletes none.

`statusline.sh` keys the cache on `sha1(repo_root/branch/HEAD_sha)` and writes
`${AIWF_STATUSLINE_CACHE_DIR:-/tmp}/aiwf-statusline-ci-<key>`. Folding HEAD into
the key is deliberate and correct — it is what stops a stale pre-commit verdict
being served for up to the TTL after a new commit. The consequence is that every
commit on every branch mints a new cache file, and the script contains no
deletion of any kind: no `rm`, no `find -mtime`, no TTL sweep, no bound on the
directory.

Measured in one devcontainer: 996 files, the oldest twelve days old. Nothing had
removed a single one.

## Why it matters

The size is negligible and that is precisely why it has gone unseen — the files
are a few bytes each, so no disk-usage investigation surfaces them. What grows
without bound is the *count*, at one per commit, forever.

It also ships. This is `internal/skills/embedded-statusline/statusline.sh`,
materialized into the repo of every consumer who runs `aiwf init --statusline`,
so the accumulation happens in their `/tmp` rather than only in aiwf's own
development environment. On a shared machine it litters a directory other users
share, and a directory entry count that only rises eventually costs more than
the bytes ever did.

G-0552 is a different subject: it bounds the Go build cache and adds a
free-space preflight for this repo's development environment, and scopes itself
there deliberately. This one is a defect in shipped content.

## Resolution shape

Two directions, and the second is stronger.

**Prune on write.** Before writing a new entry, delete `aiwf-statusline-ci-*`
older than a small multiple of the TTL. Cheap, local to the one function, and
leaves the key as it is. It costs a `find` on every statusline render, which is
a hot path.

**Bound the key instead.** Drop HEAD from the filename and keep it *inside* the
file alongside the verdict, so the entry is overwritten in place on each new
commit and the file count is bounded by the number of branches rather than by
the number of commits. The staleness guard reads the recorded sha and treats a
mismatch exactly as it treats an expired TTL, preserving the property that
folding HEAD into the key was added to protect. No sweep, no hot-path cost, and
the bound holds by construction rather than by a cleanup step that can be
skipped.

The second also answers the same question for `AIWF_STATUSLINE_CACHE_DIR`
pointed anywhere else, which a prune-by-glob would have to be careful not to
over-reach in.
