---
id: G-0134
title: Production aiwf binary not signed on Darwin (parallel to G-0133)
status: addressed
addressed_by_commit:
    - 9ad0f5eb
---
## Problem

G-0133 fixed the syspolicyd crash for `go test`'s per-package `.test` binaries by routing them through `scripts/sign-and-run.sh` via the `-exec` flag, pinned in Makefile and CI workflows. The production `aiwf` binary distributed via `go install github.com/23min/aiwf/cmd/aiwf@...` (or `go install ./cmd/aiwf` locally) lands UNSIGNED in `$(go env GOPATH)/bin/aiwf`.

This produces the same macOS Sonoma 14.8.x syspolicyd crash loop when:

- A fresh consumer runs `go install` then invokes `aiwf` (any command).
- An existing consumer reinstalls via `aiwf upgrade` (which re-execs `go install`).
- The pre-commit / pre-push hooks spawn `aiwf` as a subprocess after a host reboot or syspolicyd state reset.

Workaround: `codesign -s - -f $(which aiwf)` ad-hoc signs the binary in place. Same `-s -` flag `scripts/sign-and-run.sh` uses for test binaries.

## Surfaced via

E-0033 / M-0123 / phase 2 AC-1 commit (D-0005 attempt). The pre-commit hook hung mid-`git commit` for 13+ minutes due to the unsigned binary triggering syspolicyd; user signed the binary out-of-band to unblock.

## Proposed fix shape

Three candidates, sequenced:

1. A `make install` target that wraps `go install ./cmd/aiwf` with a `codesign -s -` step on Darwin (no-op on Linux). Document as the canonical local-install path in CLAUDE.md and the README.
2. `aiwf upgrade` re-execs `go install`, then conditionally signs on Darwin before returning. Closes the upgrade-path hole.
3. Optional structural fix: post-install hook on the published module that signs at install time. Out of scope today; G-0133's "structural fix parked" stance applies.

## Related

- G-0128 (inner test-helper binaries), G-0133 (outer `.test` binaries) — same root, different layers.
- CLAUDE.md §"Running tests on macOS — use the wrapper" — documents the test-binary side; doesn't yet address the production-binary side.

## Discipline today

Until fixed: ad-hoc sign with `codesign -s - -f $(which aiwf)` whenever `aiwf` is re-installed via `go install` or `aiwf upgrade`. Add the incantation to CLAUDE.md as part of this gap's resolution.
