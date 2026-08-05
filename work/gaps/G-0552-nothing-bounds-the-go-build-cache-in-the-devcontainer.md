---
id: G-0552
title: Nothing bounds the Go build cache in the devcontainer
status: open
discovered_in: M-0291
---
## What's missing

Nothing bounds the Go build cache in the devcontainer. It reached 84 GB on a
288 GB overlay and filled the filesystem to 100%.

## Why it matters

The failure is silent and misattributed. A full disk does not announce itself —
it surfaces as test failures, and specifically as failures in whichever tests
write the most: the ones that build binaries or create temporary repositories.
During M-0291's wrap those appeared in three different packages across three
consecutive runs, each run failing a different set, with targeted re-runs
passing. That signature reads as flakiness in the test suite, and the natural
response to it — re-run, then start hunting a race — is wasted effort aimed at
the wrong layer.

It also degrades the gate's meaning while it lasts. A green run on a
nearly-full disk is not evidence, because passing depended on which test
reached the last free blocks first.

## Options

1. **Bound the cache.** Modern Go trims the build cache automatically, and the
   trimming clearly did not keep pace here; an explicit periodic
   `go clean -cache` in container setup, or a smaller cache lifetime, makes the
   ceiling deliberate.
2. **Fail loudly instead.** A preflight in the Makefile's gate targets that
   refuses to run when free space is below a threshold, naming the disk rather
   than letting the suite report it as test failures. This is the one that
   addresses the misattribution rather than the cause.
3. **Both.** The cache bound stops the common case; the preflight catches
   whatever else fills the disk, which the cache bound would not.

Option 3 is the lean, with the preflight carrying most of the value: a bounded
cache is not the only way to fill a disk, but a gate that cannot tell a full
disk from a broken test will mislead every time.

## Scope

Encountered during M-0291's wrap. Not an aiwf kernel concern — it is a
development-environment one — but it cost a review cycle's worth of misdirected
debugging, and it will recur.
