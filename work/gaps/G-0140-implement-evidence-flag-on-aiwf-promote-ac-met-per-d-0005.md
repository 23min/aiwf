---
id: G-0140
title: Implement --evidence flag on aiwf promote AC met per D-0005
status: open
discovered_in: M-0123
---
## What's missing

Per **D-0005** (committed in M-0123 phase 1), `aiwf promote <id>/AC-N met`
under a milestone with `tdd: required` should require a `--evidence
<test-symbol>` flag binding the AC's "met" claim to a concrete test that
exercises the AC's assertion. The verb today accepts
``aiwf promote `M-NNNN/AC-N` met`` with no evidence binding.

The spec's `acRules()` in `internal/workflows/spec/rules.go` encodes:

- a **legal** cell (preconditioned on `self.evidence` non-empty),
- an **illegal** companion cell (precondition `self.evidence == ""`,
  ExpectedErrorCode `ac-evidence-missing`).

`ac-evidence-missing` is listed in `deferredImplErrorCodes` (M-0123/AC-5)
with this gap as the tracking reason.

> **Note on the tdd:required scope.** The spec cell's precondition is
> `self.evidence == ""` alone — no `parent.tdd` predicate. So the rule as
> authored fires for any AC met promote with empty evidence, regardless
> of the milestone's TDD policy. D-0005's prose framed the intent as
> "under tdd: required"; the spec cell author broadened it to all met
> promotes. The impl should pick one (probably broader-matches-spec) and
> the chosen behavior should be re-asserted against D-0005 when the impl
> lands.

## Why it matters

CLAUDE.md's "AC promotion requires mechanical evidence" rule is currently
operator-discipline only. Promoting an AC to met without a test symbol is
the failure mode that "framework correctness must not depend on the LLM's
behavior" forbids. The kernel chokepoint is the verb; the verb needs the
flag.

## Proposed fix shape

- Add `--evidence <symbol>` flag to `aiwf promote` (visible only on the
  `<id>/AC-N met` shape; reject for other promote targets).
- Write evidence into the AC's `evidence:` frontmatter field (new field,
  schema additive).
- Verb refuses with `ac-evidence-missing` when:
  * promote target is `met`,
  * `--evidence` is empty.
- Test surface: integration test against the binary with a fixture
  milestone that has a passing test symbol; verb succeeds. Same fixture
  without `--evidence`; verb refuses with the structured code.
- Once landed, remove `ac-evidence-missing` from `deferredImplErrorCodes`.

## Detailed fix outline (refined at M-0125/AC-2)

When M-0125's negative-cell driver tried to exercise this cell, the verb
returned a different error (`acs-tdd-audit` from `finalizeACPlan`'s
projection check, masking the actual chokepoint). This confirmed G-0140's
diagnosis at the cell level and surfaced concrete impl steps:

1. **Frontmatter field**: extend `entity.AcceptanceCriterion` with
   `Evidence string` (or use an existing free-text body section — design
   choice; lean is structured field for `aiwf check` queryability).
2. **Verb option**: add `Evidence string` to `verb.PromoteOptions`.
   Validate at the verb boundary: when `newStatus == StatusMet` and
   `Evidence == ""` and not `force`, return:
   ```go
   fmt.Errorf("promoting AC %s to met requires --evidence \"…\" (ac-evidence-missing); pass --force to override", id)
   ```
3. **CLI flag**: wire `--evidence "…"` on the `promote` subcommand.
   Reject it for non-AC targets (mirror the `--by` / `--superseded-by`
   shape guards in `internal/cli/promote/`).
4. **Check rule** (optional companion): a check-rule
   `ac-evidence-missing` for backfill — catches ACs that reached `met`
   before this rule landed, or via `--force`. Without this rule, the
   verb-time guard is the only chokepoint, which means historical
   evidence-less ACs stay un-flagged.
5. **Tests**:
   - `internal/verb/ac_evidence_test.go` — positive + negative cases for
     the verb.
   - Update `internal/cellcoverage/fixture.go` if AC seeding needs to
     support evidence preset.

Roughly 50–80 LOC plus tests; one focused milestone or a piece of a
follow-up epic.

## Interaction with existing M-0124 / M-0125 cells

- **M-0124 (positive driver).** The Legal cell `(AC, open, promote)` with
  `self.evidence non-empty` currently passes — but the M-0124 driver
  supplies no evidence, and the verb succeeds because there's no check.
  When this gap closes, M-0124's driver
  (`internal/policies/m0124_positive_driver_test.go::buildVerbArgs`)
  needs to start forwarding `evalCtx.Evidence` as `--evidence`; otherwise
  the Legal cell starts failing. The cellcoverage fixture already
  populates `evalCtx.Evidence = "fixture-provided evidence"` for the
  non-empty case via `SatisfyPredicate`; that string just isn't reaching
  the verb yet.
- **M-0125 (negative driver).** `ac2KnownImplGaps["ac-open-promote"]`
  points at this gap (G-0140; before consolidation it pointed at G-0164,
  which has been cancelled as a duplicate — see "History"). When the
  verb-time guard lands, remove that map entry and the cell graduates to
  end-to-end coverage via `runNegativeVerbTimeCell`.
- **M-0125 fixture customization.** `ac2ImplGapFixtureSetup["ac-open-promote"]`
  advances the AC's phase to `done` before driving the verb, so the
  incidental `acs-tdd-audit` projection finding doesn't mask the
  intended `ac-evidence-missing` chokepoint's status. Once the actual
  chokepoint exists, the fixture customization can stay (it tests the
  isolated `ac-evidence-missing` branch correctly) or be removed (the
  default fixture's `acs-tdd-audit` would fire first, which is fine
  because the verb rejects regardless of the reason).

## Closing this gap

When the impl lands:

1. Remove `"ac-evidence-missing"` from `deferredImplErrorCodes`
   (`internal/policies/m0123_ac5_drift_test.go:260`).
2. Remove `"ac-open-promote"` from `ac2KnownImplGaps`
   (`internal/policies/m0125_negative_driver_test.go`).
3. Remove `"ac-open-promote"` from `ac2ImplGapFixtureSetup` (only if the
   default fixture's `acs-tdd-audit` interaction is judged acceptable;
   see Interaction section above).
4. Update M-0124's `buildVerbArgs` to forward `evalCtx.Evidence` as
   `--evidence` (see Interaction section above).
5. Promote G-0140 to `addressed` with `--by M-NNNN` (whichever milestone
   carries the impl).

## Open questions

- Does `--evidence` validate the symbol exists in the test tree at
  promote-time, or only record it (and validate via a check rule)?
  Validation at promote-time is stronger but couples the verb to the
  language toolchain. Recording + check separation matches the kernel's
  "verb does one thing, check polices" pattern. Lean: record + check.

## History

- **M-0123 (filed):** G-0140 filed at M-0123 wrap as a follow-up to
  D-0005, citing `deferredImplErrorCodes`.
- **M-0125 (refined):** G-0164 was filed at M-0125 unaware that G-0140
  already existed (duplicate). G-0164's value-add — the M-0124
  interaction context and the closing-this-gap checklist — was merged
  into this body and G-0164 was cancelled as a duplicate.
- **Confirmed:** M-0125/AC-2's negative driver dry-run confirmed at the
  cell level that the chokepoint is missing (verb does not reject with
  `ac-evidence-missing`; an incidental `acs-tdd-audit` projection
  finding triggers instead under `tdd: required` fixtures, masking the
  actual rule).
