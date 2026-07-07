---
id: M-0237
title: 'Logger core: internal/logger package and concurrent-append safety'
status: in_progress
parent: E-0061
tdd: required
acs:
    - id: AC-1
      title: Diagnostic logging defaults off, opt-in via env then aiwf.yaml
      status: met
      tdd_phase: done
    - id: AC-2
      title: Opted-in logs land in one daily XDG-state-home file, 30-day retention
      status: met
      tdd_phase: done
    - id: AC-3
      title: Concurrent writers to the shared log file never interleave or tear a line
      status: met
      tdd_phase: done
    - id: AC-4
      title: Bound logger fields never leak the operator's home-directory path
      status: met
      tdd_phase: done
    - id: AC-5
      title: atomic_write_chokepoint.go allowlists internal/logger's append write
      status: open
      tdd_phase: done
---

## Goal

Ship `internal/logger`, an opt-in, default-off diagnostic-log package
wrapping `log/slog`, whose file writer is safe for many concurrent `aiwf`
processes to append to the same daily file with zero coordination.

## Context

No `internal/logger` package exists today; diagnostic-quality output is
either a bare `fmt.Fprintln(os.Stderr, …)` call or nothing. ADR-0017 (as
amended — see its Decision #5) specifies the concurrent-append property
(`O_APPEND`, one `Write()` per record) this milestone implements and proves.
This is the epic's foundation milestone: call-site migration (next) and
correlation wiring (after that) both need this package to exist first.

## Acceptance criteria

### AC-1 — Diagnostic logging defaults off, opt-in via env then aiwf.yaml

Precedence, highest first: `AIWF_LOG`/`AIWF_LOG_FORMAT`/`AIWF_LOG_FILE` env
vars, then `aiwf.yaml`'s `logging:` block, then default (a no-op discard
handler — no I/O, no allocation beyond the closed-form `Info` call). Unit
tests cover the full matrix: env only, yaml only, both set (env wins),
neither set (confirm zero I/O via the discard handler). Parsing and
validating the `logging:` block's three optional keys
(`level`/`format`/`destination`) in `internal/logger` is part of this AC —
`aiwf doctor` surfacing that configuration is M-0238's job, not this one's.
The block is never programmatically rewritten by a verb, so it needs no
`internal/aiwfyaml` surgical-editor entry (that package is reserved for
blocks a verb rewrites in place — `contracts:`, `areas:`, `hooks:`).

### AC-2 — Opted-in logs land in one daily XDG-state-home file, 30-day retention

Default destination `$XDG_STATE_HOME/aiwf/logs/aiwf-YYYY-MM-DD.log` (UTC
date), falling back to `~/.local/state/aiwf/logs/aiwf-YYYY-MM-DD.log` when
`XDG_STATE_HOME` is unset. The directory is created only on the first
opted-in write — never as a side effect of a plain, unopted-in invocation.
Entries older than 30 days are swept on the next invocation that touches the
directory.

### AC-3 — Concurrent writers to the shared log file never interleave or tear a line

The file handle opens with `O_APPEND`; every record is emitted as exactly
one `Write()` call — no `bufio.Writer` or other buffering that could split a
record across two syscalls. A concurrent-writer test spawns multiple
writers (goroutines, or independently-opened `*os.File` handles against the
same path, simulating separate processes) appending simultaneously and
asserts every resulting line parses cleanly, with no interleaving or
truncation. This is the property ADR-0017 Decision #5 rests on — it's what
lets many concurrent `aiwf` processes (multiple worktrees, in particular)
share one daily file with no lock.

### AC-4 — Bound logger fields never leak the operator's home-directory path

`WithVerb(verb, entity, actor)`'s field binding scrubs `/Users/<name>/` and
`/home/<name>/` fragments from any bound value (including anything derived
from `os.Args`) before binding, matching the gitleaks path-leak discipline
already enforced elsewhere in this codebase.

### AC-5 — atomic_write_chokepoint.go allowlists internal/logger's append write

`internal/policies/atomic_write_chokepoint.go`'s allowlist gains exactly one
new entry, for `internal/logger`'s file writer, with a rationale comment
pointing at ADR-0017 Decision #5 (concurrent-append, not atomic-replace —
temp+rename is the wrong pattern for a shared append-only stream). The
chokepoint's own test suite confirms no other new write site is
inadvertently exempted by the change.

## Constraints

- No log file or directory is ever created when the operator hasn't opted in
  (`AIWF_LOG` unset and no `logging:` block in `aiwf.yaml`).
- The `O_APPEND` / one-`Write()`-per-record discipline is non-negotiable —
  see ADR-0017 Decision #5.
- Single dependency: standard library only (`log/slog`), per ADR-0017's
  Consequences.

## Design notes

- ADR-0017 is the locked design; this milestone implements it, it does not
  re-derive it.
- The concurrent-writer test here (AC-3) is package-level — goroutines or
  independently-opened file handles against one path, proving the OS-level
  invariant in isolation. A full multi-process test driving real `aiwf`
  subprocesses against a shared log file is scenario tier 5 of the
  correctness-stress-harness epic (the second epic named in
  `docs/initiatives/robustness-correctness-stress-testing.md`) — not this
  milestone's job; don't duplicate that scope here.

## Surfaces touched

- `internal/logger/` (new package, including `logging:` block parsing/validation)
- `internal/policies/atomic_write_chokepoint.go` (allowlist entry)

## Out of scope

- Migrating any existing bare-stderr call site to use this package — M-0238.
- `aiwf doctor` surfacing the resolved logging configuration — M-0238.
- `correlation_id` / envelope wiring — M-0239.
- The full multi-process concurrent-writer scenario (real `aiwf` subprocess
  fan-out) — the correctness-stress-harness epic, not this one.

## Dependencies

- None — this is the epic's foundation milestone.

## References

- `docs/adr/ADR-0017-opt-in-slog-diagnostic-logging-default-off-xdg-state-home-file-route.md`
- G-0223 — implement ADR-0017 opt-in slog logging; migrate bare-stderr call sites

---

## Work log

### AC-1 — Diagnostic logging defaults off, opt-in via env then aiwf.yaml

Env/yaml/default precedence resolved independently per setting
(`ResolveConfig`); discard-when-off backed by `slog.DiscardHandler`
(`New`) · commit d0fff662 · tests 7/7

### AC-2 — Opted-in logs land in one daily XDG-state-home file, 30-day retention

`OpenDestination` resolves stderr/explicit-path/default-XDG destinations;
default directory created and swept of >30-day entries only on an
opted-in call · commit efbbc061 · tests 22/22

### AC-3 — Concurrent writers to the shared log file never interleave or tear a line

40 independently-opened file handles append 1000 records concurrently;
every line decodes as valid, uncorrupted JSON. No production code
needed — the property already held from AC-1/AC-2's design · commit
3c138e93 · tests 23/23

### AC-4 — Bound logger fields never leak the operator's home-directory path

`WithVerb` scrubs macOS/Linux home-directory fragments from verb/entity/
actor before binding, mirroring `.gitleaks.toml`'s existing
path-leak-darwin-home/path-leak-linux-home regexes · commit ef468cb9 ·
tests 29/29

## Decisions made during implementation

- The `logging:` block's parsing/validation lives in `internal/logger`
  itself, not `internal/aiwfyaml` as ADR-0017/this spec originally named
  — that package is reserved for blocks a verb programmatically rewrites
  in place (`contracts:`, `areas:`, `hooks:`), which `logging:` never is.
  ADR-0017's Consequences section is corrected to match.
- `OpenDestination` refuses to resolve the default destination when
  neither `XDG_STATE_HOME` nor `HOME` is set, rather than silently
  falling back to a bare relative path that would write into the
  process's current working directory — caught by AC-2's vacuity
  mutation probe, not anticipated by the original AC text.
- AC-5's `atomic_write_chokepoint.go` allowlist entry for
  `internal/logger`'s writer landed with AC-2 (commit efbbc061), since
  AC-2's file-opening code needs it to pass the chokepoint policy. AC-5
  itself is now just confirming the entry and promoting.

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
