---
id: G-0467
title: Lock-busy refusal emits an empty error code
status: addressed
priority: high
addressed_by_commit:
    - 568e5c8b6
---
## What's missing

`FinishVerbOutcome` (`internal/cli/cliutil/apply.go`) routes every uncoded, non-internal error to `ExitUsage` with an error envelope whose `code` field is empty. `AcquireRepoLock` (`internal/cli/cliutil/lock.go`) emits its repo-lock-busy refusal the same way — `emitErrorEnvelope(verbDisplay, "", ...)`. Those are the only two call sites in the tree that pass a literal empty code where a code belongs.

A `--format=json` consumer therefore cannot distinguish two outcomes with opposite meanings:

- **"Another aiwf process holds the lock."** The verb refused correctly; the caller should back off and retry.
- **"Something unexpected failed."** An uncoded error from anywhere inside the verb, surfaced with no identity.

Both exit 2, both carry `status: "error"` with an empty `code`. The only available discriminator is a substring match on the human-readable message — `strings.Contains(env.Error.Message, repolock.ErrBusy.Error())`, which `internal/stresstest/concurrent_writer_at_scale.go` is forced to do. That contradicts the rule that errors are compared by identity or type, never by string-matching the message.

The absent identity also costs diagnosability at the consumer. `classifyCancelOutcome` in that same file wraps only the subprocess's run error and discards the envelope when the busy match fails, so a real failure reaches CI as the bare text `exit status 2` with no indication of what broke. Occurrences of exactly that shape in the `internal/stresstest` CI history are uncharacterized for this reason.

## Why it matters

`--format=json` is the documented machine-consumable contract, and `ErrorEnvelope` already carries a `code` field for exactly this purpose — populated on the coded-error path, empty on the two paths that most need it. A caller that wants "retry on contention, fail on everything else" cannot express that against the envelope.

`ExitUsage` as the default bucket for uncoded errors compounds it: exit 2 nominally means a usage error, so a caller reading exit codes alone concludes it passed bad arguments when the cause was an I/O failure deep inside the verb.

G-0456 records the adjacent defect on a different axis — whether prelude failures emit an envelope at all, rather than what identity the envelope carries. The two resolve independently.

## Resolution shape

Give the busy refusal a stable code, either a sentinel the `entity.Code` path can surface or an explicit constant passed at the `AcquireRepoLock` call site. Decide what the uncoded-error default should be: a distinct exit code, a generic code string, or both. Then route the stress harness's busy classification through the code rather than the message, and stop discarding the envelope on the non-busy error path so the failure text survives into the log.

Whatever code is introduced is a kernel surface and must be reachable via `--help` and the embedded skill docs, not only from source.

## Where to fix

- `internal/cli/cliutil/lock.go` — the two `emitErrorEnvelope` call sites passing an empty code.
- `internal/cli/cliutil/apply.go` — `FinishVerbOutcome`'s uncoded-error default arm.
- `internal/stresstest/concurrent_writer_at_scale.go` — `parseBusyEnvelope`'s substring match and `classifyCancelOutcome`'s discarded envelope.
