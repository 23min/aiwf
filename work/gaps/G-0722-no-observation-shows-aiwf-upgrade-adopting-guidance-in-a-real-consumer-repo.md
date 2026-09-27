---
id: G-0722
title: No observation shows aiwf upgrade adopting guidance in a real consumer repo
status: open
discovered_in: M-0353
---
## What's missing

E-0095's first success criterion, and M-0353/AC-2 (deferred), claim what `aiwf upgrade` does when run unattended in a real consumer repository: install, route and report its applicable guidance, or report a handover blocker and its fix. Only a hermetic stand-in backs that claim: `TestBinary_UnattendedUpgradeAdoptsGuidance` in `internal/cli/integration/upgrade_cmd_test.go`. It replaces the `go` toolchain with a fake installer and the published catalogue at `https://github.com/23min/engineering-guidance.git` with a one-pack local catalogue, and it runs with no ai-dotfiles installation. The observation M-0353/AC-2 specifies has not been made.

M-0353 requires the observation to run against the released binary for the epic, and no release contains E-0095. Measured in the E-0095 epic worktree, where `bb0f480be` is the first commit on the epic branch that is not on `main`:

```
$ git merge-base --is-ancestor bb0f480be v0.38.0; echo $?
1
$ git tag --contains bb0f480be
$
```

## Why it matters

The hermetic test leaves out conditions the consumer repository behind E-0095 had: the published catalogue's real pack set and detection patterns, and an ai-dotfiles installation whose delivery must be handed over. Until the observation is made against the first release that contains E-0095, that release can ship with upgrade adopting nothing in such a repository, or blocked on a handover whose reported fix does not clear it, and nobody has looked.
