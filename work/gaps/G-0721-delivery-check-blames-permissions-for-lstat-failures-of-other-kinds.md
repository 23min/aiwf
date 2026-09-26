---
id: G-0721
title: Delivery check blames permissions for lstat failures of other kinds
status: open
discovered_in: M-0352
---
## What's missing

`projectguidance.CheckCompatibility` (`internal/projectguidance/compatibility.go`) refuses guidance with "cannot inspect <path>; repair its permissions, then retry" for every `lstat` failure other than a missing path or a file standing where a directory belongs: once in the `dotfiles-sync` launcher loop over PATH, and once in `compatibilityInstructions` for the personal instruction files. A failure unrelated to permissions, such as a symbolic-link loop, gets the same advice.

Measured with a build of the E-0095 epic branch at M-0352 (`dc81d7569`), in a scratch repository with an isolated HOME and Git configuration, a local guidance source listing `sample/base`, `aiwf.yaml` `hosts: [codex]` with that source, and a PATH entry `loopbin` that is a symbolic link to itself (`ln -s loopbin loopbin`):

```
$ PATH=<git dir>:/usr/bin:/bin:<scratch>/loopbin aiwf update --root r    # exit 0
Guidance:
  selected:  not adopted (guidance.packs unset)
  adopted:   none
  installed: none
  ignored:   none
  blocked:   incompatible ai-dotfiles guidance delivery: cannot inspect <scratch>/loopbin/dotfiles-sync; repair its permissions, then retry: lstat <scratch>/loopbin/dotfiles-sync: too many levels of symbolic links
```

The step ledger's `skipped .guidance (project engineering guidance)` line carries the same advice. A second run prints both again, and `.guidance/` is never created.

## Why it matters

The maintainer is told to repair permissions on a path whose permissions are not the problem, and guidance stays blocked on every run until they find the real cause themselves. A stale or looping PATH entry, which the shell itself skips without complaint, is enough to stop guidance for the whole repository.
