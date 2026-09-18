---
id: G-0696
title: AC progress rollups count a deferred AC in scope forever
status: open
---
## What's missing

Three AC progress rollups compute the in-scope denominator as every criterion
minus the cancelled ones, so a `deferred` criterion counts toward progress
permanently. `SummarizeACs` (`internal/cli/status/status.go`) sets
`p.InScope = p.Total - p.Cancelled`; `acMetTotal`
(`internal/htmlrender/default_resolver.go`) and `acRollup`
(`internal/cli/render/resolver.go`) skip only `ac.Status == entity.StatusCancelled`
while accumulating the same totals, in two near-identical loops.

Read from the source at `23b3820bb` rather than run: the defect is the
denominator's definition, and its consequence is arithmetic. A milestone with
one `met` and one `deferred` criterion has `Total` 2 and `Cancelled` 0, so
`InScope` is 2 and the rollups read `1/2` — for a criterion that is out of the
milestone's contract and will never be promoted to `met`.

The AC FSM has two terminal statuses, `deferred` and `cancelled`, and neither
claims the criterion succeeded (`entity.IsTerminalACStatus`,
`internal/entity/transition.go`). The denominator names one of them.

A fix has a constraint worth knowing before it is attempted: `SummarizeACs`'s
sibling renderer prints "ACs all cancelled" when `InScope` reaches 0
(`internal/cli/status/status.go`, pinned by `internal/cli/integration/status_cmd_test.go`).
That message is reachable today only when every criterion really is cancelled.
Excluding deferred from the denominator makes an all-deferred milestone reach
`InScope` 0 too, where the wording would be false.

## Why it matters

A milestone can be `done` with every obligation discharged and still report
itself short of its own progress, on three surfaces at once — `aiwf status`, the
rendered HTML, and `aiwf render`. The number is what a reader consults to decide
whether a milestone is finished, and it disagrees with the milestone's own
status.

The distinction the two terminal statuses exist to carry — withdrawn versus
postponed — is erased in the opposite direction from the usual failure: a
deferred criterion is treated as still owed, so deferring work makes a milestone
look permanently incomplete, which is a reason not to use the status at all.
