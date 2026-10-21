---
id: G-0695
title: promote's draft-start guard lints terminal ACs the check rules exempt
status: addressed
addressed_by_commit:
    - 09b3256d4
---
## What's missing

`aiwf check` exempts a terminal acceptance criterion from its body-completeness
rules; the promote guard that asks the same question does not. Three check
predicates skip an AC whose status is `cancelled` (`internal/check/acs.go`, two
sites, and `internal/check/entity_body.go`). `requireNonEmptyACBodiesAtMilestoneStart`
in `internal/verb/promote.go` iterates every AC and refuses on an empty body with
no status carve-out at all, so the two surfaces answer the same question
differently about the same tree.

Measured in the devcontainer (Linux) with a binary built from `23b3820bb`, in a
fresh repo after `aiwf init`: an epic, a milestone under it, one AC promoted to
`cancelled`, its body section left as the bare `### AC-1` heading the template
writes, standing on the epic branch.

```
$ aiwf check
5 findings (0 errors, 5 warnings)
exit=0

$ aiwf promote M-0001 in_progress
aiwf promote: cannot promote M-0001 to "in_progress": M-0001/AC-1 has no body
content; write prose under its `### AC-1` heading first
exit=2
```

The ids in that transcript are the throwaway repo's own, allocated by `aiwf add`
in an empty tree; they are not references to entities here.

Expected: the two surfaces agree on whether a terminal AC owes body prose.
Observed: check reports a clean tree at exit 0 while promote refuses at exit 2.

The guard's own doc comment states that `--force` is "practically inert here"
because `projectionFindings` runs unconditionally. That reasoning depends on
check raising a finding over the same AC, which it does not — so the claim does
not hold for the case above. Not separately measured.

## Why it matters

The operator's recourse is the one the refusal names, and it is unavailable: the
AC is terminal, so writing prose under it is work on a criterion already
withdrawn. Nothing else in the message points anywhere, and the surface that
would normally adjudicate — `aiwf check` — reports the tree as clean, so an
operator who consults it before promoting is told the state is fine by the tool
whose job is to say so.

The disagreement widens as the AC FSM's terminal set grows. It holds today for
`cancelled` alone; a second terminal treated as out of contract by the check
rules inherits the same split without any edit to either surface.
