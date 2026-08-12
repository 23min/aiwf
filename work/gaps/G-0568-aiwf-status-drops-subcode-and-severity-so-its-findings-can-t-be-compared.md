---
id: G-0568
title: aiwf status drops subcode and severity, so its findings can't be compared
status: open
priority: medium
discovered_in: M-0300
---
## What's missing

`aiwf status` renders a verdict: it runs the same in-memory rule pass the check
surfaces run, then reports a blocking count and a list of warning rows. The rows
carry `code`, `entity_id`, `path`, and `message` — but not `subcode`, and not
`severity`. `StatusHealthCounts` carries totals only.

A reader therefore cannot tell a declined judgment (`unresolved-unverified`,
which exists precisely so a ref-less surface can say it did not build the tier)
from a stated one. The same row shape is produced either way.

## Why it matters

`StatusFinding` cannot carry a severity that means anything. Status routes
error-severity findings into its health counts and appends only warnings to the
rows, so every row is a warning by construction; a `severity` field would be a
stored constant restating what the row's own container already determines. It
should be declined rather than added.

The subcode is the real omission, and it is smaller than this gap first claimed.
The operator-facing argument does not hold: a ref-less surface rewrites the
finding's *message* along with its subcode, so status renders "resolves to no
entity in this working tree; the cross-branch view was not built, so it may
exist on an unmerged branch" where the full check states the stronger verdict.
The declined judgment is legible in prose today, and the ambiguous row this gap
describes does not arise.

What survives is machine legibility: a row carrying `code`, `entity_id`, `path`
and `message` cannot be matched against the findings envelope every other
surface emits, so status is the one verdict surface that cannot be compared
claim-for-claim. The named consumer for that was a read-path agreement property
in an epic since cancelled, so nothing needs it today.

Disposition: carry `Subcode` when someone next touches this struct — roughly ten
lines and a test — and decline `Severity`. If no comparator materialises, close
this rather than build for it.
