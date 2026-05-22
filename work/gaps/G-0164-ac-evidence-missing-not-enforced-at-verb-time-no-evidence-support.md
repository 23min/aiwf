---
id: G-0164
title: ac-evidence-missing not enforced at verb-time (no --evidence support)
status: open
---
## Problem

The spec defines `ac-evidence-missing` as a verb-time Illegal cell:

```go
{
    Kind:              KindAC,
    FromState:         "open",
    Verb:              "promote",
    Preconditions:     []Predicate{{Subject: "self.evidence", Op: "==", Value: ""}},
    Outcome:           OutcomeIllegal,
    ExpectedErrorCode: "ac-evidence-missing",
    RejectionLayer:    RejectionLayerVerbTime,
    BlockingStrict:    true,
}
```

But the kernel doesn't enforce it at any layer:

- `verb.PromoteOptions` has no `Evidence` field.
- The `promote` CLI has no `--evidence` flag.
- `internal/verb/ac.go::promoteAC` doesn't check for evidence.
- `internal/check/` has no `ac-evidence-missing` rule.

Promoting `M-NNNN/AC-N` to `met` without evidence succeeds. Under `tdd: required` the `acs-tdd-audit` warning may surface (different code, different reason), but no rule polices the spec's stated "evidence required for met under any policy."

## Why it matters

The spec encodes a quality-discipline guard: an AC marked `met` should carry a one-line evidence note pinning what the meeting consisted of (commit SHA, test name, file pointer). Without verb-time enforcement, ACs can be marked `met` with no traceable evidence, which defeats the audit-trail purpose.

This is a real feature gap, not just a missing test. Closing it requires kernel changes spanning the promote pipeline (CLI flag, verb option, frontmatter field, check rule for backfill).

## Fix outline

The work is medium-size, ~50–80 LOC plus tests:

1. **Frontmatter field**: extend `entity.AcceptanceCriterion` with `Evidence string` (or use an existing free-text body section — design choice).
2. **Verb option**: add `Evidence string` to `verb.PromoteOptions`. Validate at the verb boundary: when `newStatus == StatusMet` and `Evidence == ""` and not `force`, return `fmt.Errorf("promoting AC %s to met requires --evidence \"…\" (ac-evidence-missing); pass --force to override", id)`.
3. **CLI flag**: wire `--evidence "…"` on the `promote` subcommand. Reject it for non-AC targets (mirror the `--by` / `--superseded-by` shape guards).
4. **Check rule**: optional companion `ac-evidence-missing` finding for backfill — catches ACs that reached `met` before this rule landed, or via `--force`.
5. **Tests**:
   - `internal/verb/ac_evidence_test.go` — positive + negative cases for the verb.
   - Update `internal/cellcoverage/fixture.go` if AC seeding needs to support evidence preset.

The M-0125/AC-2 entry un-skips automatically once the verb-time guard is in place.

## Interaction with existing M-0124 cells

The Legal cell `(AC, open, promote)` with `self.evidence non-empty` currently passes (`internal/policies/m0124_positive_driver_test.go`). Today that cell's driver supplies no evidence; the verb succeeds because there's no check. After this gap is fixed, M-0124's driver needs to start supplying `--evidence "covered by TestX"` (the cellcoverage `evalCtx.Evidence` value isn't currently translated into a verb arg). Otherwise M-0124's Legal cell starts failing.

The fix here is symmetric: both directions of the same feature.

## Closing this gap

When the impl lands:
- Remove `"ac-open-promote"` from `ac2KnownImplGaps` (`internal/policies/m0125_negative_driver_test.go`).
- Update M-0124's driver (`internal/policies/m0124_positive_driver_test.go::buildVerbArgs`) to forward `evalCtx.Evidence` as `--evidence`.

## Discovered in

M-0125/AC-2 driver dry-run.

## Status

`open`.
