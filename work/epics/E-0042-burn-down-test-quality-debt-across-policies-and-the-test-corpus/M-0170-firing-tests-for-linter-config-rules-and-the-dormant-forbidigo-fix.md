---
id: M-0170
title: Firing tests for linter-config rules and the dormant forbidigo fix
status: draft
parent: E-0042
tdd: none
---
## Deliverable

Close the linter-config vacuity surface that G-0264 exposed: golangci-lint config
rules (`forbidigo`, `depguard`, …) currently have **no firing evidence**, so a
dormant rule — one that matches zero sites — is invisible. Two parts:

1. **Fix the dormant `forbidigo` patterns.** Change the call-form `^panic\(` /
   `^os\.Exit\(` patterns to the form forbidigo v2 matches (`^panic$` /
   `^os\.Exit$`), verified against a probe carrying a library `panic` and
   `os.Exit`.
2. **Add a firing test for the linter-config rules.** A fixture carrying a
   library `panic` / `os.Exit` (and any other covered config-rule violation),
   run through `golangci-lint`, asserting the rule flags it. This is the
   firing-evidence mechanism for the golangci-config surface — parallel to the
   firing fixtures M-0166 adds for `internal/policies/` Go policies and the
   `firing_fixture_presence` meta-gate that covers them. Whether this is per-rule
   fixtures or a single meta-check is a scoping decision at milestone start.

## Why

G-0264: the `panic`/`os.Exit` forbidigo rules went dormant on a toolchain bump
and nothing noticed, because the firing-fixture meta-gate only scans
`internal/policies/*.go` — golangci-lint config rules are an uncovered surface.
This is the G-0259 pathology one surface over: a chokepoint that reads as a
guarantee while detecting nothing.

## Mechanical evidence

The firing test fails if a covered config rule stops flagging its probe — so a
future toolchain bump or config edit that silently kills a rule turns CI red,
instead of a violation slipping through unnoticed.

## Source

G-0264 (the dormant-forbidigo finding), discovered via the M-0167 `wf-rethink`.
Companion to M-0166 (firing fixtures for Go policies) and G-0259 / G-0262 (the
vacuity theme). This milestone addresses G-0264.

## Acceptance criteria

Pinned when the milestone starts.
