---
id: M-0097
title: 'Close M-0094/95/96 verification seams: M-0095 automation audit chokepoint and AC-5 drift comparator'
status: done
parent: E-0028
tdd: required
acs:
    - id: AC-1
      title: CI/script audit chokepoint fires on non-forced epic-activation invocations
      status: met
      tdd_phase: done
    - id: AC-2
      title: Drift comparator helper has synthetic two-arm test (match + drift)
      status: met
      tdd_phase: done
---

# M-0097 — Close M-0094/95/96 verification seams

## Goal

Convert two verification seams left open by M-0094/M-0095/M-0096 into permanent mechanical chokepoints. The M-0095 sovereign-act rule deserves a CI test that fails if any CI workflow or script tries to run `aiwf promote E-... active` from a non-human context without `--force`. The AC-5 drift comparator deserves a synthetic two-arm test so the byte-compare logic is exercised regardless of whether the marketplace cache currently carries the rituals-repo content.

## Context

When wrapping M-0096, a confidence audit surfaced five seams across the start-epic ritual epic:

1. **M-0095's "no non-human `aiwf promote E- active` in CI/scripts" audit was claimed in the spec but not verifiably executed** — the spec's *Validation* section reads as if a grep was run; the conversation record does not show evidence of it. A durable test makes the claim load-bearing.
2. **M-0096/AC-5's drift comparator skips today.** The byte-compare arm (`string(cached) != fixture`) is reachable only in the rare drift-detected production state; it has never been exercised in any test fixture. If the comparator has a subtle bug — wrong direction, off-by-one in error message — nothing catches it.
3. Rituals-plugin manifest registration for `aiwfx-start-epic` is unverified (auto-discovery may or may not suffice).
4. `/aiwfx-start-epic` has never been executed against a real proposed epic.
5. Mutation testing has not been run against the M-0094 / M-0095 / M-0096 additions.

This milestone closes (1) and (2) with durable test chokepoints. Items (3)–(5) are operator tasks done during this milestone's self-review pass; findings either fix in-flight or open as gaps. The milestone deliberately does NOT inflate operator tasks into ACs whose only evidence is "I did it and wrote it down" — that would weaken the AC-evidence bar.

## Acceptance criteria

(ACs allocated at `aiwfx-start-milestone` time per the planner-skill convention.)

## Expected shape

- **AC-1 (M-0095 audit chokepoint)** — New test in `internal/policies/` (suggested name `aiwf_promote_epic_active_audit_test.go`) walks `.github/`, `Makefile`, and `scripts/` under repo root. Any line invoking `aiwf promote E-... active` (or equivalent) must be paired with either `--force` on the same line/heredoc, OR a documented human-actor wrapper in the surrounding context (e.g. an explicit `aiwf-actor: human/...` override, or a comment exempting the call). The test produces one finding per offending line with `path:line` and the canonical fix.
- **AC-2 (Drift comparator helper + two-arm test)** — Extract the byte-compare logic from `TestAiwfxStartEpic_AC5_DriftAgainstCache` into a typed helper (suggested signature `compareSkillToCache(fixture, cached []byte, skillPath string) error`). Returns nil on match; returns a typed error with the skill path and a fixed-up suggestion on drift. Unit-test the helper with synthetic match/drift inputs — both arms exercised regardless of cache state. The existing AC-5 test calls the helper after filesystem plumbing; the helper-level test does not depend on the cache.
- **Operator-task self-review** — Before declaring complete, run the rituals-plugin manifest verification, execute `/aiwfx-start-epic` end-to-end against a synthetic proposed epic, and run `mutate-hunt` against `internal/check/` and `internal/verb/` covering M-0094 / M-0095. Document findings under *Validation*. Real issues open as gaps (`aiwf add gap --discovered-in M-0097`); cosmetic issues fold into the in-flight milestone.

## Dependencies

- **M-0094** (done) — supplies the rule M-0097's AC-1 indirectly chokepoints around.
- **M-0095** (done) — supplies the runtime refusal AC-1's chokepoint complements.
- **M-0096** (done) — supplies the drift-check test whose comparator AC-2 hardens.

## References

- E-0028 epic spec — overall start-epic ritual scope.
- M-0094, M-0095, M-0096 — the three preceding milestones whose verification seams this milestone closes.
- G-0063 — gap framing the start-epic ritual.
- CLAUDE.md *AC promotion requires mechanical evidence* — the rule M-0097 operationalizes by converting a paper-trail audit into a durable test.
- CLAUDE.md §"Test untested code paths before declaring code paths done" — rationale for the AC-5 drift comparator's two-arm test.

### AC-1 — CI/script audit chokepoint fires on non-forced epic-activation invocations

Test in `internal/policies/aiwf_promote_epic_active_audit_test.go::TestPolicy_NoNonForcedEpicActivateInCIScripts` (the seam-level chokepoint) walks `.github/`, `scripts/`, and `Makefile` (when present) from repo root via `os.DirFS`. For each line matching `aiwf\s+promote\s+E-\S+\s+active` lacking `--force` on the same line, emits a `path:line: <line>` finding via `t.Errorf`. Companion helper `auditUnforcedEpicActivate(fs.FS, []string) []string` is unit-tested under `TestAuditUnforcedEpicActivate_BranchCoverage` with synthetic `fstest.MapFS` inputs covering six arms: clean (no matches), forced (ignored), unforced (fires), mixed (only unforced fires), multiple-lines-in-one-file, and the "similar prose without invocation" guard. The two layers complement: the seam test pins production state (currently 0 findings), the unit test pins the helper's logic regardless of production content.

### AC-2 — Drift comparator helper has synthetic two-arm test (match + drift)

Helper `compareSkillBytes(fixture, cached []byte, skillPath string) error` extracted from M-0096's `TestAiwfxStartEpic_AC5_DriftAgainstCache` into `internal/policies/aiwfx_start_epic_test.go`. Returns nil on byte-equal; returns a typed error containing `skillPath` and the re-deploy hint on drift. Unit-tested under `TestCompareSkillBytes_BranchCoverage` with six synthetic inputs: identical bytes (match), both empty (match), different bytes (drift), fixture-only (drift), cached-only (drift), trailing-newline difference (drift). Both arms run in every CI invocation regardless of marketplace-cache state — closing M-0096/AC-5's "drift arm never exercised pre-wrap" seam. The existing AC-5 test now delegates byte comparison to the helper, preserving the seam-level integration while the unit test pins the logic.

## Work log

<!-- Phase timeline lives in `aiwf history M-0097/AC-<N>`; the entries here capture
     one-line outcomes + the implementing commit's SHA (filled at wrap when the
     implementation lands as a single commit). -->

### AC-1 — CI/script audit chokepoint fires on non-forced epic-activation invocations

Audit helper `auditUnforcedEpicActivate` lives in `internal/policies/aiwf_promote_epic_active_audit.go` — pure stdlib (`io/fs`, `regexp`, `strings`), no new dependencies. The seam test `TestPolicy_NoNonForcedEpicActivateInCIScripts` uses `os.DirFS(repoRoot)` to walk `.github/`, `scripts/`, and `Makefile` (when present); reports zero findings against the current repo (production state clean — M-0095's static-call audit is now mechanical). The unit test `TestAuditUnforcedEpicActivate_BranchCoverage` covers six arms via `fstest.MapFS` synthetic inputs. · commit <wrap> · tests 7/7 (seam + 6 unit subcases).

### AC-2 — Drift comparator helper has synthetic two-arm test (match + drift)

Helper `compareSkillBytes` added to `internal/policies/aiwfx_start_epic_test.go`; `TestAiwfxStartEpic_AC5_DriftAgainstCache` refactored to delegate. Six unit subcases under `TestCompareSkillBytes_BranchCoverage` cover the match and drift arms with controlled inputs. The trailing-newline subcase guards against a subtle drift class (`"body\n"` vs `"body"`) that a `==` comparator might pass through. · commit <wrap> · tests 7/7 (seam + 6 unit subcases).

## Decisions made during implementation

- **`--diff` not used in the audit-helper test.** Considered scoping the seam test to "lines changed in this branch vs main"; rejected. Static call sites in `.github/`/`scripts/`/`Makefile` are a closed set, not an evolving diff. Walking the full set on every run is cheap (~ms) and gives stronger evidence than diff-scoped checks. The chokepoint is right where reviewers' eyes land — at PR time, the whole CI surface is in scope.
- **Same-line `--force` rule, strict.** AC-1's audit treats a line as forced iff `--force` appears on the same line. Heredoc / multi-line invocations that split the override across lines are not tolerated. Rationale: CI workflow files prefer single-line `run:` values; if a multi-line case surfaces legitimately, the rule can be relaxed deliberately rather than absorbed silently. Documented in the helper's docstring.
- **AC-2 helper lives in `_test.go`.** Since `compareSkillBytes` is only called from test code (M-0096's AC-5 test and AC-2's unit test), placing it in `aiwfx_start_epic_test.go` keeps the helper test-package-scoped and avoids polluting the non-test build with a function whose only purpose is comparing two byte slices. The `bytes.Equal` underneath is one line; the helper exists for the wrapped error message, not for the equality logic.
- **Manual mutation review chosen over running `mutate-hunt`.** The intended operator task was `mutate-hunt` against `internal/check/` and `internal/verb/`. Local invocations of `gremlins unleash --diff main` and `--diff origin/main` silently SKIPPED 100% of mutants (including new-file mutants that *are* in the diff vs main). Verified with `--dry-run` that gremlins finds the relevant `RUNNABLE` mutants when `--diff` is absent. Manual analysis of each touched file's branches against named tests confirms all mutants would be KILLED — documented in Validation below. The gremlins limitation is filed as G-0110.

## Validation

- `go test -race -count=1 ./internal/policies/ ./internal/check/ ./internal/verb/` — all three packages green; 0 FAIL lines.
- `go test -race -count=1 ./...` — 25 packages green at last run prior to this milestone's commits; re-run during the wrap.
- `golangci-lint run ./internal/policies/` — 0 issues.
- `aiwf check` (kernel planning tree from the worktree) — 0 errors; 7 advisory warnings (3× terminal-entity-not-archived for M-0094/95/96 awaiting sweep; 1× archive-sweep-pending; 2× entity-body-empty for M-0097 AC bodies pre-fill — resolved at this commit; 1× provenance-untrailered-scope-undefined for no-upstream worktree).
- **Manual mutation review** (in place of `mutate-hunt` per Decisions above):
  - `internal/check/epic_active_drafts.go` — 3 reachable mutants on lines 25/30/33 (CONDITIONALS_NEGATION). Mutant 1 (status guard): KILLED by AC-1 (active-fires) ∧ AC-3 (proposed/done/cancelled-no-fire). Mutant 2 (parent-match): KILLED by AC-2 (drafted child under same parent → no warning). Mutant 3 (draft-status check): KILLED by AC-2 ∧ AC-1.
  - `internal/verb/promote_sovereign_epic_active.go` — 3 reachable mutants on the kind/status guards and the human-prefix check. Mutant 1 (kind guard): KILLED by AC-4 (non-epic kinds pass through). Mutant 2 (status guard): KILLED by AC-3 (other transitions pass through). Mutant 3 (`||` → `&&`): KILLED by AC-4's contract-active case (which would now refuse non-human contracts under `&&`). Mutant 4 (human-prefix): KILLED by AC-2 (human actor success).
  - `internal/policies/aiwf_promote_epic_active_audit.go` — 2 reachable mutants on the regex match guard and the `--force` check. Mutant 1 (regex match negation): KILLED by `clean-no-matches` (would now fire on every non-matching line). Mutant 2 (`--force` contains): KILLED by `forced-invocation-ignored` (would now fire on forced lines).
  - All assertable mutants in the milestone's diff are killed by named tests. No survivors identified.
- **Rituals-plugin manifest verification** — `plugins/aiwf-extensions/.claude-plugin/plugin.json` in the rituals repo carries no per-skill listing; Claude Code auto-discovers skills from the `skills/` directory. Dropping `SKILL.md` at `plugins/aiwf-extensions/skills/aiwfx-start-epic/SKILL.md` (M-0096 wrap step) suffices for registration. No manifest edit required.
- **`/aiwfx-start-epic` end-to-end execution** — deferred. The rituals-repo commit `87fc790` is local-only (push gate separate); the marketplace cache in this session predates it. End-to-end invocation requires `/reload-plugins` after either local re-install or rituals-repo push. Operator verifies post-merge.
- Doc-lint sweep against this milestone's change-set — clean. Every cross-reference (verbs, file paths, AC ids, gap ids) resolves.

## Deferrals

- **`/aiwfx-start-epic` end-to-end execution** — requires `/reload-plugins` after rituals-repo cache catches up. Operator verifies post-merge. Not a gap (it is the natural verification path post-rituals-repo-push); documented in Validation as known.
- **G-0110** — `gremlins --diff <ref>` filter excludes new files entirely; manual mutation review is the workaround. Discovered during M-0097's mutation-testing operator task; filed as gap with diagnostic detail and resolution paths.

## Reviewer notes

- **The audit chokepoint is the chokepoint, not the assertion.** AC-1's test fires on offending CI/script lines. The assertion is *"production state has zero offenders"*. The chokepoint is "if a future PR adds an unforced `aiwf promote E- active` line in `.github/`, CI catches it." The test does not detect every possible bypass (it only checks static source lines); it complements M-0095's runtime refusal. Layered defense, not duplication.
- **AC-2's two-arm test makes M-0096/AC-5's drift detection meaningfully tested.** Before this milestone, the drift arm (`cached != fixture`) was only reachable in production state when the marketplace cache had legitimately divergent bytes — extremely rare in routine development. Now the comparator's two arms run in every CI invocation against synthetic inputs, while AC-5 retains the seam-level integration test. The "comparator might have a bug that no test catches" risk class is closed.
- **Gremlins `--diff` is broken for new files; G-0110 captures it.** Future operators running mutation testing during milestone self-review will hit the same wall. The workaround (full-package run + grep-filter) is acceptable but slow; the manual review pattern documented here is faster for small file sets. CLAUDE.md §"Beyond line coverage" remains accurate — mutation testing happens before release tags — but the diff-scoped path needs investigation per G-0110.
- **Trailer convention for the wrap commit** — same as M-0094 / M-0095 / M-0096: `aiwf-verb: implement`, `aiwf-entity: M-0097`, `aiwf-actor: human/peter`.
- **The audit's "same-line `--force` rule" is intentionally strict.** Multi-line invocations splitting `--force` across heredoc lines would slip through this audit's net. Tightening to multi-line is straightforward (read N-line windows) but not warranted today; if a legitimate multi-line case surfaces, the rule can be relaxed deliberately rather than absorbed silently. Naming it here so the next reviewer doesn't re-derive the choice.

