---
id: M-0154
title: 'ADR: amend settings.json stance to consent-gated'
status: in_progress
parent: E-0039
tdd: none
acs:
    - id: AC-1
      title: ADR-0015 records the consent-gated settings.json stance
      status: met
    - id: AC-2
      title: CLAUDE.md operator-setup line states the consent-gated stance
      status: met
    - id: AC-3
      title: doctor.go stance surfaces state the consent-gated stance
      status: met
---
# M-0154 — ADR: amend settings.json stance to consent-gated

## Goal

Record the decision to amend aiwf's documented "never edits settings.json"
stance to "never edits **without explicit per-invocation consent**," so the
consent-gated wiring milestone (M-0156) builds on a ratified decision rather
than an ad-hoc relaxation.

## Context

`internal/cli/doctor/doctor.go` (the marketplace-overlap comment) and CLAUDE.md
both state aiwf never edits `settings.json`. Shipping a statusline that can wire
itself in — even with consent — revises that invariant. Per CLAUDE.md's
"Authoring an ADR" rule, the decision is recorded as a choice; *when* to act on
it stays in the planning surface, not in the ADR body.

## Acceptance criteria

<!-- Formal ACs added at start-milestone via `aiwf add ac M-0154`. Intended shape: -->

An ADR exists under `docs/adr/` with `## Context` / `## Decision` /
`## Consequences`, naming the consent mechanism (interactive `[y/N]` on a TTY or
explicit `--wire-settings`) and the `settings.local.json` default for project
scope. This milestone is the **sole owner** of the prose stance amendment: it
updates every surface that today states "aiwf never edits settings.json" — the
CLAUDE.md operator-setup line, the `doctor.go` marketplace-overlap comment, and
the `doctor.go` user-facing "aiwf will not edit your settings.json" string — to
the amended "not without explicit per-invocation consent." (M-0156, the wiring
milestone, does **not** touch this prose.) Mechanical evidence is a structural
assertion scoped to the named ADR sections and to each amended surface — asserted
within its section/string, not via a loose whole-file grep, per CLAUDE.md's
"substring assertions are not structural assertions" rule.

## Constraints

- The ADR body carries **no gate language** ("ratify after X") — decision is
  decision, per CLAUDE.md's ADR-authoring rule.
- Mechanical evidence is a structural section assertion (this milestone is
  `tdd: none` — a doc deliverable, not red-green code).

## Design notes

- Leaning a full ADR (not a lighter `decision` entity) because it revises a
  documented invariant. The FSM `proposed → accepted` via `aiwf promote` is the
  ratification surface; no bespoke status-pinning test.

## Out of scope

- Implementing the wiring (M-0156). This milestone records the decision only.

## Dependencies

- None.

## References

- [E-0039](epic.md) · `internal/cli/doctor/doctor.go` · CLAUDE.md · ADR-0014 (embed precedent)

---

## Work log

### AC-1 — ADR-0015 records the consent-gated settings.json stance

Allocated `docs/adr/ADR-0015-settings-json-edits-require-explicit-per-invocation-consent.md`
with `## Context` / `## Decision` / `## Consequences` sections; the
Decision section names the TTY `[y/N]` confirm, the `--wire-settings`
flag, and the `settings.local.json` project-scope default.
`internal/policies/m0154_stance_test.go::TestM0154_AC1_ADR0015RecordsConsentGatedStance`
pins the structural shape and the three section-scoped mentions. Tests
1/1. Closed in e4fc8c0a.

### AC-2 — CLAUDE.md operator-setup line states the consent-gated stance

CLAUDE.md `## Operator setup` section amended to distinguish the
marketplace-overlap scenario (not consent-eligible) from the statusline
opt-in (consent-eligible via TTY `[y/N]` or `--wire-settings`), with a
cross-reference to ADR-0015.
`TestM0154_AC2_CLAUDEMDOperatorSetupAmended` pins three assertions
scoped to the named section: `.claude/settings.json` preserved,
consent-mechanism phrase present, ADR-0015 cross-referenced. Tests 1/1.
Closed in 0293c165.

### AC-3 — doctor.go stance surfaces state the consent-gated stance

Both stance surfaces in `appendMarketplaceOverlapReport` amended in the
same shape: the doc-comment narrates the consent-gated stance and names
the not-consent-eligible marketplace scenario; the user-facing
`fmt.Sprintf` string carries the same amended language plus the ADR-0015
cross-reference.
`TestM0154_AC3_DoctorGoStanceSurfacesAmended` pins assertions scoped to
the function's doc-comment + body region (via the new
`extractGoFuncWithDocComment` helper). Tests 1/1. Closed in 02d02d88.

## Decisions made during implementation

- (none)

## Validation

- Tests: 3 functions in `internal/policies/m0154_stance_test.go`; all
  pass. Plus the `extractGoFuncWithDocComment` helper, exercised by
  AC-3's `t.Fatal` (function-not-found) guard.
- Full module: `go test ./...` clean; `go test -race ./internal/policies/`
  clean.
- Build: `go build ./cmd/aiwf` clean (amended doctor.go compiles).
- Lint: `golangci-lint run` 0 issues (one gocritic finding caught and
  fixed mid-cycle: `filepath.Join` over a slash-separated path).
- `aiwf check`: 0 errors on M-0154; 14 advisory warnings (pre-existing
  + the `acs-tdd-audit` line which is benign under `tdd: none`).
- Doc-lint: N/A — CLAUDE.md and ADR-0015 are not under `docs/` (well,
  ADR-0015 is under `docs/adr/`, but wf-doc-lint targets narrative
  documentation drift, not new ADR content); no broken links introduced
  by the amendments.

## Deferrals

- (none)

## Reviewer notes

- The AC granularity mirrors M-0153 — one AC per file surface (ADR /
  CLAUDE.md / doctor.go). The doctor.go AC bundles the doc-comment +
  user-facing string into one because they live in the same function
  and serve the same intent; splitting them would produce two near-
  identical regex tests that could drift independently. The single
  `extractGoFuncWithDocComment`-scoped assertion catches both surfaces
  in one structural sweep.
- The consent-mechanism assertion is a three-way OR (`--wire-settings` /
  `[y/N]` / `consent`) so the amendment prose can phrase the consent
  gating in any reasonable way without breaking the test. The
  `ADR-0015` cross-reference assertion is the strict structural marker
  that ties the prose to the ratified decision — that one cannot drift.
- `tdd: none` was deliberate: the deliverable is doc-shaped. Mechanical
  evidence is provided by the three content-assertion tests per
  CLAUDE.md's AC-promotion rule (which applies even under `tdd: none`).
- The new `extractGoFuncWithDocComment` helper handles top-level Go
  functions where the closing brace sits at column 0 (the go-fmt
  convention). It is not a general-purpose Go function extractor;
  scoping it to top-level functions is sufficient for AC-3's needs and
  for any future stance-surface assertions in similar shape.
- ADR-0015 status is `proposed` (the default from `aiwf add adr`). Per
  CLAUDE.md's "Authoring an ADR" rule, the milestone records the
  decision but does not pre-stage ratification. Promoting it to
  `accepted` is a separate, deliberate act for E-0039's later phases
  (when M-0156 actually ships the consent-gated wiring).

### AC-1 — ADR-0015 records the consent-gated settings.json stance

### AC-2 — CLAUDE.md operator-setup line states the consent-gated stance

### AC-3 — doctor.go stance surfaces state the consent-gated stance

