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

Corrective commit from wrap review: `test(archive): assert JSON
envelope content in the new format tests` · commit a347252e ·
strengthened the two archive JSON tests to parse and assert on the
envelope's `status`/`subject` fields (D1 — pin behavior, not just exit
code), per the independent review's finding.

### AC-2 — Scoped coverage-gate reports zero findings

Validation-only, no new commit. Re-ran the scoped
`TestPolicy_BranchCoverageAudit` policy test with
`AIWF_COVERAGE_BASE=2ac84846^` against a full-repo coverage profile
generated after AC-1's tests landed: zero findings across all 10 files
in this milestone's scope.

## Decisions made during implementation

- (none)

## Validation

- `go build ./...` — clean.
- `make lint` (full CI-parity set) — 0 issues.
- `make test-race` (full repo, `-race -parallel 8`) — all packages pass except one
  confirmed pre-existing, unrelated flake (`internal/stresstest`'s
  `TestMidWriteKillScenario_RealBinary_ConfirmsNoHalfWrittenFile`, a timing-sensitive
  mid-write-kill race untouched by this milestone; passes standalone).
- Scoped `TestPolicy_BranchCoverageAudit` with `AIWF_COVERAGE_BASE=2ac84846^` — zero
  findings across all 10 files in this milestone's scope.
- Independent fresh-context code-quality review, split into two parallel dimension
  reviews (doctor+selfcheck+status; the remaining 8 files) — both **approve**. No
  design-quality (`wf-rethink`) pass — this milestone introduces no new module
  boundary, abstraction, or data model, only test files and comment-only production
  edits.
- `wf-doc-lint` (scoped to the milestone's change-set) — clean; the change-set is
  entirely Go source/test files plus this spec, none intersecting the doc-lint scope.

## Deferrals

- (none) — the reviews' one repeat-instance finding (imprecise `ResolveRoot` ignore
  wording, propagated to 5 more sites) is already tracked by the open **G-0412**, not a
  new deferral.

## Reviewer notes

- Both reviewers independently confirmed the `ResolveRoot` ignore-comment wording
  ("only fails on missing aiwf.yaml + non-existent --root path") is imprecise — the
  real failure mode is `filepath.Abs`/`os.Getwd` failing; an explicit `--root` never
  stats the path at all. This is the same issue **G-0412** already tracks repo-wide
  (deliberately deferred as a future sweep, not a per-milestone fix) — one reviewer
  additionally confirmed this exact wording is pervasive (~24 sites) across the
  codebase already, predating this milestone.
- The first reviewer's finding that the two new archive JSON tests asserted only
  `rc`/commit-count (weaker than the sibling show/history additions) was fixed inline
  as a corrective commit (a347252e) before wrap — both now parse and assert on the
  JSON envelope's `status` and `subject` fields.
- The second reviewer initially reported 3 false-positive findings against a coverage
  profile file that was still being written by a concurrently-running background `go
  test` job; re-running against the finalized profile cleared them (confirmed
  non-reproducible). Recorded here so a future reader isn't confused by a stale
  intermediate result — the authoritative re-run is what's cited under Validation.
- `doctor --self-check`'s real in-process Dispatcher wiring (`internal/cli`'s own
  `init()` sets `doctor.Dispatcher = Execute`) was independently verified to have no
  import cycle: `doctor_test` (external test package) importing `internal/cli` is
  legal even though `internal/cli` imports `doctor` (production code) — the two
  `doctor`-named packages are distinct.
- Three `selfcheck.go` step-loop branches (setup/verify/verifyOutput FAIL arms) stay
  `//coverage:ignore`d — reaching them needs corrupting on-disk state a real earlier
  step in the same 29-step sequence produced, not a direct trigger; both reviewers
  confirmed this reachability analysis holds and no signature change (banned by this
  milestone's scope) would avoid it cheaply.
