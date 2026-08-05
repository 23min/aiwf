---
id: G-0557
title: Three hook installers overwrite an unreadable user hook and report success
status: open
priority: high
discovered_in: E-0077
---
## What's missing

`aiwf init` installs four git hooks, and three of them destroy a pre-existing
user hook they cannot read, reporting the result as a successful create.

`ensurePreHook` branches on the read error explicitly:

```go
case readErr != nil:
    return StepResult{}, false, fmt.Errorf("reading pre-push hook: %w", readErr)
```

`ensurePreCommitHook` and `ensureCommitMsgHook` instead derive two booleans from
the same error:

```go
hasOurMarker := readErr == nil && strings.Contains(string(existing), marker)
hasAlienHook := readErr == nil && !hasOurMarker
```

Both go false on any read fault, so the auto-migration block that would move an
existing hook aside is skipped and control falls through to the atomic write.
`os.Rename` over an unreadable file succeeds whenever the containing directory
is writable, so the user's hook is replaced, no `.local` sibling is written, no
conflict is reported, and the step ledger records a created action at exit 0.

`ensurePostCommitHook` carries both semantics at once: its status-regeneration
opt-out arm handles the read fault, and its install arm does not.

Measured 2026-08-05 against a pre-existing mode-0000 hook as a non-root user:

| installer | outcome | user hook survived |
|---|---|---|
| pre-push | error, names the unreadable hook | yes |
| pre-commit | reports created, no error | no |
| commit-msg | reports created, no error | no |
| post-commit, regeneration on | reports created, no error | no |
| post-commit, regeneration off | error, names the unreadable hook | yes |

No test exercises the read-fault path on any of the four.

## Why it matters

The failure is silent and it destroys work the operator never handed to aiwf.
The auto-migration exists precisely so that an existing hook is preserved rather
than overwritten; a read fault switches that guarantee off without saying so,
and the success row in the ledger is what the operator sees.

An unreadable hook is not exotic. A restrictive umask, a file owned by another
account in a shared checkout, or a permissions change made after the hook was
written all produce one.

The divergence also blocks the obvious refactor. The four installers are
near-identical and read as a collapse candidate (G-0472), but they carry three
distinct error semantics between them and no test pins any of them — so a
mechanical collapse would silently adopt one semantic and change the behaviour
of the other call sites, with nothing going red.

## Resolution shape

Give the three permissive sites the explicit read-fault arm the pre-push
installer already has, so a failed read returns a wrapped error rather than
deriving `false` and proceeding. Roughly three lines each.

Each site needs a test that makes the hook unreadable and asserts both that the
call fails and that the original file survives. That is the property no current
test covers, and it is what makes the fix verifiable rather than asserted.

Whether the four installers then collapse into one parameterized unit is a
separate question this fix should not prejudge. It turns on whether the
collapsed signature is simpler than what it replaces, not on the error semantics
having been reconciled.
