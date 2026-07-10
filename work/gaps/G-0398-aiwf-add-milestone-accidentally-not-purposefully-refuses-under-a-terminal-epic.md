---
id: G-0398
title: aiwf add milestone accidentally-not-purposefully refuses under a terminal epic
status: open
discovered_in: M-0244
---
## What's missing

`aiwf add milestone --epic <id>` has no dedicated precondition against
the named epic already being terminal (`done`/`cancelled` — a status
the epic FSM has no outgoing edges from). Confirmed directly:
`internal/verb/add.go`'s milestone-creation path validates the epic id
resolves and is the right kind, but reads nothing from the epic's own
`status` field.

Empirically, the attempt is already refused today — but only as an
accidental side effect, not by design. The new standing
`epic-terminal-non-terminal-children` check-rule correctly flags the
resulting tree state (terminal epic, freshly non-terminal child) as an
error-severity finding; every mutating verb runs a before/after
projection-findings gate that refuses any commit introducing a new
error-severity finding, so the add is blocked before anything lands.
Confirmed with a real repo: `aiwf add milestone --epic <done-epic>`
returns `status: findings` with a generic "did not report ok" shape,
and neither the milestone file nor a commit for it exists afterward.

## Why it matters

The refusal is real but accidental — it depends on the standing check-
rule happening to cover this exact state, wired through a generic
verb-time gate never designed with this case in mind. The operator
sees `epic-terminal-non-terminal-children`'s own message ("epic
`E-NNNN` has terminal status... but still owns non-terminal child
milestone(s)... hint: bring each listed child milestone to a terminal
status") — worded for the *detection* case (an already-existing child
gone stale), not the *creation* case (nothing existed yet; the add
itself is the problem). A dedicated guard, mirroring the shape of the
existing epic-terminal-promote guards (G-0393/G-0394), would refuse at
the right verb with a message that actually names what happened:
"cannot add a milestone under epic `E-NNNN`: it is already done."

Discovered indirectly: `internal/stresstest/verb_sequence.go`'s
`TestVerbSequenceScenario_FullWalkAcrossAllKindsPasses` property test
creates one entity per kind and walks each one's status independently.
Because it creates and fully random-walks the epic (which can land it
on `done`/`cancelled`) *before* creating the milestone that needs the
epic as `--epic` parent, the test occasionally hits this exact refusal
by accident and treats it as a hard scenario failure (it has no
tolerance for `add` being legitimately refused) — surfacing the
underlying gap as an incidental side effect of its own
independent-per-kind design, not by testing for it directly.

## Direction

Add a dedicated precondition to `aiwf add milestone`'s own verb body:
refuse when the named epic's status is terminal, with a message naming
the epic and its status. No `--force` override is obviously
correct here — unlike the promote/cancel guards, there's no legitimate
reason to *create* new work under a permanently-closed parent, so this
may not need one at all; worth a real decision before implementing
rather than assumed.

## Scope

The precondition in `aiwf add milestone`'s own verb body, a new typed
error code, and tests: a fixture where `add milestone --epic
<done-epic>` is refused with a clear, dedicated message, a fixture
where `add milestone --epic <active-epic>` still succeeds cleanly, and
confirmation the refusal names both the epic id and its terminal
status.
