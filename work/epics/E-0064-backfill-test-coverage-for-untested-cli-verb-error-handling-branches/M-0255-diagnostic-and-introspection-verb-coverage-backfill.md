---
id: M-0255
title: Diagnostic and introspection verb coverage backfill
status: in_progress
parent: E-0064
depends_on:
    - M-0252
tdd: required
acs:
    - id: AC-1
      title: Every diagnostic/introspection-group branch tested or ignored
      status: met
      tdd_phase: done
    - id: AC-2
      title: Scoped coverage-gate reports zero findings
      status: met
      tdd_phase: done
---

## Goal

Clear every branch `branch-coverage-audit` currently flags in the
diagnostic/introspection verb group — `doctor`+`selfcheck`, `status`,
`show`, `history`, `list`, `whoami`, `schema`, `template` — plus
`archive` and `authorize`, using the shared failure fixtures M-0252
builds.

## Context

M-0252 lands the reusable fixtures for the failure modes these guards
share. `doctor/selfcheck.go` carries a large concentration of flagged
sites on its own; the rest of this group is read-oriented verbs with a
thinner failure surface each. `archive` and `authorize` don't fit this
milestone's read-oriented theme — they're mutating verbs — but neither
was assigned to any of E-0064's other four milestones; folding their 10
remaining flagged lines in here (rather than a sixth milestone) keeps
the epic's "zero findings" success criterion reachable without adding a
milestone for two files.

## Acceptance criteria

### AC-1 — Every diagnostic/introspection-group branch tested or ignored

Every branch `branch-coverage-audit` flags (base = the commit before
M-0238/AC-3's rename, `2ac84846^`) within
`internal/cli/{doctor,status,show,history,list,whoami,schema,template,
archive,authorize}` carries either a passing test (reusing M-0252's
fixtures where the failure mode matches) or a
`//coverage:ignore <reason>`.

### AC-2 — Scoped coverage-gate reports zero findings

`make coverage-gate`'s underlying policy test, run with `AIWF_COVERAGE_BASE`
set to the pre-M-0238 commit, reports zero findings for the files listed
in AC-1.

## Constraints

- Reuse M-0252's fixtures for shared failure modes rather than
  reimplementing them per file.
- Per-site judgment only: real test where triggerable, honest
  `//coverage:ignore <reason>` otherwise.

## Out of scope

- Entity-lifecycle, contract, bulk-input, and non-CLI infra files —
  M-0252, M-0253, M-0254, and M-0256's job.
- Any change to error-handling behavior beyond what's needed to make a
  branch testable.

## Dependencies

- M-0252 — its shared fixtures must exist before this milestone starts.

## References

- **E-0064** — parent epic.
- **M-0252** — shared fixtures this milestone consumes.

## Work log

### AC-1 — Every diagnostic/introspection-group branch tested or ignored

~20 new tests plus 33 `//coverage:ignore` annotations, closing all 55
branch-coverage-audit findings across doctor+selfcheck, status,
authorize, archive, show, history, list, schema, template, and whoami
· commit 0d683494 · tests 20/20

`doctor --self-check` wired the real in-process Dispatcher (via
`internal/cli`'s own `init()`) for a genuine full 29-step run rather
than faking it. `show`/`history` each needed one new real-scenario
test (an authorized entity's Scopes section; `[reason:]`/
`[audit-only:]` chips) because the closest existing coverage ran
through a subprocess-compiled binary (`testutil.RunBin`), invisible to
`go test`'s own `-coverprofile` instrumentation. `archive.go`'s 2
findings were real output-format tests (NoOp/dry-run JSON envelopes),
not error guards — zero ignores needed there.

### AC-2 — Scoped coverage-gate reports zero findings

Validation-only, no new commit. Re-ran the scoped
`TestPolicy_BranchCoverageAudit` policy test with
`AIWF_COVERAGE_BASE=2ac84846^` against a full-repo coverage profile
generated after AC-1's tests landed: zero findings across all 10 files
in this milestone's scope.

## Decisions made during implementation

- (none)

## Validation

## Deferrals

## Reviewer notes
