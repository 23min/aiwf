---
id: G-0166
title: RejectionLayerCheckTime cells rejected at verb-time by kernel
status: open
discovered_in: M-0125
---
## What's missing

The M-0123 spec table marks two cells with `RejectionLayer: RejectionLayerCheckTime` (i.e. the cell is supposed to be enforced post-write by `aiwf check`, with the verb itself accepting the transition cleanly):

1. **`gap.open → addressed` with `self.addressed_by == ""`** — `ExpectedErrorCode: gap-resolved-has-resolver` (warning).
2. **`AC.open → met` under `parent.tdd == required` with `self.tdd_phase != done`** — `ExpectedErrorCode: acs-tdd-audit` (error).

In practice the kernel rejects both at verb-time, not at check-time:

- **Gap cell**: `internal/verb/promote.go::requireResolverForResolutionClass` (added by G-0096) hand-rolls a verb-time guard that returns `"promoting a gap to \"addressed\" requires --by <entity-id> or --by-commit <sha> so the gap-resolved-has-resolver rule is satisfied; pass --force to override"`. The verb refuses to write the illegal state.
- **AC cell**: `internal/verb/ac.go::finalizeACPlan` runs `projectionFindings` (pre-write `check.Run` delta). Because `acs-tdd-audit` is severity `error` under `tdd: required`, `check.HasErrors(fs)` is true and the verb returns the projection's findings instead of writing. The verb refuses to write the illegal state.

The kernel being stricter than the spec is design-aligned (belt-and-suspenders): an illegal state never lands on disk in the first place. The check rule itself remains active as a backstop — if a future caller bypasses the verb (hand-edits the markdown, uses `--force` where the verb supports it, or imports a malformed entity), the rule still fires. But the cell's *spec axis* (RejectionLayer = CheckTime) doesn't match its *implementation axis* (rejected at verb-time).

## Why it matters

M-0125's AC-3 is "Per-cell negative driver: check-time rejection (finding-code present)." The driver's contract is:

1. Build fixture, bring entity to FromState.
2. Satisfy preconditions.
3. **Run the cell's verb (expected to succeed)**.
4. Run `aiwf check --format=json`.
5. Assert the finding code appears in the envelope.

Step 3 fails for both check-time cells because the kernel rejects at verb-time. The driver detects the failure and skips the cell via `ac3KnownImplGaps` pointing here.

Today AC-3 has *zero* cells running through the full check-time pipeline. The driver, floor assertion, and staleness cross-check still provide value (any future check-time cell added to the spec without a skip entry surfaces immediately), but the "verb succeeds → check fires" path is not exercised end-to-end. The cells' check-rule backstop is exercised in `internal/check/*_test.go`'s unit tests — that's not the same as AC-3's driver-level integration assertion, but it is real coverage of the rule's logic.

## Resolution shape

Two viable paths:

**(A) Reclassify the cells in the spec.** Mark both as `RejectionLayer: RejectionLayerVerbTime` — the axis description matches kernel behavior. AC-3's enumeration would then drop to zero check-time cells; the floor assertion (currently ≥2) would need to be removed or replaced with a "this surface exists for future cells" sentinel.

**(B) Soften the kernel's verb-time chokepoint for these cells.** Remove `requireResolverForResolutionClass` (or scope it to `--force`-able only); remove the projection-findings pre-write check (or make it skip the codes the spec marks as CheckTime). Then the verb succeeds and the check rule's backstop role is the only chokepoint. This is conceptually cleaner (one chokepoint per rule) but loses the belt-and-suspenders strength.

Path (A) is preferred unless the kernel team has independent reasons to want check-only enforcement for these rules. Path (B) is unlikely without a separate design argument — the kernel's pre-write projection is a deliberate strength.

Either path lets AC-3's driver participate in coverage of these cells (post-A by removing them from CheckTime; post-B by them succeeding at verb-time).

## Where to fix

- **Path A**: `internal/workflows/spec/rules.go` — flip `RejectionLayer: RejectionLayerCheckTime` → `RejectionLayerVerbTime` on the two cells. The AC-3 driver's enumeration handles the change automatically.
- **Path B**: `internal/verb/promote.go` (resolver guard) + `internal/verb/ac.go::finalizeACPlan` (projection guard).

## Related

- M-0123 (spec table authoring; introduced the RejectionLayer axis)
- M-0125/AC-3 (this gap surfaced during its red phase)
- M-0131 (spec audit; may pick up the reclassification)
- G-0096 (verb-time resolver requirement; the gap cell's chokepoint)
