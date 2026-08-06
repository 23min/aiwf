---
id: G-0561
title: The install verbs report an operator's permission fault as an internal error
status: open
priority: medium
---
## What's missing

`aiwf init` and `aiwf update` map every error out of the install pipeline to
exit 3 — the code reserved for an internal fault — and print the wrapped error
with no statement of what the operator should do next.

The whole classification is one arm, duplicated at
`internal/cli/initcmd/initcmd.go:118` and `internal/cli/update/update.go:121`:

```go
if err != nil {
    cliutil.Errorf("aiwf init: %v\n", err)
    return cliutil.ExitInternal
}
```

Measured 2026-08-06 against a repo carrying a mode-0000 hook, as a non-root
user:

```
$ aiwf init
aiwf init: reading pre-commit hook: open /path/.git/hooks/pre-commit: permission denied
$ echo $?
3
```

`aiwf update` reports the same code and the same shape of message.

The condition is the permissions on the operator's own file. Nothing about it is
internal to aiwf, and the remedy — restore read access and re-run, which
refreshes whatever the aborted run already installed — appears nowhere the
operator can see it.

## Why it matters

The exit code is what a caller reads to decide whether it hit a bug worth
reporting or a condition it can repair. A provisioning script, a CI step, and an
operator triaging a failed setup all get the same wrong answer, and the message
gives them nothing to act on.

The install path is the one place a refusal is *expected* to be actionable: it
fires precisely because aiwf declined to touch a file the operator owns. Naming
that an internal error inverts the finding.

The closed exit-code set — 0 ok, 1 findings, 2 usage, 3 internal — has no member
for an environment precondition. So the fix is not a different constant chosen in
passing: which existing code this class belongs to, or whether the set needs
another member, is the decision this gap carries.

## Resolution shape

Two parts, separable, and worth taking in this order.

The message is the cheap half and can land alone. The refusing seam knows the
path and the fault; what it does not say is that restoring read access and
re-running fixes it. `aiwf check` already carries per-finding hints for exactly
this reason — verb errors have no equivalent, which is why the guidance has
nowhere to live today.

The exit code is the real question, and it is a CLI convention every verb shares
rather than an `init` detail. Three ways it can go: fold a precondition fault
into exit 2 on the reading that the operator can repair their environment; keep 3
and settle that it means "aiwf cannot proceed" rather than "aiwf is broken";
or widen the set. Widening changes a contract shared by every verb and every
caller parsing it, so it wants a decision entity rather than an inline pick.

Whichever way it lands, the catch-all arm is duplicated across the install
verbs, so the blast radius is both of them and any verb later written to the
same shape.

## Related

- G-0557 — the read-fault refusal this error path now fires from; closing it took
  the number of installers reaching this message from one to four
