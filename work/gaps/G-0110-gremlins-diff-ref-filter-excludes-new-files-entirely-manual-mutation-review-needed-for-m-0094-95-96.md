---
id: G-0110
title: gremlins --diff <ref> filter excludes new files entirely; manual mutation review needed for M-0094/95/96
status: open
discovered_in: M-0097
---

## What's missing

The `--diff <ref>` flag on `gremlins unleash` (used by `.github/workflows/mutate-hunt.yml` when scoped to a diff target) excludes mutants in files that are *entirely new in the branch*, not just lines unchanged versus the diff target. Observed during M-0097's operator-task self-review: `gremlins unleash --diff main ./internal/check` and `--diff origin/main ./internal/check` both reported 192 SKIPPED mutants and 0 runnable, even though `internal/check/epic_active_drafts.go` is a new file added in this branch (3 mutants per dry-run without `--diff`).

Confirmed: a no-`--diff` dry-run lists the three mutants as `RUNNABLE` (`epic_active_drafts.go:25:16`, `:30:37`, `:33:16`). With `--diff` in either form, the same mutants become `SKIPPED`.

Behavior matches across `--diff main` and `--diff origin/main`, ruling out a missing-remote-ref explanation. Gremlins is `v0.6.0` or similar; the in-repo `mutate-hunt` workflow does not currently pass a diff target (it mutates the full pattern), so this issue does not impair the workflow as configured — it only impairs ad-hoc scoped runs during milestone self-review.

## Why it matters

CLAUDE.md §"Beyond line coverage" prescribes mutation testing "before tagging a release or after a substantive test-suite change." When a milestone adds new logic to a large package, scoping the mutation run to the milestone's diff is the natural way to keep the run fast and the survivor triage relevant. With `--diff` broken for new files, the operator either runs the full package (slow, noisy triage) or skips mutation testing entirely (loses the evidence).

The M-0097 self-review fell back to **manual mutation analysis** on the three affected files (`internal/check/epic_active_drafts.go`, `internal/verb/promote_sovereign_epic_active.go`, `internal/policies/aiwf_promote_epic_active_audit.go`). Each branch was walked against the existing AC tests and found to be KILLED by at least one named test. Documented in M-0097's *Validation* section. This is acceptable evidence for the milestone but does not scale — operators need a working diff-scoped mutation run for future milestones.

## Resolution paths

- Investigate gremlins's `--diff` semantics in source (`github.com/go-gremlins/gremlins`); confirm whether the new-file-skip is a known limitation or a config issue.
- If confirmed upstream, file an issue and reference it in `.github/workflows/mutate-hunt.yml`'s comment block alongside the existing `--workers 1` / `--timeout-coefficient 15` rationale.
- Document the workaround (full-package run + grep-filter the survivors) in CLAUDE.md §"Beyond line coverage" so the next operator does not re-derive it.
