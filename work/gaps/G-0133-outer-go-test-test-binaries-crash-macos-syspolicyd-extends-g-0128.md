---
id: G-0133
title: Outer go-test .test binaries crash macOS syspolicyd (extends G-0128)
status: addressed
discovered_in: M-0120
addressed_by_commit:
    - 09399b79
---
## What's wrong

G-0128 (archived) closed the first layer of the macOS Sonoma 14.8.x syspolicyd
crash by ad-hoc signing the `aiwf` binaries that `AiwfBinary` and `BuildBinary`
(centralized in `internal/cli/cliutil/testutil/proc.go` after M-0118's test
relocation), and the M-080 AC-6 helper at `internal/policies/m080_test.go`,
build under `$TMPDIR` for in-process test orchestration. The fix was incomplete:
**Go's `go test` compiles the per-package test binary itself (`<pkg>.test`) and
exec's it directly**. Those outer binaries are unsigned on Intel macOS by
default — Apple Silicon auto-ad-hoc-signs, Intel does not.

When the kernel asks syspolicyd to assess an unsigned `aiwf.test` or
`policies.test` binary, the same `Security::CodeSigning::MachORep::signingData`
crash on the `syspolicyd.violation_metrics` queue fires, with the same
downstream effect: launchd throttles restarts to once per 20 minutes, every
execve on the host stalls during the throttle window. Terminal lockups return.

## How to recognize it

Same signature as G-0128, but the `binary_path=` field in the unified log
around each crash names a `…/go-build*/b001/<pkg>.test` path rather than (or in
addition to) a `…/T/TestBinary_*/001/aiwf` path. From the 2026-05-19 crash
window on the diagnosing machine:

```
binary_path=/private/var/folders/.../T/go-build*/b001/aiwf.test
binary_path=/private/var/folders/.../T/go-build*/b001/policies.test
```

Both are unsigned (`errSecCSUnsigned`, `MacOS error -67062`).

## Diagnostic record (2026-05-19)

Three syspolicyd crashes within ~25 min on the same machine:

- 11:06:20, 11:10:27, 11:29:52
- All three correspond to assessments of unsigned `aiwf.test`, `policies.test`,
  and (separately) inner helper-built `aiwf` binaries
- G-0128's helper-side codesign step shipped on 2026-05-18; today's crashes came
  from the outer test-binary layer that the prior fix did not cover

## Mitigation

Wrap every `go test`-spawned binary through `go test -exec=<wrapper>`. The
wrapper ad-hoc signs the binary on Darwin before exec'ing it.

The wrapper at `scripts/sign-and-run.sh`:

```bash
#!/bin/bash
set -euo pipefail
if [[ "$(uname)" == "Darwin" ]]; then
  codesign --sign - --force "$1" 2>/dev/null || true
fi
exec "$@"
```

Wiring:

- `Makefile` test targets pass `-exec=$(TEST_EXEC)` where
  `TEST_EXEC := $(CURDIR)/scripts/sign-and-run.sh`
- `.github/workflows/{go,flake-hunt,fuzz}.yml` test invocations carry
  `-exec=./scripts/sign-and-run.sh` (Linux no-ops via the `uname` check)
- `CLAUDE.md` under *Go conventions → Testing* describes the wrapper and links
  back to this gap

## Why not GOFLAGS

`GOFLAGS=-exec=…` works but is a per-shell-session config — easy to forget on
fresh terminals or in scripts that don't inherit it. Pinning the flag in the
Makefile and CI workflows makes the wrap mandatory whenever tests run through
the documented surface. The drift-prevention chokepoint is the Makefile /
workflow files; tests run via `make test`, `make test-race`, or CI all carry
the wrap.

The inline codesign step from G-0128's fix (in `AiwfBinary` / `BuildBinary`
at `internal/cli/cliutil/testutil/proc.go` and the m080 AC-6 build at
`internal/policies/m080_test.go`) becomes redundant once this wrap lands but
stays for now — ~50ms per build, harmless. A follow-up gap can remove it if
the duplication earns the entry.

## Related

- G-0128 (archived) — `macOS syspolicyd crashes parsing unsigned test binaries
  (Sonoma 14.8.x)`. First-layer fix; this gap is the second layer.
- G-0127 (open) — `Integration tests fork/exec deadlock on macOS under -race +
  parallel`. Separate macOS-only test-infra issue; reinforces the case for a
  Linux devcontainer.
- A devcontainer migration would side-step both bug classes. Discussed across
  the same diagnostic session; not yet decided.
