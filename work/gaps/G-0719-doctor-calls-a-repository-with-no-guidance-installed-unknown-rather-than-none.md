---
id: G-0719
title: Doctor calls a repository with no guidance installed unknown rather than none
status: addressed
discovered_in: M-0352
addressed_by_commit:
    - 414aa0bff
---
## What's missing

With no guidance installed, `aiwf doctor` prints `installed revision: unknown (no intact owned index)` while the guidance section of `aiwf update` prints `installed: none` for the same repository. Doctor falls back to `projectguidance.UnknownRevision` whenever `Inspection.Revision` is empty (`internal/cli/doctor/project_guidance.go`), which covers both an absent installation and a recorded index that is not intact; the report uses `projectguidance.InstalledState` (`internal/projectguidance/state.go`), which distinguishes them. The constant's comment scopes "unknown" to an index that is not intact.

Measured with a build of the E-0095 epic branch at M-0352 (`f5a2c53f8`), in a scratch repository with an isolated HOME and Git configuration, `aiwf.yaml` selecting `sample/base` from an absent source so nothing installs:

```
$ aiwf update --root r2 | grep installed:
  installed: none
$ aiwf doctor --root r2 | grep "installed revision"
project policy: installed revision: unknown (no intact owned index); upstream freshness not checked
```

## Why it matters

Doctor calls a repository with nothing installed "unknown (no intact owned index)", which reads as a damaged installation, while the update report on the same repository says nothing is installed; doctor's next line (`.guidance/index.md: selected guidance is not installed; run aiwf update`) is the only hint that nothing was ever there. The two verbs are meant to describe guidance state in the same words.
