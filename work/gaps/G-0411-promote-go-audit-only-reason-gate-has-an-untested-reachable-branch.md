---
id: G-0411
title: promote.go audit-only --reason gate has an untested reachable branch
status: addressed
discovered_in: M-0253
addressed_by_commit:
    - 3c20ac67
---
## What's missing

`internal/cli/promote/promote.go:128` (the `gateFlag = "--audit-only"`
reassignment inside the `--force`/`--audit-only` + empty-`--reason`
guard) is reachable — `aiwf promote <id> <status> --audit-only` with no
`--reason` hits it — but has zero test coverage. M-0253's AC-1 didn't
catch it because `branch-coverage-audit` is diff-scoped to lines
changed since a fixed base commit, and this exact line predates that
base (only its sibling `force`-arm was touched), so the mechanical
gate never flagged it. Found by an independent reviewer during
M-0253's wrap, not by the audit tool.

## Why it matters

This is precisely the class of gap E-0064 exists to close — an
untested CLI-verb error-handling branch — but it's invisible to the
epic's own mechanical detection method (the diff-scoped audit against
the pre-M-0238 base) because the line's last change predates that
base. None of E-0064's flagged-file milestones will close this one on
their own, since it falls outside every milestone's scoped flagged-line
set even though `promote.go` itself is M-0253's file. The fix is a
one-line addition — a second case in `TestRun_ForceRequiresReason`:
`auditOnly: true, reason: ""` — small enough to land as its own
`wf-patch` rather than waiting on a future milestone.