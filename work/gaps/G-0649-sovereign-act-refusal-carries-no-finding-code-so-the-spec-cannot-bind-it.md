---
id: G-0649
title: Sovereign-act refusal carries no finding code, so the spec cannot bind it
status: open
discovered_in: M-0324
---
## What's missing

`requireHumanActorForSovereignAct` (`internal/verb/promote_sovereign_act.go`)
returns a bare `fmt.Errorf`. It carries no finding code, which has two
observable consequences.

At the CLI, `FinishVerbOutcome` classifies an error by asking
`entity.Code(err)`: a coded error exits `ExitFindings` (1), an
`internalError` exits `ExitInternal` (3), and everything else falls to
`ExitUsage` (2). A sovereign refusal takes the default arm, so it reports
as a usage error rather than as the legality refusal it is — the same
violation class `aiwf check` reports at exit 1 once the act has landed.

In the legal-workflow spec, both code-oriented drift arms in
`internal/policies/m0123_ac5_drift_test.go` skip any rule whose
`ExpectedErrorCode` is empty. A `GlobalRules()` entry for sovereignty can
therefore be declared but not bound to the implementation the way every
other illegal cell is.

## Why it matters

The spec's bidirectional closure is what keeps the legality encoding and
the kernel from drifting apart. A rule that cannot carry a code sits
outside that closure: it documents the sovereignty gate without pairing it
against the impl, so the gate could be removed and the spec would stay
green.

Fixing it means giving the refusal a code, adding the matching
`hintTable` entry that `PolicyFindingCodesHaveHints` requires, and
covering it per `PolicyFindingCodesHaveTests`. That changes the exit code
of the shipped `epic proposed → active` gate from 2 to 1 — a user-visible
change to a surface consumers and pipelines already read, so it needs a
decision of its own rather than riding along with the milestone that
found it.
