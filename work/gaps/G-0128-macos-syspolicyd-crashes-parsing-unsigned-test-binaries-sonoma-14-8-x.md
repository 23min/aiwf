---
id: G-0128
title: macOS syspolicyd crashes parsing unsigned test binaries (Sonoma 14.8.x)
status: addressed
discovered_in: M-0120
addressed_by_commit:
    - 360387cf
---
## What's wrong

On macOS Sonoma 14.8.1 (`iMac20,1`, x86_64), `/usr/libexec/syspolicyd` segfaults parsing the Mach-O code-signing data of a freshly-built Go test binary in `$TMPDIR`. Fault site:

```
Security::Universal::architecture() const
Security::CodeSigning::MachORep::signingData()
…
SecStaticCodeCheckValidityWithErrors
```

dispatched on the `syspolicyd.violation_metrics` queue when Gatekeeper records the unsigned-binary "violation" telemetry event. The crash is `EXC_BAD_ACCESS` / `SIGSEGV` with `KERN_INVALID_ADDRESS at 0x8` — a null-pointer deref inside Apple's `Security` framework.

`syspolicyd` is consulted on every `execve()`. Repeated crashes (29 consecutive observed in one diagnostic window) cause `launchd` to throttle restarts to once per 20 minutes. During the throttle window every new process launch stalls waiting for syspolicyd to return, presenting as locked-up terminals, frozen tab completion, and frozen VS Code helper launches.

## How to recognize it

```sh
ls -lt /Library/Logs/DiagnosticReports/syspolicyd-*.ips | head
```

A stream of `syspolicyd-YYYY-MM-DD-HHMMSS.ips` reports at 15–25 minute intervals is the signature. Each `.ips` carries the same fault site and a `consecutiveCrashCount` that climbs across the sequence.

To identify which binary triggered a specific crash:

```sh
log show --predicate 'process == "syspolicyd"' \
  --start '<crash-time>-30s' --end '<crash-time>+30s' \
  | grep -E 'binary_path|GK performScan'
```

The `binary_path=` field names the binary syspolicyd was validating.

## Diagnostic record (2026-05-18)

- macOS 14.8.1 (23J30), iMac20,1, x86_64
- 13 consecutive `syspolicyd-*.ips` reports in 4 hours; `consecutiveCrashCount: 29`, `throttleTimeout: 1200`
- Faulting thread queue: `syspolicyd.violation_metrics`
- Trigger binary: `/private/var/folders/.../T/TestBinary_CheckVerbose_ByteIdenticalToBaseline*/001/aiwf`
- syspolicyd reports `MacOS error: -67062` (`errSecCSUnsigned`) immediately before crashing

The trigger pattern:

- Go's `go build` on Intel macOS produces unsigned Mach-O binaries (Apple Silicon auto-ad-hoc-signs; x86_64 does not)
- Tests under `cmd/aiwf/binary_integration_test.go`, `cmd/aiwf/integration_test.go`, and `internal/policies/m080_test.go` build a fresh binary per run and immediately `exec.Command()` it
- The kernel asks syspolicyd to assess the unsigned binary on every execve
- Apple's parser dereferences a null `Universal` (multi-arch wrapper) struct when no signature is present, taking down the daemon
- The telemetry event queue persists across restarts → daemon crashes again on relaunch → throttle

## Mitigations (landed)

Ad-hoc sign every test-built binary before `exec`. Adds `codesign --sign - --force <bin>` after `go build` in the three test-build helpers:

- `cmd/aiwf/integration_test.go` — `aiwfBinary` (sync.Once-shared)
- `cmd/aiwf/binary_integration_test.go` — `buildBinary` (per-test, custom ldflags)
- `internal/policies/m080_test.go` — `/tmp/aiwf-m080` build

Gated on `runtime.GOOS == "darwin"`. Harmless on Apple Silicon (re-signs an already-ad-hoc-signed binary). Cost ~50ms per build. The ad-hoc signature shifts Gatekeeper from "unsigned → violation_metrics" (which crashes) to "ad-hoc signed → ad-hoc validation" (which doesn't).

## Recovery (after the fact)

When syspolicyd is already in the crash-throttle loop, queued telemetry events keep crashing every restart. Reboot drains the queue. In this session, syspolicyd recovered cleanly 53 minutes after the trigger stopped — no reboot was needed once new unsigned binaries stopped landing.

## Root-cause investigation (deferred)

This is an Apple bug, not an aiwf bug:

- `Security::Universal::architecture()` should null-check before dereferencing
- The `violation_metrics` queue should be poison-resistant — one malformed input should not stall every subsequent `execve` on the host

Worth filing via Feedback Assistant if a minimal Mach-O reproducer can be isolated. Not blocking — the codesign workaround is robust.

## Related

- G-0127 (`Integration tests fork/exec deadlock on macOS under -race + parallel`) — also a macOS-only test-infra issue; reinforces the case for a Linux-based devcontainer.
- Devcontainer migration would side-step this entire class. Discussed in the same session; not yet decided.
