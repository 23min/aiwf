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

Two consequences, one for people and one for machines.

For an operator, a warning row that says a reference "resolves to no entity in
this working tree" reads identically whether the surface established that or
merely could not check. The subcode is what carries the difference, and status
drops it.

For any tool comparing surfaces, status is structurally incomparable: its rows
cannot be matched against the findings envelope the check surfaces emit. M-0300's
read-path agreement property has to fall back to comparing status on its blocking
count alone, which is sound in one direction only and rests on a conjunction the
harness does not own. Carrying subcode and severity on `StatusFinding` would let
status be compared claim-for-claim like every other verdict surface.
