---
id: G-0471
title: No chokepoint detects a verb run by a binary older than the worktree's source
status: open
priority: high
discovered_in: M-0281
---
## What's missing

Working on the kernel means the `aiwf` on `PATH` was built from older source than
the working tree. Every verb then runs older logic — reads *and* writes — with no
signal. Nothing compares the running binary against the source it is operating on.

Two predecessors bound this gap rather than covering it.

**G-0147** is titled "Worktree aiwf binary discipline lacks a mechanical
chokepoint" and is `addressed`. Its resolution added a `make diag-aiwf`
convenience target and a CLAUDE.md section. That section still states, as current
truth, that nothing mechanically blocks a stale-PATH call. The hazard is
documented; the chokepoint its title names does not exist.

**G-0176** shipped real detection — `internal/cli/doctor/binary_staleness.go`
compares a pseudo-version binary's base SHA against `refs/remotes/origin/main`.
Two properties leave this case uncovered:

- It **skips tagged releases by shape**, on the reasoning that they are covered by
  the `latest:` row. That row answers whether the binary is behind the newest
  *published* tag. It does not answer whether the binary is behind the *working
  tree*, and a developer many commits past a tag holds a maximally-stale binary
  that the check declines to examine.
- It lives in `doctor`, which is opt-in. The failure mode arrives precisely when
  an operator does not think to run `doctor`, so a guard reachable only by
  choosing to look cannot catch it.

## Why it matters

The framing "stale binary gives stale answers" understates it. A stale binary
**writes**: it commits entity frontmatter, renames files, materializes `.claude/`
artifacts, and stamps trailers, all using superseded logic, and every result looks
correct because it *is* correct for the code that ran.

Measured over one milestone-wrap session in this repo, with `v0.30.0` on `PATH`
against a working tree well past that tag:

- A milestone's own acceptance criterion appeared to fail. A same-state `retitle`
  returned exit 2 where the criterion requires an exit-0 NoOp. The convergence was
  present in the source and absent from the binary. The investigation that
  followed was spent on working code.
- `aiwf update` materialized the **binary's** embedded skills rather than the
  tree's. `doctor` afterwards reported seven drifted, among them the skill for the
  verb that session had changed. The guidance surface an assistant reads was a
  version behind the code it describes.
- `doctor` was silent throughout, by design, because the binary carried a tag.

The second item is the sharpest: the artifacts that tell an AI how to operate aiwf
were installed by a version that predates the behavior they document. This repo
holds that framework correctness must not depend on an assistant remembering a
rule, and the rule here is one a human is equally free to forget — the
authoritative pre-push hook is wired `PATH`-relative, so in the kernel repo the
local chokepoint validates a different program than the one being pushed.

## Options

1. **Widen the shipped detector.** Drop the tagged-release skip when the module
   path matches the kernel repo, and compare against the working tree's `HEAD`
   rather than `origin/main`. Smallest change, reuses machinery that already
   carries tests, and keeps the advisory-only severity. Leaves the check in
   `doctor`, so it still requires looking.
2. **Move the check ahead of the verb.** A mutating verb invoked inside the kernel
   repo compares binary against source before it writes, and warns — or refuses
   under a strict flag. This is the chokepoint G-0147 names: `doctor` is the wrong
   home for a guard against something that happens when `doctor` is not run.
3. **Have the git hooks build from source** when they detect the kernel repo, so
   pre-commit and pre-push validate the code being committed rather than whatever
   is installed. Closes the self-referential hole most completely, and pays a
   build on every hook invocation.

Options 1 and 2 compose and are the lean: widen the comparison so it covers the
tagged case, then place it where a verb reaches it rather than where an operator
opts in. Option 3 is correct in principle but taxes every commit, and CI already
builds the pushed code from source.

Whichever is chosen, closing this one should not repeat G-0147's shape — a
resolution that documents the hazard leaves the gap open under a closed status.
