---
id: G-0718
title: Pending guidance advice names a rerun that disabled maintenance cannot finish
status: open
discovered_in: M-0352
---
## What's missing

With `guidance.enabled: false` and a leftover `.guidance/.aiwf-pending`, the guidance section of `aiwf update` and `aiwf init`, and `aiwf doctor`, advise "installation incomplete; rerun aiwf update before using the pack set". A rerun cannot clear it: disabled maintenance skips installation, so the recovery record stays. The wording is `projectguidance.PendingState` in `internal/projectguidance/state.go`, printed by `internal/cli/cliutil/guidance_report.go` and `internal/cli/doctor/project_guidance.go` regardless of maintenance.

Measured with a build of the E-0095 epic branch at M-0352 (`f5a2c53f8`), in a scratch repository with an isolated HOME and Git configuration and a local guidance source listing `sample/base`: install once with maintenance enabled, write `{}` to `.guidance/.aiwf-pending`, then set `enabled: false`.

```
$ aiwf update --root r1
Guidance:
  selected:  sample/base
  maintenance disabled; installed policy remains in use
  …
  pending:   installation incomplete; rerun aiwf update before using the pack set
$ aiwf update --root r1 | grep pending:
  pending:   installation incomplete; rerun aiwf update before using the pack set
$ ls r1/.guidance/.aiwf-pending
r1/.guidance/.aiwf-pending
$ aiwf doctor --root r1 | grep pending
project policy: .guidance/.aiwf-pending: installation incomplete; rerun aiwf update before using the pack set
```

## Why it matters

A maintainer who disabled maintenance with an interrupted installation on disk follows the advice, reruns update, and gets the same line back. The `maintenance disabled` line printed above it does not say that maintenance must be re-enabled for the rerun to finish the installation, so the pack set stays partial while the pending line keeps naming a rerun as the fix.
