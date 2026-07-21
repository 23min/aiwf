---
id: M-0127
title: Relocate docs/pocv3/ contents and sweep cross-references
status: in_progress
parent: E-0034
depends_on:
    - M-0126
tdd: none
acs:
    - id: AC-1
      title: docs/pocv3/ files relocated/archived per TRIAGE.md
      status: met
    - id: AC-2
      title: Fixture path constants updated for legal-workflows files
      status: met
    - id: AC-3
      title: Zero dangling docs/pocv3 references
      status: met
    - id: AC-4
      title: aiwf check and link integrity clean
      status: met
---

## Goal

Execute the moves recorded in M-0126's `TRIAGE.md` table. Update every cross-reference to `docs/pocv3/` across the repo (~163 files at planning time including markdown, Go source under `internal/`, and embedded skill markdown). At milestone close, `docs/pocv3/` no longer exists, `aiwf check` is clean, and a repo-wide link check is clean.

## Context

**Gated on E-0033 wrap.** E-0033's Pass A (M-0121) audits `docs/pocv3/design/design-decisions.md` and other normative docs as primary citation sources, and writes new tests under `internal/policies/`. The relocate sweep touches the same files. Running concurrently invites silent drift and merge friction; the framework's "correctness must not depend on LLM behavior" rule applies here too.

Full AC body, design notes, and surfaces-touched section drafted at `aiwfx-start-milestone` time. The shape will refine post-Triage when the actual file set + target paths are known.

## Acceptance criteria

### AC-1 — docs/pocv3/ files relocated/archived per TRIAGE.md

Every one of `TRIAGE.md`'s 42 rows is executed: `relocate` rows land at their recorded target path, `archive` rows land under `docs/archive/pocv3/`, and the one `supersede-with-entity` row (`observability-surfaces-plan.md` → G-0433) has its source archived alongside the rest. `docs/pocv3/` contains zero files afterward.

### AC-2 — Fixture path constants updated for legal-workflows files

`internal/policies/m0121_audit_catalog_test.go` and `internal/policies/m0122_first_principles_catalog_test.go` resolve `docs/design/legal-workflows-audit.md` and `docs/design/legal-workflows-first-principles.md` respectively (the `TRIAGE.md`-recorded relocate targets), and both test files pass unmodified in their assertions otherwise.

### AC-3 — Zero dangling docs/pocv3 references

A repo-wide structural test asserts no live Go source (`internal/`, `cmd/`), embedded skill markdown, or top-level doc (`docs/`, `README.md`, `CONTRIBUTING.md`) contains the literal substring `docs/pocv3`, with a narrow, explicit allowlist for deliberately historical mentions (`CHANGELOG.md`; this epic's own `work/` planning-tree prose narrating the migration).

### AC-4 — aiwf check and link integrity clean

`aiwf check` reports no new findings attributable to the sweep, and a repo-wide markdown-link-integrity pass (the `wf-doc-lint` check 5 heuristic, run mechanically) reports zero broken links caused by the move.

## Out of scope

- Re-classifying any file beyond what M-0126's `TRIAGE.md` records. If triage was wrong, file a gap; do not silently revise during the sweep.
- Writing the CLAUDE.md hierarchy section (M-0128's job).
- Landing the drift chokepoint (M-0129's job).

## Dependencies

- M-0126 (Triage) — done. Provides the disposition table this milestone executes.
- E-0033 (Pin legal kernel-verb workflows mechanically) — wrapped. Removes the file-conflict window.

## References

- **E-0034** — parent epic.
- **G-0132** — `aiwf render roadmap --write` blocked by dangling refs in source epic bodies. Worth resolving alongside the sweep if the renderer-canonicalization fix is in scope, since this milestone is sweeping cross-references anyway.

## Work log

### AC-1 — docs/pocv3/ files relocated/archived per TRIAGE.md

All 42 rows executed (`git mv`, plain filesystem operations, not aiwf verbs — `docs/pocv3/` holds no entities); mechanically verified every target exists and every source is gone; `docs/pocv3/` removed entirely · commit 3f6e14a6 · tests 1/1

### AC-2 — Fixture path constants updated for legal-workflows files

`internal/policies/m0121_audit_catalog_test.go` and `m0122_first_principles_catalog_test.go` repointed to `docs/design/`; both files' full test suites pass · commit 3f6e14a6 · tests 2/2

### AC-3 — Zero dangling docs/pocv3 references

Swept ~121 files (Go source, embedded skills, docs/, ADRs, CI config, live planning-tree bodies); added `TestM0127_AC3_NoDanglingDocsPocv3References` under `internal/policies/`, vacuity-checked (injected a stray reference, confirmed the test caught it, reverted) · commit 3f6e14a6 · tests 1/1

### AC-4 — aiwf check and link integrity clean

`aiwf check` reports only the pre-existing G-0434 false positives (now also firing on M-0127, same reused-id pattern as M-0126) and the expected no-upstream advisory — zero findings attributable to the sweep. A repo-wide markdown-link-integrity pass (matching `wf-doc-lint` check 5: fenced/inline-code spans excluded, directory targets treated as valid) reports zero broken links outside `docs/archive/**`, the frozen historical snapshot of the retired tree · commit 3f6e14a6 · tests clean

### Post-review corrections (AC-2, AC-3)

Independent review (dispatched at wrap) requested two changes: restore M-0126's still-valid `AC-2`/`AC-4`/`AC-5` tests, wholesale-deleted when only the `AC-1`/`AC-3` sub-test (which required the now-retired `docs/pocv3/` to exist) was actually obsolete; and widen AC-3's own test scope to `work/` and `CHANGELOG.md`, matching the AC's own spec text — the existing allowlist entries turned out to already be exactly correct, so the widened scan passed unmodified. Also fixed two pre-existing markdown bugs the relocation newly exposed to the `markdown-lint` CI job (an unescaped table-cell pipe and a genuine CommonMark nested-fence bug that was silently leaking half of two worked examples into live document structure), confirmed present at the original pre-migration file, not introduced by this move · commit 0add9d81 · tests: full suite green, `make lint` 0 issues, `npx markdownlint-cli2` 0 issues outside gitignored `.claude/`

## Decisions made during implementation

None rising to an ADR/`D-NNN`-worthy architectural decision. The one scoping call worth recording — narrowing `PolicyDesignDocAnchors` to `docs/design/` (its direct successor) rather than broadening it to all of `docs/`, after the broadened version surfaced unrelated pre-existing content issues in `docs/explorations/surveys/` and `docs/research/` that this milestone has no standing to fix — is captured directly in that function's doc comment and in AC-4's Work log entry above.

## Validation

`make lint` (golangci-lint, full set) — 0 issues. `go build ./...` — green. `go test -count=1 ./...` — green (one `TestWorktreeRitualsCheckHook_NotAWorktreeExitsZeroSilently` "text file busy" flake observed once in a full-suite run, confirmed unrelated to this diff by 3x isolated re-run and by re-running the full suite clean afterward). `make coverage-gate` — diff-scoped branch coverage clean, firing-fixture presence clean, no stale allowlist entries. `aiwf check` — clean except the two pre-existing, already-tracked G-0434 false positives (M-0126, M-0127) and the expected no-upstream advisory. `npx markdownlint-cli2` against the exact CI job config — 0 issues outside the gitignored `.claude/` tree (never seen by CI on a fresh checkout).

Independent review: three fresh-context reviewers dispatched in parallel, sliced by concern (physical-move correctness, Go source changes, cross-reference sweep) per `wf-review-code`'s guidance for large milestones. Physical-move and cross-reference-sweep reviewers both verdicted **approve**, zero blocking findings, after independently re-deriving evidence (filesystem/blob-hash checks, a from-scratch 275-link repo-wide scanner) rather than trusting the Work log. The Go-source reviewer verdicted **request-changes** with two legitimate blocking findings, both fixed in the "Post-review corrections" Work log entry above and confirmed mechanically (re-running the affected tests, `make lint`, `aiwf check`) rather than by a second fresh-reviewer dispatch, since both fixes were concrete and independently re-verifiable rather than judgment calls. `wf-rethink` (design-quality lens) was not run — this milestone introduced no new module/package boundary, core abstraction, or data model, only a file relocation and two narrow policy-scope adjustments.

## Deferrals

- (none) — every AC's scope landed; no residual M-0127 work was punted.

## Reviewer notes

- **G-0074/G-0075/G-0092 remain `open`.** All three gaps are entirely about `docs/pocv3/`'s own retirement and are slated for supersession by this epic (per M-0126's References section) rather than living cross-references this milestone should edit around — deliberately left unswept. Flagged by two independent reviewers as an **epic-wrap obligation**: `aiwfx-wrap-epic E-0034` should promote/supersede these three so the merged tree doesn't carry open gaps naming an absent directory indefinitely. Not an M-0127 defect; tracked here so it isn't lost before the epic wrap.
- **`docs/archive/**` intentionally left with broken internal links.** `docs/archive/pocv3/README.md` and other archived files still link to siblings that used to sit right next to them before relocation (e.g. `overview.md`, `architecture.md`). This is deliberate — the archive is a frozen historical snapshot per ADR-0004, same forget-by-default convention as `CHANGELOG.md` — not an oversight. Independently confirmed by a reviewer (18/18 and 3/3 broken-as-expected on two archived files).
- **Discoverability-channel scan now includes `docs/archive/`** (no archive-dir skip in `readDiscoverabilityChannels`, unlike the narrower `PolicyDesignDocAnchors`). Accepted as a theoretical-only false-negative — a finding code or config field documented *only* in archived content would count as discoverable — since in practice every code/field is also covered by the always-scanned embedded `aiwf-check` SKILL.md.
- **markdown-lint CI job newly lints the relocated top-level docs** (`docs/architecture.md`, `docs/overview.md`, `docs/workflows.md`, `docs/skill-author-guide.md`, `docs/design/*`, `docs/migration/*`) now that they're out from under the retired `docs/pocv3/` exclude. Verified locally against the exact CI glob config (`npx markdownlint-cli2`) — clean; the two pre-existing bugs it surfaced are fixed in this milestone's "Post-review corrections" work.



