---
id: G-0432
title: aiwf version/doctor and JSON envelope resolve version differently
status: open
priority: low
discovered_in: M-0126
---
## Problem

`aiwf version` and `aiwf doctor` resolve the binary's version via `ResolvedVersion()` in `internal/cli/root.go`, which prefers the ldflags stamp embedded by `make install` / `aiwf upgrade`. Every `--format=json` command's envelope (`{tool, version, ...}`) resolves the same fact via `version.Current()` (Go build info), a distinct code path. For a binary built outside the blessed `make install`/`aiwf upgrade` path (e.g. a bare `go install`, or certain CI/devcontainer builds), the two report different version strings for the same binary — a single-source-of-truth (C1) violation. `PolicyEnvelopeVersionSource` pins the envelope's derivation; nothing pins the human-print site to match it. Surfaced by the 2026-06-16 health-scorecard audit (C1 scored Weak) and its accompanying gap-triage pass (Candidate B), verified real but never filed.

## Direction

Route both `aiwf version`/`aiwf doctor`'s human-facing print and the JSON envelope through one resolver. Extend `PolicyEnvelopeVersionSource` (or add a sibling policy) to also assert the human-print site matches the envelope's derivation.